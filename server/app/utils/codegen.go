package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// TokensPerSnippet must match the browser's game generator.
const TokensPerSnippet = 120

type CodeGen struct {
	logger *zap.SugaredLogger
	client *http.Client
	url    string
}

type generateRequest struct {
	Name   string `json:"name"`
	Seed   uint64 `json:"seed"`
	Tokens int    `json:"tokens"`
}

type generateResponse struct {
	Code string `json:"code"`
}

func NewCodeGen(logger *zap.SugaredLogger) CodeGen {
	url := strings.TrimSpace(viper.GetString("CodeGen.service_url"))
	if strings.EqualFold(url, "disabled") || strings.EqualFold(url, "off") {
		url = ""
	}
	return CodeGen{
		logger: logger,
		client: &http.Client{Timeout: 8 * time.Second},
		url:    url,
	}
}

// Generate asks the Rust resrap service for a deterministic snippet. The
// request and response are deliberately identical to the WASM wrapper so the
// server and the browser produce byte-for-byte the same code.
func (c *CodeGen) Generate(ctx context.Context, language string, seed uint64, tokens int) (string, error) {
	if tokens <= 0 {
		tokens = TokensPerSnippet
	}
	if c.url == "" {
		return "", fmt.Errorf("codegen service url is not configured")
	}
	body, err := json.Marshal(generateRequest{Name: language, Seed: seed, Tokens: tokens})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("codegen service unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("codegen service status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode codegen response: %w", err)
	}
	if out.Code == "" {
		return "", fmt.Errorf("codegen returned an empty snippet")
	}
	return out.Code, nil
}

// GenerateSnippet returns a snippet and never fails: if the generator is
// unavailable a small bundled snippet is used so a match can still finish.
// The selected snippet is sent to every client, so scoring remains correct.
func (c *CodeGen) GenerateSnippet(ctx context.Context, language string, seed uint64) (string, bool) {
	code, err := c.Generate(ctx, language, seed, TokensPerSnippet)
	if err == nil {
		return code, true
	}
	if c.url != "" && c.logger != nil {
		c.logger.Warnw("falling back to bundled snippet", "language", language, "error", err)
	}
	return FallbackSnippet(language), false
}

// SnippetStartOffset reproduces the browser's "skip the imports" rule. Both
// sides must agree on where a run begins or results will not be comparable.
func SnippetStartOffset(seed uint64, text string) int {
	const skipLines = 15
	const tailRoom = 1400

	parts := strings.Split(text, "\n")
	offsets := make([]int, len(parts))
	acc := 0
	for i, part := range parts {
		offsets[i] = acc
		acc += len(part) + 1
	}

	var candidates []int
	for i := skipLines; i < len(parts); i++ {
		if len(strings.TrimSpace(parts[i])) <= 4 {
			continue
		}
		if len(text)-offsets[i] < tailRoom {
			continue
		}
		candidates = append(candidates, i)
	}

	startLine := 0
	if len(candidates) > 0 {
		pick := DeriveSeed(seed, 1)
		startLine = candidates[pick%uint64(len(candidates))]
	}

	cursorAt := offsets[startLine]
	for cursorAt < len(text) && strings.ContainsRune(" \n\t", rune(text[cursorAt])) {
		cursorAt++
	}
	// Short snippets (notably the bundled fallbacks) have no line that
	// leaves 1400 bytes of tail room. Starting at the offset of their last
	// line can point at the final newline and produce an empty target. Fall
	// back to the beginning so the run always has something to type.
	if cursorAt >= len(text) {
		return 0
	}
	return cursorAt
}

// DeriveSeed mirrors app/src/lib/core/codegen.ts:deriveSeed (SplitMix64).
func DeriveSeed(seed uint64, salt uint64) uint64 {
	const gamma = uint64(0x9e3779b97f4a7c15)
	z := seed + salt*gamma + gamma
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// FallbackSnippet is intentionally tiny; it only runs when the generator
// service is down.
func FallbackSnippet(language string) string {
	switch language {
	case "c":
		return "#include <stdio.h>\n\nint main(void) {\n    int total = 0;\n    for (int i = 0; i < 10; i++) {\n        total += i;\n    }\n    printf(\"%d\\n\", total);\n    return 0;\n}\n"
	case "cpp":
		return "#include <iostream>\n#include <vector>\n\nint main() {\n    std::vector<int> values{1, 2, 3, 4};\n    int total = 0;\n    for (int value : values) {\n        total += value;\n    }\n    std::cout << total << std::endl;\n    return 0;\n}\n"
	case "go":
		return "package main\n\nimport \"fmt\"\n\nfunc main() {\n    total := 0\n    for i := 0; i < 10; i++ {\n        total += i\n    }\n    fmt.Println(total)\n}\n"
	case "haskell":
		return "module Main where\n\nmain :: IO ()\nmain = do\n    let values = [1, 2, 3, 4]\n    print (sum values)\n"
	case "html":
		return "<!doctype html>\n<html>\n  <head>\n    <title>Type</title>\n  </head>\n  <body>\n    <h1>OSDTYP</h1>\n    <p>Keep typing.</p>\n  </body>\n</html>\n"
	case "java":
		return "import java.util.*;\n\npublic class Main {\n    public static void main(String[] args) {\n        int total = 0;\n        for (int i = 0; i < 10; i++) {\n            total += i;\n        }\n        System.out.println(total);\n    }\n}\n"
	case "js":
		return "function sum(values) {\n    return values.reduce((total, value) => total + value, 0);\n}\n\nconst values = [1, 2, 3, 4];\nconsole.log(sum(values));\n"
	case "python":
		return "def main():\n    values = [1, 2, 3, 4]\n    total = sum(values)\n    print(total)\n\nif __name__ == \"__main__\":\n    main()\n"
	case "rust":
		return "fn main() {\n    let mut total = 0;\n    for value in 0..10 {\n        total += value;\n    }\n    println!(\"{}\", total);\n}\n"
	case "sql":
		return "CREATE TABLE players (\n    id INTEGER PRIMARY KEY,\n    name TEXT NOT NULL\n);\n\nSELECT name FROM players ORDER BY name;\n"
	default:
		return "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"keep typing\")\n}\n"
	}
}
