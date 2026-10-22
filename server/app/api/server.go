package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"osdtyp/app/api/auth"
	"osdtyp/app/core"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"
	"osdtyp/app/services"
	"osdtyp/app/utils"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Server struct {
	logger   *zap.SugaredLogger
	engine   *gin.Engine
	services *services.ServiceLayer
	core     *core.CodeCore
	db       *postgresql.Database
	rdb      *redis.Client
}

func NewServer(logger *zap.SugaredLogger) (*Server, error) {
	db, err := postgresql.ConnectDatabase(logger)
	if err != nil {
		return nil, err
	}
	rdb, err := redis.NewClient()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return NewServerWithDeps(logger, db, rdb)
}

// NewServerWithDeps is used by tests and lets callers inject sqlite/miniredis.
func NewServerWithDeps(logger *zap.SugaredLogger, db *postgresql.Database, rdb *redis.Client) (*Server, error) {
	c := core.NewCodeCore(logger, db, rdb)
	svc, err := services.NewServiceLayer(logger, c, db)
	if err != nil {
		return nil, err
	}
	if err := svc.Bootstrap(context.Background()); err != nil {
		logger.Warnw("bootstrap failed", "error", err)
	}
	c.Start()

	s := &Server{logger: logger, services: svc, core: c, db: db, rdb: rdb}
	s.engine = gin.New()
	s.engine.Use(ginzap.Ginzap(logger.Desugar(), time.RFC3339, true))
	s.engine.Use(ginzap.RecoveryWithZap(logger.Desugar(), true))
	gin.DefaultWriter = utils.ZapWriter{Logger: logger}
	gin.DefaultErrorWriter = utils.ZapWriter{Logger: logger}
	s.engine.Use(cors.New(s.corsConfig()))
	s.engine.Use(auth.OptionalAuth())
	s.engine.Use(s.idempotency())
	s.routes()
	return s, nil
}

func (s *Server) corsConfig() cors.Config {
	frontend := strings.TrimRight(viper.GetString("Core.frontend_url"), "/")
	origins := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if frontend != "" {
		origins = append(origins, frontend)
	}
	return cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Idempotency-Key", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Idempotent-Replay"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

func (s *Server) routes() {
	r := s.engine
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"reply": "pong"}) })
	r.GET("/login/github", s.githubLogin)
	r.GET("/auth/github/callback", s.githubCallback)
	r.GET("/login/github/fake", s.devLoginGET)
	r.POST("/auth/dev-login", s.devLogin)

	protected := r.Group("")
	protected.Use(auth.RequireAuth())
	{
		protected.GET("/auth/me", s.whoami)
		protected.POST("/auth/logout", s.logout)

		protected.GET("/state", s.getState)
		protected.POST("/state/ack", s.ackResults)

		protected.GET("/user/whoami", s.whoami)
		protected.GET("/user/search", s.searchUsers)
		protected.GET("/user/friends", s.friends)
		protected.POST("/user/follow", s.follow)
		protected.POST("/user/unfollow", s.unfollow)
		protected.GET("/user/:username/profile", s.profile)

		protected.GET("/ranked/leaderboard", s.leaderboard)
		protected.POST("/ranked/queue", s.joinQueue)
		protected.DELETE("/ranked/queue", s.leaveQueue)
		protected.GET("/ranked/queue", s.queueStatus)
		protected.GET("/leaderboard", s.leaderboard)

		protected.GET("/room/list", s.listRooms)
		protected.GET("/room/mine", s.myRooms)
		protected.POST("/room/create", s.createRoom)
		protected.GET("/room/:code", s.getRoom)
		protected.PATCH("/room/:code", s.updateRoom)
		protected.DELETE("/room/:code", s.closeRoom)
		protected.POST("/room/:code/join", s.joinRoom)
		protected.POST("/room/:code/leave", s.leaveRoom)
		protected.POST("/room/:code/start", s.startRoomGame)
		protected.POST("/room/:code/kick", s.kickUser)
		protected.POST("/room/:code/ban", s.banUser)
		protected.POST("/room/:code/unban", s.unbanUser)
		protected.POST("/room/:code/promote", s.promoteUser)
		protected.POST("/room/:code/demote", s.demoteUser)
		protected.POST("/room/:code/invite", s.inviteUser)
		protected.POST("/room/:code/bot", s.addBot)
		protected.GET("/room/:code/link", s.roomLink)
		protected.POST("/room/:code/contest", s.createContest)
		protected.GET("/room/:code/contests", s.listContests)

		protected.GET("/invites", s.listInvites)
		protected.POST("/invites/:id/accept", s.acceptInvite)
		protected.POST("/invites/:id/decline", s.declineInvite)
		protected.GET("/invite/:id/accept", s.acceptInvite)
		protected.GET("/invite/:id/decline", s.declineInvite)

		protected.GET("/game/:id", s.getGame)
		protected.POST("/game/:id/leave", s.leaveGame)
		protected.POST("/game/:id/ack", s.ackResults)
		protected.GET("/game/:id/results", s.getGame)
		protected.GET("/contest/:id", s.getContest)
		protected.GET("/contests/:id", s.getContest)

		protected.GET("/ws", s.websocket)
		protected.GET("/user/imonline", s.websocket)

		// Compatibility aliases for clients written before the V1 routes.
		protected.GET("/user/join-lobby", s.joinQueue)
		protected.DELETE("/user/join-lobby", s.leaveQueue)
		protected.POST("/user/join-lobby", s.joinQueue)
		protected.GET("/room/join", s.joinRoomQuery)
	}
}

func (s *Server) StartServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = viper.GetString("Core.port")
	}
	if port == "" {
		port = ":8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	s.logger.Infof("OSDType server listening on %s", port)
	if err := s.engine.Run(port); err != nil && err != http.ErrServerClosed {
		s.logger.Fatalw("server stopped", "error", err)
	}
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) Close() {
	s.core.Stop()
	if s.rdb != nil {
		_ = s.rdb.Close()
	}
	if s.db != nil {
		_ = s.db.Close()
	}
}
