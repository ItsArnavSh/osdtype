package matchmaker

import (
	"fmt"
	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"
	"osdtyp/app/utils"
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
	db      *postgresql.Database
}

func NewMatchMaker(rdb *redis.RedisClient, logger *zap.SugaredLogger, ac *game.ActiveGames, sessions *usersession.ActiveSessions, db *postgresql.Database) Matchmaker {
	return Matchmaker{
		rdb:     rdb,
		logger:  logger,
		lobby:   make(map[entity.LobbyType]*btree.BTree),
		ac:      ac,
		session: sessions,
		db:      db,
	}

}
func (m *Matchmaker) Initialize() {
	// Initialize different lobbies
	for _, duration := range []entity.LobbyType{entity.SPRINT, entity.STANDARD, entity.MARATHON} {
		m.lobby[duration] = btree.New(2) // degree 2 BTree per duration
	}
}
func (m *Matchmaker) AddToGlobalLobby(userid uint32, current_rank uint16, duration entity.LobbyType) error {
	if m.session.Users[userid] == nil {
		m.logger.Info("Total ", len(m.session.Users))
		for k, v := range m.session.Users {
			m.logger.Info("Finding ", userid)
			m.logger.Debug("Member ", k, v)
		}
		return fmt.Errorf("Empty\n\n\n\n\n")
	}
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
	sig := make(chan []entity.WPMRes)
	m.ac.NewGame(players, duration, sig)
	m.updateRanks(<-sig)
}

// Post match rank changes
func (m *Matchmaker) updateRanks(leaderboard []entity.WPMRes) {
	var ranks []uint16
	var pos []uint16
	for i, entry := range leaderboard {
		user, err := m.db.GetUser(entry.ID)
		if err != nil {
			return
		}
		ranks = append(ranks, user.CurrentRank)
		pos = append(pos, uint16(i+1))
	}

	updated := utils.UpdateElo(ranks, pos)
	for i, entry := range leaderboard {
		err := m.db.ChangeRank(entry.ID, updated[i])
		if err != nil {
			return
		}
	}
}
