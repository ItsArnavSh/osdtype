package api

import "osdtype/application/auth"

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
	}

	// Admin Route
	admingroup := s.gin_engine.Group("/admin")
	admingroup.Use(auth.AdminMiddleware())
	{
		admingroup.GET("/ping", s.admin_ping)
		admingroup.POST("/insert-snippet", s.insert_snippet)
	}
}
