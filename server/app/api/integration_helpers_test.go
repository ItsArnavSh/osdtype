//go:build integration

package api_test

import "time"

// timeFixture returns a distinct, deterministic timestamp so seeded contests
// have a stable ordering without depending on the wall clock.
func timeFixture(i int) time.Time {
	return time.Date(2099, 1, 1, 0, i, 0, 0, time.UTC)
}
