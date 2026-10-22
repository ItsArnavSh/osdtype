package game

import (
	"context"
	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FinishHook persists the finished match, applies Elo and returns the final
// leaderboard (including rating deltas).
type FinishHook func(ctx context.Context, game entity.Game, players []entity.GamePlayer, recordings []entity.Recording, flags []entity.CheatFlag) ([]entity.ResultView, error)

type StartOptions struct {
	Game    entity.Game
	Players []entity.GamePlayer
	Users   map[uint32]entity.User
	Bots    map[uint32]entity.BotConfig
}

type Manager struct {
	logger   *zap.SugaredLogger
	hub      *hub.Hub
	onFinish FinishHook

	mu       sync.RWMutex
	games    map[uint32]*runtime
	userGame map[uint32]uint32
}

func NewManager(logger *zap.SugaredLogger, h *hub.Hub) *Manager {
	return &Manager{
		logger:   logger,
		hub:      h,
		games:    make(map[uint32]*runtime),
		userGame: make(map[uint32]uint32),
	}
}

func (m *Manager) SetFinishHook(hook FinishHook) { m.onFinish = hook }

// Start registers a game and begins its clock. The caller is responsible for
// persisting the game and setting StartsAt/EndsAt.
func (m *Manager) Start(ctx context.Context, opts StartOptions) error {
	if opts.Game.StartsAt == nil || opts.Game.EndsAt == nil {
		now := time.Now().UTC()
		start := now.Add(3 * time.Second)
		end := start.Add(time.Duration(opts.Game.Duration) * time.Second)
		opts.Game.StartsAt = &start
		opts.Game.EndsAt = &end
	}

	rt := newRuntime(m, opts)
	m.mu.Lock()
	if _, exists := m.games[opts.Game.ID]; exists {
		m.mu.Unlock()
		return nil
	}
	m.games[opts.Game.ID] = rt
	for _, p := range opts.Players {
		m.userGame[p.UserID] = opts.Game.ID
	}
	m.mu.Unlock()

	runCtx := context.Background()
	rt.broadcast(hub.Ev("game_start", rt.view(0)))
	go rt.run(runCtx)
	for uid, cfg := range opts.Bots {
		go rt.runBot(runCtx, uid, cfg)
	}
	return nil
}

func (m *Manager) HandleInput(userID uint32, kp entity.Keypress) bool {
	m.mu.RLock()
	id, ok := m.userGame[userID]
	rt := m.games[id]
	m.mu.RUnlock()
	if !ok || rt == nil {
		return false
	}
	rt.applyInput(userID, kp)
	return true
}

func (m *Manager) IsInGame(userID uint32) (uint32, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.userGame[userID]
	return id, ok
}

// Snapshot returns live game state used for reconnects and /state.
func (m *Manager) Snapshot(userID uint32) (entity.GameView, bool) {
	m.mu.RLock()
	id, ok := m.userGame[userID]
	rt := m.games[id]
	m.mu.RUnlock()
	if !ok || rt == nil {
		return entity.GameView{}, false
	}
	return rt.view(userID), true
}

// Leave forfeits the caller's run. The game keeps running for everyone else.
func (m *Manager) Leave(ctx context.Context, userID uint32) (uint32, bool) {
	m.mu.RLock()
	id, ok := m.userGame[userID]
	rt := m.games[id]
	m.mu.RUnlock()
	if !ok || rt == nil {
		return 0, false
	}
	rt.leave(userID)
	m.hub.SendToUser(userID, hub.Ev("left", map[string]any{"game_id": id, "reason": "player_left"}))
	return id, true
}

// Abandon stops a game without producing results (e.g. room closed).
func (m *Manager) Abandon(gameID uint32) {
	m.mu.Lock()
	rt := m.games[gameID]
	if rt == nil {
		m.mu.Unlock()
		return
	}
	rt.mu.Lock()
	rt.finished = true
	rt.mu.Unlock()
	m.removeLocked(gameID)
	m.mu.Unlock()
	rt.closeDone()
}

func (m *Manager) remove(gameID uint32) {
	m.mu.Lock()
	m.removeLocked(gameID)
	m.mu.Unlock()
}

func (m *Manager) removeLocked(gameID uint32) {
	rt := m.games[gameID]
	if rt == nil {
		return
	}
	delete(m.games, gameID)
	for uid := range rt.players {
		if m.userGame[uid] == gameID {
			delete(m.userGame, uid)
		}
	}
}

func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.games)
}
