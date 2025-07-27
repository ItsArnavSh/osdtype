package api

import (
	"context"
	"osdtype/application/services/anticheat"
	"osdtype/database"

	"github.com/asaskevich/EventBus"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func StartServer(ctx context.Context, log *zap.Logger, db *database.Queries) {
	r := gin.Default()
	bus := EventBus.New() //For Decoupled Anticheat
	antiCheat := anticheat.AntiCheat{Query: db, Logger: log}
	bus.Subscribe("cheatcheck", antiCheat.RunAntiCheat)
	GitHubAuth(log, r)
	wshandler := WSHandler{query: db, logger: log, bus: bus}
	r.GET("ws", wshandler.wsHandler)
	r.Run(":8080")
}
