package core

import (
	"context"
	"osdtyp/app/core/bots"
	"osdtyp/app/core/game"
	"osdtyp/app/core/hub"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/core/scheduler"
	"osdtyp/app/core/state"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"

	"go.uber.org/zap"
)

// CodeCore wires the long-lived services that are not HTTP handlers.
type CodeCore struct {
	DB         *postgresql.Database
	Redis      *redis.Client
	Hub        *hub.Hub
	Games      *game.Manager
	Matchmaker *matchmaker.Matchmaker
	State      *state.Store
	Bots       *bots.Manager
	Scheduler  *scheduler.Scheduler

	ctx    context.Context
	cancel context.CancelFunc
}

func NewCodeCore(logger *zap.SugaredLogger, db *postgresql.Database, rdb *redis.Client) *CodeCore {
	ctx, cancel := context.WithCancel(context.Background())
	h := hub.New(logger, rdb)
	games := game.NewManager(logger, h)
	mm := matchmaker.New(logger, rdb, db, h, games)
	st := state.New(logger, rdb, db, games)
	b := bots.New()
	sch := scheduler.New(logger, db)
	return &CodeCore{
		DB:         db,
		Redis:      rdb,
		Hub:        h,
		Games:      games,
		Matchmaker: mm,
		State:      st,
		Bots:       b,
		Scheduler:  sch,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (c *CodeCore) Start() {
	c.Matchmaker.Start(c.ctx)
	c.Scheduler.Start(c.ctx)
}

func (c *CodeCore) Stop() { c.cancel() }
