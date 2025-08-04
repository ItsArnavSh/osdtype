package room

import (
	"osdtype/database"

	"go.uber.org/zap"
)

type RoomHandler struct {
	db      *database.Queries
	logger  *zap.Logger
	actions map[string]func(info string) error
}

//Create Room
//
