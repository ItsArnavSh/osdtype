package game

import (
	"sync/atomic"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// ActiveGames is the entry point for running a round.
//
// The queue of live games used to be a slice that was appended to and never
// read: rounds were launched and ran to completion without being tracked. It
// has been replaced with a counter, because a slice nothing reads is just a
// memory leak with a logging statement.
type ActiveGames struct {
	logger  *zap.SugaredLogger
	codeGen utils.CodeGen
	live    atomic.Int64
}

// NewActiveGames builds the round runner.
func NewActiveGames(logger *zap.SugaredLogger) *ActiveGames {
	return &ActiveGames{
		logger:  logger,
		codeGen: utils.NewCodeGen(logger),
	}
}

// NewGame runs a round and blocks until it finishes.
//
// It blocks for the length of the round, so every caller must put it on its own
// goroutine. The scheduler used to call it inline, which meant one contest
// stalled every other lobby, contest and task for as long as its round ran.
func (a *ActiveGames) NewGame(players []entity.PlayerItem, duration time.Duration, sig chan []entity.WPMRes) {
	if len(players) < 2 {
		a.logger.Warnw("refusing to run a round with fewer than two players", "players", len(players))
		return
	}

	a.live.Add(1)
	defer a.live.Add(-1)

	a.logger.Debugw("starting round", "players", len(players), "duration", duration)

	NewGameHandler(&a.codeGen, players, a.logger, duration, sig).Run()
}

// NewGameAsync runs a round on its own goroutine.
//
// This is the form callers should use. It returns immediately, and the
// leaderboard is delivered on sig when the round finishes.
func (a *ActiveGames) NewGameAsync(players []entity.PlayerItem, duration time.Duration, sig chan []entity.WPMRes) {
	go a.NewGame(players, duration, sig)
}

// Live reports how many rounds are in progress, for the status endpoint.
func (a *ActiveGames) Live() int64 { return a.live.Load() }
