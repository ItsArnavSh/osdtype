package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

type apiEnv struct {
	server *Server
	http   *httptest.Server
	db     *postgresql.Database
	rdb    *rediswrap.Client
}

func newAPIEnv(t *testing.T) *apiEnv {
	t.Helper()
	viper.Reset()
	viper.Set("Core.frontend_url", "http://localhost:5173")
	viper.Set("Core.dev_login", true)
	viper.Set("Core.jwt_key", "test-key")
	viper.Set("Game.countdown_ms", 20)
	viper.Set("Matchmaking.tick_ms", 25)
	viper.Set("Matchmaking.base_range", 500)
	viper.Set("Matchmaking.bot_after_seconds", 60)
	viper.Set("Contests.lobby_minutes", 1)

	codegen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "package generated\n\nfunc generated() {}\n// " + strings.Repeat("typing ", 200),
		})
	}))
	viper.Set("CodeGen.service_url", codegen.URL)

	logger := zap.NewNop().Sugar()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	gdb, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	db := postgresql.NewFromGorm(gdb, logger)
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mr := miniredis.RunT(t)
	raw := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	rdb := rediswrap.NewFromRedis(raw)

	server, err := NewServerWithDeps(logger, db, rdb)
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	httpServer := httptest.NewServer(server.Engine())

	t.Cleanup(func() {
		httpServer.Close()
		server.Close()
		codegen.Close()
		_ = raw.Close()
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
		mr.Close()
	})
	return &apiEnv{server: server, http: httpServer, db: db, rdb: rdb}
}

type apiClient struct {
	t      *testing.T
	base   string
	cookie *http.Cookie
}

func (a *apiEnv) client(t *testing.T, username string) *apiClient {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username})
	resp, err := http.Post(a.http.URL+"/auth/dev-login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("dev login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dev login status: %d", resp.StatusCode)
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("no auth cookie")
	}
	return &apiClient{t: t, base: a.http.URL, cookie: cookies[0]}
}

func (c *apiClient) do(method, path string, body any, headers map[string]string) (*http.Response, []byte) {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(c.cookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return out
}
