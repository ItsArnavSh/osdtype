package scheduler

import (
	"context"
	"encoding/json"
	"time"

	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

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

	// cancel ends the scheduling loop. StartScheduler derives its own context
	// from it, so Stop ends the loop without the caller having to keep the
	// context it passed in.
	cancel context.CancelFunc
}

// wakeBuffer is how many pending wakeups can queue before NewTask has to fall
// back to the database. The channel used to be unbuffered, so scheduling a
// contest blocked its HTTP request until the scheduler loop happened to be
// parked in its select.
const wakeBuffer = 64

// StartScheduler runs the loop on a context the scheduler owns, so Stop can
// end it without the caller holding a cancel function.
func (s *Scheduler) StartScheduler() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.Run(ctx)
}

// Stop ends the scheduling loop and waits briefly for it to unwind.
//
// The loop is blocked in a select on a task that may be hours away, so Stop
// relies on the cancel rather than a nudge down the wake channel: a nudge
// would only make it re-read the queue and park again.
func (s *Scheduler) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
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
	s.TaskHandler(ctx, task)
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
func (s *Scheduler) TaskHandler(ctx context.Context, task entity.Task) {
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
		s.openLobby(ctx, task, contest)
	case entity.LOBBY:
		s.runGame(ctx, contest)
	}
}

// openLobby creates the contest's room and re-queues the contest so the game
// starts once players have had time to join.
//
// A contest's lobby is an ordinary room. It used to be an integer minted by
// the in-memory lobby map, which meant a contest lobby could not be joined by
// anyone who was not already in the process's memory, and its roster could not
// be listed by anything.
func (s *Scheduler) openLobby(ctx context.Context, task entity.Task, contest entity.Contest) {
	if s.lobby == nil {
		s.logger.Errorw("no lobby manager is wired up, cannot open a lobby", "job_id", task.JobID)
		return
	}

	room, err := s.lobby.CreateGame(ctx, entity.Room{
		Name:      contest.Title(),
		Desc:      contest.Data,
		Public:    entity.PRIVATE,
		Capacity:  contestCapacity,
		Duration:  contest.Duration,
		CreatorID: contest.HostID,
	})
	if err != nil {
		s.logger.Errorw("could not create the contest lobby", "job_id", task.JobID, "error", err)
		return
	}

	contest.LobbyID = room.ID
	contest.Status = entity.LOBBY
	if err := s.db.UpdateContest(contest); err != nil {
		s.logger.Errorw("could not record the contest lobby", "job_id", task.JobID, "error", err)
		return
	}

	task.Time = task.Time.Add(lobbyOpenFor)
	if err := s.db.NewTask(task); err != nil {
		s.logger.Errorw("could not re-queue the contest", "job_id", task.JobID, "error", err)
	}
}

// runGame starts the contest and stores the leaderboard.
//
// It launches the round on its own goroutine and then waits on the result, so
// the scheduling loop is free to keep working while a contest plays out. It
// used to call the round runner inline, which meant a single contest stalled
// every other lobby, contest and task for the length of its round.
func (s *Scheduler) runGame(ctx context.Context, contest entity.Contest) {
	if s.lobby == nil {
		s.logger.Errorw("no lobby manager is wired up, cannot start the game", "job_id", contest.JobID)
		return
	}

	sig := make(chan []entity.WPMRes, 1)

	// Warn the roster before starting, so members who are idle in a tab get a
	// countdown rather than silently missing the contest.
	s.lobby.WarnContest(ctx, contest.LobbyID, contest.JobID, int(contestDurationWarning.Seconds()))

	if err := s.lobby.StartAsync(ctx, contest.LobbyID, contest.Duration.Duration(), sig); err != nil {
		s.logger.Errorw("could not start the contest game", "job_id", contest.JobID, "error", err)
		return
	}

	contest.Status = entity.STARTED
	if err := s.db.UpdateContest(contest); err != nil {
		s.logger.Errorw("could not mark the contest started", "job_id", contest.JobID, "error", err)
		return
	}

	timeout := time.NewTimer(contestRoundTimeout)
	defer stopTimer(timeout)

	select {
	case leaderboard := <-sig:
		if err := s.finishContest(contest, leaderboard); err != nil {
			s.logger.Errorw("could not store the leaderboard", "job_id", contest.JobID, "error", err)
		}
	case <-ctx.Done():
		s.logger.Infow("contest abandoned with the scheduler", "job_id", contest.JobID)
	case <-timeout.C:
		s.logger.Errorw("contest game timed out", "job_id", contest.JobID)
	}
}

// finishContest stores a finished contest's leaderboard and its per-player
// runs, so a contest result is visible in a player's history afterwards.
func (s *Scheduler) finishContest(contest entity.Contest, leaderboard []entity.WPMRes) error {
	lbJSON, err := json.Marshal(leaderboard)
	if err != nil {
		return err
	}

	contest.Status = entity.ENDED
	contest.Leaderboard = lbJSON
	if err := s.db.UpdateContest(contest); err != nil {
		return err
	}

	now := time.Now()
	for i, res := range leaderboard {
		run := entity.Run{
			ID:         taskIDs.GenerateID(),
			UserID:     res.ID,
			Mode:       contest.Duration,
			Solo:       false,
			WPM:        res.WPM,
			Raw:        res.RAW,
			Accuracy:   res.Accuracy,
			Correct:    res.Correct,
			Wrong:      res.Wrong,
			DurationMS: contest.Duration.Duration().Milliseconds(),
			RankBefore: res.RatingBefore,
			RankAfter:  res.RatingAfter,
			Language:   contest.Lang,
			Passed:     true,
			PlayedAt:   now.Add(time.Duration(i) * time.Second),
		}
		if _, err := s.db.SaveRun(context.Background(), run); err != nil {
			s.logger.Errorw("could not store a contest run", "job_id", contest.JobID, "user_id", res.ID, "error", err)
		}
	}

	s.logger.Infow("contest finished", "job_id", contest.JobID, "players", len(leaderboard))
	return nil
}

// Contest scheduling parameters.
const (
	// lobbyOpenFor is how long a contest's lobby accepts players between
	// opening and starting.
	lobbyOpenFor = 5 * time.Minute
	// contestRoundTimeout is how long to wait for a contest round to report a
	// leaderboard before giving up on it.
	contestRoundTimeout = 10 * time.Minute
	// contestCapacity is how many players a contest room holds.
	contestCapacity = 6
	// contestDurationWarning is the notice given to members before a contest
	// starts.
	contestDurationWarning = 30 * time.Second
)

// taskIDs mints ids for rows the scheduler writes.
var taskIDs = utils.NewGenerator()

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

func NewScheduler(logger *zap.SugaredLogger, db *postgresql.Database, ml *controlledlobby.ControlledLobby) (*Scheduler, error) {
	return &Scheduler{
		logger:   logger,
		db:       db,
		mro:      entity.Task{},
		lobby:    ml,
		wakechan: make(chan entity.Task, wakeBuffer),
	}, nil
}
