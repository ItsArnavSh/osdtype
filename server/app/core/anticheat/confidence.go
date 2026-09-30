package anticheat

import "osdtyp/app/utils"

// StandardDeviationTest scores how much a run's timing varies.
//
// The old version compared the raw standard deviation against a fixed
// threshold of 15ms, which conflated two different things: a fast typist has a
// small deviation purely by typing fast, so a legitimately quick run scored the
// same as a constant-rate script. It also used an int32 accumulator that
// overflowed on any run over a few thousand keystrokes.
//
// The coefficient of variation divides the deviation by the mean, so it
// measures variation relative to the player's own pace: a slow careful typist
// and a fast sloppy one both clear it, and a fixed-interval typer does not.
//
// It returns a positive contribution when the run passes.
func (a *AntiCheat) StandardDeviationTest(gaps []int32) int {
	if len(gaps) < 2 {
		return standardDeviationConfidence
	}
	if utils.CoefficientOfVariation(gaps) < minVariation {
		return -standardDeviationConfidence
	}
	return standardDeviationConfidence
}

// ShortestInterval scores the fastest gap in a run.
//
// A sub-30ms gap is a switch bounce or an injected event rather than a human
// keystroke. A handful of them is normal when someone is typing a repeated
// pattern, so the signal only fires when a large share of the run is under the
// floor, which a script produces and a person does not.
func (a *AntiCheat) ShortestInterval(gaps []int32) int {
	if len(gaps) < 2 {
		return shortestIntervalConfidence
	}
	if utils.FractionBelow(gaps, humanFloorMS) > maxFastFraction {
		return -shortestIntervalConfidence
	}
	return shortestIntervalConfidence
}

// Uniformity scores how many distinct intervals a run contains.
//
// This is the cheapest reliable signal against a fixed-interval typer: it
// repeats one number, so a long run has almost no distinct values. Human
// typing has a different gap every time.
func (a *AntiCheat) Uniformity(gaps []int32) int {
	if len(gaps) < 2 {
		return uniformityConfidence
	}
	ratio := float64(utils.DistinctValues(gaps)) / float64(len(gaps))
	if ratio < minDistinctRatio {
		return -uniformityConfidence
	}
	return uniformityConfidence
}

// Confidence weights for each signal.
const (
	standardDeviationConfidence = 2
	shortestIntervalConfidence  = 4
	// Uniformity is weighted highest because it has no false positives to
	// speak of: real typing has a unique gap per keystroke.
	uniformityConfidence = 6
)
