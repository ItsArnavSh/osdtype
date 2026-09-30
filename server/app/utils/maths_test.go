//go:build unit

package utils

import (
	"math"
	"testing"
)

func TestCumulativeToDiffsEmpty(t *testing.T) {
	if got := CumulativeToDiffs(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	if got := CumulativeToDiffs([]int32{100}); got != nil {
		t.Fatalf("a single sample has no gaps, got %v", got)
	}
}

func TestCumulativeToDiffs(t *testing.T) {
	// The first sample is dropped, not differenced against zero: the gap from
	// the round starting to the first keystroke is how long the player took to
	// settle in, not how fast they type. The old version returned it as a gap
	// and inflated every mean it was later used in.
	got := CumulativeToDiffs([]int32{100, 250, 400})
	want := []int32{150, 150}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestCumulativeToDiffsDiscardsNonPositiveGaps(t *testing.T) {
	// Timestamps can arrive out of order or repeated when a client's clock is
	// coarse. A negative or zero gap is a broken measurement, not evidence of
	// automation, so it is dropped rather than kept: keeping it would make the
	// anticheat report a verdict derived from the client's bug.
	got := CumulativeToDiffs([]int32{100, 50, 200, 200, 300})
	want := []int32{150, 100}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestCumulativeToDiffsDoesNotMutateInput(t *testing.T) {
	// The anticheat keeps the cumulative series for its own statistics, so a
	// conversion that reordered or wrote through it would corrupt both.
	in := []int32{100, 250, 400}
	before := append([]int32(nil), in...)
	CumulativeToDiffs(in)
	for i := range in {
		if in[i] != before[i] {
			t.Fatalf("the input was modified: %v became %v", before, in)
		}
	}
}

func TestStandardDeviationTooShort(t *testing.T) {
	if got := StandardDeviation(nil); got != 0 {
		t.Errorf("expected 0 for nil, got %v", got)
	}
	if got := StandardDeviation([]int32{42}); got != 0 {
		t.Errorf("expected 0 for a single element, got %v", got)
	}
}

func TestStandardDeviationIdenticalValues(t *testing.T) {
	if got := StandardDeviation([]int32{7, 7, 7, 7}); got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestStandardDeviationKnownValue(t *testing.T) {
	// {2,4,4,4,5,5,7,9} has a population standard deviation of exactly 2.
	got := StandardDeviation([]int32{2, 4, 4, 4, 5, 5, 7, 9})
	if math.Abs(got-2.0) > 1e-9 {
		t.Errorf("expected 2, got %v", got)
	}
}

// TestStandardDeviationDoesNotOverflow is the regression that motivated the
// rewrite: the old accumulator summed into an int32, so a long run wrapped and
// produced a nonsense mean, and therefore a nonsense deviation.
func TestStandardDeviationDoesNotOverflow(t *testing.T) {
	// 40 000 samples averaging 60 000 sums to 2.4e9, which is past MaxInt32.
	const n = 40_000
	arr := make([]int32, n)
	for i := range arr {
		arr[i] = 60_000
	}

	got := StandardDeviation(arr)
	if got != 0 {
		t.Errorf("identical values should have a deviation of 0, got %v", got)
	}
	if m := Mean(arr); m != 60_000 {
		t.Errorf("the mean overflowed: got %v, want 60000", m)
	}
}

func TestMeanTooShort(t *testing.T) {
	if got := Mean(nil); got != 0 {
		t.Errorf("expected 0 for nil, got %v", got)
	}
	if got := Mean([]int32{7}); got != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}

func TestCoefficientOfVariation(t *testing.T) {
	// A perfectly even cadence has no relative variation, however slow it is.
	if got := CoefficientOfVariation([]int32{100, 100, 100, 100}); got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
	// The measure is scale invariant, which is the reason for using it at all:
	// a typist who keeps an even rhythm has the same relative variation whether
	// they are typing at 50ms or 12ms a keystroke, and the anticheat has to
	// judge them the same. Raw deviation would call the fast one a bot.
	steady := CoefficientOfVariation([]int32{50, 50, 50, 50, 100, 100, 100, 100})
	brisk := CoefficientOfVariation([]int32{200, 200, 200, 200, 400, 400, 400, 400})
	if math.Abs(brisk-steady) > 1e-9 {
		t.Errorf("the same rhythm at four times the pace scored %v against %v; the measure is not scale invariant", brisk, steady)
	}
	if got := CoefficientOfVariation([]int32{0, 0}); got != 0 {
		t.Errorf("a zero mean should report 0 rather than divide by zero, got %v", got)
	}
	if got := CoefficientOfVariation([]int32{10}); got != 0 {
		t.Errorf("a single sample has no spread, got %v", got)
	}
}

func TestFindMinIgnoringFirstShortInput(t *testing.T) {
	if got := FindMinIgnoringFirst(nil); got != 0 {
		t.Errorf("expected 0 for nil, got %d", got)
	}
	if got := FindMinIgnoringFirst([]int32{5}); got != 0 {
		t.Errorf("expected 0 for a single element, got %d", got)
	}
}

// TestFindMinIgnoringFirst documents the intent: the first element is the
// start of the run and is skipped, so a fast opening keystroke cannot be
// mistaken for a burst of automation.
func TestFindMinIgnoringFirst(t *testing.T) {
	got := FindMinIgnoringFirst([]int32{1, 50, 120, 90})
	if got != 50 {
		t.Errorf("expected 50, got %d", got)
	}
}

func TestPercentile(t *testing.T) {
	arr := []int32{50, 10, 40, 20, 30}
	if got := Percentile(arr, 0); got != 10 {
		t.Errorf("p0 = %d, want the minimum 10", got)
	}
	if got := Percentile(arr, 100); got != 50 {
		t.Errorf("p100 = %d, want the maximum 50", got)
	}
	if got := Percentile(arr, 50); got != 30 {
		t.Errorf("p50 = %d, want 30", got)
	}
	if got := Percentile(nil, 50); got != 0 {
		t.Errorf("an empty sample has no percentile, got %d", got)
	}
}

func TestPercentileDoesNotSortInPlace(t *testing.T) {
	// The anticheat holds the live interval slice; sorting it in place would
	// reorder the keystroke log underneath the other statistics.
	arr := []int32{50, 10, 40, 20, 30}
	before := append([]int32(nil), arr...)
	Percentile(arr, 50)
	for i := range arr {
		if arr[i] != before[i] {
			t.Fatalf("the sample was reordered: %v became %v", before, arr)
		}
	}
}

func TestCountBelowAndFractionBelow(t *testing.T) {
	arr := []int32{10, 20, 30, 40, 50}
	if got := CountBelow(arr, 30); got != 2 {
		t.Errorf("CountBelow = %d, want 2", got)
	}
	if got := FractionBelow(arr, 30); math.Abs(got-0.4) > 1e-9 {
		t.Errorf("FractionBelow = %v, want 0.4", got)
	}
	if got := FractionBelow(nil, 30); got != 0 {
		t.Errorf("an empty sample has no fraction, got %v", got)
	}
	if got := CountBelow(arr, 0); got != 0 {
		t.Errorf("nothing is below zero, got %d", got)
	}
}

func TestDistinctValues(t *testing.T) {
	// A scripted run repeats the same interval; a human almost never does.
	scripted := []int32{80, 80, 80, 80, 80, 80, 80, 80}
	if got := DistinctValues(scripted); got != 1 {
		t.Errorf("a scripted run should have one distinct value, got %d", got)
	}
	human := []int32{71, 133, 98, 120, 145, 87, 160, 101}
	if got := DistinctValues(human); got != 8 {
		t.Errorf("a human run should have eight distinct values, got %d", got)
	}
	if got := DistinctValues(nil); got != 0 {
		t.Errorf("an empty sample has no values, got %d", got)
	}
}

func TestFindMinIgnoringFirstAtTwo(t *testing.T) {
	// Two samples is the smallest input that has anything to compare, and the
	// off-by-one here would have silently returned the first sample.
	if got := FindMinIgnoringFirst([]int32{900, 100}); got != 100 {
		t.Errorf("expected 100, got %d", got)
	}
}
