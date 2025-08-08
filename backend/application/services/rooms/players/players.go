package players

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"osdtype/application/auth"
	"osdtype/application/entity"
	livetype "osdtype/application/services/typing"
	"osdtype/application/services/typing/wpm"
	"osdtype/application/util"
	"osdtype/database"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
)

type Player struct {
	Conn          *websocket.Conn
	Rec           chan []byte
	Username      string
	Typed_snippet string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewPlayer(ctx *gin.Context) (Player, error) {
	username, err := auth.GetUser(ctx)
	if err != nil {
		return Player{}, err
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return Player{}, err
	}
	return Player{Conn: conn, Rec: make(chan []byte), Username: username, Typed_snippet: ""}, err
}

type PlayerHub struct {
	Players    map[*Player]bool
	Register   chan *Player
	Disconnect chan *Player
	Keychan    chan KeyPress
}
type KeyPress struct {
	UserName string
	action   string
}

func NewPlayerHub() PlayerHub {
	return PlayerHub{
		Players:    make(map[*Player]bool),
		Register:   make(chan *Player),
		Disconnect: make(chan *Player),
		Keychan:    make(chan KeyPress),
	}
}
func (p *Player) SendSnippet(snippet string) {
	p.Conn.WriteMessage(websocket.TextMessage, []byte(snippet))
}

func (p *PlayerHub) RunHub(wg *sync.WaitGroup, key chan KeyPress) {
	//Call all when the timer is up for all players
	defer wg.Done()
	for {
		select {
		case v := <-p.Register:
			p.Players[v] = true
		case v := <-p.Disconnect:
			if _, ok := p.Players[v]; ok {
				delete(p.Players, v)
				close(v.Rec)
			}
		case keyp := <-p.Keychan:
			//Decide what to do when player actually sends a keychan
			//Nothing is done here with the key its just there...
			key <- keyp
		}
	}
}
func (p *Player) RecordKeystrokes(ctx context.Context, ess entity.Essentials, snippet database.LanguageStore, config entity.GameConf, keychan chan KeyPress) error {
	run_id := uuid.NewString()
	rec := entity.Recording{
		OriginalID: snippet.ID,
	}
	var typChan = make(chan entity.KeyDef)
	typeStruct := livetype.Typer{
		Query:   ess.Db,
		Logger:  *ess.Logger,
		Rec:     rec,
		KeyChan: typChan,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go typeStruct.LiveSave(ctx, &wg)

	defer wg.Wait()
	//Max deadline before crash
	p.Conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	first_hit := livetype.GetKeyStrokes(p.Conn)
	start_time := first_hit.Time
	latest_time := int64(0)
	typChan <- first_hit
	for {
		// Read message from client

		keystroke := livetype.GetKeyStrokes(p.Conn)
		emp := entity.KeyDef{}
		if keystroke == emp {
			break
		}
		latest_time = keystroke.Time
		typChan <- keystroke
		keychan <- KeyPress{UserName: p.Username, action: keystroke.Delta}

	}
	close(typChan)
	//After this the frontend will send us the snippet as text
	// First we send it a "end" message
	p.Conn.WriteMessage(websocket.TextMessage, []byte("end"))

	_, usersnippet, err := p.Conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return err
	}
	st_time := time.Unix(start_time, 0)
	//Save the events in the database as Pending
	wpm := wpm.Calculate_WPM(entity.WPM{OriginalSnippet: snippet.Snippet, UserSnippet: string(usersnippet), DurationMS: latest_time - start_time})
	wpm_json, err := json.Marshal(wpm)
	if err != nil {
		return err
	}
	err = p.Conn.WriteMessage(websocket.TextMessage, wpm_json)
	if err != nil {
		ess.Logger.Error("Could not send the WPM data")
	}
	user := p.Username
	ess.Db.CreateTypeRun(ctx, database.CreateTypeRunParams{
		StartTime: pgtype.Timestamp{Time: st_time},
		ID:        run_id,
		RunData:   rec.Recording,
		UserID:    user,
		Language:  config.Language,
		Wpm:       wpm.WPM,
		RawWpm:    wpm.RAW,
		SnippetID: snippet.ID,
		Delta:     util.GenerateDeltas(snippet.Snippet, string(usersnippet)),
	})
	//Set off to the anticheat software non blocking
	ess.Bus.Publish("cheatcheck", typChan)
	return nil
}
