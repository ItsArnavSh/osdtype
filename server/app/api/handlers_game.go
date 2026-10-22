package api

import (
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/services"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
)

func (s *Server) getGame(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	view, err := s.services.GetGameView(c, uid, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (s *Server) leaveGame(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	state, err := s.services.LeaveGame(c, uid, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "state": state})
}

func (s *Server) websocket(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	conn, err := utils.UpgradeToWebSocket(c)
	if err != nil {
		s.logger.Warnw("websocket upgrade failed", "user_id", uid, "error", err)
		return
	}
	s.core.Hub.Serve(uid, conn)
}

func (s *Server) createContest(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.CreateContestRequest
	if !bindJSON(c, &req) {
		return
	}
	contest, err := s.services.CreateContest(c, uid, c.Param("code"), req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, contest)
}

func (s *Server) listContests(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	contests, err := s.services.ListContests(c, c.Param("code"), offset, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"contests": contests})
}

func (s *Server) getContest(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	contest, err := s.services.GetContest(c, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, contest)
}
