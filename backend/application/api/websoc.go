package api

import (
	"log"
	"net/http"
	livetype "osdtype/application/services/typing"
	"osdtype/database"
	"strconv"

	"github.com/asaskevich/EventBus"
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
	bus    EventBus.Bus
}

func (w *WSHandler) wsHandler(c *gin.Context) {
	//Todo: Create a channel thingy to communicate the errors to the frontend
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Get and check the language parameter from the query
	lang := c.Query("lang")
	if lang == "" {
		w.logger.Error("Language Parameter Not Set")
		return
	}
	time := c.Query("time")
	if time == "" {
		w.logger.Error("Time duration not set")
		return
	}
	duration, err := strconv.Atoi(time)
	if err != nil {
		w.logger.Error("Time duration not a number")
	}
	//Todo: Do something with the error
	_ = livetype.ConductTest(lang, duration, c, w.logger, w.query, conn, w.bus)
}
