package utils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestCodeGenRequestAndResponse(t *testing.T) {
	viper.Reset()
	var got generateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = io.WriteString(w, `{"code":"package main"}`)
	}))
	defer srv.Close()
	viper.Set("CodeGen.service_url", srv.URL)

	cg := NewCodeGen(nil)
	code, err := cg.Generate(context.Background(), "go", 12345, 120)
	if err != nil {
		t.Fatal(err)
	}
	if code != "package main" {
		t.Fatalf("code = %q", code)
	}
	if got.Name != "go" || got.Seed != 12345 || got.Tokens != 120 {
		t.Fatalf("request = %+v", got)
	}
}

func TestCodeGenFallback(t *testing.T) {
	viper.Reset()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	viper.Set("CodeGen.service_url", srv.URL)

	cg := NewCodeGen(nil)
	code, ok := cg.GenerateSnippet(context.Background(), "python", 1)
	if ok {
		t.Fatal("expected fallback")
	}
	if !strings.Contains(code, "def main") {
		t.Fatalf("unexpected fallback: %q", code)
	}
}

func TestDeriveSeedMatchesBrowser(t *testing.T) {
	cases := map[uint64]uint64{
		12345:                3778200017661327597,
		0:                    7960286522194355700,
		18446744073709551615: 16834447057089888969,
	}
	for seed, want := range cases {
		if got := DeriveSeed(seed, 1); got != want {
			t.Errorf("DeriveSeed(%d) = %d want %d", seed, got, want)
		}
	}
}

func TestSnippetStartOffset(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 80; i++ {
		b.WriteString("    let generated_value_")
		b.WriteString(strings.Repeat("x", 40))
		b.WriteString(" = 1;\n")
	}
	text := b.String()
	offset := SnippetStartOffset(99, text)
	if offset <= 0 || offset >= len(text) {
		t.Fatalf("offset = %d", offset)
	}
	if text[offset] == ' ' || text[offset] == '\n' || text[offset] == '\t' {
		t.Fatalf("offset points at whitespace")
	}
	if again := SnippetStartOffset(99, text); again != offset {
		t.Fatalf("not deterministic: %d vs %d", offset, again)
	}
	if other := SnippetStartOffset(100, text); other != offset {
		// Different seeds may legitimately pick another line, but the result
		// must always be a valid cursor.
		if other < 0 || other >= len(text) {
			t.Fatalf("invalid other offset")
		}
	}
}
