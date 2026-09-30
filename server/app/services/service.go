package services

import (
	"errors"

	"osdtyp/app/core"
	"osdtyp/app/core/anticheat"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// ErrNoLiveSession is returned when a request needs a WebSocket and the caller
// has none.
//
// Everything real-time in the product needs one: there is nowhere to send a
// seed, a countdown or an invitation without it. Saying so plainly is better
// than letting a request half-succeed and leave a player in a lobby they will
// never be sent a game for.
var ErrNoLiveSession = errors.New("open a connection before doing that")

// ServiceLayer is the seam between the HTTP handlers and everything below.
//
// Handlers call it; it calls the database, the session registry and the core
// services. It exists so a handler is a translation of HTTP into a call, with
// no rules of its own, and so a rule is testable without a request.
type ServiceLayer struct {
	db      *postgresql.Database
	logger  *zap.SugaredLogger
	int_gen utils.Generator
	core    *core.CodeCore

	// anticheat judges submitted runs.
	anticheat anticheat.AntiCheat

	// elo is the rating engine, shared with the matchmaker so a private lobby
	// and a ranked queue cannot disagree about how a rating moves.
	elo utils.EloEngine
}

// NewServiceLayer builds the seam.
//
// The core handle is required. The previous version accepted nil and stored
// it, so the first request that touched a session nil-dereferenced instead of
// failing at construction where the mistake was.
func NewServiceLayer(logger *zap.SugaredLogger, core *core.CodeCore, db *postgresql.Database) (ServiceLayer, error) {
	if core == nil {
		return ServiceLayer{}, errors.New("services: a core handle is required")
	}
	if db == nil {
		return ServiceLayer{}, errors.New("services: a database handle is required")
	}
	svc := ServiceLayer{
		logger:  logger,
		db:      db,
		int_gen: utils.NewGenerator(),
		core:    core,
	}
	svc.init()
	return svc, nil
}

// DB exposes the database to the handlers that need a read it does not wrap.
//
// It is deliberately not a general escape hatch: only the reads with no
// business logic behind them come through here.
func (s *ServiceLayer) DB() *postgresql.Database { return s.db }

// Core exposes the core services, for the same reason as DB.
func (s *ServiceLayer) Core() *core.CodeCore { return s.core }

// anticheat judges submitted runs. It is held rather than constructed per
// request so a future rule can carry state, and so its thresholds are
// resolved once.
func (s *ServiceLayer) init() {
	s.anticheat = anticheat.NewAntiCheat(s.logger)
}
