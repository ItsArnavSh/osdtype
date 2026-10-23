package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func testContext(method, target string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, target, nil)
	return c
}

func TestTokenFromRequestPrefersBearerHeader(t *testing.T) {
	c := testContext(http.MethodGet, "/auth/me")
	c.Request.Header.Set("Authorization", "Bearer header-token")
	c.Request.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})

	if got := tokenFromRequest(c); got != "header-token" {
		t.Fatalf("tokenFromRequest() = %q, want header-token", got)
	}
}

func TestTokenFromRequestAcceptsQueryOnlyForWebSocketUpgrade(t *testing.T) {
	c := testContext(http.MethodGet, "/ws?token=ws-token")
	c.Request.Header.Set("Upgrade", "websocket")

	if got := tokenFromRequest(c); got != "ws-token" {
		t.Fatalf("tokenFromRequest() = %q, want ws-token", got)
	}
}

func TestTokenFromRequestIgnoresQueryForOrdinaryRequests(t *testing.T) {
	c := testContext(http.MethodGet, "/auth/me?token=leaked-token")

	if got := tokenFromRequest(c); got != "" {
		t.Fatalf("tokenFromRequest() = %q, want empty", got)
	}
}
