package testsupport

import "strconv"

func formatUint(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
