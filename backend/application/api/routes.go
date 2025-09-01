package api

import (
	"osdtype/application/auth"
)

// Setting up routes
func (s *Server) SetRoutes() {
	//Public Routes
	s.GitHubAuth() //Sets up routes for github
	publicgroup := s.gin_engine.Group("/")
	{
		publicgroup.GET("/health", s.ping)
		publicgroup.GET("/get-snippet", s.get_snippet)
	}
	//User Routes -> Protected
	usergroup := s.gin_engine.Group("/user")
	usergroup.Use(auth.AuthMiddleware())
	{
		usergroup.GET("/whoami", s.whoami)
		usergroup.GET("/ws", s.wsHandler)
		roomgroup := usergroup.Group("/room")
		{
			roomgroup.POST("create-room", s.room_handler.Create_room)
			roomgroup.POST("add-player", s.room_handler.Add_player)
			roomgroup.POST("change-player-perms", s.room_handler.Change_player_perms)
			roomgroup.POST("remove-player", s.room_handler.Remove_player)
			gameroom := roomgroup.Group("/game")
			gameroom.POST("start-game", s.create_game)

		}
	}

	// Admin Route
	admingroup := s.gin_engine.Group("/admin")
	admingroup.Use(auth.AdminMiddleware())
	{
		admingroup.GET("/ping", s.admin_ping)
		admingroup.POST("/insert-snippet", s.insert_snippet)
	}
}
