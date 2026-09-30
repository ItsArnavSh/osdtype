//go:build unit

package utils

import (
	"math"
	"testing"

	"osdtyp/app/entity"
)

func TestCleanSnippetStripsWhitespace(t *testing.T) {
	got := cleanSnippet("a b\nc")
	want := []rune{'a', 'b', 'c'}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestCleanSnippetHandlesEmptyAndUnicode(t *testing.T) {
	if got := cleanSnippet("   \n\n  "); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
	// Multi-byte runes must survive intact.
	got := cleanSnippet("héllo")
	if len(got) != 5 {
		t.Fatalf("expected 5 runes, got %d (%v)", len(got), got)
	}
}

func TestCalculateWPMZeroDuration(t *testing.T) {
	// A zero duration would divide by zero; the function must return the zero
	// result instead of +Inf.
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "hello",
		UserSnippet:     "hello",
		DurationMS:      0,
	})
	if got.WPM != 0 || got.RAW != 0 {
		t.Fatalf("expected the zero result, got %+v", got)
	}
}

func TestCalculateWPMZeroInput(t *testing.T) {
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "hello",
		UserSnippet:     "",
		DurationMS:      1000,
	})
	if got.WPM != 0 || got.RAW != 0 {
		t.Fatalf("expected the zero result, got %+v", got)
	}
}

func TestCalculateWPMPerfectRun(t *testing.T) {
	// 5 characters in 1 minute is the canonical 1 WPM.
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "hello",
		UserSnippet:     "hello",
		DurationMS:      60000,
	})
	if math.Abs(float64(got.RAW)-1.0) > 0.001 {
		t.Errorf("expected raw 1.0, got %v", got.RAW)
	}
	if math.Abs(float64(got.Accuracy)-1.0) > 0.001 {
		t.Errorf("expected accuracy 1.0, got %v", got.Accuracy)
	}
	if got.Correct != 5 || got.Wrong != 0 {
		t.Errorf("expected 5 correct and 0 wrong, got %d/%d", got.Correct, got.Wrong)
	}
	if math.Abs(float64(got.WPM)-1.0) > 0.001 {
		t.Errorf("expected net 1.0, got %v", got.WPM)
	}
}

func TestCalculateWPMCountsMistakes(t *testing.T) {
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "hello",
		UserSnippet:     "hxllo",
		DurationMS:      60000,
	})
	if got.Correct != 4 || got.Wrong != 1 {
		t.Errorf("expected 4 correct and 1 wrong, got %d/%d", got.Correct, got.Wrong)
	}
	if math.Abs(float64(got.Accuracy)-0.8) > 0.001 {
		t.Errorf("expected accuracy 0.8, got %v", got.Accuracy)
	}
}

// TestCalculateWPMCountsOverflowCharacters covers typing past the end of the
// snippet: the extra characters must be counted as errors.
func TestCalculateWPMCountsOverflowCharacters(t *testing.T) {
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "abc",
		UserSnippet:     "abcdef",
		DurationMS:      60000,
	})
	if got.Correct != 3 {
		t.Errorf("expected 3 correct, got %d", got.Correct)
	}
	if got.Wrong != 3 {
		t.Errorf("expected 3 wrong (the overflow), got %d", got.Wrong)
	}
}

func TestCalculateWPMIgnoresWhitespaceWhenComparing(t *testing.T) {
	// "h e l l o" typed with different spacing is the same text.
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "h e l l o",
		UserSnippet:     "hello",
		DurationMS:      60000,
	})
	if got.Wrong != 0 {
		t.Errorf("expected no mistakes, got %d wrong", got.Wrong)
	}
}

// TestCalculateWPMTruncatedInput covers finishing early: only the overlapping
// prefix is compared.
func TestCalculateWPMTruncatedInput(t *testing.T) {
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "abcdef",
		UserSnippet:     "abc",
		DurationMS:      60000,
	})
	if got.Correct != 3 || got.Wrong != 0 {
		t.Errorf("expected 3 correct and 0 wrong, got %d/%d", got.Correct, got.Wrong)
	}
}

func TestCalculateWPMRawIsNotScaledByAccuracy(t *testing.T) {
	// raw counts everything typed; net applies accuracy. They must differ when
	// the run has mistakes.
	got := Calculate_WPM(entity.WPM{
		OriginalSnippet: "hello",
		UserSnippet:     "xxxxx",
		DurationMS:      60000,
	})
	if got.RAW == got.WPM {
		t.Errorf("expected raw and net to differ, both are %v", got.RAW)
	}
	if got.WPM != 0 {
		t.Errorf("expected net 0 for a fully wrong run, got %v", got.WPM)
	}
	if got.Correct != 0 {
		t.Errorf("expected 0 correct, got %d", got.Correct)
	}
}
