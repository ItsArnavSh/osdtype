// Package testconfig configures process-wide settings for the test suite.
//
// It deliberately depends on nothing inside the application so that any
// package, including ones whose own tests reach into unexported fields, can
// import it without creating a cycle.
package testconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// JWTKey is the signing key the suite runs under.
const JWTKey = "test_secret_key"

// Configure points viper at the throwaway test database and makes sure a JWT
// key is present.
//
// ConnectDatabase reads its DSN from viper, so any test that wants to use the
// real connection path has to populate those keys. Values come from the
// environment, defaulting to the local throwaway database started by
// "just test-db-up" (port 5433, deliberately different from the dev database).
func Configure(t testing.TB) {
	t.Helper()

	// A JWT key must exist before any test mints a token. t.Setenv restores the
	// previous value when the test finishes.
	if os.Getenv("JWTKEY") == "" {
		t.Setenv("JWTKEY", JWTKey)
	}

	set("DB.host", "TEST_DB_HOST", "localhost")
	set("DB.user", "TEST_DB_USER", "user")
	set("DB.password", "TEST_DB_PASSWORD", "password")
	set("DB.dbname", "TEST_DB_NAME", "typedata")
	set("DB.port", "TEST_DB_PORT", "5433")
	set("Core.port", "TEST_CORE_PORT", ":8080")
	// Point the codegen client at a closed port so no test can accidentally
	// depend on the real snippet service.
	set("CodeGen.service_url", "TEST_CODEGEN_URL", "http://127.0.0.1:1/generate")

	// go test runs package binaries in parallel, so two packages sharing one set
	// of tables would delete each other's rows mid-test. Each binary gets its
	// own schema; public stays on the path so extensions still resolve.
	set("DB.search_path", "TEST_DB_SCHEMA", schemaForBinary()+",public")
}

// schemaForBinary derives a schema name unique to the running test binary,
// which is named after the package under test, e.g. "scheduler.test".
func schemaForBinary() string {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".test")
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '_'
		}
	}, name)
	if name == "" {
		name = "suite"
	}
	return "test_" + name
}

// Reset clears viper's global state. Viper is process global, so a stray key
// would silently leak into the next test.
func Reset(t testing.TB) {
	t.Helper()
	viper.Reset()
}

func set(key, env, def string) {
	v := os.Getenv(env)
	if v == "" {
		v = def
	}
	viper.Set(key, v)
}
