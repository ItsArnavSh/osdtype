package room

import (
	"osdtype/application/entity"
)

type RoomHandler struct {
	essentials entity.Essentials
	actions    map[string]func(info string) error
}
