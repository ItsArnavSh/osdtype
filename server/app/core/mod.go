package core

import (
	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/core/game"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/core/usersession"

	"go.uber.org/zap"
)

//This package houses all the core backend services that are not exactly "event based" from the internal library

type CodeCore struct {
	Matchmaker  matchmaker.Matchmaker
	ManualLobby controlledlobby.ControlledLobby
	ActiveGames game.ActiveGames
	Sessions    usersession.ActiveSessions
}

func NewCodeCore(logger *zap.SugaredLogger) CodeCore {
	games := game.NewActiveGames(logger)
	session := usersession.NewActiveSessions()
	return CodeCore{
		ActiveGames: games,
		Matchmaker:  matchmaker.NewMatchMaker(nil, logger, &games, &session),
		Sessions:    session,
		ManualLobby: controlledlobby.NewControlledLobby(logger, &games, &session),
	}
}
func (c *CodeCore) BootCodeCore() {
	{ //Matchmaker stuff
		c.Matchmaker.Initialize()
		go c.Matchmaker.BackgroundMatchmaker()
	}

}
