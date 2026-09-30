package api

import (
	"os"
	"strings"
	"time"

	"osdtyp/app/api/auth"
	"osdtyp/app/core"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/services"
	"osdtyp/app/utils"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Server struct {
	logger     *zap.SugaredLogger
	gin_engine *gin.Engine
	services   services.ServiceLayer
	core       *core.CodeCore
}

func NewServer(logger *zap.SugaredLogger) (Server, error) {
	db, err := postgresql.ConnectDatabase(logger)
	if err != nil {
		return Server{}, err
	}
	return NewServerWithDB(logger, db)
}

// NewServerWithDB builds a Server around an already-connected database.
// Splitting this out of NewServer keeps the wiring in one place and lets tests
// supply their own database instead of dialing one from configuration.
func NewServerWithDB(logger *zap.SugaredLogger, db postgresql.Database) (Server, error) {
	r := newEngine(logger)
	codeCore, err := core.NewCodeCore(logger, &db)
	if err != nil {
		return Server{}, err
	}
	service, err := services.NewServiceLayer(logger, &codeCore, &db)
	if err != nil {
		return Server{}, err
	}
	return Server{logger: logger, gin_engine: r, services: service, core: &codeCore}, nil
}

// newEngine builds the gin engine with logging, recovery and CORS attached.
func newEngine(logger *zap.SugaredLogger) *gin.Engine {
	r := gin.New()
	{ // Configuring the Gin Logger to use the zap instead of its own logger
		r.Use(ginzap.Ginzap(logger.Desugar(), time.RFC3339, true))
		r.Use(ginzap.RecoveryWithZap(logger.Desugar(), true))
		gin.DefaultWriter = utils.ZapWriter{Logger: logger}
		gin.DefaultErrorWriter = utils.ZapWriter{Logger: logger}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	return r
}

// Engine exposes the underlying gin engine. Used by tests and by any caller
// that wants to mount the routes on its own server.
func (s *Server) Engine() *gin.Engine {
	return s.gin_engine
}

// SetupRoutes mounts every endpoint.
//
// The table is grouped by the resource a person is thinking about rather than
// by handler, and each group carries the auth its members actually need. Two
// things about the old table were bugs rather than style: `/user/imonline` was
// the WebSocket upgrade, so the name said one thing and the handler did
// another, and `invite-to-lobby` was registered without a leading slash, which
// gin silently mounted somewhere nobody looked.
func (s *Server) SetupRoutes() {
	{ // General routes, no auth.
		s.gin_engine.GET("/ping", s.ping)
		s.gin_engine.GET("/get-user", s.getuser)
		// The leaderboard is readable without signing in: a ladder is the one
		// page a stranger should be able to see.
		s.gin_engine.GET("/leaderboard", s.leaderboard)
		// The game's vocabulary, so no client hardcodes the mode, language or
		// tier lists.
		s.gin_engine.GET("/options", s.getOptions)
	}

	{ // The caller's own account and connection.
		user := s.gin_engine.Group("/user")
		user.Use(auth.AuthMiddleware())
		user.GET("/whoami", s.whoami)
		// The WebSocket. The name says what it is now.
		user.GET("/session", s.joinsession)
		user.GET("/status", s.sessionStatus)
	}

	{ // Finding people and inviting them.
		social := s.gin_engine.Group("/user")
		social.Use(auth.AuthMiddleware())
		social.GET("/friends", s.friends)
		social.GET("/search", s.searchPlayers)
		social.POST("/follow", s.follow)
		social.POST("/unfollow", s.unfollow)
		social.GET("/online", s.isOnline)
		// The invite is a POST: it changes the invitee's state, and its target
		// belongs in a body rather than a query.
		social.POST("/invite/:id", s.invitePlayerToLobby)
		social.POST("/invite", s.invitePlayerToLobby)
		// Kept so a bookmark from before the route was named still invites.
		social.GET("/invite-to-lobby", s.invitePlayerToLobby)
	}

	{ // The inbox.
		inbox := s.gin_engine.Group("/user")
		inbox.Use(auth.AuthMiddleware())
		inbox.GET("/notifications", s.getNotifications)
		inbox.GET("/notifications/count", s.getNotificationCount)
		inbox.POST("/notifications/read", s.markNotificationsRead)
	}

	{ // Ranked play and the solo run behind it.
		ranked := s.gin_engine.Group("/user")
		ranked.Use(auth.AuthMiddleware())
		ranked.POST("/queue", s.queueRanked)
		ranked.DELETE("/queue", s.unqueueRanked)
		ranked.GET("/queue", s.queueStatus)
		// The old path and spelling still work.
		ranked.GET("/join-lobby", s.queueRanked)
		ranked.GET("/leave-lobby", s.unqueueRanked)
		ranked.POST("/run", s.submitRun)
		ranked.GET("/runs", s.getRuns)
		ranked.GET("/rank", s.getRankHistory)
	}

	{ // Private game lobbies, by share code.
		lobby := s.gin_engine.Group("/lobby")
		lobby.Use(auth.AuthMiddleware())
		lobby.POST("/create", s.createLobby)
		// A POST because it takes a password, which must not end up in a
		// referrer header.
		lobby.POST("/join", s.joinLobby)
		lobby.POST("/leave", s.leaveLobby)
		lobby.POST("/:id/start", s.startLobby)
		lobby.GET("/:id", s.getLobby)
		// The invitation link's target. It is a second segment so it does not
		// collide with /:id: gin refuses to mount a static route beside a
		// wildcard at the same level, and a six character code and a numeric id
		// in one slot would be a guess either way.
		lobby.GET("/code/:code", s.getLobbyByCode)
	}

	room_group := s.gin_engine.Group("/room")
	room_group.Use(auth.AuthMiddleware())
	{ // Room-related endpoints
		room_group.POST("/create", s.CreateRoom)     // POST /room/create
		room_group.POST("/add-member", s.AddMember)  // POST /room/add-member
		room_group.POST("/promote", s.PromoteToMod)  // POST /room/promote
		room_group.POST("/demote", s.DemoteToMember) // POST /room/demote
		room_group.POST("/block", s.BlockUser)       // POST /room/block
		room_group.POST("/unblock", s.UnBlockUser)   // POST /room/unblock
		room_group.POST("/remove", s.RemoveUser)     // POST /room/remove
		room_group.GET("/list", s.GetRoomList)       // GET /room/list?index=0
	}

	// Contest-related endpoints
	{
		room_group.POST("/contest/create", s.CreateContest)  // POST /room/contest/create
		room_group.GET("/contest/list", s.GetContests)       // GET /room/contest/list?room_id=123&index=0
		room_group.GET("/contest/:job_id", s.GetContestData) // GET /room/contest/456
	}

	// Auth Route
	s.GitHubAuth()
	s.FakeGitHubAuth()
}
func (s *Server) StartServer() {
	port := os.Getenv("PORT")
	if port == "" {
		// Fall back to Viper config for local development
		port = viper.GetString("Core.port")
	}

	// Ensure port has colon prefix for Gin
	if port != "" && !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	// Booting all the internal services
	s.logger.Debug("Booting Core")
	go s.core.BootCodeCore()

	if port == "" {
		s.logger.Errorf("Port not found in config or environment")
		return
	}

	s.logger.Infof("Server is running on %s", port)
	if err := s.gin_engine.Run(port); err != nil {
		s.logger.Errorf("server stopped: %v", err)
	}
}
