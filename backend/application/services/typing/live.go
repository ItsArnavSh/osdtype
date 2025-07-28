package livetype

import (
	"encoding/json"
	"log"
	"osdtype/application/entity"
	"osdtype/application/services/wpm"
	"osdtype/database"
	"sync"
	"time"

	EventBus "github.com/asaskevich/EventBus"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func ConductTest(lang string, tim int, c *gin.Context, logger *zap.Logger, query *database.Queries, conn *websocket.Conn, bus EventBus.Bus) error {
	run_id := uuid.NewString()
	snippet, err := query.GetRandomSnippetByLanguage(c.Request.Context(), lang)
	if err != nil {
		logger.Error("Could not load snippet", zap.Error(err))
		//Todo: Optionally, send an error response over websocket
		return err
	}

	// Now, create the Recording and Typer structs, AFTER snippet is fetched
	rec := entity.Recording{
		OriginalID: snippet.ID,
	}
	var typChan = make(chan entity.KeyDef)
	typeStruct := Typer{
		Query:   query,
		Logger:  *logger,
		Rec:     rec,
		KeyChan: typChan,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go typeStruct.LiveSave(c.Request.Context(), &wg)

	defer wg.Wait()

	first_hit := GetKeyStrokes(conn)
	start_time := first_hit.Time
	typChan <- first_hit
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
	}
	close(typChan)
	st_time := time.Unix(start_time, 0)
	//Save the events in the database as Pending
	wpm := wpm.Calculate_WPM(entity.WPM{OriginalSnippet: snippet.Snippet, UserSnippet: ""})
	query.CreateTypeRun(c.Request.Context(), database.CreateTypeRunParams{StartTime: pgtype.Timestamp{Time: st_time}, ID: run_id, RunData: rec.Recording, UserID: "", Language: lang, Wpm: wpm.WPM, RawWpm: wpm.RAW})
	//Set off to the anticheat software non blocking
	bus.Publish("cheatcheck", typChan)
	return nil
}
func GetKeyStrokes(conn *websocket.Conn) entity.KeyDef {

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return entity.KeyDef{}
	}
	var keystroke entity.KeyDef
	json.Unmarshal(msg, &keystroke)
	return keystroke
}
