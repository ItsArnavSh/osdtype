package player

import (
	"encoding/json"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type OutGoing struct { //Common comms channel for the GameHandler
	PlayerID      uint64          `json:"player_id"`
	CurrentPoints uint16          `json:"current_points"`
	Update        entity.Keypress //For the frontend to caliberate
}

// Handling the player logic
// Will receive messages from the player, and handle them
type Player struct {
	State     strings.Builder
	ID        uint64
	Rank      uint16
	In        chan entity.Keypress
	Out       chan OutGoing //Global Out
	LocalOut  chan OutGoing //To send updates to ws
	Logger    *zap.SugaredLogger
	WebSocIn  <-chan []byte
	WebSocOut chan<- any
	//Maintain a outqueue to send appended messages instead of one by one via ws
	history   []entity.Keypress
	Snippet   string
	CloseTime time.Duration
	Duration  time.Duration
	WG        *sync.WaitGroup
}

func (p *Player) PlayerOutUpdate() {

	for game := range p.LocalOut {
		p.WebSocOut <- game
		if game.PlayerID == 0 {
			close(p.In)
			close(p.LocalOut)
			p.WG.Done()
			return
		}
	}
	for range p.LocalOut {
	}
	p.WG.Done()
}

func (p *Player) PlayerInRoutine() {
	//The websocket interface that gets updates from here
	// And also updates this person

	for message := range p.WebSocIn {
		var msg entity.Keypress
		if err := json.Unmarshal(message, &msg); err != nil {
			p.Logger.Warnf("Invalid JSON:  %v", err)
			continue
		}
		if p.CloseTime == 0 { //Hasent been set yet
			starttime := time.Millisecond * time.Duration(msg.TimeMS)
			p.CloseTime = p.Duration + starttime
		}
		//Will update the code
		if p.CloseTime > time.Millisecond*time.Duration(msg.TimeMS) {
			p.HandlePress(msg)
		}
	}
}
func (p *Player) HandlePress(keypress entity.Keypress) {
	switch keypress.Action {
	case entity.KEYPRESS:
		p.State.WriteString(keypress.Value)

	case entity.BACKSPACE:
		current := p.State.String()
		if strings.HasSuffix(current, keypress.Value) {
			p.State.Reset()
			p.State.WriteString(current[:len(current)-len(keypress.Value)])
		}
	}
	p.Send(keypress)
}

func (p *Player) Send(keypress entity.Keypress) {

	update := OutGoing{
		PlayerID: p.ID,
		Update:   keypress,
	}

	select {
	case p.Out <- update:
	default:
	}
}
func (p *Player) CalculateScore() entity.WPMRes {
	input := entity.WPM{
		OriginalSnippet: p.Snippet,
		UserSnippet:     p.State.String(),
		DurationMS:      p.Duration.Milliseconds(),
	}
	wpm_res := utils.Calculate_WPM(input)
	wpm_res.ID = p.ID
	return wpm_res
}
