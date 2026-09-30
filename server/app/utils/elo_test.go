//go:build unit

package utils

import (
	"testing"
)

func TestUpdateEloEmptyInput(t *testing.T) {
	got := UpdateElo(nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected an empty result, got %v", got)
	}
}

// TestUpdateEloMismatchedLengths makes sure bad input is passed straight
// through rather than panicking.
func TestUpdateEloMismatchedLengths(t *testing.T) {
	cur := []uint16{1000, 1200}
	got := UpdateElo(cur, []uint16{1})
	if len(got) != 2 || got[0] != 1000 || got[1] != 1200 {
		t.Fatalf("expected the input to be returned unchanged, got %v", got)
	}
}

// TestUpdateEloSinglePlayerIs a regression test.
//
// The K factor is 32/(n-1). With n == 1 that is 32/0, which is +Inf. Every
// per-opponent delta then became NaN, and converting NaN to uint16 yields 0,
// so a lone player's rank was reset to zero. The ranking must be preserved.
func TestUpdateEloSinglePlayerIs(t *testing.T) {
	got := UpdateElo([]uint16{1750}, []uint16{1})
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0] != 1750 {
		t.Fatalf("a single player's rank changed: got %d, want 1750", got[0])
	}
}

func TestUpdateEloTwoPlayersWinnerGainsLoserDrops(t *testing.T) {
	// Equal starting ranks, so the winner and loser should move in opposite
	// directions by the same amount.
	got := UpdateElo([]uint16{1000, 1000}, []uint16{1, 2})

	if got[0] <= 1000 {
		t.Errorf("first place should have gained, got %d", got[0])
	}
	if got[1] >= 1000 {
		t.Errorf("second place should have lost, got %d", got[1])
	}
	if got[0] == got[1] {
		t.Errorf("the two ranks should have diverged, both are %d", got[0])
	}
}

func TestUpdateEloTieLeavesRanksEqual(t *testing.T) {
	got := UpdateElo([]uint16{1000, 1000}, []uint16{1, 1})
	if got[0] != got[1] {
		t.Errorf("a tie should keep ranks equal, got %d and %d", got[0], got[1])
	}
}

func TestUpdateEloUnderdogCanGainOnAnUpset(t *testing.T) {
	// A heavily favored player finishes last.
	got := UpdateElo([]uint16{2000, 1000}, []uint16{2, 1})
	if got[0] >= 2000 {
		t.Errorf("the favorite should have lost rating, got %d", got[0])
	}
	if got[1] <= 1000 {
		t.Errorf("the underdog should have gained rating, got %d", got[1])
	}
}

// TestUpdateEloNeverNegative guards the clamp that keeps the uint16 conversion
// well defined.
func TestUpdateEloNeverNegative(t *testing.T) {
	got := UpdateElo([]uint16{0, 0, 0}, []uint16{3, 2, 1})
	for i, v := range got {
		if float64(v) < 0 {
			t.Fatalf("entry %d is negative: %d", i, v)
		}
	}
}

// TestUpdateEloZeroSumAcrossPair checks the classic property: total rating is
// conserved apart from per-player rounding.
func TestUpdateEloZeroSumAcrossPair(t *testing.T) {
	got := UpdateElo([]uint16{1200, 1000}, []uint16{1, 2})
	delta := int64(got[0]) - 1200 + (int64(got[1]) - 1000)
	if delta > 2 || delta < -2 {
		t.Errorf("total rating drifted by %d", delta)
	}
}
