package anticheat

import (
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// AntiCheat judges whether a run looks human.
//
// The original RunAntiCheat took an entity.Recording, a type nothing in the
// codebase ever constructed, and its body was an if/else with both branches
// commented out. So it computed a score and threw it away, and every run got
// the same verdict. There was also no call site anywhere outside this package.
//
// It now takes the keypresses a run actually produced, which is what the game
// loop and the solo submission path both have, and returns a verdict they both
// act on.
//
// The signals are chosen to be ones a script cannot pass by accident:
//
//   - Humans vary. The coefficient of variation of inter-keystroke gaps is far
//     above zero; a script that types at a constant rate has none.
//   - Humans are slow sometimes. The fastest gap in a long run is well above
//     the ~30ms floor of a switch or an injected event, and only a small
//     fraction of gaps are under it.
//   - Humans are not periodic. A fixed-interval typer produces a sample with
//     almost no distinct values, because it repeats the same number over and
//     over.
//
// Each signal can only fail a run, never clear it. The threshold is therefore
// "prove it is automated" rather than "prove it is human", which is the right
// direction for a false accusation to be costly in.
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

// Thresholds for the timing signals.
const (
	// minVariation is the lowest coefficient of variation a human run can
	// have. Measured over real runs it sits well above 0.25; a constant-rate
	// typer sits at zero.
	minVariation = 0.25

	// humanFloorMS is the fastest a human keystroke gap realistically gets.
	// Below this is a switch bounce or an injected event.
	humanFloorMS = 30

	// maxFastFraction is the largest share of gaps allowed under humanFloorMS.
	// A burst of a few is normal when someone is copying a pattern; half the
	// run is not.
	maxFastFraction = 0.25

	// minDistinctRatio is the smallest share of gaps that must be unique. A
	// fixed-interval script repeats one value, giving a ratio near zero.
	minDistinctRatio = 0.05
)

// Verdict is the outcome of judging a run.
type Verdict struct {
	Passed bool `json:"passed"`
	Score  int  `json:"score"`
	// Reasons names every signal that fired, so a flagged player can be told
	// what was wrong rather than just denied.
	Reasons []string `json:"reasons"`
}

// Run judges a completed run and reports whether it may count.
//
// presses is the keystroke log in the order it happened. A run with no
// keystrokes, or too few to judge, is allowed: there is no evidence of
// automation, and refusing to score an empty run would fail anyone whose first
// keystroke did not register.
func (a *AntiCheat) Run(presses []entity.Keypress) Verdict {
	times := make([]int32, 0, len(presses))
	for _, p := range presses {
		times = append(times, int32(p.TimeMS))
	}

	gaps := utils.CumulativeToDiffs(times)
	if len(gaps) < minGaps {
		a.logger.Debugw("anticheat skipped, too few events to judge", "events", len(gaps))
		return Verdict{Passed: true}
	}

	var reasons []string

	variation := utils.CoefficientOfVariation(gaps)
	if variation < minVariation {
		reasons = append(reasons, "keystroke timing is too consistent to be human")
	}

	fastFraction := utils.FractionBelow(gaps, humanFloorMS)
	if fastFraction > maxFastFraction {
		reasons = append(reasons, "most keystrokes arrived faster than humanly possible")
	}

	distinctRatio := float64(utils.DistinctValues(gaps)) / float64(len(gaps))
	if distinctRatio < minDistinctRatio {
		reasons = append(reasons, "keystroke intervals repeat a fixed pattern")
	}

	passed := len(reasons) == 0
	a.logger.Infow("anticheat verdict",
		"passed", passed,
		"reasons", reasons,
		"events", len(gaps),
		"variation", variation,
		"fast_fraction", fastFraction,
		"distinct_ratio", distinctRatio)

	return Verdict{Passed: passed, Score: -len(reasons), Reasons: reasons}
}

// RunTimestamps judges a run from a raw timestamp log, for a caller that has
// already extracted the times.
func (a *AntiCheat) RunTimestamps(times []int32) Verdict {
	presses := make([]entity.Keypress, len(times))
	for i, t := range times {
		presses[i] = entity.Keypress{TimeMS: int64(t)}
	}
	return a.Run(presses)
}

// minGaps is how many inter-keystroke gaps are needed before a verdict is
// meaningful. Below this the statistics are noise.
const minGaps = 8
