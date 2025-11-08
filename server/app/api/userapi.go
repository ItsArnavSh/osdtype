package api

import (
	"net/http"
	"osdtyp/app/api/auth"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) whoami(c *gin.Context) {
	userStr, err := auth.GetUserID(c)
	s.logger.Debug("Reached Here", userStr)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to get user ID"})
		return
	}
	userID, err := strconv.ParseUint(userStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	userdata, err := s.services.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user data"})
		return
	}
	c.JSON(http.StatusOK, userdata)
}
