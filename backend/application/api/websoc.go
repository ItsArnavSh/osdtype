package api

import (
	"log"
	"net/http"
	livetype "osdtype/application/services/typing"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) wsHandler(c *gin.Context) {
	//Todo: Create a channel thingy to communicate the errors to the frontend
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Get and check the language parameter from the query
	snip_id := c.Query("snip_id")
	if snip_id == "" {
		s.essen.Logger.Error("Language Parameter Not Set")
		return
	}
	time := c.Query("time")
	if time == "" {
		s.essen.Logger.Error("Time duration not set")
		return
	}
	duration, err := strconv.Atoi(time)
	if err != nil {
		s.essen.Logger.Error("Time duration not a number")
	}
	//Todo: Do something with the error
	//Modify conduct test to get snippet from snip id
	_ = livetype.ConductTest(snip_id, duration, c, s.essen.Logger, s.essen.Db, conn, s.essen.Bus)
}
