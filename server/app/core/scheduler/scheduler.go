package scheduler

import (
	"context"
	"encoding/json"
	"time"

	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"

	"go.uber.org/zap"
)

/*
 *
 * We will first of all maintain a var NextJob which basically stores the data for the next task
 * Everytime something is passed thru the backend, its first checked if it can be that
 * If yes, the var is updated, the data is saved in the db nonetheless
 * After the NextJob actually takes place, we check the db for the most recent job
 * If there are multiple, pick them all up in an array and start them at set time
 *
 */

// For Scheduling the lobby and organizing the competition
type Scheduler struct {
	logger   *zap.SugaredLogger
	mro      entity.Task // Most recent object idk
	db       *postgresql.Database
	lobby    *controlledlobby.ControlledLobby
	wakechan chan entity.Task
}

// wakeBuffer is how many pending wakeups can queue before NewTask has to fall
// back to the database. The channel used to be unbuffered, so scheduling a
// contest blocked its HTTP request until the scheduler loop happened to be
// parked in its select.
const wakeBuffer = 64

func (s *Scheduler) StartScheduler() {
	s.Run(context.Background())
}

// Run drives the scheduling loop until ctx is canceled.
//
// The old implementation only ever parked in a `select {}` with no way out, so
// every embedding leaked a goroutine and tests could not isolate themselves
// from a loop that was still draining the queue.
func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Debugln("Scheduler Started")

	// idleWait is how long to sleep when there is nothing scheduled. Nothing is
	// discarded, so waking early or late is harmless.
	const idleWait = time.Hour

	for {
		if ctx.Err() != nil {
			s.logger.Debugln("Scheduler stopped")
			return
		}

		// Read the earliest task. It stays in the queue until it has actually
		// run, so a nudge while waiting cannot lose it.
		task, err := s.db.PeekRecentTask(ctx)
		if err != nil {
			s.logger.Errorw("scheduler could not read the queue", "error", err)
			// Back off rather than spinning on a broken database.
			if !sleep(ctx, time.Second) {
				return
			}
			continue
		}

		if task.JobID == 0 {
			s.logger.Debugln("No tasks in queue")
			timer := time.NewTimer(idleWait)
			select {
			case <-ctx.Done():
				stopTimer(timer)
				return
			case <-s.wakechan:
			case <-timer.C:
			}
			stopTimer(timer)
			continue
		}

		s.logger.Infow("Next task", "job_id", task.JobID, "at", task.Time)

		wait := time.Until(task.Time)
		if wait <= 0 {
			s.runTask(ctx, task)
			continue
		}

		// A dedicated timer rather than time.After, so the timer is released
		// when a nudge arrives instead of lingering until the deadline.
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return
		case <-timer.C:
			s.runTask(ctx, task)
		case <-s.wakechan:
			// Re-read the queue: the newcomer may be due sooner. The task we
			// were waiting on is still stored, so nothing is lost.
		}
		stopTimer(timer)
	}
}

// sleep waits for d, returning false if the context was canceled first.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer stopTimer(timer)
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// runTask removes a task from the queue and hands it to the handler.
func (s *Scheduler) runTask(ctx context.Context, task entity.Task) {
	if err := s.db.DeleteTask(ctx, task.JobID); err != nil {
		s.logger.Errorw("could not remove the finished task", "job_id", task.JobID, "error", err)
	}
	s.TaskHandler(task)
}

// stopTimer releases a timer, draining the channel if it already fired.
func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

// TaskHandler drives one task through its state machine.
//
// A contest is a two step job: the first task creates a lobby and re-queues the
// same contest five minutes later so players have time to join, the second
// starts the game and stores the leaderboard.
func (s *Scheduler) TaskHandler(task entity.Task) {
	if task.Category != entity.CONTEST {
		// The scheduler is deliberately general; other categories are simply
		// added here later on.
		return
	}

	contest, err := s.db.GetContestData(task.JobID)
	if err != nil {
		s.logger.Errorw("could not load the contest for a task", "job_id", task.JobID, "error", err)
		return
	}

	switch contest.Status {
	case entity.UPCOMING:
		s.openLobby(task, contest)
	case entity.LOBBY:
		s.runGame(contest)
	}
}

// openLobby creates the contest's lobby and re-queues the contest so the game
// starts once players have had time to join.
func (s *Scheduler) openLobby(task entity.Task, contest entity.Contest) {
	if s.lobby == nil {
		s.logger.Errorw("no lobby manager is wired up, cannot open a lobby", "job_id", task.JobID)
		return
	}

	contest.LobbyID = s.lobby.CreateNewLobby()
	contest.Status = entity.LOBBY
	if err := s.db.UpdateContest(contest); err != nil {
		s.logger.Errorw("could not record the contest lobby", "job_id", task.JobID, "error", err)
		return
	}

	task.Time = task.Time.Add(5 * time.Minute)
	if err := s.db.NewTask(task); err != nil {
		s.logger.Errorw("could not re-queue the contest", "job_id", task.JobID, "error", err)
	}
}

// runGame starts the contest and waits for the leaderboard.
func (s *Scheduler) runGame(contest entity.Contest) {
	if s.lobby == nil {
		s.logger.Errorw("no lobby manager is wired up, cannot start the game", "job_id", contest.JobID)
		return
	}

	sig := make(chan []entity.WPMRes)
	if err := s.lobby.StartGameFromLobby(contest.LobbyID, contest.Duration.Duration(), sig); err != nil {
		s.logger.Errorw("could not start the contest game", "job_id", contest.JobID, "error", err)
		return
	}

	contest.Status = entity.STARTED
	if err := s.db.UpdateContest(contest); err != nil {
		s.logger.Errorw("could not mark the contest started", "job_id", contest.JobID, "error", err)
		return
	}

	timeout := time.NewTimer(10 * time.Minute)
	defer stopTimer(timeout)

	select {
	case leaderboard := <-sig:
		lbJSON, err := json.Marshal(leaderboard)
		if err != nil {
			s.logger.Errorw("could not encode the leaderboard", "job_id", contest.JobID, "error", err)
			return
		}
		contest.Status = entity.ENDED
		contest.Leaderboard = lbJSON
		if err := s.db.UpdateContest(contest); err != nil {
			s.logger.Errorw("could not store the leaderboard", "job_id", contest.JobID, "error", err)
		}
	case <-timeout.C:
		s.logger.Errorw("contest game timed out", "job_id", contest.JobID)
	}
}

// NewTask records a task and nudges the scheduler loop.
//
// The task is written to the database first so it is durable even if the
// scheduler is busy, then the loop is woken with a non-blocking send. Blocking
// here would stall whichever request triggered the schedule.
func (s *Scheduler) NewTask(task entity.Task) {
	s.logger.Debug("Pushing task")

	if err := s.db.NewTask(task); err != nil {
		s.logger.Errorw("could not persist task", "job_id", task.JobID, "error", err)
		return
	}

	select {
	case s.wakechan <- task:
	default:
		// The loop is already backed up. The row is in the database, so the
		// next poll will pick it up; dropping the nudge only delays it.
		s.logger.Debugw("scheduler busy, task will be picked up on the next poll",
			"job_id", task.JobID)
	}
}

func NewScheduler(logger *zap.SugaredLogger, db *postgresql.Database, ml *controlledlobby.ControlledLobby) (Scheduler, error) {
	return Scheduler{
		logger:   logger,
		db:       db,
		mro:      entity.Task{},
		lobby:    ml,
		wakechan: make(chan entity.Task, wakeBuffer),
	}, nil
}
