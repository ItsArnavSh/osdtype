package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type CodeGen struct {
	logger *zap.SugaredLogger
	client *http.Client
	url    string
}

type generateRequest struct {
	Name   string `json:"name"`
	Seed   uint32 `json:"seed"`
	Tokens int    `json:"tokens"`
}

type generateResponse struct {
	Code []string `json:"code"`
}

func NewCodeGen(logger *zap.SugaredLogger) CodeGen {
	return CodeGen{
		logger: logger,
		client: &http.Client{},
		url:    viper.GetString("CodeGen.service_url"),
	}
}

// Generate asks the snippet service for code in the named language.
//
// The service is a separate Rust process, so a failure here means a player gets
// an empty snippet rather than a crash. Every failure path is logged and turned
// into an empty string.
func (c *CodeGen) Generate(ctx context.Context, name string, seed uint32, tokens int) string {
	c.logger.Infof("Generating code for %s", name)

	reqBody := generateRequest{
		Name:   name,
		Seed:   seed,
		Tokens: tokens,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("marshal failed: ", err)
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(data))
	if err != nil {
		c.logger.Error("could not build the codegen request: ", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("snippet service unavailable: ", err)
		return ""
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warnw("could not close the codegen response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.logger.Errorf("snippet service status: %d", resp.StatusCode)
		return ""
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Error("decode failed: ", err)
		return ""
	}
	return strings.Join(result.Code, "")
}
