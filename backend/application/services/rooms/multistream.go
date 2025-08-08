package room

import (
	"context"
	"fmt"
	"osdtype/application/entity"
	"osdtype/application/services/rooms/players"
	"osdtype/application/services/rooms/viewers"
	"sync"

	"github.com/gin-gonic/gin"
)

type GameHandler struct {
	players         players.PlayerHub
	viewers         viewers.ViewerHub
	snippet         string
	playerSnippet   []string
	denyConnections bool
	roomid          string
	essentials      entity.Essentials
	gamConf         entity.GameConf
}

func NewGameHandler(roomid string, ess entity.Essentials, gameConf entity.GameConf) GameHandler {
	return (GameHandler{
		players:         players.NewPlayerHub(),
		viewers:         viewers.NewViewerHub(),
		snippet:         "",
		playerSnippet:   nil,
		denyConnections: false,
		roomid:          roomid,
		essentials:      ess,
		gamConf:         gameConf,
	})
}

func (g *GameHandler) ReadyCompetition() {
	var wg sync.WaitGroup
	wg.Add(2)
	//Boot up background processes
	go g.players.RunHub(&wg)
	go g.viewers.RunHub(&wg)
	wg.Wait()
}

func (g *GameHandler) StartCompetition(ctx context.Context) error {
	g.denyConnections = true
	//if start game is called, players cannot enroll
	//Send the snippet to all the players
	snippet, err := g.essentials.Db.GetRandomSnippetByLanguage(ctx, "javascript")
	if err != nil {
		return err
	}
	for player := range g.players.Players {
		go player.SendSnippet(snippet.Snippet)
	}
	//Now all have got the data, so we can safely begin taking responses.
	return nil
}

func (g *GameHandler) RegisterForGame(ctx *gin.Context, gameid string) error {
	if g.denyConnections {
		return fmt.Errorf("Cannot Register once contest has started")
	}
	//New Player will connect via websocket too
	player, err := players.NewPlayer(ctx)
	if err != nil {
		return err
	}
	g.players.Register <- &player
	return nil
}
