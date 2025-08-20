package api

import "github.com/gin-gonic/gin"

// All the public routes inside the app
func (s *Server) ping(g *gin.Context) {

	g.JSON(200, gin.H{"reply": "pong"})
}
