package players

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Player struct {
	conn          *websocket.Conn
	rec           chan []byte
	username      string
	typed_snippet string
}
type PlayerHub struct {
	Players    map[*Player]bool
	Register   chan *Player
	Disconnect chan *Player
	Keychan    chan []byte
}

func NewPlayerHub() PlayerHub {
	return PlayerHub{
		Players:    make(map[*Player]bool),
		Register:   make(chan *Player),
		Disconnect: make(chan *Player),
		Keychan:    make(chan []byte),
	}
}
func (p *Player) sendSnippet(snippet string) {
	p.conn.WriteMessage(websocket.TextMessage, []byte(snippet))
}

func (p *PlayerHub) RunHub(wg *sync.WaitGroup) {
	//Call all when the timer is up for all players
	defer wg.Done()
	for {
		select {
		case v := <-p.Register:
			p.Players[v] = true
		case v := <-p.Disconnect:
			if _, ok := p.Players[v]; ok {
				delete(p.Players, v)
				close(v.rec)
			}
		case _ = <-p.Keychan:
			//Decide what to do when player actually sends a keychan
		}
	}
}
