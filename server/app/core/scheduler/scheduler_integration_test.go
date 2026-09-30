//go:build integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/testconfig"
)

func newScheduler(t *testing.T) (*Scheduler, postgresql.Database) {
	t.Helper()
	testconfig.Configure(t)

	db, err := postgresql.ConnectDatabase(zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("could not connect to the test database: %v", err)
	}

	// Start from an empty queue.
	if err := db.RawDB().Where("1 = 1").Delete(&entity.Task{}).Error; err != nil {
		t.Fatalf("could not clear the task table: %v", err)
	}

	// A nil lobby is fine for the paths exercised here: no task is ever due, and
	// the handler now reports a missing lobby instead of dereferencing it.
	sch, err := NewScheduler(zap.NewNop().Sugar(), &db, nil)
	if err != nil {
		t.Fatalf("could not build the scheduler: %v", err)
	}
	return sch, db
}

// TestNewTaskDoesNotBlock is a regression test.
//
// NewTask sent on an unbuffered channel, so scheduling a contest blocked until
// the scheduler loop happened to be parked in its select. With the loop not
// running at all, which is the case in tests and in any embedding that has not
// booted the core yet, the send never returned and the caller's request hung
// forever.
func TestNewTaskDoesNotBlock(t *testing.T) {
	sch, db := newScheduler(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		sch.NewTask(entity.Task{Category: entity.CONTEST, JobID: 5001, Time: time.Now().Add(time.Hour)})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("NewTask blocked, the caller would hang")
	}

	// The task must be durable regardless of whether a loop is running.
	if _, err := db.PeekRecentTask(context.Background()); err != nil {
		t.Fatalf("could not read the queue: %v", err)
	}
	var count int64
	if err := db.RawDB().Model(&entity.Task{}).Where("job_id = ?", 5001).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the task to be persisted, found %d rows", count)
	}
}

// TestNewTaskSurvivesAFullWakeBuffer covers the overflow path: the nudge is
// dropped but the row is already in the database.
func TestNewTaskSurvivesAFullWakeBuffer(t *testing.T) {
	sch, db := newScheduler(t)

	// Fill the wake buffer without any loop consuming it.
	total := wakeBuffer + 20
	for i := 0; i < total; i++ {
		sch.NewTask(entity.Task{
			Category: entity.CONTEST,
			JobID:    uint32(6000 + i),
			Time:     time.Now().Add(time.Hour),
		})
	}

	var count int64
	if err := db.RawDB().Model(&entity.Task{}).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != int64(total) {
		t.Fatalf("expected all %d tasks to be persisted, found %d", total, count)
	}
}

// TestPeekDoesNotRemove is a regression test.
//
// The scheduler used to pop-and-delete the earliest task and then wait for it.
// Any nudge arriving during that wait threw the popped task away, so it never
// ran. Peeking keeps the row until the task has actually been handled.
func TestPeekDoesNotRemove(t *testing.T) {
	sch, db := newScheduler(t)
	sch.NewTask(entity.Task{Category: entity.CONTEST, JobID: 7001, Time: time.Now().Add(time.Hour)})
	ctx := context.Background()

	if _, err := db.PeekRecentTask(ctx); err != nil {
		t.Fatalf("peek failed: %v", err)
	}

	var count int64
	if err := db.RawDB().Model(&entity.Task{}).Where("job_id = ?", 7001).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != 1 {
		t.Fatalf("peek removed the task, %d rows remain", count)
	}
}

func TestPeekReturnsTheEarliestTask(t *testing.T) {
	sch, db := newScheduler(t)
	now := time.Now()

	sch.NewTask(entity.Task{JobID: 8002, Time: now.Add(2 * time.Hour)})
	sch.NewTask(entity.Task{JobID: 8001, Time: now})

	task, err := db.PeekRecentTask(context.Background())
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if task.JobID != 8001 {
		t.Errorf("expected the earlier task 8001, got %d", task.JobID)
	}
}

func TestPeekOnAnEmptyQueue(t *testing.T) {
	_, db := newScheduler(t)

	task, err := db.PeekRecentTask(context.Background())
	if err != nil {
		t.Fatalf("an empty queue should not be an error, got %v", err)
	}
	if task.JobID != 0 {
		t.Errorf("expected the zero task, got %+v", task)
	}
}

func TestDeleteTask(t *testing.T) {
	sch, db := newScheduler(t)
	ctx := context.Background()
	sch.NewTask(entity.Task{JobID: 9001, Time: time.Now()})

	if err := db.DeleteTask(ctx, 9001); err != nil {
		t.Fatalf("could not delete: %v", err)
	}

	var count int64
	if err := db.RawDB().Model(&entity.Task{}).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected the queue to be empty, %d rows remain", count)
	}
}

// TestStartSchedulerRunsADueTask drives the real loop end to end.
func TestStartSchedulerRunsADueTask(t *testing.T) {
	sch, db := newScheduler(t)

	// Already due, so the loop should pick it up immediately. The category is
	// CONTEST with a job id that has no contest behind it, so TaskHandler
	// returns early; the point is that the task leaves the queue.
	sch.NewTask(entity.Task{Category: entity.CONTEST, JobID: 9101, Time: time.Now().Add(-time.Minute)})

	// The loop is tied to a context so it can be stopped again. Left running it
	// would keep polling and consuming rows out from under the other tests.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := db.RawDB().Model(&entity.Task{}).Where("job_id = ?", 9101).Count(&count).Error; err == nil && count == 0 {
			cancel()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("the scheduler never consumed the due task")
}

// TestRunStopsOnContextCancel checks the loop can actually be shut down, which
// is what lets the rest of the suite trust that no scheduler is still running.
func TestRunStopsOnContextCancel(t *testing.T) {
	sch, _ := newScheduler(t)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		sch.Run(ctx)
	}()

	// Let it get as far as parking on the empty queue, then stop it.
	time.Sleep(250 * time.Millisecond)
	cancel()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Run ignored the canceled context and kept going")
	}
}
