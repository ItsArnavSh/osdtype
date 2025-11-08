package matchmaker

import (
	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/internal/redis"
	"sync"
	"time"

	"github.com/google/btree"
	"go.uber.org/zap"
)

// Automatically scan redis for all the available people in ranked lobby
// And pairs them, triggering the main game
// For v1 using local memory
// Migrate this functionality to redis later

type Matchmaker struct {
	rdb     *redis.RedisClient
	logger  *zap.SugaredLogger
	lobby   map[entity.LobbyType]*btree.BTree
	mu      sync.RWMutex
	ac      *game.ActiveGames
	session *usersession.ActiveSessions
}

func NewMatchMaker(rdb *redis.RedisClient, logger *zap.SugaredLogger, ac *game.ActiveGames) *Matchmaker {
	mm := &Matchmaker{
		rdb:    rdb,
		logger: logger,
		lobby:  make(map[entity.LobbyType]*btree.BTree),
		ac:     ac,
	}

	// Initialize different lobbies
	for _, duration := range []entity.LobbyType{entity.SPRINT, entity.STANDARD, entity.MARATHON} {
		mm.lobby[duration] = btree.New(2) // degree 2 BTree per duration
	}

	return mm
}
func (m *Matchmaker) AddToGlobalLobby(userid uint64, current_rank uint16, duration entity.LobbyType) error {
	in, out := m.session.Users[userid].Subscribe()
	m.lobby[duration].ReplaceOrInsert(entity.PlayerItem{ID: userid, Rank: current_rank, JoinedAt: time.Now(), IN: in, OUT: out})
	//m.rdb.JoinLobby(c.Request.Context(), 0, userid, current_rank)
	return nil
}

// Todo: Build a more fair matchmaker

func (m *Matchmaker) BackgroundMatchmaker() {
	for _, typ := range []entity.LobbyType{entity.SPRINT, entity.STANDARD, entity.MARATHON} {
		go func(lobby_type entity.LobbyType) {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				m.mu.Lock()
				if m.lobby[lobby_type].Len() < 2 {
					m.mu.Unlock()
					continue
				}

				// get oldest player (based on join time)
				oldestItem := m.lobby[lobby_type].Min()
				if oldestItem == nil {
					m.mu.Unlock()
					continue
				}
				oldest := oldestItem.(entity.PlayerItem)
				waitTime := time.Since(oldest.JoinedAt)

				// if the oldest player hasn't waited enough yet, skip
				const minWait = 3 * time.Second
				const maxWait = 8 * time.Second
				if waitTime < minWait {
					m.mu.Unlock()
					continue
				}

				// find nearby players
				var group []entity.PlayerItem
				lowerBound := int(oldest.Rank) - 200
				upperBound := int(oldest.Rank) + 200

				m.lobby[lobby_type].Ascend(func(i btree.Item) bool {
					player := i.(entity.PlayerItem)
					if int(player.Rank) < lowerBound {
						return true
					}
					if int(player.Rank) > upperBound {
						return false
					}
					group = append(group, player)
					return len(group) < 10 // max 10 players
				})

				// start match if conditions met
				if len(group) >= 2 || waitTime >= maxWait {
					for _, p := range group {
						m.lobby[lobby_type].Delete(p)
					}
					m.startMatch(group, typ.Duration())
				}

				m.mu.Unlock()
			}
		}(typ)
	}
}

func (m *Matchmaker) startMatch(players []entity.PlayerItem, duration time.Duration) {
	m.logger.Infof("Starting new game")
	m.ac.NewGame(players, duration)
}
