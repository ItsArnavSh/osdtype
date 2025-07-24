package main

import (
	"context"
	"osdtype/application/api"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	log, _ := zap.NewProduction()
	defer log.Sync()
	api.StartServer(ctx, log)
}
