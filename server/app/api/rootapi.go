package api

import "github.com/gin-gonic/gin"

func (s *Server) ping(g *gin.Context) {
	s.logger.Infof("Ping endpoint hit from: %s", g.Request.RemoteAddr)
	g.JSON(200, gin.H{"reply": "pong"})
}
