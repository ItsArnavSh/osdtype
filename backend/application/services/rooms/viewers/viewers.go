package viewers

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Viewer struct {
	conn *websocket.Conn
	send chan []byte
}

func (v *Viewer) writePump() {
	for msg := range v.send {
		v.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

type ViewerHub struct {
	Viewers    map[*Viewer]bool
	Register   chan *Viewer
	Unregister chan *Viewer
	Broadcast  chan []byte
}

func NewViewerHub() ViewerHub {
	return ViewerHub{
		Viewers:    make(map[*Viewer]bool),
		Register:   make(chan *Viewer),
		Unregister: make(chan *Viewer),
		Broadcast:  make(chan []byte),
	}
}

func (vh *ViewerHub) RunHub(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case v := <-vh.Register:
			vh.Viewers[v] = true
		case v := <-vh.Unregister:
			if _, ok := vh.Viewers[v]; ok {
				delete(vh.Viewers, v)
				close(v.send)
			}
		case msg := <-vh.Broadcast:
			for v := range vh.Viewers {
				select {
				case v.send <- msg: // Send to viewer's channel
				default:
					// Viewer is slow or dead
					close(v.send)
					delete(vh.Viewers, v)
				}
			}
		}
	}
}
