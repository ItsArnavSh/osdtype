package api

import (
	"net/http"
	"strconv"

	"osdtyp/app/api/auth"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) createPracticeRun(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	var req services.PracticeRunRequest
	if !bindJSON(c, &req) {
		return
	}
	run, err := s.services.CreatePracticeRun(c, uid, req)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

func (s *Server) savePracticeRun(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	run, err := s.services.SavePracticeRun(c, uid, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

func (s *Server) gameReplays(c *gin.Context) {
	uid, err := auth.GetUserID(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	bundle, err := s.services.GameReplays(c, uid, id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, bundle)
}

func (s *Server) publicReplay(c *gin.Context) {
	gameID, err := strconv.ParseUint(c.Param("game"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{Error: "invalid game id", Code: "bad_request"})
		return
	}
	userID, err := strconv.ParseUint(c.Param("user"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{Error: "invalid user id", Code: "bad_request"})
		return
	}
	bundle, err := s.services.PublicReplay(c, uint32(gameID), uint32(userID))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, bundle)
}
