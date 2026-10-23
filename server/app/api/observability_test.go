package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRequestIDHeaderAndErrorBody(t *testing.T) {
	env := newAPIEnv(t)
	client := &apiClient{t: t, base: env.http.URL}

	resp, _ := client.do(http.MethodGet, "/ping", nil, nil)
	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Fatal("missing generated X-Request-ID")
	}

	const supplied = "client-request-123"
	resp, _ = client.do(http.MethodGet, "/ping", nil, map[string]string{"X-Request-ID": supplied})
	if got := resp.Header.Get("X-Request-ID"); got != supplied {
		t.Fatalf("request id = %q, want %q", got, supplied)
	}

	resp, raw := client.do(http.MethodPost, "/bugs", []byte("{"), map[string]string{"X-Request-ID": supplied})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid json status %d: %s", resp.StatusCode, raw)
	}
	body := decode[apiError](t, raw)
	if body.RequestID != supplied {
		t.Fatalf("error request id = %q, want %q", body.RequestID, supplied)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	env := newAPIEnv(t)
	client := &apiClient{t: t, base: env.http.URL}

	resp, raw := client.do(http.MethodGet, "/metrics", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics status %d: %s", resp.StatusCode, raw)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("metrics content type = %q", contentType)
	}
	body := string(raw)
	for _, metric := range []string{
		"osdtyp_uptime_seconds",
		"osdtyp_http_requests_total",
		"osdtyp_http_responses_total{class=\"2xx\"}",
		"osdtyp_go_goroutines",
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("metrics missing %q:\n%s", metric, body)
		}
	}
}

func TestMetricsToken(t *testing.T) {
	env := newAPIEnv(t)
	viper.Set("Metrics.token", "metrics-secret")
	client := &apiClient{t: t, base: env.http.URL}

	resp, _ := client.do(http.MethodGet, "/metrics", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated metrics status %d", resp.StatusCode)
	}

	resp, raw := client.do(http.MethodGet, "/metrics", nil, map[string]string{
		"Authorization": "Bearer metrics-secret",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated metrics status %d: %s", resp.StatusCode, raw)
	}
}
