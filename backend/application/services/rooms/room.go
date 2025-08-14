package room

import (
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RoomHandler struct {
	essentials entity.Essentials
	actions    map[string]func(*gin.Context, []byte) error
}

func NewRoomHandler(db *database.Queries, logger *zap.Logger) RoomHandler {
	rh := RoomHandler{}
	var actions = map[string]func(*gin.Context, []byte) error{
		"create_room":        rh.create_room,
		"add_player":         rh.add_player,
		"change_player_perm": rh.change_player_perms,
		"remove_player":      rh.remove_player,
	}
	rh.actions = actions
	rh.essentials = entity.Essentials{Db: db, Logger: logger}
	return rh
}
