package api

import (
	"log"
	"net/http"
	"osdtype/application/services/livetype"
	"osdtype/database"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	query  *database.Queries
	logger *zap.Logger
}

func (w *WSHandler) wsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	//	const ws = new WebSocket("ws://localhost:8080/ws?token=abc123&lang=en");
	lang := c.Query("lang")
	//Also add auth to this request later on
	if lang == "" {
		w.logger.Error("Language Parameter Not Set")
		return
	}
	typeStruct := livetype.Typer{}
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		// Read message from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received: %s", msg)

		// Echo back
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}
