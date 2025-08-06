package room

import (
	"context"
	"fmt"
	"osdtype/application/services/rooms/players"
	"osdtype/application/services/rooms/viewers"
	"sync"
)

type GameHandler struct {
	players         players.PlayerHub
	viewers         viewers.ViewerHub
	snippet         string
	playerSnippet   []string
	denyConnections bool
	roomid          string
}

func NewGameHandler(roomid string) GameHandler {
	return (GameHandler{
		players:         players.NewPlayerHub(),
		viewers:         viewers.NewViewerHub(),
		snippet:         "",
		playerSnippet:   nil,
		denyConnections: false,
	})
}

func (g *GameHandler) ReadyCompetition() {
	var wg sync.WaitGroup
	wg.Add(2)
	//Boot up background processes
	go g.players.RunHub(&wg)
	go g.viewers.RunHub(&wg)

}

func (g *GameHandler) StartCompetition(snippet string) {
	g.denyConnections = true
	//Once the final 5 sec timer starts players cannot enroll
	//

}
func (g *GameHandler) RegisterForGame(ctx context.Context) error {
	if g.denyConnections {
		return fmt.Errorf("Cannot Register once contest has started")
	}
	return nil
}
