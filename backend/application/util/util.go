package util

func CumulativeToDiffs(arr []int64) []int64 {
	if len(arr) == 0 {
		return nil
	}
	diff := make([]int64, len(arr))
	var prev int64
	for i, v := range arr {
		diff[i] = v - prev
		prev = v
	}
	return diff
}
