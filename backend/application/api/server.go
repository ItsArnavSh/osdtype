package api

import (
	"context"
	"osdtype/application/auth"
	"osdtype/application/entity"
	"osdtype/application/services/anticheat"
	langauge "osdtype/application/services/language"
	room "osdtype/application/services/rooms"
	"osdtype/database"

	"github.com/asaskevich/EventBus"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	//Todo: Remove bus from essentials...its not
	essen        entity.Essentials
	bus          EventBus.Bus
	active_games room.ActiveGames
	gin_engine   *gin.Engine
	anti_cheat   anticheat.AntiCheat
}

func (s *Server) StartServer(ctx context.Context, log *zap.Logger, db *database.Queries) {
	GitHubAuth(log, r)

	r.Use(auth.AuthMiddleware())
	ws := r.Group("/", auth.AuthMiddleware())
	ess := entity.Essentials{Db: db, Logger: log, Bus: bus}

	_ = room.NewActiveGames(ess) //Shift to server
	//Todo: Remove these with essentials
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

func NewServer(ctx context.Context, essen entity.Essentials) (Server, error) {

	r := gin.Default()
	da_bus := EventBus.New() //For Decoupled Anticheat

	antiCheat := anticheat.AntiCheat{Query: essen.Db, Logger: essen.Logger}
	da_bus.Subscribe("cheatcheck", antiCheat.RunAntiCheat)

	active_games := room.NewActiveGames(essen)
	return Server{
		gin_engine:   r,
		essen:        essen,
		bus:          da_bus,
		active_games: active_games,
		anti_cheat:   antiCheat,
	}, nil
}
