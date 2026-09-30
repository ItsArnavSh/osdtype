//go:build integration

package postgresql

import (
	"os"
	"testing"

	"go.uber.org/zap"

	"osdtyp/app/internal/testconfig"
)

// connect dials the throwaway test database through the same DSN assembly the
// application uses, so the tests exercise the real connection path rather than a
// hand-rolled one.
func connect(t *testing.T) Database {
	t.Helper()
	testconfig.Configure(t)

	db, err := ConnectDatabase(zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("could not reach the test database on %s:%s: %v",
			os.Getenv("TEST_DB_HOST"), os.Getenv("TEST_DB_PORT"), err)
	}
	return db
}
