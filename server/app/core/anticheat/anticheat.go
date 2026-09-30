package anticheat

import (
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type AntiCheat struct {
	logger *zap.SugaredLogger
}

// NewAntiCheat builds a scorer. A nil logger is tolerated so a bare
// &AntiCheat{} stays usable from tests.
func NewAntiCheat(logger *zap.SugaredLogger) AntiCheat {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return AntiCheat{logger: logger}
}

// RunAntiCheat scores a finished recording and reports whether it passed.
//
// The body used to be an if/else whose two branches were both commented-out
// queries, so the function computed a score and then threw it away: every run
// had the same outcome and nothing observed the result. The verdict is now
// returned and logged.
//
// The scoring convention is a negative total means the run looks automated, a
// positive total means it looks human, and only a positive total passes.
func (a *AntiCheat) RunAntiCheat(rec entity.Recording) (passed bool, score int) {
	diffs := utils.CumulativeToDiffs(rec.Timestamps)
	score = a.StandardDeviationTest(diffs) + a.ShortestInterval(diffs)
	passed = score > 0

	verdict := "flagged"
	if passed {
		verdict = "cleared"
	}
	a.logger.Debugw("anticheat verdict",
		"run_id", rec.RunID,
		"verdict", verdict,
		"score", score)

	return passed, score
}
