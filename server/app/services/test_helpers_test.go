package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"osdtyp/app/core"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	rediswrap "osdtyp/app/internal/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	sqlite "github.com/glebarez/sqlite"
)

type testEnv struct {
	svc    *ServiceLayer
	core   *core.CodeCore
	db     *postgresql.Database
	rdb    *rediswrap.Client
	mr     *miniredis.Miniredis
	gdb    *gorm.DB
	server *httptest.Server
}

func newTestEnv(t *testing.T, snippet string) *testEnv {
	t.Helper()
	return newTestEnvWith(t, snippet, nil)
}

func newTestEnvWith(t *testing.T, snippet string, setup func()) *testEnv {
	t.Helper()
	viper.Reset()
	viper.Set("Core.frontend_url", "http://localhost:5173")
	viper.Set("Game.countdown_ms", 15)
	viper.Set("Matchmaking.tick_ms", 25)
	viper.Set("Matchmaking.base_range", 500)
	viper.Set("Matchmaking.growth_range", 50)
	viper.Set("Matchmaking.max_players", 4)
	viper.Set("Matchmaking.bot_after_seconds", 30)
	viper.Set("Contests.lobby_minutes", 1)
	if setup != nil {
		setup()
	}

	if snippet == "" {
		snippet = generatedSnippet()
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name   string `json:"name"`
			Seed   uint64 `json:"seed"`
			Tokens int    `json:"tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": snippet})
	}))
	viper.Set("CodeGen.service_url", srv.URL)

	logger, _ := zap.NewDevelopment()
	logger = logger.Named("test")
	sugar := logger.Sugar()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	gdb, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := postgresql.NewFromGorm(gdb, sugar)
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	mr := miniredis.RunT(t)
	raw := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	rdb := rediswrap.NewFromRedis(raw)
	coreLayer := core.NewCodeCore(sugar, db, rdb)
	svc, err := NewServiceLayer(sugar, coreLayer, db)
	if err != nil {
		t.Fatalf("services: %v", err)
	}
	if err := svc.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	env := &testEnv{svc: svc, core: coreLayer, db: db, rdb: rdb, mr: mr, gdb: gdb, server: srv}
	t.Cleanup(func() {
		coreLayer.Stop()
		srv.Close()
		_ = raw.Close()
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
		mr.Close()
	})
	return env
}

func generatedSnippet() string {
	var b strings.Builder
	b.WriteString("package generated\n\nimport \"fmt\"\n\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "// generated line %02d %s\n", i, strings.Repeat("x", 46))
		b.WriteString("func generated() { fmt.Println(\"keep typing\") }\n")
	}
	return b.String()
}

func (e *testEnv) login(t *testing.T, name string) entity.User {
	t.Helper()
	u, err := e.svc.LoginUser(context.Background(), name, "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("login %s: %v", name, err)
	}
	return u
}

func (e *testEnv) state(t *testing.T, uid uint32) StateView {
	t.Helper()
	st, err := e.svc.State(context.Background(), uid)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	return st
}

func nonWhitespace(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var _ = os.Getenv
var _ = time.Now

func viperSetBotAfter(seconds int) {
	viper.Set("Matchmaking.bot_after_seconds", seconds)
}
