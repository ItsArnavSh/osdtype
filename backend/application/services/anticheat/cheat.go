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

	time_diff := util.CumulativeToDiffs(rec.Timestamps)

	is_flagged := a.StandardDeviationTest(time_diff) + a.ShortestInterval(time_diff)
	if is_flagged > 0 {
		a.Query.FlagTypeRun(ctx, rec.RunID)
	} else {
		a.Query.VerifyTypeRun(ctx, rec.RunID)
	}

}
