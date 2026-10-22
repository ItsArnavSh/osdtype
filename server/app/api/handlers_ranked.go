package api

import (
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"

	"github.com/gin-gonic/gin"
)

type queueRequest struct {
	Duration int `json:"duration"`
}

func (s *Server) requestedDuration(c *gin.Context) int {
	if raw := c.Query("duration"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	}
	if c.Request.ContentLength > 0 {
		var req queueRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			return req.Duration
		}
	}
	return 0
}

func (s *Server) joinQueue(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	status, err := s.services.JoinQueue(c, uid, s.requestedDuration(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"queue": status})
}

func (s *Server) leaveQueue(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	duration := s.requestedDuration(c)
	if duration == 0 {
		duration = 90
	}
	if err := s.services.LeaveQueue(c, uid, duration); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) queueStatus(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	duration := s.requestedDuration(c)
	if duration == 0 {
		duration = 90
	}
	status, ok, err := s.services.QueueStatus(c, uid, duration)
	if err != nil {
		s.fail(c, err)
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{"queue": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queue": status})
}
