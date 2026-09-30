package player

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type OutGoing struct { // Common comms channel for the GameHandler
	PlayerID      uint32          `json:"player_id"`
	CurrentPoints uint16          `json:"current_points"`
	Update        entity.Keypress // For the frontend to caliberate
}

// Handling the player logic
// Will receive messages from the player, and handle them
type Player struct {
	State     strings.Builder
	Name      string
	ID        uint32
	Rank      uint16
	In        chan entity.Keypress
	Out       chan OutGoing // Global Out
	LocalOut  chan OutGoing // To send updates to ws
	Logger    *zap.SugaredLogger
	WebSocIn  <-chan []byte
	WebSocOut chan<- any
	// Maintain a outqueue to send appended messages instead of one by one via ws
	Snippet   string
	CloseTime time.Time
	Duration  time.Duration
}

func (p *Player) PlayerInRoutine(wg *sync.WaitGroup) {
	defer wg.Done()

	p.Logger.Infow("player input routine started",
		"player_id", p.ID,
	)

	ticker := time.NewTicker(30 * time.Second)

	for {
		select {
		case message, ok := <-p.WebSocIn:
			if !ok {
				// Player disconnected
				p.Logger.Infow("websocket input channel closed",
					"player_id", p.ID,
				)
				ticker.Stop()
				return
			}
			p.Logger.Debugw("message received",
				"player_id", p.ID,
				"raw", string(message),
			)

			var msg entity.Keypress
			if err := json.Unmarshal(message, &msg); err != nil {
				p.Logger.Warnw("invalid json received",
					"player_id", p.ID,
					"error", err,
				)
				continue
			}

			if p.CloseTime.IsZero() {
				starttime := time.UnixMilli(msg.TimeMS)
				p.CloseTime = starttime.Add(p.Duration)

				p.Logger.Infow("player started typing",
					"player_id", p.ID,
					"start_time", starttime,
					"close_time", p.CloseTime,
				)

				ticker.Stop()
				ticker = time.NewTicker(p.Duration)
			}

			p.HandlePress(msg)

		case <-ticker.C:
			ticker.Stop()

			p.Logger.Infow("player timer expired",
				"player_id", p.ID,
			)

			return
		}
	}
}
func (p *Player) HandlePress(keypress entity.Keypress) {
	p.Logger.Debugw("handling keypress",
		"player_id", p.ID,
		"action", keypress.Action,
		"value", keypress.Value,
	)

	switch keypress.Action {
	case entity.KEYPRESS:
		p.State.WriteString(keypress.Value)
		p.Logger.Debugln("Len of string is", p.State.Len())

	case entity.BACKSPACE:
		current := p.State.String()
		if strings.HasSuffix(current, keypress.Value) {
			p.State.Reset()
			p.State.WriteString(current[:len(current)-len(keypress.Value)])
		}
	}

	// p.Send(keypress)
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
	p.Logger.Debugw("calculating score",
		"player_id", p.ID,
		"typed", p.State.Len(),
		"snippet_len", len(p.Snippet),
	)

	input := entity.WPM{
		// Only compare against the part of the snippet the player could
		// possibly have reached. Slicing to State.Len() panicked when a player
		// typed more characters than the snippet contains, which any client
		// could trigger by simply keeping the keyboard held down.
		OriginalSnippet: p.reachableSnippet(),
		UserSnippet:     p.State.String(),
		DurationMS:      p.Duration.Milliseconds(),
	}

	p.Logger.Infow("calculating score",
		"player_id", p.ID,
		"typed_length", len(input.UserSnippet),
	)

	wpm_res := utils.Calculate_WPM(input)
	wpm_res.Name = p.Name

	p.Logger.Infow("score calculated",
		"player_id", p.ID,
		"wpm", wpm_res.WPM,
		"accuracy", wpm_res.Accuracy,
	)

	return wpm_res
}

// reachableSnippet returns as much of the snippet as the player typed, capped
// at the snippet length. A player who overshoots the snippet gets the whole
// snippet back; Calculate_WPM counts the excess characters as errors.
func (p *Player) reachableSnippet() string {
	n := p.State.Len()
	if n > len(p.Snippet) {
		n = len(p.Snippet)
	}
	if n < 0 {
		n = 0
	}
	return p.Snippet[:n]
}
