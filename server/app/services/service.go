package services

import (
	"context"
	"osdtyp/app/core"
	"osdtyp/app/core/bots"
	"osdtyp/app/core/game"
	"osdtyp/app/core/hub"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/core/state"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"
	"osdtyp/app/utils"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type ServiceLayer struct {
	logger  *zap.SugaredLogger
	db      *postgresql.Database
	core    *core.CodeCore
	intGen  *utils.Generator
	codegen utils.CodeGen

	frontendURL  string
	lobbyMinutes int
	countdown    time.Duration
}

func NewServiceLayer(logger *zap.SugaredLogger, c *core.CodeCore, db *postgresql.Database) (*ServiceLayer, error) {
	gen := utils.NewGenerator()
	s := &ServiceLayer{
		logger:       logger,
		db:           db,
		core:         c,
		intGen:       &gen,
		codegen:      utils.NewCodeGen(logger),
		frontendURL:  viper.GetString("Core.frontend_url"),
		lobbyMinutes: viper.GetInt("Contests.lobby_minutes"),
		countdown:    time.Duration(viper.GetInt("Game.countdown_ms")) * time.Millisecond,
	}
	if s.frontendURL == "" {
		s.frontendURL = "http://localhost:5173"
	}
	if s.lobbyMinutes <= 0 {
		s.lobbyMinutes = 2
	}
	if s.countdown <= 0 {
		s.countdown = 3 * time.Second
	}

	c.Games.SetFinishHook(s.onGameFinished)
	c.Matchmaker.SetCreator(s)
	c.Scheduler.SetController(s)
	c.Hub.SetHandlers(s.onSocketMessage, s.onSocketConnect, s.onSocketDisconnect)
	return s, nil
}

func (s *ServiceLayer) g() *game.Manager           { return s.core.Games }
func (s *ServiceLayer) hub() *hub.Hub              { return s.core.Hub }
func (s *ServiceLayer) mm() *matchmaker.Matchmaker { return s.core.Matchmaker }
func (s *ServiceLayer) state() *state.Store        { return s.core.State }
func (s *ServiceLayer) rdb() *redis.Client         { return s.core.Redis }
func (s *ServiceLayer) bots() *bots.Manager        { return s.core.Bots }

// Bootstrap creates the reserved bot accounts. It is safe to call on every
// boot because usernames are unique.
func (s *ServiceLayer) Bootstrap(ctx context.Context) error {
	for _, persona := range s.bots().All() {
		cfg, _ := s.bots().Config(persona)
		name := bots.Username(persona)
		if _, err := s.db.GetUserFromName(ctx, name); err == nil {
			continue
		}
		user := s.newUser(name, cfg.Rating, true)
		if err := s.db.AddUser(ctx, user); err != nil {
			s.logger.Warnw("could not seed bot", "persona", persona, "error", err)
		}
	}
	return nil
}
