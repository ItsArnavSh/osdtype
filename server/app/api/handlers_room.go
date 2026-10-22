package api

import (
	"net/http"
	"strconv"
	"strings"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

func idParam(c *gin.Context, name string) (uint32, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{Error: "invalid id", Code: "bad_request"})
		return 0, false
	}
	return uint32(v), true
}

func (s *Server) createRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.CreateRoomRequest
	if !bindJSON(c, &req) {
		return
	}
	room, err := s.services.CreateRoom(c, uid, req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, room)
}

func (s *Server) listRooms(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	rooms, err := s.services.ListRooms(c, uid, false, offset, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

func (s *Server) myRooms(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	rooms, err := s.services.ListRooms(c, uid, true, offset, limit)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

func (s *Server) getRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	room, err := s.services.GetRoom(c, uid, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

type joinRoomRequest struct {
	Password string `json:"password"`
}

func (s *Server) joinRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req joinRoomRequest
	if c.Request.ContentLength > 0 {
		if !bindJSON(c, &req) {
			return
		}
	}
	room, err := s.services.JoinRoom(c, uid, c.Param("code"), req.Password)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) joinRoomQuery(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	code := c.Query("code")
	if code == "" {
		code = c.Query("room")
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, apiError{Error: "room code is required", Code: "bad_request"})
		return
	}
	room, err := s.services.JoinRoom(c, uid, code, c.Query("password"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) leaveRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.services.LeaveRoom(c, uid, c.Param("code")); err != nil {
		s.fail(c, err)
		return
	}
	state, err := s.services.State(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "state": state})
}

func (s *Server) updateRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.UpdateRoomRequest
	if !bindJSON(c, &req) {
		return
	}
	room, err := s.services.UpdateRoom(c, uid, c.Param("code"), req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) closeRoom(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.services.CloseRoom(c, uid, c.Param("code")); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) startRoomGame(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.StartGameRequest
	if c.Request.ContentLength > 0 {
		if !bindJSON(c, &req) {
			return
		}
	}
	view, err := s.services.StartRoomGame(c, uid, c.Param("code"), req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type targetRequest struct {
	UserID uint32 `json:"user_id"`
}

func (s *Server) roomTarget(c *gin.Context) (uint32, bool) {
	var req targetRequest
	if !bindJSON(c, &req) {
		return 0, false
	}
	if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, apiError{Error: "user_id is required", Code: "bad_request"})
		return 0, false
	}
	return req.UserID, true
}

func (s *Server) kickUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	target, ok := s.roomTarget(c)
	if !ok {
		return
	}
	if err := s.services.KickUser(c, uid, c.Param("code"), target); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) banUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	target, ok := s.roomTarget(c)
	if !ok {
		return
	}
	if err := s.services.BanUser(c, uid, c.Param("code"), target); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) unbanUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	target, ok := s.roomTarget(c)
	if !ok {
		return
	}
	if err := s.services.UnbanUser(c, uid, c.Param("code"), target); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) promoteUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	target, ok := s.roomTarget(c)
	if !ok {
		return
	}
	if err := s.services.PromoteUser(c, uid, c.Param("code"), target); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) demoteUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	target, ok := s.roomTarget(c)
	if !ok {
		return
	}
	if err := s.services.DemoteUser(c, uid, c.Param("code"), target); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) inviteUser(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.InviteRequest
	if !bindJSON(c, &req) {
		return
	}
	invite, err := s.services.InviteUser(c, uid, c.Param("code"), req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, invite)
}

func (s *Server) listInvites(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	invites, err := s.services.ListInvites(c, uid)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invites": invites})
}

func (s *Server) acceptInvite(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	room, err := s.services.AcceptInvite(c, uid, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	state, _ := s.services.State(c, uid)
	c.JSON(http.StatusOK, gin.H{"room": room, "state": state})
}

func (s *Server) declineInvite(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	if err := s.services.DeclineInvite(c, uid, id); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type botRequest struct {
	Persona string `json:"persona"`
}

func (s *Server) addBot(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req botRequest
	if !bindJSON(c, &req) {
		return
	}
	persona := entity.Persona(strings.ToUpper(strings.TrimSpace(req.Persona)))
	found := false
	for _, p := range entity.Personas {
		if p == persona {
			found = true
		}
	}
	if !found {
		c.JSON(http.StatusBadRequest, apiError{Error: "persona must be LEWIS, MAX or LANCE", Code: "bad_request"})
		return
	}
	room, err := s.services.AddBot(c, uid, c.Param("code"), persona)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) roomLink(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	room, err := s.services.GetRoom(c, uid, c.Param("code"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": room.Code, "invite_link": room.InviteLink, "has_password": room.HasPassword})
}
