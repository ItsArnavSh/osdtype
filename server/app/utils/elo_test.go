package utils

import (
	"osdtyp/app/entity"
	"testing"
)

func TestUpdateEloTwoPlayers(t *testing.T) {
	got := UpdateElo([]int{1000, 1000}, []int{1, 2})
	if got[0] != 1016 || got[1] != 984 {
		t.Fatalf("unexpected elo: %v", got)
	}
	if sum := got[0] + got[1]; sum != 2000 {
		t.Fatalf("elo is not zero-sum: %v", got)
	}
}

func TestUpdateEloFreeForAll(t *testing.T) {
	got := UpdateElo([]int{1000, 1000, 1000}, []int{1, 2, 3})
	if got[0] <= got[1] || got[1] <= got[2] {
		t.Fatalf("order not preserved: %v", got)
	}
}

func TestUpdateEloFloor(t *testing.T) {
	got := UpdateElo([]int{0, 2000}, []int{2, 1})
	if got[0] < 0 {
		t.Fatalf("rating went below floor: %v", got)
	}
}

func TestRankTiers(t *testing.T) {
	cases := []struct {
		rating int
		tier   string
		div    int
	}{
		{0, "Bronze", 4},
		{199, "Bronze", 1},
		{200, "Silver", 4},
		{400, "Gold", 4},
		{800, "Diamond", 4},
		{1000, "Diamond", 2},
		{1200, "Master", 4},
		{1600, "Grandmaster", 0},
		{2500, "Grandmaster", 0},
	}
	for _, c := range cases {
		got := entity.RankFor(c.rating)
		if got.Tier != c.tier || got.Division != c.div {
			t.Errorf("rating %d: got %s/%d want %s/%d", c.rating, got.Tier, got.Division, c.tier, c.div)
		}
	}
}
