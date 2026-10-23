package postgresql

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDatabaseDSNFromURL(t *testing.T) {
	viper.Reset()
	viper.Set("DB.url", "postgresql://postgres.project:secret@db.example.test:5432/postgres?sslmode=require")
	dsn, target, err := databaseDSN()
	if err != nil {
		t.Fatalf("databaseDSN: %v", err)
	}
	if dsn != "postgresql://postgres.project:secret@db.example.test:5432/postgres?sslmode=require" {
		t.Fatalf("unexpected dsn: %s", dsn)
	}
	if target != "db.example.test:5432/postgres" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestDatabaseDSNFromURLAddsExplicitSSLMode(t *testing.T) {
	viper.Reset()
	t.Setenv("DB_SSLMODE", "require")
	viper.Set("DB.url", "postgresql://postgres.project:secret@db.example.test:5432/postgres")
	dsn, _, err := databaseDSN()
	if err != nil {
		t.Fatalf("databaseDSN: %v", err)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("sslmode missing: %s", dsn)
	}
}

func TestDatabaseDSNFromPartsUsesSSLMode(t *testing.T) {
	viper.Reset()
	viper.Set("DB.host", "db.example.test")
	viper.Set("DB.user", "postgres.project")
	viper.Set("DB.password", "p@ss word")
	viper.Set("DB.dbname", "postgres")
	viper.Set("DB.port", 5432)
	viper.Set("DB.sslmode", "require")
	dsn, _, err := databaseDSN()
	if err != nil {
		t.Fatalf("databaseDSN: %v", err)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("sslmode missing: %s", dsn)
	}
	if !strings.Contains(dsn, "p%40ss+word") && !strings.Contains(dsn, "p%40ss%20word") {
		t.Fatalf("password was not escaped: %s", dsn)
	}
}
