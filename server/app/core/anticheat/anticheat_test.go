//go:build unit

package anticheat

import (
	"testing"

	"osdtyp/app/entity"
)

// The scoring convention across this package is: a negative score means the run
// looks automated, a positive score means it looks human. RunAntiCheat sums the
// two tests and treats anything greater than zero as cleared.

// humanIntervals mimics a person typing with natural variation.
func humanIntervals() []int32 {
	return []int32{120, 95, 180, 140, 210, 160, 90, 175, 130, 200, 145, 165, 110, 190, 155, 125, 205, 150, 135, 170}
}

// botIntervals mimics an automated typer: very low variance and sub-40ms gaps.
func botIntervals() []int32 {
	return []int32{12, 12, 12, 11, 12, 12, 12, 12, 11, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12}
}

func TestStandardDeviationFlagsBots(t *testing.T) {
	a := &AntiCheat{}
	got := a.StandardDeviationTest(botIntervals())
	if got >= 0 {
		t.Errorf("expected a machine-regular run to look automated, got a score of %d", got)
	}
}

func TestStandardDeviationClearsHumans(t *testing.T) {
	a := &AntiCheat{}
	got := a.StandardDeviationTest(humanIntervals())
	if got <= 0 {
		t.Errorf("expected a varied human run to be cleared, got a score of %d", got)
	}
}

func TestShortestIntervalFlagsBots(t *testing.T) {
	a := &AntiCheat{}
	got := a.ShortestInterval(botIntervals())
	if got >= 0 {
		t.Errorf("expected a sub-40ms gap to look automated, got a score of %d", got)
	}
}

func TestShortestIntervalClearsHumans(t *testing.T) {
	a := &AntiCheat{}
	got := a.ShortestInterval(humanIntervals())
	if got <= 0 {
		t.Errorf("expected a human run to be cleared, got a score of %d", got)
	}
}

// TestEmptyRunIsFlagged documents the default. With no timestamps the shortest
// interval is reported as 0, which is below the 40ms threshold, so an empty
// recording is treated as automated. Failing closed is the safer behavior.
func TestEmptyRunIsFlagged(t *testing.T) {
	a := &AntiCheat{}
	if got := a.ShortestInterval(nil); got >= 0 {
		t.Errorf("expected an empty run to be flagged, got %d", got)
	}
	if got := a.StandardDeviationTest(nil); got >= 0 {
		t.Errorf("expected an empty run to be flagged, got %d", got)
	}
}

// TestCombinedScoreClearsHumans is the end-to-end check of the scoring rule
// RunAntiCheat relies on.
func TestCombinedScoreClearsHumans(t *testing.T) {
	a := &AntiCheat{}
	diffs := []int32{0, 120, 95, 180, 140, 210, 160, 90, 175, 130, 200, 145, 165, 110, 190, 155, 125, 205, 150, 135, 170}
	if total := a.StandardDeviationTest(diffs) + a.ShortestInterval(diffs); total <= 0 {
		t.Errorf("expected a human run to score above zero, got %d", total)
	}
}

// TestRunAntiCheatReportsAVerdict is a regression test.
//
// RunAntiCheat used to have an if/else whose branches were both commented out,
// so the score it computed was discarded and no caller could learn the outcome.
func TestRunAntiCheatReportsAVerdict(t *testing.T) {
	a := NewAntiCheat(nil)

	human := entity.Recording{RunID: "human", Timestamps: cumulative(humanIntervals())}
	if passed, score := a.RunAntiCheat(human); !passed {
		t.Errorf("expected a human run to pass, got score %d", score)
	}

	bot := entity.Recording{RunID: "bot", Timestamps: cumulative(botIntervals())}
	if passed, score := a.RunAntiCheat(bot); passed {
		t.Errorf("expected a bot run to be flagged, got score %d", score)
	}
}

// cumulative turns a list of intervals into the cumulative timestamps a
// recording actually stores.
func cumulative(intervals []int32) []int32 {
	out := make([]int32, 0, len(intervals))
	var total int32
	for _, d := range intervals {
		total += d
		out = append(out, total)
	}
	return out
}

func TestConfidenceConstants(t *testing.T) {
	if standard_deviation_confidence != 2 {
		t.Errorf("unexpected standard deviation confidence: %d", standard_deviation_confidence)
	}
	if shortest_interval_confidence != 4 {
		t.Errorf("unexpected shortest interval confidence: %d", shortest_interval_confidence)
	}
}
