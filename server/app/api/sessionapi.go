package api

import (
	"net/http"
	"osdtyp/app/api/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) joinsession(g *gin.Context) {
	userid, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("failed to get userID", "error", err)
		g.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	err = s.services.StartSession(g, userid)
	if err != nil {
		s.logger.Errorw("failed to start session", "userid", userid, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start session"})
		return
	}

	g.JSON(http.StatusOK, gin.H{"message": "session started successfully"})
}
