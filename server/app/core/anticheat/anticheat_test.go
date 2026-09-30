//go:build unit

package anticheat

import (
	"strings"
	"testing"

	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

// The anticheat reads three signals and can only fail a run, never clear it.
// The threshold is "prove it is automated" rather than "prove it is human",
// because the cost of a false accusation is much higher than the cost of a
// missed one.

// humanIntervals is a measured cadence rather than a tidy one.
//
// The first version of this fixture was a neat list of round numbers in the
// 90-210ms range, which turned out to have a coefficient of variation of 0.224
// — just under the 0.25 threshold, so a run built to look human was flagged.
// That was the fixture being wrong rather than the threshold: real typing is
// far noisier than that, because of double-letter rolls, word boundaries and
// the pauses where somebody re-reads the next few words. The values below are
// what that looks like, and the CV is asserted in
// TestHumanFixtureIsAboveTheThreshold so the fixture cannot drift back under
// it without a test failing.
func humanIntervals() []int32 {
	return []int32{
		143, 61, 288, 112, 96, 201, 74, 341, 128, 167,
		55, 259, 103, 187, 141, 79, 312, 122, 176, 88,
		233, 119, 164, 67, 205, 134, 98, 276, 151, 83,
		119, 224, 71, 163, 108, 245, 92, 182, 137, 59,
	}
}

// botIntervals is a fixed-interval typer: no variation at all, and every gap
// below the human floor. All three signals should catch it.
func botIntervals() []int32 {
	out := make([]int32, 40)
	for i := range out {
		out[i] = 12
	}
	return out
}

// metronomeIntervals is the awkward case: a script typing at a perfectly
// plausible human pace. It has no variation and repeats one number, but nothing
// in it is individually impossible, so it is what the variation and uniformity
// signals exist for.
func metronomeIntervals() []int32 {
	out := make([]int32, 40)
	for i := range out {
		out[i] = 140
	}
	return out
}

// cumulative turns a list of intervals into the timestamps a run actually
// produces, which is the shape the run path hands over.
func cumulative(intervals []int32) []int32 {
	out := make([]int32, 0, len(intervals))
	var total int32
	for _, d := range intervals {
		total += d
		out = append(out, total)
	}
	return out
}

// TestHumanFixtureIsAboveTheThreshold guards the fixture itself. Without it, a
// future edit that tidies the numbers up would quietly turn every test in this
// file into a false accusation, and the failure would read as the anticheat
// working.
func TestHumanFixtureIsAboveTheThreshold(t *testing.T) {
	got := (&AntiCheat{}).StandardDeviationTest(humanIntervals())
	if got <= 0 {
		cv := utils.CoefficientOfVariation(humanIntervals())
		t.Fatalf("the human fixture scores as automated; its coefficient of variation is %.3f and the threshold is %.2f",
			cv, minVariation)
	}
}

func TestStandardDeviationFlagsBots(t *testing.T) {
	got := (&AntiCheat{}).StandardDeviationTest(botIntervals())
	if got >= 0 {
		t.Errorf("a constant-rate run should look automated, got %d", got)
	}
}

func TestStandardDeviationClearsHumans(t *testing.T) {
	got := (&AntiCheat{}).StandardDeviationTest(humanIntervals())
	if got <= 0 {
		t.Errorf("a varied human run should be cleared, got %d", got)
	}
}

// TestStandardDeviationIsPaceInvariant is the reason the signal is a
// coefficient of variation rather than a raw deviation in milliseconds: the
// same rhythm at four times the pace is the same run to a person, and the old
// threshold called the fast one a bot.
func TestStandardDeviationIsPaceInvariant(t *testing.T) {
	slow := make([]int32, 0, 20)
	fast := make([]int32, 0, 20)
	for _, g := range humanIntervals() {
		slow = append(slow, g)
		fast = append(fast, g/4)
	}

	if (&AntiCheat{}).StandardDeviationTest(fast) <= 0 {
		t.Errorf("a fast typist with a human rhythm was flagged: %v", fast)
	}
	if (&AntiCheat{}).StandardDeviationTest(slow) <= 0 {
		t.Errorf("a slow typist with a human rhythm was flagged: %v", slow)
	}
}

func TestShortestIntervalFlagsBots(t *testing.T) {
	got := (&AntiCheat{}).ShortestInterval(botIntervals())
	if got >= 0 {
		t.Errorf("an all-sub-30ms run should look automated, got %d", got)
	}
}

func TestShortestIntervalClearsHumans(t *testing.T) {
	got := (&AntiCheat{}).ShortestInterval(humanIntervals())
	if got <= 0 {
		t.Errorf("a human run should be cleared, got %d", got)
	}
}

// TestShortestIntervalToleratesBursts covers the case the threshold exists for:
// a few very fast keystrokes is normal when someone is typing a repeated
// pattern, and flagging that is a false accusation.
func TestShortestIntervalToleratesBursts(t *testing.T) {
	gaps := append([]int32{9, 8, 11}, humanIntervals()...)
	if got := (&AntiCheat{}).ShortestInterval(gaps); got <= 0 {
		t.Errorf("a handful of fast keystrokes in a human run was flagged, got %d", got)
	}
}

// TestTooShortToJudge documents the deliberate choice on thin input.
//
// The old package treated an empty recording as automated, on the reasoning
// that failing closed is safer. It is the wrong way round: the same reasoning
// refuses to score anyone whose first keystroke did not register, and an
// accused player with no way to produce a counter-example is the worst outcome
// this package can produce. There is no evidence either way, so there is no
// verdict.
func TestTooShortToJudge(t *testing.T) {
	a := NewAntiCheat(nil)
	for _, tc := range []struct {
		name string
		run  []entity.Keypress
	}{
		{"nil", nil},
		{"empty", []entity.Keypress{}},
		{"one", []entity.Keypress{{TimeMS: 100}}},
		{"below the minimum", make([]entity.Keypress, minGaps)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := a.Run(tc.run)
			if !v.Passed {
				t.Errorf("a run too short to judge was flagged: %v", v.Reasons)
			}
			if len(v.Reasons) != 0 {
				t.Errorf("a run with no verdict still produced reasons: %v", v.Reasons)
			}
		})
	}
}

func TestConfidenceConstants(t *testing.T) {
	if standardDeviationConfidence != 2 {
		t.Errorf("unexpected standard deviation confidence: %d", standardDeviationConfidence)
	}
	if shortestIntervalConfidence != 4 {
		t.Errorf("unexpected shortest interval confidence: %d", shortestIntervalConfidence)
	}
	// Uniformity is weighted above the others because it has no false positives
	// to speak of: real typing has a different gap every keystroke. If a
	// threshold change ever made it noisier than the others, the weighting
	// would need revisiting with it.
	if uniformityConfidence <= shortestIntervalConfidence {
		t.Errorf("uniformity is weighted %d, not above the shortest interval's %d",
			uniformityConfidence, shortestIntervalConfidence)
	}
}

func TestUniformityFlagsAFixedIntervalTyper(t *testing.T) {
	if got := (&AntiCheat{}).Uniformity(metronomeIntervals()); got >= 0 {
		t.Errorf("a fixed-interval run should look automated, got %d", got)
	}
}

func TestUniformityClearsHumans(t *testing.T) {
	if got := (&AntiCheat{}).Uniformity(humanIntervals()); got <= 0 {
		t.Errorf("a varied human run should be cleared, got %d", got)
	}
}

// TestRunReportsAVerdict is the end-to-end check, and the regression for the
// original: RunAntiCheat used to have an if/else whose branches were both
// commented out, so the score it computed was discarded and no caller could
// learn the outcome.
func TestRunReportsAVerdict(t *testing.T) {
	a := NewAntiCheat(nil)

	human := a.RunTimestamps(cumulative(humanIntervals()))
	if !human.Passed {
		t.Errorf("a human run was flagged: %v", human.Reasons)
	}
	if human.Score != 0 {
		t.Errorf("a passing run should score zero, got %d", human.Score)
	}
	if len(human.Reasons) != 0 {
		t.Errorf("a passing run should report no reasons, got %v", human.Reasons)
	}

	bot := a.RunTimestamps(cumulative(botIntervals()))
	if bot.Passed {
		t.Error("a bot run was cleared")
	}
	if bot.Score >= 0 {
		t.Errorf("a flagged run should score negative, got %d", bot.Score)
	}
	if len(bot.Reasons) == 0 {
		t.Error("a flagged run must say why, or the player cannot appeal it")
	}
}

// TestRunNamesEverySignalThatFired checks the reason list is the whole story:
// a player told a run was rejected should be able to see all three objections
// at once, not discover them one submission at a time.
func TestRunNamesEverySignalThatFired(t *testing.T) {
	a := NewAntiCheat(nil)
	v := a.RunTimestamps(cumulative(botIntervals()))
	if len(v.Reasons) != 3 {
		t.Errorf("a script should trip all three signals, got %d: %v", len(v.Reasons), v.Reasons)
	}
	for _, r := range v.Reasons {
		if !strings.HasSuffix(r, "human") && !strings.Contains(r, "possible") && !strings.Contains(r, "pattern") {
			t.Errorf("a reason a person could not act on: %q", r)
		}
	}
}

// TestRunFlagsAConstantRateTyperThatIsNotAlsoFast covers the case where the
// fast-keystroke signal cannot help: a script typing at a perfectly plausible
// human pace, where only the variation and uniformity signals have anything to
// say. It is also the reason those two signals exist at all.
func TestRunFlagsAConstantRateTyperThatIsNotAlsoFast(t *testing.T) {
	a := NewAntiCheat(nil)

	v := a.RunTimestamps(cumulative(metronomeIntervals()))
	if v.Passed {
		t.Fatal("a constant-interval run at a plausible speed was cleared")
	}
	for _, r := range v.Reasons {
		if strings.Contains(r, "faster than humanly possible") {
			t.Errorf("a 140ms run was reported as impossibly fast: %v", v.Reasons)
		}
	}
	if len(v.Reasons) < 2 {
		t.Errorf("expected both the variation and the uniformity signal to fire, got %v", v.Reasons)
	}
}

func TestRunAndRunTimestampsAgree(t *testing.T) {
	a := NewAntiCheat(nil)
	times := cumulative(botIntervals())

	presses := make([]entity.Keypress, len(times))
	for i, t := range times {
		presses[i] = entity.Keypress{TimeMS: int64(t)}
	}

	fromKeys := a.Run(presses)
	fromTimes := a.RunTimestamps(times)
	if fromKeys.Passed != fromTimes.Passed || fromKeys.Score != fromTimes.Score {
		t.Errorf("the two entry points disagree: %+v against %+v", fromKeys, fromTimes)
	}
}

// TestRunIgnoresOutOfOrderTimestamps is a client-robustness case: a coarse
// clock produces repeated or backwards timestamps, and that must not be
// mistaken for automation. A bot could exploit the reverse too, so this is a
// judgement call, but refusing to judge a run whose clock is broken is better
// than accusing the player.
func TestRunIgnoresOutOfOrderTimestamps(t *testing.T) {
	a := NewAntiCheat(nil)
	times := cumulative(humanIntervals())
	// Duplicate every other timestamp, as a millisecond-resolution clock does.
	messy := make([]int32, 0, len(times)*2)
	for _, t := range times {
		messy = append(messy, t, t)
	}

	if v := (&a).RunTimestamps(messy); !v.Passed {
		t.Errorf("a run with a coarse clock was flagged: %v", v.Reasons)
	}
}
