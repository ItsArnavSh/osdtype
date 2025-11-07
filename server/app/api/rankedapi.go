package api

import (
	"net/http"
	"osdtyp/app/entity"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) joinLobby(g *gin.Context) {
	useridStr := g.Query("userid")
	rankStr := g.Query("rank")
	durationStr := g.Query("duration")

	// Check for missing parameters
	if useridStr == "" || rankStr == "" || durationStr == "" {
		s.logger.Warnw("missing query parameters", "userid", useridStr, "rank", rankStr, "duration", durationStr)
		g.JSON(http.StatusBadRequest, gin.H{
			"error": "missing required query parameters: userid, rank, duration",
		})
		return
	}

	// Parse user ID
	userid, err := strconv.ParseUint(useridStr, 10, 64)
	if err != nil {
		s.logger.Warnw("invalid userid", "userid", useridStr, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid userid format"})
		return
	}

	// Parse rank
	rankVal, err := strconv.ParseUint(rankStr, 10, 16)
	if err != nil {
		s.logger.Warnw("invalid rank", "rank", rankStr, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid rank format"})
		return
	}
	currentRank := uint16(rankVal)

	// Parse duration into lobby type
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

	// Try to add to global lobby
	if err := s.core.Matchmaker.AddToGlobalLobby(g, userid, currentRank, lobbyType); err != nil {
		s.logger.Errorw("failed to join lobby", "userid", userid, "rank", currentRank, "lobbyType", lobbyType, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join lobby"})
		return
	}

	// Success
	s.logger.Infow("user joined lobby", "userid", userid, "rank", currentRank, "lobbyType", lobbyType)
	g.JSON(http.StatusOK, gin.H{"message": "joined lobby successfully"})
}
