package api

import (
	"io"

	"github.com/gin-gonic/gin"
)

func (s *Server) create_game(g *gin.Context) {
	inst, err := io.ReadAll(g.Request.Body)
	if err != nil {
		//Todo: Return a proper error
		return
	}

	gameHandler, err := s.active_games.NewGame(g.Request.Context(), inst)
	gameHandler.ReadyCompetition()
	//Now the players can start tuning in
}
