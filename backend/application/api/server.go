package api

import (
	"context"
	"osdtype/application/entity"
	"osdtype/application/services/anticheat"
	room "osdtype/application/services/rooms"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/asaskevich/EventBus"
	"github.com/gin-gonic/gin"
)

type Server struct {
	//Todo: Remove bus from essentials...its not
	essen        entity.Essentials
	bus          EventBus.Bus
	active_games room.ActiveGames
	gin_engine   *gin.Engine
	anti_cheat   anticheat.AntiCheat
	room_handler room.RoomHandler
}

func (s *Server) StartServer(ctx context.Context) {
	s.gin_engine.Run(":8080")
}

func NewServer(ctx context.Context, essen entity.Essentials) (Server, error) {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	da_bus := EventBus.New() //For Decoupled Anticheat

	antiCheat := anticheat.AntiCheat{Query: essen.Db, Logger: essen.Logger}
	da_bus.Subscribe("cheatcheck", antiCheat.RunAntiCheat)

	active_games := room.NewActiveGames(essen)
	roomHandler := room.NewRoomHandler(essen.Db, essen.Logger)
	return Server{
		gin_engine:   r,
		essen:        essen,
		bus:          da_bus,
		active_games: active_games,
		anti_cheat:   antiCheat,
		room_handler: roomHandler,
	}, nil
}
