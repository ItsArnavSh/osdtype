package core

import (
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
// Sessions and ManualLobby are held as pointers: both embed a mutex, so
// storing them by value would copy the lock and leave the copy guarding
// nothing.
type CodeCore struct {
	Matchmaker  matchmaker.Matchmaker
	ManualLobby *controlledlobby.ControlledLobby
	ActiveGames game.ActiveGames
	Scheduler   scheduler.Scheduler
	Sessions    *usersession.ActiveSessions
	Database    *postgresql.Database
}

func NewCodeCore(logger *zap.SugaredLogger, db *postgresql.Database) (CodeCore, error) {
	games := game.NewActiveGames(logger)
	session := usersession.NewActiveSessions(logger)
	ml := controlledlobby.NewControlledLobby(logger, &games, &session)
	sch, err := scheduler.NewScheduler(logger, db, &ml)
	if err != nil {
		return CodeCore{}, err
	}
	// Database is kept on the struct so subsystems that were not handed it
	// directly can still reach it.
	return CodeCore{
		ActiveGames: games,
		Matchmaker:  matchmaker.NewMatchMaker(nil, logger, &games, &session, db),
		Sessions:    &session,
		ManualLobby: &ml,
		Scheduler:   sch,
		Database:    db,
	}, nil
}
func (c *CodeCore) BootCodeCore() {
	{ // Matchmaker stuff
		// Initialize starts the matchmaker's worker goroutine, so calling it
		// more than once would spawn a second worker on the same queues.
		c.Matchmaker.Initialize()
	}
	{ // Scheduler stuff
		go c.Scheduler.StartScheduler()
	}
	// Park forever: the caller runs this in its own goroutine and the scheduler
	// and matchmaker workers are expected to live for the process lifetime.
	select {}
}
