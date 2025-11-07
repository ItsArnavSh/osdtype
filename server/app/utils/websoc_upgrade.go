package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Reusable upgrader with optional origin check
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: tighten this later (e.g. check JWT or allowed origins)
		return true
	},
}

// UpgradeToWebSocket upgrades a Gin HTTP request to a WebSocket connection
func UpgradeToWebSocket(c *gin.Context) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
