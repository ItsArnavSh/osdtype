package api

import (
	"encoding/json"
	"net/http"
	"osdtyp/app/api/auth"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) follow(g *gin.Context) {
	userid, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("failed to get userID", "error", err)
		g.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	otherUsername := g.Query("user")
	if otherUsername == "" {
		s.logger.Warnw("missing user query parameter", "userid", userid)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing user query parameter"})
		return
	}

	otherUser, err := s.services.GetUserFromName(g.Request.Context(), otherUsername)
	if err != nil {
		s.logger.Warnw("failed to get other user", "username", otherUsername, "error", err)
		g.JSON(http.StatusNotFound, gin.H{"error": "user to follow not found"})
		return
	}

	if err := s.services.FollowUser(g.Request.Context(), userid, otherUser.ID); err != nil {
		s.logger.Errorw("failed to follow user", "userid", userid, "other_user_id", otherUser.ID, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to follow user"})
		return
	}

	g.JSON(http.StatusOK, gin.H{"message": "user followed successfully"})
}

func (s *Server) unfollow(g *gin.Context) {
	userid, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("failed to get userID", "error", err)
		g.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	otherUsername := g.Query("user")
	if otherUsername == "" {
		s.logger.Warnw("missing user query parameter", "userid", userid)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing user query parameter"})
		return
	}

	otherUser, err := s.services.GetUserFromName(g.Request.Context(), otherUsername)
	if err != nil {
		s.logger.Warnw("failed to get other user", "username", otherUsername, "error", err)
		g.JSON(http.StatusNotFound, gin.H{"error": "user to unfollow not found"})
		return
	}

	if err := s.services.UnfollowUser(g.Request.Context(), userid, otherUser.ID); err != nil {
		s.logger.Errorw("failed to unfollow user", "userid", userid, "other_user_id", otherUser.ID, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unfollow user"})
		return
	}

	g.JSON(http.StatusOK, gin.H{"message": "user unfollowed successfully"})
}

func (s *Server) joinControlledLobby(g *gin.Context) {
	userid, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("unauthenticated joinControlledLobby attempt", "error", err)
		g.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	lobbyidStr := g.Query("lobbyid")
	if lobbyidStr == "" {
		s.logger.Warnw("missing lobbyid query parameter", "userid", userid)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing lobbyid parameter"})
		return
	}

	lobbyid64, err := strconv.ParseUint(lobbyidStr, 10, 32)
	if err != nil {
		s.logger.Warnw("invalid lobbyid format", "lobbyid", lobbyidStr, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid lobbyid format"})
		return
	}
	lobbyid := uint32(lobbyid64)
	if err := s.services.JoinControlledLobby(userid, lobbyid); err != nil {
		s.logger.Errorw("failed to join controlled lobby", "userid", userid, "lobbyid", lobbyid, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join lobby"})
		return
	}

	s.logger.Infow("user joined controlled lobby", "userid", userid, "lobbyid", lobbyid)
	g.JSON(http.StatusOK, gin.H{"message": "joined controlled lobby successfully"})
}

func (s *Server) invitePlayerToLobby(g *gin.Context) {
	invitorID, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("unauthenticated invitePlayerToLobby attempt", "error", err)
		g.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	inviteeStr := g.Query("invitee")
	if inviteeStr == "" {
		s.logger.Warnw("missing invitee query parameter", "invitor", invitorID)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing invitee parameter"})
		return
	}

	invitee, err := strconv.ParseUint(inviteeStr, 10, 32)
	if err != nil {
		s.logger.Warnw("invalid invitee format", "invitee", inviteeStr, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitee format"})
		return
	}

	lobbyidStr := g.Query("lobbyid")
	if lobbyidStr == "" {
		s.logger.Warnw("missing lobbyid query parameter", "invitor", invitorID)
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing lobbyid parameter"})
		return
	}

	lobbyid, err := strconv.ParseUint(lobbyidStr, 10, 32)
	if err != nil {
		s.logger.Warnw("invalid lobbyid format", "lobbyid", lobbyidStr, "error", err)
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid lobbyid format"})
		return
	}

	// Get user info for invitor to extract username
	invitorUser, err := s.services.GetUser(g.Copy().Request.Context(), invitorID)
	if err != nil {
		s.logger.Errorw("failed to get invitor user info", "invitorID", invitorID, "error", err)
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invitor info"})
		return
	}

	// Call invite service and handle errors if any
	s.services.InvitePlayerToLobby(invitorUser.Username, uint32(invitee), uint32(lobbyid))
	s.logger.Infow("player invited to lobby", "invitor", invitorUser.Username, "invitee", invitee, "lobbyid", lobbyid)
	g.JSON(http.StatusOK, gin.H{"message": "invitation sent"})
}

func (s *Server) searchPlayers(g *gin.Context) {
	username := g.Query("name")
	if username == "" {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "No username in query params"})
		return
	}
	users, err := s.services.SearchUsers(g.Request.Context(), username)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "Error Fetching Users"})
	}
	users_json, err := json.Marshal(users)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "Error Fetching Users"})
	}
	g.JSON(http.StatusOK, users_json)
}
