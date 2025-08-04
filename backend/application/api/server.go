package api

import (
	"context"
	"osdtype/application/auth"
	"osdtype/application/entity"
	"osdtype/application/services/anticheat"
	langauge "osdtype/application/services/language"
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

	r.Use(auth.AuthMiddleware())
	ws := r.Group("/", auth.AuthMiddleware())
	wshandler := WSHandler{query: db, logger: log, bus: bus}
	r.GET("get", func(c *gin.Context) {
		var lang_data entity.LangData
		err := c.ShouldBindBodyWithJSON(&lang_data)
		if err != nil {
			log.Error(err.Error())
		}
		langauge.InsertSnippet(ctx, *db, lang_data.Language, lang_data.Snippet)
	})
	ws.GET("ws", wshandler.wsHandler)
	r.Run(":8080")
}
