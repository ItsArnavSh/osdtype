package controlledlobby

import (
	"fmt"
	"osdtyp/app/core/game"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//For Rooms and friends 1v1 where the user decides the lobby

type ControlledLobby struct {
	logger    *zap.SugaredLogger
	ac        *game.ActiveGames
	lobby     map[uint64][]entity.PlayerItem
	generator utils.Generator
}

func NewControlledLobby(logger *zap.SugaredLogger, ac *game.ActiveGames) ControlledLobby {
	return ControlledLobby{
		logger:    logger,
		ac:        ac,
		generator: utils.NewGenerator(),
		lobby:     make(map[uint64][]entity.PlayerItem),
	}
}
func (c *ControlledLobby) CreateNewLobby() uint64 {
	lobby_id := c.generator.GenerateID()
	return lobby_id
}
func (c *ControlledLobby) JoinControlledLobby(g *gin.Context, userid, lobby_id uint64) error {
	websoc, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	c.lobby[lobby_id] = append(c.lobby[lobby_id], entity.PlayerItem{ID: userid, Websock: websoc})
	return nil
}
func (c *ControlledLobby) StartGameFromLobby(lobby_id uint64, duration time.Duration) error {
	players := c.lobby[lobby_id]
	if len(players) == 0 {
		return fmt.Errorf("Lobby not found in memory")
	}
	delete(c.lobby, lobby_id)
	c.ac.NewGame(players, duration)
	return nil
}
