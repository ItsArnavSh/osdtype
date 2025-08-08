package entity

import (
	"osdtype/database"

	"github.com/asaskevich/EventBus"
	"go.uber.org/zap"
)

type Essentials struct {
	Db     *database.Queries
	Logger *zap.Logger
	Bus    EventBus.Bus
}
