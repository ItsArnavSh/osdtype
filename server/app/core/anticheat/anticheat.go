package anticheat

import (
	"context"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type AntiCheat struct {
	db     postgresql.Database
	logger *zap.SugaredLogger
}

func (a *AntiCheat) RunAntiCheat(ctx context.Context, rec entity.Recording) {

	time_diff := utils.CumulativeToDiffs(rec.Timestamps)

	is_flagged := a.StandardDeviationTest(time_diff) + a.ShortestInterval(time_diff)
	if is_flagged > 0 {
		//a.Query.FlagTypeRun(ctx, rec.RunID)
	} else {
		//a.Query.VerifyTypeRun(ctx, rec.RunID)
	}

}
