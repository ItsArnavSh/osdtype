package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
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
func (c *CodeGen) Generate(name string, seed uint32, tokens int) string {
	c.logger.Infof("Generating code for %s", name)

	reqBody := generateRequest{
		Name:   name,
		Seed:   seed,
		Tokens: tokens,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("marshal failed:", err)
		return ""
	}

	resp, err := c.client.Post(c.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		c.logger.Error("Rust service unavailable:", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Errorf("Rust service status: %d", resp.StatusCode)
		return ""
	}

	var result generateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		c.logger.Error("decode failed:", err)
		return ""
	}
	fmt.Print(result.Code)
	return strings.Join(result.Code, "")
}
