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
	rh.essentials = entity.Essentials{Db: db, Logger: logger}
	return rh
}
