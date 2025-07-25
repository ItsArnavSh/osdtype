package anticheat

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/asaskevich/EventBus"
	"go.uber.org/zap"
)

type AntiCheat struct {
	Query  *database.Queries
	Logger *zap.Logger
	Bus    *EventBus.Bus
}

func (a *AntiCheat) RunAntiCheat(ctx context.Context, rec entity.Recording) {
	//This will trigger all the anticheat functions one by one for this specific run and then flag or verify the run
	//Check for Standard Deviation in typing speed, near zero can be flagged
	// Also check for pasting...too much text was saved in too little span of time

	is_flagged := false
	if is_flagged {
		a.Query.FlagTypeRun(ctx, rec.RunID)
	} else {
		a.Query.VerifyTypeRun(ctx, rec.RunID)
	}

}
