package utils

import "math"

// UpdateElo applies free-for-all Elo to a leaderboard. positions[i] is the
// finishing position (1-based) of current[i]. Returns the new ratings.
func UpdateElo(current []int, positions []int) []int {
	n := len(current)
	if n == 0 || len(positions) != n {
		return current
	}
	if n == 1 {
		return []int{current[0]}
	}

	newElo := make([]float64, n)
	for i := range n {
		newElo[i] = float64(current[i])
	}

	k := 32.0 / float64(n-1)
	for i := range n {
		curElo := float64(current[i])
		delta := 0.0
		for j := range n {
			if i == j {
				continue
			}
			oppElo := float64(current[j])
			var score float64
			switch {
			case positions[i] < positions[j]:
				score = 1
			case positions[i] == positions[j]:
				score = 0.5
			default:
				score = 0
			}
			expected := 1.0 / (1.0 + math.Pow(10, (oppElo-curElo)/400.0))
			delta += math.Round(k * (score - expected))
		}
		newElo[i] += delta
	}

	out := make([]int, n)
	for i := range n {
		if newElo[i] < 0 {
			newElo[i] = 0
		}
		out[i] = int(math.Round(newElo[i]))
	}
	return out
}

// EloDelta returns the delta for each player. Mirrors UpdateElo.
func EloDelta(current []int, positions []int) []int {
	updated := UpdateElo(current, positions)
	out := make([]int, len(current))
	for i := range current {
		out[i] = updated[i] - current[i]
	}
	return out
}
