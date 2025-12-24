package utils

import "math"

func UpdateElo(current []uint16, scores []uint16) []uint16 {
	n := len(current)
	if n == 0 || len(scores) != n {
		return current
	}

	// Work in float internally
	newElo := make([]float32, n)
	for i := range n {
		newElo[i] = float32(current[i])
	}

	K := float32(32.0 / float32(n-1))

	for i := range n {
		curPlace := scores[i]
		curElo := float32(current[i])

		var eloChange float32 = 0

		for j := range n {
			if i == j {
				continue
			}

			oppPlace := scores[j]
			oppElo := float32(current[j])

			// S value
			var S float32
			if curPlace < oppPlace {
				S = 1.0
			} else if curPlace == oppPlace {
				S = 0.5
			} else {
				S = 0.0
			}

			// Expected score
			EA := float32(
				1.0 / (1.0 + math.Pow(10, float64((oppElo-curElo)/400.0))),
			)

			// Same rounding behavior as Java
			eloChange += float32(math.Round(float64(K * (S - EA))))
		}

		newElo[i] = curElo + eloChange
	}

	// Convert back to uint16
	out := make([]uint16, n)
	for i := range n {
		if newElo[i] < 0 {
			newElo[i] = 0 // safety clamp
		}
		out[i] = uint16(newElo[i])
	}

	return out
}
