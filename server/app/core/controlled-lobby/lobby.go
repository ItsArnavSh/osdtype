package controlledlobby

import (
	"errors"
	"sync"
	"time"

	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// For Rooms and friends 1v1 where the user decides the lobby
// Also used by Contest
type ControlledLobby struct {
	logger *zap.SugaredLogger
	ac     *game.ActiveGames
	// lobby is written by the HTTP handler goroutines that join a lobby and read
	// and deleted by the scheduler goroutine that starts the game, so every
	// access has to hold the mutex. Concurrent map writes crash the process.
	mu        sync.Mutex
	lobby     map[uint32][]entity.PlayerItem
	generator utils.Generator
	session   *usersession.ActiveSessions
}

func NewControlledLobby(logger *zap.SugaredLogger, ac *game.ActiveGames, session *usersession.ActiveSessions) ControlledLobby {
	return ControlledLobby{
		logger:    logger,
		ac:        ac,
		generator: utils.NewGenerator(),
		lobby:     make(map[uint32][]entity.PlayerItem),
		session:   session,
	}
}
func (c *ControlledLobby) CreateNewLobby() uint32 {
	lobby_id := c.generator.GenerateID()
	return lobby_id
}
func (c *ControlledLobby) JoinControlledLobby(userid, lobby_id uint32) error {
	session := c.session.GetSession(userid)
	if session == nil {
		// Subscribing to a session that does not exist dereferenced a nil
		// pointer. The caller is an HTTP handler, so gin turned the panic into
		// a bare 500 with no explanation.
		c.logger.Warnf("user %d tried to join lobby %d without a live session", userid, lobby_id)
		return errors.New("no live session for this user")
	}

	in, out := session.Subscribe()
	c.addPlayer(lobby_id, entity.PlayerItem{ID: userid, IN: in, OUT: out})
	return nil
}

// addPlayer puts a player on a lobby's roster.
func (c *ControlledLobby) addPlayer(lobby_id uint32, p entity.PlayerItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lobby[lobby_id] = append(c.lobby[lobby_id], p)
}
func (c *ControlledLobby) StartGameFromLobby(lobby_id uint32, duration time.Duration, sig chan []entity.WPMRes) error {
	// Take the roster out under the lock and then release it: NewGame blocks
	// for the whole round, and holding the lock through that would stop anyone
	// from joining a different lobby for the duration of the match.
	c.mu.Lock()
	players := c.lobby[lobby_id]
	delete(c.lobby, lobby_id)
	c.mu.Unlock()

	if len(players) == 0 {
		return errors.New("lobby not found in memory")
	}
	c.ac.NewGame(players, duration, sig)
	return nil
}
func (c *ControlledLobby) RemoveLobby(lobbyid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lobby, lobbyid)
}

// LobbySize reports how many players are waiting in a lobby. Used by tests and
// diagnostics.
func (c *ControlledLobby) LobbySize(lobbyid uint32) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.lobby[lobbyid])
}
