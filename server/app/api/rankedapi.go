package api

import (
	"net/http"
	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
)

func (s *Server) joinLobby(g *gin.Context) {
	userid, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("invalid userid", "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid userid format"})
		return
	}

	rankStr, err := s.services.GetRank(g.Request.Context(), userid)
	if err != nil {
		s.logger.Warnw("invalid rank", "userid", userid, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid rank format"})
		return
	}

	durationStr := g.Query("duration")
	if durationStr == "" {
		s.logger.Warnw("missing duration", "userid", userid)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing duration parameter"})
		return
	}

	// Check for zero values explicitly
	if userid == 0 {
		s.logger.Warnw("invalid userid or rank", "userid", userid, "rank", rankStr)
		g.JSON(http.StatusBadRequest, gin.H{"error": "userid or rank cannot be zero"})
		return
	}

	var lobbyType entity.LobbyType
	switch durationStr {
	case "30":
		lobbyType = entity.SPRINT
	case "90":
		lobbyType = entity.STANDARD
	case "300":
		lobbyType = entity.MARATHON
	default:
		s.logger.Warnw("invalid duration", "duration", durationStr)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration; must be one of 30, 90, 300"})
		return
	}

	if err := s.core.Matchmaker.AddToGlobalLobby(userid, rankStr, lobbyType); err != nil {
		s.logger.Errorw("failed to join lobby", "userid", userid, "rank", rankStr, "lobbyType", lobbyType, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join lobby"})
		return
	}

	s.logger.Infow("user joined lobby", "userid", userid, "rank", rankStr, "lobbyType", lobbyType)
	g.JSON(http.StatusOK, gin.H{"message": "joined lobby successfully"})
}
