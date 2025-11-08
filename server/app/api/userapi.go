package api

import (
	"net/http"
	"osdtyp/app/api/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) whoami(c *gin.Context) {
	userID, err := auth.GetUserID(c)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to get user ID"})
		return
	}

	userdata, err := s.services.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user data"})
		return
	}
	c.JSON(http.StatusOK, userdata)
}

func (s *Server) getuser(c *gin.Context) {
	username := c.Query("user")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameter: user"})
		return
	}
	user, err := s.services.GetUserFromName(c.Request.Context(), username)
	if err != nil {
		// You can check for not found error type if your service defines it
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	// Return user object as JSON
	c.JSON(http.StatusOK, user)
}
