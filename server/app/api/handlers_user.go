package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/core/hub"
	"osdtyp/app/services"

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

func (s *Server) getProfileSettings(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	settings, err := s.services.ProfileSettings(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, settings)
}

func (s *Server) updateProfileSettings(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.ProfileSettingsRequest
	if !bindJSON(c, &req) {
		return
	}
	settings, err := s.services.UpdateProfileSettings(c, uid, req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, settings)
}

func (s *Server) exportUserData(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	export, err := s.services.ExportUserData(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	payload, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		s.fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="osdtyp-%d-export.json"`, uid))
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

func (s *Server) deleteRunHistory(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	deleted, err := s.services.DeleteRunHistory(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted, "pinned_kept": true, "best_kept": true})
}

func (s *Server) leaderboard(c *gin.Context) {
	uid, _ := auth.GetUserID(c)
	duration, _ := strconv.Atoi(c.DefaultQuery("duration", "90"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	view, err := s.services.Leaderboard(c, uid, duration, offset, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) searchLeaderboard(c *gin.Context) {
	duration, _ := strconv.Atoi(c.DefaultQuery("duration", "90"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	view, err := s.services.SearchLeaderboard(c, duration, c.Query("q"), limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
