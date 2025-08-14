package viewers

import (
	"net/http"
	"osdtype/application/auth"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Viewer struct {
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewViewer(ctx *gin.Context) (Viewer, error) {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return Viewer{}, err
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return Viewer{}, err
	}
	return Viewer{Conn: conn, UserID: user, Send: make(chan []byte)}, nil
}
func (v *Viewer) writePump() {
	for msg := range v.Send {
		v.Conn.WriteMessage(websocket.TextMessage, msg)
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
				close(v.Send)
			}
		case msg := <-vh.Broadcast:
			for v := range vh.Viewers {
				select {
				case v.Send <- msg: // Send to viewer's channel
				default:
					// Viewer is slow or dead
					close(v.Send)
					delete(vh.Viewers, v)
				}
			}
		}
	}
}
