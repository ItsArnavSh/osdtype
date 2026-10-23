package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestSecurityHeaders(t *testing.T) {
	env := newAPIEnv(t)

	resp, err := http.Get(env.http.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status: %d", resp.StatusCode)
	}
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for key, want := range headers {
		if got := resp.Header.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("missing content security policy")
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("missing request id")
	}
}

func TestOversizedRequestBodyRejected(t *testing.T) {
	env := newAPIEnv(t)

	body := strings.NewReader(strings.Repeat("x", int(maxRequestBodyBytes)+1))
	req, err := http.NewRequest(http.MethodPost, env.http.URL+"/bugs", body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413: %s", resp.StatusCode, raw)
	}
	payload := decode[apiError](t, raw)
	if payload.Code != "payload_too_large" {
		t.Fatalf("code = %q, want payload_too_large", payload.Code)
	}
}

func TestProductionConfigRejectsInsecureDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("GIN_MODE", "release")
	viper.Set("Core.dev_login", false)
	viper.Set("Core.jwt_key", "dev-insecure-change-me")
	viper.Set("Core.cookie_secure", true)
	viper.Set("Core.cookie_samesite", "none")
	if err := validateProductionConfig(); err == nil {
		t.Fatal("expected default JWT key to be rejected")
	}

	viper.Set("Core.jwt_key", strings.Repeat("a", 32))
	if err := validateProductionConfig(); err != nil {
		t.Fatalf("strong key rejected: %v", err)
	}

	viper.Set("Core.cookie_secure", false)
	if err := validateProductionConfig(); err == nil {
		t.Fatal("expected SameSite=None without Secure to be rejected")
	}
}
