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
}

func TestCumulativeToDiffs(t *testing.T) {
	got := CumulativeToDiffs([]int32{100, 250, 400})
	want := []int32{100, 150, 150}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestCumulativeToDiffsHandlesDecrease(t *testing.T) {
	// A reset or seek can make the cumulative series drop, which must show up
	// as a negative delta rather than being clamped.
	got := CumulativeToDiffs([]int32{100, 50})
	if len(got) != 2 || got[0] != 100 || got[1] != -50 {
		t.Fatalf("expected [100 -50], got %v", got)
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
