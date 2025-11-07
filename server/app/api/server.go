package api

import (
	"osdtyp/app/api/auth"
	"osdtyp/app/core"
	"osdtyp/app/services"
	"osdtyp/app/utils"
	"time"

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
	service, err := services.NewServiceLayer(logger)

	core := core.NewCodeCore(logger)
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
		root_group.GET("/join-lobby", s.joinLobby)
	}
	user_group := s.gin_engine.Group("/user")
	user_group.Use(auth.AuthMiddleware())
	{
		user_group.GET("/whoami", s.whoami)
	}
	//Auth Route
	s.GitHubAuth()
}
func (s *Server) StartServer() {
	port := viper.GetString("Core.port")
	//Booting all the internal services
	s.logger.Debug("Booting Core")
	s.core.BootCodeCore()
	if port == "" {
		s.logger.Errorf("Port not found in config")
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
