package anticheat

import (
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/asaskevich/EventBus"
	"go.uber.org/zap"
)

type AntiCheat struct {
	query  *database.Queries
	logger *zap.Logger
	bus    *EventBus.Bus
}

func (a *AntiCheat) RunAntiCheat(rec entity.Recording) {
	//This will trigger all the anticheat functions one by one for this specific run and then flag or verify the run
	//Check for Standard Deviation in typing speed, near zero can be flagged
	// Also check for pasting...too much text was saved in too little span of time
}
