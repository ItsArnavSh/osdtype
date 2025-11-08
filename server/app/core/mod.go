package core

import (
	"osdtyp/app/core/game"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/core/usersession"

	"go.uber.org/zap"
)

//This package houses all the core backend services that are not exactly "event based" from the internal library

type CodeCore struct {
	Matchmaker *matchmaker.Matchmaker

	ActiveGames game.ActiveGames
	Sessions    usersession.ActiveSessions
}

func NewCodeCore(logger *zap.SugaredLogger) CodeCore {
	games := game.NewActiveGames(logger)
	return CodeCore{
		ActiveGames: games,
		Matchmaker:  matchmaker.NewMatchMaker(nil, logger, &games),
		Sessions:    usersession.NewActiveSessions(),
	}
}
func (c *CodeCore) BootCodeCore() {
	go c.Matchmaker.BackgroundMatchmaker()
}
