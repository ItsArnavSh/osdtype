package utils

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Upgrader is intentionally strict about origins. During local development
// the configured frontend URL is allowed; same-origin requests are always ok.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // native clients and tests
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true
		}
		allowed := os.Getenv("FRONTEND_URL")
		if allowed == "" {
			allowed = "http://localhost:5173"
		}
		return strings.EqualFold(strings.TrimRight(origin, "/"), strings.TrimRight(allowed, "/"))
	},
}

func UpgradeToWebSocket(c *gin.Context) (*websocket.Conn, error) {
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}
