package utils

import (
	"math"
	"osdtyp/app/entity"
	"testing"
)

func TestCalculateWPM(t *testing.T) {
	res := Calculate_WPM(entity.WPM{
		OriginalSnippet: "the quick brown fox",
		UserSnippet:     "the quick brown fox",
		DurationMS:      60000,
	})
	if res.Correct != 16 {
		t.Fatalf("correct = %d", res.Correct)
	}
	if math.Abs(res.RAW-3.2) > 0.001 || math.Abs(res.WPM-3.2) > 0.001 {
		t.Fatalf("wpm = %f raw = %f", res.WPM, res.RAW)
	}
	if math.Abs(res.Accuracy-1.0) > 0.001 {
		t.Fatalf("accuracy = %f", res.Accuracy)
	}
}

func TestCalculateWPMWithMistakes(t *testing.T) {
	res := Calculate_WPM(entity.WPM{
		OriginalSnippet: "abcdefghij",
		UserSnippet:     "abcdeXghij",
		DurationMS:      30000,
	})
	if res.Correct != 9 || res.Wrong != 1 {
		t.Fatalf("correct=%d wrong=%d", res.Correct, res.Wrong)
	}
	if math.Abs(res.Accuracy-0.9) > 0.001 {
		t.Fatalf("accuracy = %f", res.Accuracy)
	}
	if math.Abs(res.WPM-(float64(10)/5.0/0.5*0.9)) > 0.001 {
		t.Fatalf("wpm = %f", res.WPM)
	}
}

func TestCalculateWPMEmpty(t *testing.T) {
	res := Calculate_WPM(entity.WPM{OriginalSnippet: "abc", UserSnippet: "", DurationMS: 1000})
	if res.WPM != 0 || res.Accuracy != 0 {
		t.Fatalf("expected zero result, got %+v", res)
	}
}
