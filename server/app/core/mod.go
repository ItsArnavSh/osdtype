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

//This package houses all the core backend services that are not exactly "event based" from the internal library

type CodeCore struct {
	Matchmaker  matchmaker.Matchmaker
	ManualLobby controlledlobby.ControlledLobby
	ActiveGames game.ActiveGames
	Scheduler   scheduler.Scheduler
	Sessions    usersession.ActiveSessions
	Database    *postgresql.Database
}

func NewCodeCore(logger *zap.SugaredLogger, db *postgresql.Database) CodeCore {
	games := game.NewActiveGames(logger)
	session := usersession.NewActiveSessions()
	ml := controlledlobby.NewControlledLobby(logger, &games, &session)
	sch, err := scheduler.NewScheduler(logger, db, &ml)
	if err != nil {
		return CodeCore{}
	}
	return CodeCore{
		ActiveGames: games,
		Matchmaker:  matchmaker.NewMatchMaker(nil, logger, &games, &session),
		Sessions:    session,
		ManualLobby: ml,
		Scheduler:   sch,
	}
}
func (c *CodeCore) BootCodeCore() {
	{ //Matchmaker stuff
		c.Matchmaker.Initialize()
		go c.Matchmaker.BackgroundMatchmaker()
	}
	{ //Scheduler stuff
		go c.Scheduler.StartScheduler()
	}

}
