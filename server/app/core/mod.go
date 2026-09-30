package core

import (
	"context"
	"errors"

	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/core/game"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/core/scheduler"
	"osdtyp/app/core/usersession"
	"osdtyp/app/internal/postgresql"

	"go.uber.org/zap"
)

// This package houses all the core backend services that are not exactly "event based" from the internal library

// CodeCore owns every long-lived service in the process.
//
// Every field is a pointer. Sessions, the lobby manager and the scheduler each
// hold a mutex, so storing them by value would copy the lock and leave the copy
// guarding nothing: a start racing a join would then be synchronized by two
// unrelated mutexes.
type CodeCore struct {
	Matchmaker  *matchmaker.Matchmaker
	ManualLobby *controlledlobby.ControlledLobby
	ActiveGames *game.ActiveGames
	Scheduler   *scheduler.Scheduler
	Sessions    *usersession.ActiveSessions
	Database    *postgresql.Database
}

// NewCodeCore wires the process together.
//
// It returns an error rather than panicking on a failed scheduler build,
// because the previous version called os.Exit from inside a constructor: a
// caller in a test or a tool had no way to observe the failure and no way to
// skip the subsystem that could not be built.
func NewCodeCore(logger *zap.SugaredLogger, db *postgresql.Database) (CodeCore, error) {
	if db == nil {
		return CodeCore{}, errors.New("core: a database handle is required")
	}

	games := game.NewActiveGames(logger)
	sessions := usersession.NewActiveSessions(logger)
	lobby := controlledlobby.NewControlledLobby(logger, games, sessions, db)

	sch, err := scheduler.NewScheduler(logger, db, lobby)
	if err != nil {
		return CodeCore{}, err
	}

	return CodeCore{
		ActiveGames: games,
		Matchmaker:  matchmaker.NewMatchMaker(logger, games, sessions, db),
		Sessions:    sessions,
		ManualLobby: lobby,
		Scheduler:   sch,
		Database:    db,
	}, nil
}

// BootCodeCore starts the long-lived workers and blocks forever.
//
// The caller runs this in its own goroutine: the matchmaker worker and the
// scheduler loop are expected to live for the process lifetime, and there is
// nothing after the select for control to reach.
func (c *CodeCore) BootCodeCore() {
	// Matchmaker.Initialize starts the worker goroutine, so calling it more
	// than once would put a second worker on the same queues.
	c.Matchmaker.Initialize()

	go c.Scheduler.StartScheduler()

	select {}
}

// Shutdown stops the workers that have a stop condition.
//
// The session registry is drained too, so a reload does not leave a
// disconnect hook pointing at a dead map. It is not a full graceful shutdown:
// in-flight rounds are abandoned, which is the right trade for a reload.
func (c *CodeCore) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}

	c.Scheduler.Stop()
	c.Matchmaker.Stop()
	c.Sessions.Drain()

	return nil
}
