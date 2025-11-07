package utils

import (
	"strings"

	"go.uber.org/zap"
)

type ZapWriter struct {
	Logger *zap.SugaredLogger
}

func (w ZapWriter) Write(p []byte) (n int, err error) {
	w.Logger.Debug(strings.TrimSpace(string(p)))
	return len(p), nil
}
