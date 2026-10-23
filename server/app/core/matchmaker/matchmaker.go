package matchmaker

import (
	"context"
	"fmt"
	"osdtyp/app/core/game"
	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Creator is implemented by the service layer. The matchmaker only decides
// who plays whom; creating games and applying ratings stays in one place.
type Creator interface {
	StartRankedMatch(ctx context.Context, players []entity.User, durationSec int) error
	PickRankedBot(ctx context.Context, rating, duration int) (entity.User, bool)
}

type Matchmaker struct {
	logger  *zap.SugaredLogger
	rdb     *redis.Client
	db      *postgresql.Database
	hub     *hub.Hub
	games   *game.Manager
	creator Creator

	mu      sync.Mutex
	started bool
	stop    chan struct{}

	baseRange   int
	growthRange int
	maxPlayers  int
	botAfter    time.Duration
	tick        time.Duration

	lastNotify map[uint32]time.Time
}

func New(logger *zap.SugaredLogger, rdb *redis.Client, db *postgresql.Database, h *hub.Hub, games *game.Manager) *Matchmaker {
	base := viper.GetInt("Matchmaking.base_range")
	if base <= 0 {
		base = 100
	}
	growth := viper.GetInt("Matchmaking.growth_range")
	if growth <= 0 {
		growth = 50
	}
	maxPlayers := viper.GetInt("Matchmaking.max_players")
	if maxPlayers < 2 {
		maxPlayers = 4
	}
	botAfter := time.Duration(viper.GetInt("Matchmaking.bot_after_seconds")) * time.Second
	if botAfter <= 0 {
		botAfter = 20 * time.Second
	}
	tick := time.Duration(viper.GetInt("Matchmaking.tick_ms")) * time.Millisecond
	if tick <= 0 {
		tick = 500 * time.Millisecond
	}
	return &Matchmaker{
		logger:      logger,
		rdb:         rdb,
		db:          db,
		hub:         h,
		games:       games,
		baseRange:   base,
		growthRange: growth,
		maxPlayers:  maxPlayers,
		botAfter:    botAfter,
		tick:        tick,
		lastNotify:  make(map[uint32]time.Time),
	}
}

func (m *Matchmaker) SetCreator(c Creator) { m.creator = c }

func (m *Matchmaker) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.stop = make(chan struct{})
	m.mu.Unlock()

	ticker := time.NewTicker(m.tick)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stop:
				return
			case <-ticker.C:
				for _, duration := range []int{30, 90, 300} {
					m.matchDuration(ctx, duration)
					m.notifyQueued(ctx, duration)
				}
			}
		}
	}()
}

func (m *Matchmaker) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		close(m.stop)
		m.started = false
	}
}

func (m *Matchmaker) Add(ctx context.Context, uid uint32, rating int, duration int) error {
	now := time.Now().UTC()
	if err := m.rdb.QueueAdd(ctx, duration, uid, rating, now); err != nil {
		return err
	}
	_ = m.rdb.SetState(ctx, uid, redis.UserState{Status: "queue", QueueDuration: duration})
	m.hub.SendToUser(uid, hub.Ev("queue_joined", map[string]any{
		"duration":  duration,
		"rating":    rating,
		"joined_at": now,
	}))
	return nil
}

func (m *Matchmaker) Remove(ctx context.Context, uid uint32, duration int) error {
	if err := m.rdb.QueueRemove(ctx, duration, uid); err != nil {
		return err
	}
	st, ok, _ := m.rdb.GetState(ctx, uid)
	if ok && st.Status == "queue" {
		_ = m.rdb.SetState(ctx, uid, redis.UserState{Status: "idle"})
	}
	m.hub.SendToUser(uid, hub.Ev("queue_left", map[string]any{"duration": duration}))
	return nil
}

type QueueStatus struct {
	Duration int       `json:"duration"`
	Position int64     `json:"position"`
	Waiting  int       `json:"waiting"`
	JoinedAt time.Time `json:"joined_at"`
}

func (m *Matchmaker) QueueSize(ctx context.Context) int {
	total := 0
	for _, duration := range []int{30, 90, 300} {
		items, err := m.rdb.QueueItems(ctx, duration)
		if err == nil {
			total += len(items)
		}
	}
	return total
}

func (m *Matchmaker) Status(ctx context.Context, uid uint32, rating int, duration int) (QueueStatus, bool, error) {
	items, err := m.rdb.QueueItems(ctx, duration)
	if err != nil {
		return QueueStatus{}, false, err
	}
	var pos int64
	var joined time.Time
	found := false
	for _, it := range items {
		if it.UserID == uid {
			joined = it.JoinedAt
			found = true
			break
		}
		if it.Rating > rating {
			pos++
		}
	}
	if !found {
		return QueueStatus{}, false, nil
	}
	return QueueStatus{Duration: duration, Position: pos, Waiting: len(items), JoinedAt: joined}, true, nil
}

func (m *Matchmaker) matchDuration(ctx context.Context, duration int) {
	ok, err := m.rdb.AcquireLock(ctx, fmt.Sprintf("matchmaker:%d", duration), 2*time.Second)
	if err != nil || !ok {
		return
	}
	defer m.rdb.ReleaseLock(ctx, fmt.Sprintf("matchmaker:%d", duration))

	items, err := m.rdb.QueueItems(ctx, duration)
	if err != nil || len(items) == 0 {
		return
	}

	claimed := make(map[uint32]bool)
	now := time.Now().UTC()

	for i := 0; i < len(items); i++ {
		a := items[i]
		if claimed[a.UserID] {
			continue
		}
		if _, inGame := m.games.IsInGame(a.UserID); inGame {
			_ = m.rdb.QueueRemove(ctx, duration, a.UserID)
			continue
		}

		wait := now.Sub(a.JoinedAt)
		spread := m.baseRange + int(wait.Seconds())*m.growthRange
		group := []redis.QueueItem{a}
		for j := i + 1; j < len(items) && len(group) < m.maxPlayers; j++ {
			b := items[j]
			if claimed[b.UserID] {
				continue
			}
			if abs(b.Rating-a.Rating) <= spread {
				group = append(group, b)
			}
		}

		if len(group) < 2 {
			if wait >= m.botAfter && m.creator != nil {
				bot, ok := m.creator.PickRankedBot(ctx, a.Rating, duration)
				if ok {
					users, err := m.loadUsers(ctx, []uint32{a.UserID, bot.ID})
					if err == nil && len(users) == 2 {
						_ = m.rdb.QueueRemove(ctx, duration, a.UserID)
						if err := m.creator.StartRankedMatch(ctx, users, duration); err != nil {
							m.logger.Errorw("failed to start bot match", "error", err)
						}
					}
				}
			}
			continue
		}

		for _, it := range group {
			_ = m.rdb.QueueRemove(ctx, duration, it.UserID)
			claimed[it.UserID] = true
		}
		ids := make([]uint32, 0, len(group))
		for _, it := range group {
			ids = append(ids, it.UserID)
		}
		users, err := m.loadUsers(ctx, ids)
		if err != nil || len(users) < 2 {
			m.logger.Warnw("match candidates vanished", "error", err)
			continue
		}
		if m.creator != nil {
			if err := m.creator.StartRankedMatch(ctx, users, duration); err != nil {
				m.logger.Errorw("failed to start ranked match", "error", err)
				for _, u := range users {
					_ = m.rdb.QueueAdd(ctx, duration, u.ID, u.RatingFor(duration), now)
				}
			}
		}
	}
}

func (m *Matchmaker) loadUsers(ctx context.Context, ids []uint32) ([]entity.User, error) {
	users, err := m.db.ListUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	// Preserve queue order.
	byID := make(map[uint32]entity.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	out := make([]entity.User, 0, len(ids))
	for _, id := range ids {
		if u, ok := byID[id]; ok {
			out = append(out, u)
		}
	}
	if len(out) != len(ids) {
		return out, fmt.Errorf("some queued users no longer exist")
	}
	return out, nil
}

func (m *Matchmaker) notifyQueued(ctx context.Context, duration int) {
	items, err := m.rdb.QueueItems(ctx, duration)
	if err != nil {
		return
	}
	now := time.Now()
	for i, it := range items {
		if last, ok := m.lastNotify[it.UserID]; ok && now.Sub(last) < 2*time.Second {
			continue
		}
		m.lastNotify[it.UserID] = now
		m.hub.SendToUser(it.UserID, hub.Ev("queue_status", map[string]any{
			"duration": duration,
			"position": i,
			"waiting":  len(items),
			"waited":   int(now.Sub(it.JoinedAt).Seconds()),
		}))
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
