package api

import (
	"context"
	"fmt"
	"net/http"
	"osdtype/application/auth"
	"osdtype/application/entity"
	"osdtype/application/services/anticheat"
	langauge "osdtype/application/services/language"
	room "osdtype/application/services/rooms"
	"osdtype/database"
	"time"

	"github.com/gin-contrib/cors"

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
	GitHubAuth(log, s.gin_engine) // e.g. /login is public
	s.gin_engine.GET("/api/me", func(c *gin.Context) {
		// 1. Get the token from the cookie
		jwtToken, err := c.Cookie("token")
		fmt.Println("Token is: ", jwtToken)
		if err != nil {
			// If no cookie is present, the user is not logged in
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}

		// 2. Validate the JWT (your auth.ValidateJWT function would do this)
		// For this example, we'll assume auth.ValidateJWT returns the username
		username, err := auth.ValidateJWT(jwtToken)
		if err != nil {
			fmt.Println(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// 3. Return the user's information
		c.JSON(http.StatusOK, gin.H{"login": username})
	})
	// Public routes (no auth)
	s.gin_engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected routes
	ws := s.gin_engine.Group("/")
	ws.Use(auth.AuthMiddleware())

	ess := entity.Essentials{Db: db, Logger: log, Bus: s.bus}
	_ = room.NewActiveGames(ess) //Shift to server

	// Example protected GET
	ws.GET("get", func(c *gin.Context) {
		var lang_data entity.LangData
		if err := c.ShouldBindJSON(&lang_data); err != nil {
			log.Error(err.Error())
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		langauge.InsertSnippet(ctx, *db, lang_data.Language, lang_data.Snippet)
	})

	// Protected WS
	wshandler := WSHandler{query: db, logger: log, bus: s.bus}
	ws.GET("ws", wshandler.wsHandler)

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
	return Server{
		gin_engine:   r,
		essen:        essen,
		bus:          da_bus,
		active_games: active_games,
		anti_cheat:   antiCheat,
	}, nil
}
