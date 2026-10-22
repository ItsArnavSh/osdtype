package api

import (
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/core/hub"

	"github.com/gin-gonic/gin"
)

func (s *Server) getState(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	state, err := s.services.State(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, state)
}

func (s *Server) ackResults(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	state, err := s.services.AckResults(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, state)
}

func (s *Server) searchUsers(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		q = c.Query("name")
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	users, err := s.services.SearchUsers(c, q, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (s *Server) friends(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	list, err := s.services.Friends(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": list})
}

type followRequest struct {
	User string `json:"user"`
}

func (s *Server) follow(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	name := c.Query("user")
	if name == "" && c.Request.ContentLength > 0 {
		var req followRequest
		if bindJSON(c, &req) {
			name = req.User
		} else {
			return
		}
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, apiError{Error: "user is required", Code: "bad_request"})
		return
	}
	target, err := s.services.GetUserFromName(c, name)
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.services.FollowUser(c, uid, target.ID); err != nil {
		s.fail(c, err)
		return
	}
	s.core.Hub.SendToUser(target.ID, hub.Ev("friend_update", gin.H{"user_id": uid, "action": "follow"}))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) unfollow(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	name := c.Query("user")
	if name == "" && c.Request.ContentLength > 0 {
		var req followRequest
		if bindJSON(c, &req) {
			name = req.User
		} else {
			return
		}
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, apiError{Error: "user is required", Code: "bad_request"})
		return
	}
	target, err := s.services.GetUserFromName(c, name)
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.services.UnfollowUser(c, uid, target.ID); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) profile(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	profile, err := s.services.Profile(c, uid, c.Param("username"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (s *Server) leaderboard(c *gin.Context) {
	uid, _ := auth.GetUserID(c)
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := s.services.Leaderboard(c, offset, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	myPos := int64(0)
	if uid != 0 {
		if user, err := s.services.GetUser(c, uid); err == nil {
			if pos, err := s.db.CountRatingAbove(c, user.Rating); err == nil {
				myPos = pos + 1
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": rows, "my_position": myPos})
}
