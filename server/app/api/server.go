package api

import (
	"os"
	"osdtyp/app/api/auth"
	"osdtyp/app/core"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/services"
	"osdtyp/app/utils"
	"strings"
	"time"

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

func NewServer(logger *zap.SugaredLogger) Server {
	r := gin.New()
	{ //Configuring the Gin Logger to use the zap instead of its own logger
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
	db, err := postgresql.ConnectDatabase(logger)
	if err != nil {
		return Server{}
	}
	core := core.NewCodeCore(logger, &db)
	service, err := services.NewServiceLayer(logger, &core, &db)
	if err != nil {
		return Server{}
	}
	return Server{logger: logger, gin_engine: r, services: service, core: &core}
}
func (s *Server) SetupRoutes() {
	//Setting up general routes
	root_group := s.gin_engine.Group("/")
	{
		root_group.GET("/ping", s.ping)
		root_group.GET("get-user", s.getuser)
	}
	user_group := s.gin_engine.Group("/user")
	user_group.Use(auth.AuthMiddleware())
	{
		user_group.GET("/whoami", s.whoami)
		//whoami
		user_group.GET("/join-lobby", s.joinLobby)
		//join-lobby?duration=30
		user_group.GET("/imonline", s.joinsession)
		//imonline
		user_group.POST("/follow", s.follow)
		//follow?user=name
		user_group.POST("/unfollow", s.unfollow)
		//unfollow?user=name
		user_group.GET("/join-clobby", s.joinControlledLobby)
		//join-clobby?lobbyid=id
		user_group.GET("invite-to-lobby", s.invitePlayerToLobby)
		//invite-to-lobby?invitee=name
	}
	room_group := s.gin_engine.Group("/room")

	room_group.Use(auth.AuthMiddleware())
	// Room-related endpoints
	{
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
	//Auth Route
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

	//Booting all the internal services
	s.logger.Debug("Booting Core")
	s.core.BootCodeCore()

	if port == "" {
		s.logger.Errorf("Port not found in config or environment")
		return
	}

	s.logger.Infof("Server is running on %s", port)
	s.gin_engine.Run(port)
}
func SetRouter() *gin.Engine { //For running tests
	logger, _ := zap.NewDevelopment()
	serv := NewServer(logger.Sugar())
	serv.SetupRoutes()
	return serv.gin_engine
}
