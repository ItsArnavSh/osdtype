package entity

import (
	"math"
	"testing"
)

func TestBotWPMScalesWithRating(t *testing.T) {
	cfg := BotConfig{RelativeSpeed: 1.5}
	last := 0.0
	for _, rating := range []int{400, 600, 1000, 1500, 2000} {
		cfg.Rating = rating
		wpm := cfg.BotWPM()
		if wpm <= last {
			t.Fatalf("wpm at %d = %.1f, not above previous %.1f", rating, wpm, last)
		}
		last = wpm
	}
}

func TestBotWPMIsNearHumanPaceAtElo(t *testing.T) {
	cases := []struct {
		rating int
		want   float64
	}{
		{rating: 400, want: 30},
		{rating: 1000, want: 60},
		{rating: 2000, want: 110},
	}
	for _, tc := range cases {
		cfg := BotConfig{Rating: tc.rating, RelativeSpeed: 1.5}
		if got := cfg.BotWPM(); math.Abs(got-tc.want) > 0.01 {
			t.Errorf("BotWPM(%d) = %.2f, want %.2f", tc.rating, got, tc.want)
		}
	}
}

func TestBotWPMPersonaSpreadStaysSmall(t *testing.T) {
	slow := BotConfig{Rating: 800, RelativeSpeed: 1.2}.BotWPM()
	normal := BotConfig{Rating: 800, RelativeSpeed: 1.5}.BotWPM()
	fast := BotConfig{Rating: 800, RelativeSpeed: 1.8}.BotWPM()
	if !(slow < normal && normal < fast) {
		t.Fatalf("persona speeds not ordered: %.1f %.1f %.1f", slow, normal, fast)
	}
	if fast/slow > 1.2 {
		t.Fatalf("persona spread too large: %.2fx", fast/slow)
	}
}
