package api

import (
	"net/http"

	"osdtyp/app/api/auth"
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
