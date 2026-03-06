package game

import (
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"sync"
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
func (a *ActiveGames) NewGame(players []entity.PlayerItem, duration time.Duration, sig chan []entity.WPMRes) {
	a.logger.Debug("Duration is ", duration)
	gh := NewGameHandler(&a.code_gen, players, a.logger, duration, sig)
	var wg sync.WaitGroup
	for _, player := range gh.Player {
		wg.Add(1)
		go player.PlayerInRoutine(&wg)
	}
	//Dont really wait for it here
	go gh.GlobalBroadcaster()
	//Added a delay just in case one of the ws is slower
	wg.Wait()
	gh.EndLiveStream()

}
