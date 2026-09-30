//go:build unit

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// newPingRouter builds a gin engine carrying only the unauthenticated /ping
// route. Testing it in isolation keeps the unit suite free of any database
// dependency.
func newPingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop().Sugar()

	srv := Server{logger: logger, gin_engine: gin.New()}
	srv.gin_engine.GET("/ping", srv.ping)
	return srv.gin_engine
}

func TestPingReturnsPong(t *testing.T) {
	router := newPingRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"reply":"pong"}` {
		t.Fatalf(`expected {"reply":"pong"}, got %s`, got)
	}
}

// TestPingSetsJSONContentType guards the client contract: the frontend reads
// the body with res.json(), so a non-JSON content type would break it.
func TestPingSetsJSONContentType(t *testing.T) {
	router := newPingRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", ct)
	}
}
