package utils

import "math"

func CumulativeToDiffs(arr []int32) []int32 {
	if len(arr) == 0 {
		return nil
	}
	diff := make([]int32, len(arr))
	var prev int32
	for i, v := range arr {
		diff[i] = v - prev
		prev = v
	}
	return diff
}

func StandardDeviation(arr []int32) float64 {
	n := len(arr)
	if n == 0 || n == 1 {
		return 0
	}

	// Calculate mean as float32
	var sum int32
	for _, v := range arr {
		sum += v
	}
	mean := float64(sum) / float64(n)

	// Calculate variance
	var variance float64
	for _, v := range arr {
		variance += math.Pow(float64(v)-mean, 2)
	}
	variance = variance / float64(n)

	return math.Sqrt(variance)
}
func FindMinIgnoringFirst(arr []int32) int32 {
	if len(arr) <= 1 {
		return 0
	}

	min := arr[1] // Start with second element
	for _, v := range arr[2:] {
		if v < min {
			min = v
		}
	}
	return min
}
