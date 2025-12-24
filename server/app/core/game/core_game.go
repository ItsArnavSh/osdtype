package game

import (
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"time"

	"go.uber.org/zap"
)

type ActiveGames struct {
	logger   *zap.SugaredLogger
	running  []GameHandler
	code_gen utils.CodeGen
}

func NewActiveGames(logger *zap.SugaredLogger) ActiveGames {
	return ActiveGames{
		logger:   logger,
		running:  nil,
		code_gen: utils.NewCodeGen(logger),
	}

}
func (a *ActiveGames) NewGame(players []entity.PlayerItem, duration time.Duration, sig chan struct{}) {
	a.logger.Debug("Duration is ", duration)
	gh := NewGameHandler(&a.code_gen, players, a.logger, duration)
	go gh.GlobalBroadcaster()
	//Added a delay just in case one of the ws is slower
	time.AfterFunc(duration+500*time.Millisecond, gh.EndLiveStream)
	sig <- struct{}{}

}
