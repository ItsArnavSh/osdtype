package controlledlobby

import (
	"fmt"
	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"time"

	"go.uber.org/zap"
)

//For Rooms and friends 1v1 where the user decides the lobby

type ControlledLobby struct {
	logger    *zap.SugaredLogger
	ac        *game.ActiveGames
	lobby     map[uint64][]entity.PlayerItem
	generator utils.Generator
	session   *usersession.ActiveSessions
}

func NewControlledLobby(logger *zap.SugaredLogger, ac *game.ActiveGames, session *usersession.ActiveSessions) ControlledLobby {
	return ControlledLobby{
		logger:    logger,
		ac:        ac,
		generator: utils.NewGenerator(),
		lobby:     make(map[uint64][]entity.PlayerItem),
		session:   session,
	}
}
func (c *ControlledLobby) CreateNewLobby() uint64 {
	lobby_id := c.generator.GenerateID()
	return lobby_id
}
func (c *ControlledLobby) JoinControlledLobby(userid, lobby_id uint64) error {
	in, out := c.session.GetSession(userid).Subscribe()
	c.lobby[lobby_id] = append(c.lobby[lobby_id], entity.PlayerItem{ID: userid, IN: in, OUT: out})
	return nil
}
func (c *ControlledLobby) StartGameFromLobby(lobby_id uint64, duration time.Duration, sig chan struct{}) error {
	players := c.lobby[lobby_id]
	if len(players) == 0 {
		return fmt.Errorf("Lobby not found in memory")
	}
	delete(c.lobby, lobby_id)
	c.ac.NewGame(players, duration, sig)
	return nil
}
func (c *ControlledLobby) RemoveLobby(lobbyid uint64) {
	delete(c.lobby, lobbyid)
}
