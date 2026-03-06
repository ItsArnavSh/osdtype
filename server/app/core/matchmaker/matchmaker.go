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

type Matchmaker struct {
	rdb     *redis.RedisClient
	logger  *zap.SugaredLogger
	lobby   map[entity.LobbyType]*btree.BTree
	mu      sync.RWMutex
	ac      *game.ActiveGames
	session *usersession.ActiveSessions
	db      *postgresql.Database
}

func NewMatchMaker(
	rdb *redis.RedisClient,
	logger *zap.SugaredLogger,
	ac *game.ActiveGames,
	sessions *usersession.ActiveSessions,
	db *postgresql.Database,
) Matchmaker {
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
	for _, typ := range []entity.LobbyType{
		entity.SPRINT,
		entity.STANDARD,
		entity.MARATHON,
	} {
		m.lobby[typ] = btree.New(2)
	}

	go m.worker()
}

func (m *Matchmaker) AddToGlobalLobby(userid uint32, rank uint16, typ entity.LobbyType) error {

	user := m.session.Users[userid]
	if user == nil {
		return fmt.Errorf("user session not found")
	}

	in, out := user.Subscribe()

	player := entity.PlayerItem{
		ID:       userid,
		Rank:     rank,
		JoinedAt: time.Now(),
		IN:       in,
		OUT:      out,
	}

	m.mu.Lock()
	m.lobby[typ].ReplaceOrInsert(player)
	m.mu.Unlock()

	return nil
}

func (m *Matchmaker) worker() {

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {

		for _, typ := range []entity.LobbyType{
			entity.SPRINT,
			entity.STANDARD,
			entity.MARATHON,
		} {

			players := m.tryMatch(typ)

			if len(players) >= 2 {

				go m.startMatch(players, typ.Duration())
			}
		}
	}
}

func (m *Matchmaker) tryMatch(typ entity.LobbyType) []entity.PlayerItem {

	m.mu.Lock()
	defer m.mu.Unlock()

	tree := m.lobby[typ]

	if tree.Len() < 2 {
		return nil
	}

	oldestItem := tree.Min()
	if oldestItem == nil {
		return nil
	}

	oldest := oldestItem.(entity.PlayerItem)

	wait := time.Since(oldest.JoinedAt)

	baseRange := 100
	extra := int(wait.Seconds()) * 50

	lower := int(oldest.Rank) - (baseRange + extra)
	upper := int(oldest.Rank) + (baseRange + extra)

	var group []entity.PlayerItem

	tree.Ascend(func(i btree.Item) bool {

		p := i.(entity.PlayerItem)

		if int(p.Rank) < lower {
			return true
		}

		if int(p.Rank) > upper {
			return false
		}

		group = append(group, p)

		return len(group) < 6
	})

	if len(group) < 2 {
		return nil
	}

	for _, p := range group {
		tree.Delete(p)
	}

	return group
}

func (m *Matchmaker) startMatch(players []entity.PlayerItem, duration time.Duration) {

	m.logger.Infof("Starting match with %d players", len(players))

	sig := make(chan []entity.WPMRes)

	m.ac.NewGame(players, duration, sig)

	go func() {

		select {

		case res := <-sig:
			m.updateRanks(res)
			m.logger.Infoln("Scores have been updated")
		case <-time.After(2 * time.Minute):
			m.logger.Error("game result timeout")
		}

	}()
}

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
