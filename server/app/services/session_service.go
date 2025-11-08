package services

import (
	"github.com/gin-gonic/gin"
)

func (s *ServiceLayer) StartSession(g *gin.Context, userid uint64) error {
	user, err := s.db.GetUser(g.Request.Context(), userid)
	if err != nil {
		return err
	}
	//A consistent websocket connection will be established
	// For the lifetime of the session
	// Will automatically disconnect when the user closes his tab
	return s.core.Sessions.NewUserSession(g, user.ID)
}
