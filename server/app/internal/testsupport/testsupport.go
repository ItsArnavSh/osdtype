// Package testsupport builds an application server wired to a test database
// and hands back a ready-to-use *gin.Engine.
//
// It depends on testconfig rather than inlining the configuration, so packages
// whose own tests reach into unexported fields can share the same setup without
// creating an import cycle.
package testsupport

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"osdtyp/app/api"
	"osdtyp/app/api/auth"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/testconfig"
)

// Logger returns a development logger. The gin loggers are silenced by
// gin.TestMode so test output stays readable; use -v to see everything.
func Logger() *zap.SugaredLogger {
	gin.SetMode(gin.TestMode)
	l, err := zap.NewDevelopment()
	if err != nil {
		return zap.NewNop().Sugar()
	}
	return l.Sugar()
}

// NewTestDB connects to the test database described by the TEST_DB_* env
// variables, running migrations. It fails the calling test if the database is
// unreachable: an integration test that cannot reach its database should error
// loudly rather than pass vacuously.
func NewTestDB(t testing.TB) postgresql.Database {
	t.Helper()
	testconfig.Configure(t)

	db, err := postgresql.ConnectDatabase(Logger())
	if err != nil {
		t.Fatalf("could not connect to the test database: %v", err)
	}
	return db
}

// NewRouter builds the full server, routes included, against the test database
// and returns its engine. This is the integration entry point.
func NewRouter(t testing.TB) *gin.Engine {
	t.Helper()
	db := NewTestDB(t)
	srv, err := api.NewServerWithDB(Logger(), db)
	if err != nil {
		t.Fatalf("could not build the server: %v", err)
	}
	srv.SetupRoutes()
	return srv.Engine()
}

// TokenForUser mints a valid JWT whose subject is the given user id, matching
// what AuthMiddleware expects in the "userID" context value.
func TokenForUser(t testing.TB, userID uint32) string {
	t.Helper()
	tok, err := auth.GenerateJWT(formatUint(userID))
	if err != nil {
		t.Fatalf("could not mint a JWT: %v", err)
	}
	return tok
}
