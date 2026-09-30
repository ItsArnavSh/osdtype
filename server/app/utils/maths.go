package utils

import "math"

// Statistics helpers for the anticheat.
//
// The original StandardDeviation accumulated its sum into an int32, which
// overflows once a run has more than a few thousand keystrokes: a 90 second
// round at 90 WPM is about 13500 events, and summing their timestamps passes
// 2^31 comfortably. An overflowing sum produces a nonsense mean, so the
// deviation the anticheat scored on was garbage for exactly the longest runs,
// which are the ones most worth checking.

// CumulativeToDiffs turns timestamps relative to the start of a run into the
// gaps between consecutive events.
//
// The first sample is dropped rather than differenced against zero. The gap
// from the round starting to the first keystroke is how long the player took
// to settle in, not how fast they type, so including it inflated the mean and
// masked real variation.
func CumulativeToDiffs(arr []int32) []int32 {
	if len(arr) < 2 {
		return nil
	}

	diff := make([]int32, 0, len(arr)-1)
	for i := 1; i < len(arr); i++ {
		d := arr[i] - arr[i-1]
		// A non-positive gap means the client sent timestamps out of order or
		// reused one. Counting it as zero would make the run look automated,
		// which is a verdict the caller should never get from a bad clock, so
		// the sample is discarded.
		if d <= 0 {
			continue
		}
		diff = append(diff, d)
	}
	return diff
}

// Mean returns the arithmetic mean of a sample. It returns zero for an empty
// or single-element sample, which is the honest answer: there is no spread to
// describe.
func Mean(arr []int32) float64 {
	if len(arr) == 0 {
		return 0
	}
	var sum float64
	for _, v := range arr {
		sum += float64(v)
	}
	return sum / float64(len(arr))
}

// StandardDeviation is the population standard deviation of a sample, in the
// same unit as the input.
//
// The accumulation is done in float64 throughout. Summing timestamps into an
// int32 overflows well inside a single long round, which produced a meaningless
// mean and therefore a meaningless deviation.
func StandardDeviation(arr []int32) float64 {
	n := len(arr)
	if n < 2 {
		return 0
	}

	mean := Mean(arr)

	var variance float64
	for _, v := range arr {
		d := float64(v) - mean
		variance += d * d
	}
	variance /= float64(n)

	return math.Sqrt(variance)
}

// CoefficientOfVariation is the standard deviation expressed as a fraction of
// the mean.
//
// This is the more useful measure for keystroke timing, because a slow typist
// and a fast typist should both be judged against their own pace. Raw
// deviation in milliseconds is a function of speed as much as of humanity: a
// very fast typist has a smaller deviation purely by typing faster.
func CoefficientOfVariation(arr []int32) float64 {
	if len(arr) < 2 {
		return 0
	}
	mean := Mean(arr)
	if mean == 0 {
		return 0
	}
	return StandardDeviation(arr) / mean
}

// FindMinIgnoringFirst returns the smallest value after the first.
//
// The first interval is skipped because it measures the player's reaction time
// to the round starting, which is the slowest keystroke of the run by
// construction and would otherwise always win the minimum.
func FindMinIgnoringFirst(arr []int32) int32 {
	if len(arr) <= 1 {
		return 0
	}
	min := arr[1]
	for _, v := range arr[2:] {
		if v < min {
			min = v
		}
	}
	return min
}

// Percentile returns the value at a percentile, using nearest-rank. p is in
// the range 0 to 100.
//
// The sort happens before the range is clamped. The previous version returned
// arr[0] for p <= 0 and arr[len-1] for p >= 100, which is the minimum and the
// maximum only if the caller happened to have sorted the sample already — and
// a keystroke log arrives in the order it was typed, so p0 answered with
// however long the player took to start and p100 with however long they took
// to finish.
func Percentile(arr []int32, p float64) int32 {
	if len(arr) == 0 {
		return 0
	}

	// Copy before sorting: callers pass live slices and sorting in place would
	// reorder a keystroke log.
	sorted := make([]int32, len(arr))
	copy(sorted, arr)
	insertionSort(sorted)

	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	// Nearest-rank: the smallest value at or below which at least p percent of
	// the sample falls, which is ceil(n*p/100) - 1. The previous version used
	// round(n*p/100), which for five samples at p50 picked the fourth rather
	// than the median — so the anticheat's "half the run was faster than X"
	// was off by one sample, and the error was largest for exactly the short
	// runs a bot would submit.
	rank := int(math.Ceil(float64(len(sorted))*p/100)) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

func insertionSort(a []int32) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

// CountBelow returns how many values in the sample are below a threshold.
//
// It is how "most of this run was faster than humanly possible" is
// expressed, as distinct from "one keystroke was", which is just a slip.
func CountBelow(arr []int32, threshold int32) int {
	n := 0
	for _, v := range arr {
		if v < threshold {
			n++
		}
	}
	return n
}

// FractionBelow is CountBelow as a fraction of the sample size.
func FractionBelow(arr []int32, threshold int32) float64 {
	if len(arr) == 0 {
		return 0
	}
	return float64(CountBelow(arr, threshold)) / float64(len(arr))
}

// DistinctValues counts how many distinct values a sample has.
//
// A pasted or scripted run produces long runs of identical intervals; human
// typing essentially never does. This is the cheapest reliable signal for
// that, and it needs no tuning.
func DistinctValues(arr []int32) int {
	if len(arr) == 0 {
		return 0
	}
	seen := make(map[int32]struct{}, len(arr))
	for _, v := range arr {
		seen[v] = struct{}{}
	}
	return len(seen)
}
