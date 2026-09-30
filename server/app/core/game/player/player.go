package player

import (
	"strings"
	"sync"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

// Player is one participant in a running game.
//
// It owns the typing state and the outbound half of the live broadcast. The
// inbound half is the session lease in Session, which is closed if the socket
// dies or another consumer takes the socket over.
type Player struct {
	// State is what the player has typed so far. It is only touched by
	// PlayerInRoutine, which runs on its own goroutine, so it needs no lock.
	State strings.Builder

	Name     string
	ID       uint32
	Rank     uint16
	Duration time.Duration

	// Session is the lease on this player's socket. GlobalBroadcaster sends
	// live keystrokes through it.
	Session entity.Socket

	// Local receives this player's own keystrokes after they have been
	// broadcast, so a client can render its own input without the server
	// echoing a separate message.
	Local chan entity.KeystrokeFrame

	// Broadcast is the shared fan-out queue every player in the game writes
	// to and one reader drains.
	Broadcast chan entity.KeystrokeFrame

	Snippet string
	Logger  *zap.SugaredLogger

	started  time.Time
	finished bool
}

// New builds a player bound to a lease and a shared broadcast queue.
func New(item entity.PlayerItem, snippet string, bcast chan entity.KeystrokeFrame, logger *zap.SugaredLogger, duration time.Duration) *Player {
	return &Player{
		Name:      item.Name,
		ID:        item.ID,
		Rank:      item.Rank,
		Duration:  duration,
		Session:   item.Session,
		Snippet:   snippet,
		Logger:    logger,
		Broadcast: bcast,
		Local:     make(chan entity.KeystrokeFrame, 32),
	}
}

// PlayerInRoutine reads this player's frames and applies them until the round
// ends.
//
// The clock starts on the first keystroke rather than when the game begins, so
// a player who joins a countdown late is not charged for the countdown. It
// ends either when the round's duration elapses or when the lease closes,
// which is what makes a disconnect return promptly instead of sitting out the
// whole round.
func (p *Player) PlayerInRoutine(wg *sync.WaitGroup) {
	defer wg.Done()

	p.Logger.Infow("player routine started", "player_id", p.ID)

	var deadline <-chan time.Time
	timer := time.NewTimer(p.Duration)
	defer timer.Stop()
	deadline = timer.C

	for {
		select {
		case message, ok := <-p.Session.Recv():
			if !ok {
				// The lease was revoked: either the socket died or something
				// else took the socket over.
				p.Logger.Infow("player input stream ended", "player_id", p.ID)
				return
			}

			frame, err := entity.UnmarshalClientFrame(message)
			if err != nil {
				p.Logger.Warnw("unparseable frame", "player_id", p.ID, "error", err)
				continue
			}

			switch frame.Type {
			case entity.ClientLeave:
				p.Logger.Infow("player left", "player_id", p.ID)
				return

			case entity.ClientPing:
				p.Session.Send(entity.Encode(struct {
					Type string `json:"type"`
				}{Type: entity.FramePong}))
				continue

			case entity.ClientKeypress:
				// fall through to handling below
			default:
				p.Logger.Debugw("ignoring frame in game",
					"player_id", p.ID, "frame_type", frame.Type)
				continue
			}

			if frame.Value == "" {
				// An empty keypress carries no information and would corrupt
				// the typed state, so it is dropped rather than applied.
				continue
			}

			if p.started.IsZero() {
				p.started = time.Now()
				// Re-arm the deadline for a full round from the first keystroke.
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(p.Duration)
				deadline = timer.C
				p.Logger.Infow("player started typing", "player_id", p.ID)
			}

			p.HandlePress(frame.Keypress)

		case <-deadline:
			p.Logger.Infow("round timer expired", "player_id", p.ID)
			p.finish()
			return
		}
	}
}

// HandlePress applies one input event to the typed state and broadcasts it.
//
// The broadcast used to be commented out, which meant the entire live
// keystroke pipeline had no writer: the game handler's fan-out queue was
// never fed, so no player ever saw another player type.
func (p *Player) HandlePress(keypress entity.Keypress) {
	before := p.State.Len()

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

	// Only tell everyone about events that actually changed the state, which
	// keeps a spam of backspaces over a character that is not there from
	// flooding the room.
	if p.State.Len() == before {
		return
	}

	frame := entity.KeystrokeFrame{
		Type:          entity.FrameKeystroke,
		PlayerID:      p.ID,
		CurrentPoints: cappedPoints(p.State.Len()),
		Update:        keypress,
	}

	select {
	case p.Broadcast <- frame:
	default:
		// The fan-out reader is behind. Dropping a live keystroke is
		// harmless: it is a cosmetic broadcast, and the score is computed from
		// the typed state, not from what was broadcast.
		p.Logger.Debugw("broadcast queue full, dropping keystroke", "player_id", p.ID)
	}

	select {
	case p.Local <- frame:
	default:
	}
}

// finish marks the player as done so their score is final.
func (p *Player) finish() {
	p.finished = true
}

// CalculateScore scores the run.
func (p *Player) CalculateScore() entity.WPMRes {
	typed := p.State.String()

	// Only compare against the part of the snippet the player could possibly
	// have reached. Slicing to State.Len() panicked when a player typed more
	// characters than the snippet contains, which any client could trigger by
	// simply holding a key down.
	input := entity.WPM{
		OriginalSnippet: p.reachableSnippet(),
		UserSnippet:     typed,
		DurationMS:      p.Duration.Milliseconds(),
	}

	res := utils.Calculate_WPM(input)
	res.Name = p.Name
	res.ID = p.ID

	p.Logger.Infow("score calculated",
		"player_id", p.ID,
		"typed_length", len(typed),
		"wpm", res.WPM,
		"accuracy", res.Accuracy)

	return res
}

// reachableSnippet returns as much of the snippet as the player typed, capped
// at the snippet length. A player who overshoots gets the whole snippet back,
// and the excess characters count as errors.
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

// maxPoints is the largest progress value that fits the wire field.
const maxPoints = 1<<16 - 1

// cappedPoints clamps a typed-character count into the range the broadcast
// frame can carry, so a player who holds a key down cannot wrap the counter.
func cappedPoints(n int) uint16 {
	if n < 0 {
		return 0
	}
	if n > maxPoints {
		return maxPoints
	}
	return uint16(n)
}
