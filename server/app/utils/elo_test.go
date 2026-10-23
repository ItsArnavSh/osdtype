package utils

import "testing"

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
