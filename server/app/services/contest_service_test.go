package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"osdtyp/app/entity"
)

func futureStart() time.Time {
	return time.Now().UTC().Add(2 * time.Minute)
}

func TestContestPublicLifecycle(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "contest-alice")
	bob := e.login(t, "contest-bob")
	charlie := e.login(t, "contest-charlie")

	view, err := e.svc.CreateContest(ctx, alice.ID, CreateContestRequest{
		Title:       "Friday Go Sprint",
		Description: "# rules\n\nBe **fast**.",
		Accent:      "#00a19c",
		Visibility:  entity.ContestPublic,
		StartsAt:    futureStart(),
		Duration:    90,
		Language:    "go",
		MaxPlayers:  2,
	})
	if err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if !view.IsHost || !view.IsEnrolled || view.EntryCount != 1 {
		t.Fatalf("host should be auto-enrolled: %+v", view)
	}
	if view.ShareLink == "" || view.Status != entity.ContestUpcoming {
		t.Fatalf("incomplete contest view: %+v", view)
	}

	list, err := e.svc.ListContests(ctx, 0)
	if err != nil {
		t.Fatalf("public list: %v", err)
	}
	if len(list.Contests) != 1 || list.Contests[0].Code != view.Code {
		t.Fatalf("public contest missing from board: %+v", list)
	}
	if len(list.Trending) != 1 || len(list.Mine) != 0 {
		t.Fatalf("guest list shape: %+v", list)
	}
	own, err := e.svc.ListContests(ctx, alice.ID)
	if err != nil {
		t.Fatalf("own list: %v", err)
	}
	if len(own.Mine) != 1 || !own.Mine[0].IsHost {
		t.Fatalf("hosted contest missing from mine: %+v", own.Mine)
	}

	joined, err := e.svc.EnrollContest(ctx, bob.ID, view.Code, "")
	if err != nil {
		t.Fatalf("enroll bob: %v", err)
	}
	if !joined.IsEnrolled || joined.EntryCount != 2 {
		t.Fatalf("bob not enrolled: %+v", joined)
	}
	again, err := e.svc.EnrollContest(ctx, bob.ID, view.Code, "")
	if err != nil || again.EntryCount != 2 {
		t.Fatalf("duplicate enroll should be idempotent: %+v %v", again, err)
	}
	if _, err := e.svc.EnrollContest(ctx, charlie.ID, view.Code, ""); !errors.Is(err, ErrFull) {
		t.Fatalf("expected full contest, got %v", err)
	}
	if _, err := e.svc.UpdateContest(ctx, bob.ID, view.Code, UpdateContestRequest{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-host should not edit: %v", err)
	}
	left, err := e.svc.LeaveContest(ctx, bob.ID, view.Code)
	if err != nil {
		t.Fatalf("leave contest: %v", err)
	}
	if left.IsEnrolled || left.EntryCount != 1 {
		t.Fatalf("bob still enrolled: %+v", left)
	}

	cancelled, err := e.svc.CancelContest(ctx, alice.ID, view.Code)
	if err != nil {
		t.Fatalf("cancel contest: %v", err)
	}
	if cancelled.Status != entity.ContestCancelled {
		t.Fatalf("contest not cancelled: %+v", cancelled)
	}
	after, err := e.svc.ListContests(ctx, 0)
	if err != nil {
		t.Fatalf("list after cancel: %v", err)
	}
	if len(after.Contests) != 0 {
		t.Fatalf("cancelled contest still public: %+v", after.Contests)
	}
}

func TestContestPrivatePasswordAndVisibility(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "private-alice")
	bob := e.login(t, "private-bob")

	view, err := e.svc.CreateContest(ctx, alice.ID, CreateContestRequest{
		Title:       "Invite only",
		Description: "secret route",
		Visibility:  entity.ContestPrivate,
		Password:    "open-sesame",
		StartsAt:    futureStart(),
		Duration:    30,
		Language:    "rust",
		MaxPlayers:  4,
	})
	if err != nil {
		t.Fatalf("create private contest: %v", err)
	}
	if !view.RequiresPassword {
		t.Fatalf("private contest should require password: %+v", view)
	}
	public, err := e.svc.ListContests(ctx, bob.ID)
	if err != nil {
		t.Fatalf("public list: %v", err)
	}
	if len(public.Contests) != 0 || len(public.Trending) != 0 {
		t.Fatalf("private contest leaked into public listings: %+v", public)
	}
	detail, err := e.svc.GetContestByCode(ctx, bob.ID, view.Code)
	if err != nil {
		t.Fatalf("private detail by link: %v", err)
	}
	if detail.Participants != nil {
		t.Fatalf("private roster leaked before enrollment: %+v", detail.Participants)
	}
	if _, err := e.svc.EnrollContest(ctx, bob.ID, view.Code, ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected password required, got %v", err)
	}
	if _, err := e.svc.EnrollContest(ctx, bob.ID, view.Code, "wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected wrong password, got %v", err)
	}
	joined, err := e.svc.EnrollContest(ctx, bob.ID, view.Code, "open-sesame")
	if err != nil {
		t.Fatalf("enroll with password: %v", err)
	}
	if !joined.IsEnrolled || len(joined.Participants) != 2 {
		t.Fatalf("roster not visible after enrollment: %+v", joined)
	}
	if _, err := e.svc.CreateContest(ctx, alice.ID, CreateContestRequest{
		Title:      "bad public password",
		Visibility: entity.ContestPublic,
		Password:   "nope",
		StartsAt:   futureStart(),
		Duration:   30,
		Language:   "go",
		MaxPlayers: 4,
	}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("public contest accepted a password: %v", err)
	}
}

func TestContestSchedulerStartsAndCancels(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "scheduler-alice")
	bob := e.login(t, "scheduler-bob")

	makeContest := func(title string) ContestView {
		t.Helper()
		view, err := e.svc.CreateContest(ctx, alice.ID, CreateContestRequest{
			Title:      title,
			Visibility: entity.ContestPublic,
			StartsAt:   futureStart(),
			Duration:   30,
			Language:   "go",
			MaxPlayers: 4,
		})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		return view
	}

	// Two humans: the scheduler should launch a hidden contest game and set
	// both players' state to that game.
	launch := makeContest("launch me")
	if _, err := e.svc.EnrollContest(ctx, bob.ID, launch.Code, ""); err != nil {
		t.Fatalf("enroll bob: %v", err)
	}
	stored, err := e.db.GetContest(ctx, launch.ID)
	if err != nil {
		t.Fatalf("load launch contest: %v", err)
	}
	stored.StartsAt = time.Now().UTC().Add(-time.Second)
	if err := e.db.SaveContest(ctx, &stored); err != nil {
		t.Fatalf("move contest into the past: %v", err)
	}
	if err := e.svc.HandleContestTask(ctx, entity.Task{Category: entity.CONTEST, JobID: launch.ID}); err != nil {
		t.Fatalf("handle launch task: %v", err)
	}
	started, err := e.db.GetContest(ctx, launch.ID)
	if err != nil {
		t.Fatalf("load started contest: %v", err)
	}
	if started.Status != entity.ContestStarted || started.GameID == 0 || started.RoomID == 0 {
		t.Fatalf("contest did not start: %+v", started)
	}
	if st := e.state(t, bob.ID); st.State != "game" || st.Game == nil || st.Game.ID != started.GameID {
		t.Fatalf("enrolled player was not routed to the contest game: %+v", st)
	}
	if st := e.state(t, alice.ID); st.State != "game" {
		t.Fatalf("host was not routed to the contest game: %+v", st)
	}

	// One human: the contest must cancel instead of starting an empty room.
	cancel := makeContest("cancel me")
	stored, err = e.db.GetContest(ctx, cancel.ID)
	if err != nil {
		t.Fatalf("load cancel contest: %v", err)
	}
	stored.StartsAt = time.Now().UTC().Add(-time.Second)
	if err := e.db.SaveContest(ctx, &stored); err != nil {
		t.Fatalf("move cancel contest into the past: %v", err)
	}
	if err := e.svc.HandleContestTask(ctx, entity.Task{Category: entity.CONTEST, JobID: cancel.ID}); err != nil {
		t.Fatalf("handle cancel task: %v", err)
	}
	cancelled, err := e.db.GetContest(ctx, cancel.ID)
	if err != nil {
		t.Fatalf("load cancelled contest: %v", err)
	}
	if cancelled.Status != entity.ContestCancelled || cancelled.GameID != 0 {
		t.Fatalf("single-player contest should cancel: %+v", cancelled)
	}
}
