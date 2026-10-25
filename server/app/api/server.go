package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	"go.uber.org/zap/zapcore"
)

type Server struct {
	logger   *zap.SugaredLogger
	engine   *gin.Engine
	services *services.ServiceLayer
	core     *core.CodeCore
	db       *postgresql.Database
	rdb      *redis.Client
	metrics  *requestMetrics
}

func NewServer(logger *zap.SugaredLogger) (*Server, error) {
	if err := validateProductionConfig(); err != nil {
		return nil, err
	}
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

	s := &Server{logger: logger, services: svc, core: c, db: db, rdb: rdb, metrics: newRequestMetrics()}
	s.engine = gin.New()
	s.engine.Use(requestIDMiddleware())
	s.engine.Use(securityHeaders())
	s.engine.Use(s.metrics.middleware())
	s.engine.Use(ginzap.GinzapWithConfig(logger.Desugar(), &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/healthz", "/readyz", "/metrics"},
		Context: func(c *gin.Context) []zapcore.Field {
			fields := []zapcore.Field{zap.String("request_id", requestID(c))}
			if uid, err := auth.GetUserID(c); err == nil {
				fields = append(fields, zap.Uint32("user_id", uid))
			}
			return fields
		},
	}))
	s.engine.Use(ginzap.RecoveryWithZap(logger.Desugar(), true))
	gin.DefaultWriter = utils.ZapWriter{Logger: logger}
	gin.DefaultErrorWriter = utils.ZapWriter{Logger: logger}
	s.engine.Use(cors.New(s.corsConfig()))
	s.engine.Use(bodyLimit(maxRequestBodyBytes))
	s.engine.Use(auth.OptionalAuth())
	s.engine.Use(s.activityMiddleware())
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
		AllowHeaders:     []string{"Content-Type", "Authorization", "Idempotency-Key", "X-Requested-With", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "Idempotent-Replay", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

func (s *Server) routes() {
	r := s.engine
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"reply": "pong"}) })
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", s.readiness)
	r.GET("/metrics", s.metrics.handler)
	r.POST("/bugs", s.createBugReport)
	r.GET("/login/github", s.githubLogin)
	r.GET("/auth/github/callback", s.githubCallback)
	r.GET("/login/github/fake", s.devLoginGET)
	r.POST("/auth/dev-login", s.devLogin)
	// Share links must work without an account.
	r.GET("/replay/:game/:user", s.publicReplay)

	// Contest pages and their calendar links are shareable without an
	// account. Private contests are protected by their unguessable code.
	r.GET("/contests", s.listContests)
	r.GET("/contests/:code", s.getContest)
	r.GET("/contests/:code/calendar", s.contestCalendar)

	// The admin surface is deliberately invisible: every non-admin request,
	// including an unauthenticated one, gets the same 404 as an unknown route.
	admin := r.Group("/admin")
	admin.Use(s.requireAdmin())
	{
		admin.GET("/overview", s.adminOverview)
		admin.GET("/bugs", s.adminBugs)
		admin.PATCH("/bugs/:id", s.adminUpdateBug)
		admin.GET("/users", s.adminUsers)
		admin.POST("/users/:id/admin", s.adminGrant)
		admin.DELETE("/users/:id/admin", s.adminRevoke)
		admin.GET("/activity", s.adminActivity)
		admin.GET("/health", s.adminHealth)
		admin.GET("/logs", s.adminLogs)
		admin.GET("/events", s.adminEvents)
		admin.GET("/audits", s.adminAudits)
	}

	protected := r.Group("")
	protected.Use(auth.RequireAuth())
	{
		protected.GET("/auth/me", s.whoami)
		protected.POST("/auth/logout", s.logout)

		protected.GET("/state", s.getState)
		protected.POST("/state/ack", s.ackResults)
		protected.GET("/bugs/mine", s.listMyBugReports)

		protected.GET("/user/whoami", s.whoami)
		protected.GET("/user/search", s.searchUsers)
		protected.GET("/user/friends", s.friends)
		protected.GET("/user/settings", s.getProfileSettings)
		protected.PATCH("/user/settings", s.updateProfileSettings)
		protected.GET("/user/export", s.exportUserData)
		protected.DELETE("/user/runs", s.deleteRunHistory)
		protected.POST("/user/follow", s.follow)
		protected.POST("/user/unfollow", s.unfollow)
		protected.GET("/user/:username/profile", s.profile)

		protected.GET("/ranked/leaderboard", s.leaderboard)
		protected.GET("/ranked/leaderboard/search", s.searchLeaderboard)
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

		protected.POST("/contests", s.createContest)
		protected.PATCH("/contests/:code", s.updateContest)
		protected.DELETE("/contests/:code", s.cancelContest)
		protected.POST("/contests/:code/enroll", s.enrollContest)
		protected.DELETE("/contests/:code/enroll", s.leaveContest)
		protected.POST("/contests/:code/stages/:position/attempts", s.startContestAttempt)

		protected.GET("/invites", s.listInvites)
		protected.POST("/invites/:id/accept", s.acceptInvite)
		protected.POST("/invites/:id/decline", s.declineInvite)
		protected.GET("/invite/:id/accept", s.acceptInvite)
		protected.GET("/invite/:id/decline", s.declineInvite)

		protected.GET("/game/:id", s.getGame)
		protected.POST("/game/:id/leave", s.leaveGame)
		protected.POST("/game/:id/ack", s.ackResults)
		protected.GET("/game/:id/results", s.getGame)
		protected.GET("/game/:id/replays", s.gameReplays)
		protected.PATCH("/game/:id/run", s.updateRun)
		protected.POST("/practice/runs", s.createPracticeRun)
		protected.POST("/practice/runs/:id/save", s.savePracticeRun)

		protected.GET("/ws", s.websocket)
		protected.GET("/user/imonline", s.websocket)

		// Compatibility aliases for clients written before the V1 routes.
		protected.GET("/user/join-lobby", s.joinQueue)
		protected.DELETE("/user/join-lobby", s.leaveQueue)
		protected.POST("/user/join-lobby", s.joinQueue)
		protected.GET("/room/join", s.joinRoomQuery)
	}
}

func (s *Server) readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	if err := s.db.PingContext(ctx); err != nil {
		s.logger.Warnw("readiness check failed", "dependency", "postgres", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	if s.rdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	if err := s.rdb.Ping(ctx); err != nil {
		s.logger.Warnw("readiness check failed", "dependency", "redis", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) activityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid, err := auth.GetUserID(c); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := s.services.RecordActivity(ctx, uid, time.Now().UTC()); err != nil {
				s.logger.Warnw("could not record activity", "user_id", uid, "error", err)
			}
			cancel()
		}
		c.Next()
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

	srv := &http.Server{
		Addr:              port,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	s.services.StartMonitoring()
	s.services.RecordEvent(context.Background(), "info", "server", "server started", map[string]any{"port": port})
	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.logger.Infof("OSDType server listening on %s", port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatalw("server stopped", "error", err)
		}
	case <-shutdownSignal.Done():
		s.logger.Info("shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Warnw("graceful shutdown timed out", "error", err)
			_ = srv.Close()
		}
	}
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) Close() {
	s.services.StopMonitoring()
	s.services.RecordEvent(context.Background(), "info", "server", "server stopping", nil)
	s.core.Stop()
	if s.rdb != nil {
		_ = s.rdb.Close()
	}
	if s.db != nil {
		_ = s.db.Close()
	}
}
