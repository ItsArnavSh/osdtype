package anticheat

import (
	"context"
	"osdtype/application/entity"
	"osdtype/application/util"
	"osdtype/database"

	"go.uber.org/zap"
)

type AntiCheat struct {
	Query  *database.Queries
	Logger *zap.Logger
}

func (a *AntiCheat) RunAntiCheat(ctx context.Context, rec entity.Recording) {
	//This will trigger all the anticheat functions one by one for this specific run and then flag or verify the run
	//Check for Standard Deviation in typing speed, near zero can be flagged
	// Also check for pasting...too much text was saved in too little span of time

	time_diff := util.CumulativeToDiffs(rec.Timestamps)
	is_flagged := false
	if is_flagged {
		a.Query.FlagTypeRun(ctx, rec.RunID)
	} else {
		a.Query.VerifyTypeRun(ctx, rec.RunID)
	}

}
