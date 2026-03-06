package utils

import (
	"bytes"
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
		url:    viper.GetString("CodeGen.service_url"), // e.g. http://localhost:8081/generate
	}
}

func (c *CodeGen) Generate(name string, seed uint32, tokens int) string {

	reqBody := generateRequest{
		Name:   name,
		Seed:   seed,
		Tokens: tokens,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("Failed to marshal request:", err)
		return ""
	}

	resp, err := c.client.Post(c.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		c.logger.Error("Rust CodeGen service unavailable:", err)
		return ""
	}
	defer resp.Body.Close()

	var result generateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		c.logger.Error("Failed to decode response:", err)
		return ""
	}

	// merge tokens
	return strings.Join(result.Code, "")
}
