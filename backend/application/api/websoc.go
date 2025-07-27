package api

import (
	"encoding/json"
	"log"
	"net/http"
	"osdtype/application/entity"
	"osdtype/application/services/livetype"
	"osdtype/database"
	"sync"

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

	// Get snippet using language (no need for rec/typeStruct yet)
	snippet, err := w.query.GetRandomSnippetByLanguage(c.Request.Context(), lang)
	if err != nil {
		w.logger.Error("Could not load snippet", zap.Error(err))
		// Optionally, send an error response over websocket
		return
	}

	// Now, create the Recording and Typer structs, AFTER snippet is fetched
	rec := entity.Recording{
		OriginalID: snippet.ID,
	}
	var typChan = make(chan entity.KeyDef)
	typeStruct := livetype.Typer{
		Query:   w.query,
		Logger:  *w.logger,
		Rec:     rec,
		KeyChan: typChan,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go typeStruct.LiveSave(c.Request.Context(), &wg)

	defer wg.Wait()
	for {
		// Read message from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		var keystroke entity.KeyDef
		json.Unmarshal(msg, &keystroke)
		typChan <- keystroke
		w.bus.Publish("cheatcheck", typChan)
	}

}
