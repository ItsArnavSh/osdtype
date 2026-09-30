//go:build integration

package testsupport

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
)

// ResetTestData empties the tables the integration suite writes to.
//
// The tests share one throwaway database, so without this a row seeded by an
// earlier test would collide with a later one on the primary key. Deleting the
// child tables first keeps referential order tidy.
func ResetTestData(t testing.TB, db postgresql.Database) {
	t.Helper()

	ctx := context.Background()
	models := []any{
		&entity.Room_User{},
		&entity.Room{},
		&entity.Contest{},
		&entity.Task{},
		&entity.Friends{},
		&entity.User{},
	}
	for _, m := range models {
		if err := db.RawDB().WithContext(ctx).Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("could not clear %T: %v", m, err)
		}
	}
}

// NewCleanRouter is the usual integration entry point: a fresh server against
// a test database whose tables have just been emptied.
func NewCleanRouter(t testing.TB) *gin.Engine {
	t.Helper()
	ResetTestData(t, NewTestDB(t))
	return NewRouter(t)
}
