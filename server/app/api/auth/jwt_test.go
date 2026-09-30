//go:build unit

package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestMain(m *testing.M) {
	// The signing key is resolved from the environment on each call, so
	// setting it here is enough.
	_ = os.Setenv("JWTKEY", "test_secret_key")
	code := m.Run()
	_ = os.Unsetenv("JWTKEY")
	os.Exit(code)
}

// signWithKey lets the tamper tests build a token with an arbitrary secret.
func signWithKey(t *testing.T, claims jwt.RegisteredClaims, key []byte) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("could not sign: %v", err)
	}
	return tok
}

func TestGenerateAndValidateRoundTrip(t *testing.T) {
	tok, err := GenerateJWT("12345")
	if err != nil {
		t.Fatalf("could not generate a token: %v", err)
	}
	got, err := ValidateJWT(tok)
	if err != nil {
		t.Fatalf("could not validate the token: %v", err)
	}
	if got != "12345" {
		t.Fatalf("expected subject 12345, got %q", got)
	}
}

func TestValidateRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "not-a-token", "a.b.c", "...."} {
		if _, err := ValidateJWT(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestValidateRejectsTamperedToken(t *testing.T) {
	tok, err := GenerateJWT("1")
	if err != nil {
		t.Fatalf("could not generate a token: %v", err)
	}
	// Flip a character in the signature.
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape: %s", tok)
	}
	sig := []byte(parts[2])
	sig[0] = 'z'
	if _, err := ValidateJWT(strings.Join(parts[:2], ".") + "." + string(sig)); err == nil {
		t.Fatal("expected a tampered token to be rejected")
	}
}

func TestValidateRejectsExpiredToken(t *testing.T) {
	tok := signWithKey(t, jwt.RegisteredClaims{
		Subject:   "1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}, []byte(os.Getenv("JWTKEY")))

	if _, err := ValidateJWT(tok); err == nil {
		t.Fatal("expected an expired token to be rejected")
	}
}

// TestValidateRejectsForeignKey covers a token signed with the right algorithm
// but the wrong secret.
func TestValidateRejectsForeignKey(t *testing.T) {
	tok := signWithKey(t, jwt.RegisteredClaims{
		Subject:   "1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}, []byte("a_completely_different_secret"))

	if _, err := ValidateJWT(tok); err == nil {
		t.Fatal("expected a token signed with another key to be rejected")
	}
}

// TestValidateRejectsEmptySubject guards against a token that carries no
// subject: the handlers treat a missing user as anonymous, so an empty subject
// should be refused rather than producing user id 0.
func TestValidateRejectsEmptySubject(t *testing.T) {
	tok := signWithKey(t, jwt.RegisteredClaims{
		Subject:   "",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}, []byte(os.Getenv("JWTKEY")))

	got, err := ValidateJWT(tok)
	if err == nil && got == "" {
		t.Fatal("expected an empty subject to be rejected")
	}
}

// TestValidateRejectsNonHMAC guards against the classic algorithm-confusion
// attack where a token claims to be signed with "none".
func TestValidateRejectsNonHMAC(t *testing.T) {
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject: "1",
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("could not sign: %v", err)
	}
	if _, err := ValidateJWT(tok); err == nil {
		t.Fatal("expected an alg=none token to be rejected")
	}
}

// ---------------------------------------------------------------- middleware

func newMiddlewareEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/probe", func(c *gin.Context) {
		uid, err := GetUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"authed": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"authed": true, "uid": uid})
	})
	return r
}

func TestMiddlewarePassesThroughWithoutToken(t *testing.T) {
	w := httptest.NewRecorder()
	newMiddlewareEngine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareAcceptsBearerHeader(t *testing.T) {
	tok, err := GenerateJWT("777")
	if err != nil {
		t.Fatalf("could not generate a token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	w := httptest.NewRecorder()
	newMiddlewareEngine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "777") {
		t.Errorf("expected the user id in the body, got %s", w.Body.String())
	}
}

func TestMiddlewareAcceptsCookie(t *testing.T) {
	tok, err := GenerateJWT("888")
	if err != nil {
		t.Fatalf("could not generate a token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: tok})

	w := httptest.NewRecorder()
	newMiddlewareEngine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer garbage")

	w := httptest.NewRecorder()
	newMiddlewareEngine().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// TestSetAndGetUserID is a regression test.
//
// SetUserID stored a uint32 in the gin context while GetUserID read it back
// with c.GetString, which yields "" for any non-string value. The pair therefore
// never round-tripped and every caller saw "No Active User Logged in" even
// though a user had just been set.
func TestSetAndGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	SetUserID(c, 4242)

	uid, err := GetUserID(c)
	if err != nil {
		t.Fatalf("expected to read the user id back, got %v", err)
	}
	if uid != 4242 {
		t.Fatalf("expected 4242, got %d", uid)
	}
}

func TestGetUserIDWithoutValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if _, err := GetUserID(c); err == nil {
		t.Fatal("expected an error when no user is set")
	}
}

func TestIsAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	empty, _ := gin.CreateTestContext(httptest.NewRecorder())
	if IsAuth(empty) {
		t.Error("expected IsAuth to be false on a fresh context")
	}

	SetUserID(empty, 1)
	if !IsAuth(empty) {
		t.Error("expected IsAuth to be true once a user is set")
	}
}

func TestGetUserIDHandlesStringFromMiddleware(t *testing.T) {
	// The middleware stores the token subject, which is a string. Reading it
	// back has to work for that shape too.
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("userID", strconv.FormatUint(99, 10))

	uid, err := GetUserID(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != 99 {
		t.Fatalf("expected 99, got %d", uid)
	}
}
