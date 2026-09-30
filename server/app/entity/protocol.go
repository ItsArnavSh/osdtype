package entity

import (
	"encoding/json"
	"strconv"
	"time"
)

// This file is the single definition of every frame that crosses the user
// WebSocket. The TypeScript mirror lives in
// app/src/lib/core/api/protocol.ts; the two are kept in step by hand.
//
// Every server frame is a flat JSON object with a "type" discriminator,
// because the client needs to route on it without inspecting the rest of the
// payload. Frames are handed to a session already marshaled, so the session
// never touches JSON and cannot accidentally base64 a byte slice the way a
// WriteJSON of a []byte would.
//
// Frame types the server sends.
const (
	// FrameSeed opens a game. It carries the generator seed and the language
	// it selected, so a client that has the grammar can regenerate the exact
	// same snippet locally without the server streaming code over the wire.
	FrameSeed = "seed"

	// FrameCountdown precedes a game start, counting down to zero.
	FrameCountdown = "countdown"
	// FrameStart is the zero of that countdown: type now.
	FrameStart = "start"

	// FrameKeystroke is a live broadcast of one player's input.
	FrameKeystroke = "keystroke"

	// FrameEnd carries the final leaderboard and closes the game.
	FrameEnd = "end"

	// FrameNotification delivers something out of band: an invitation, a
	// match being found, a contest about to begin.
	FrameNotification = "notification"

	// FrameMatchFound tells a queued player their game is being set up.
	FrameMatchFound = "match_found"
	// FrameQueueStatus tells a queued player where they are in the queue.
	FrameQueueStatus = "queue_status"
	// FrameContestStarting tells members a contest lobby is about to run.
	FrameContestStarting = "contest_starting"

	// FrameRoster carries a room's current membership.
	FrameRoster = "roster"

	// FrameError reports something the player did wrong, in a form the client
	// can show verbatim.
	FrameError = "error"
	// FramePong answers a client ping.
	FramePong = "pong"
)

// Frame types the client sends.
const (
	// ClientKeypress is a single input event. It is sent flat rather than
	// nested in a payload object because it is the highest frequency frame in
	// the protocol and there is no reason to make it larger.
	ClientKeypress = "keypress"
	// ClientReady says a lobby member is ready to start.
	ClientReady = "ready"
	// ClientLeave says a player is abandoning a lobby or a game.
	ClientLeave = "leave"
	// ClientPing keeps an idle socket alive through intermediaries.
	ClientPing = "ping"
)

// Encode marshals a frame for the wire.
//
// A frame that cannot be marshaled is replaced with an error frame rather
// than dropped, so a bug in one frame type cannot silently disconnect a game.
func Encode(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"error","message":"could not encode frame"}`)
	}
	return raw
}

// SeedFrame opens a game.
type SeedFrame struct {
	Type       string `json:"type"`
	Value      string `json:"value"`
	Lang       string `json:"lang"`
	Tokens     int    `json:"tokens"`
	DurationMS int64  `json:"duration_ms"`
}

// NewSeedFrame describes a game about to start.
func NewSeedFrame(seed uint32, lang Language, tokens int, d time.Duration) SeedFrame {
	return SeedFrame{
		Type:       FrameSeed,
		Value:      strconv.FormatUint(uint64(seed), 10),
		Lang:       lang.String(),
		Tokens:     tokens,
		DurationMS: d.Milliseconds(),
	}
}

// CountdownFrame counts down to a game starting.
type CountdownFrame struct {
	Type    string `json:"type"`
	Seconds int    `json:"seconds"`
}

// KeystrokeFrame is a live broadcast of one player's input.
//
// PlayerID is the player the keystroke came from, CurrentPoints is how many
// characters they have produced so far, and Update is the raw event so a
// client can render it against the snippet.
type KeystrokeFrame struct {
	Type          string   `json:"type"`
	PlayerID      uint32   `json:"player_id"`
	CurrentPoints uint16   `json:"current_points"`
	Update        Keypress `json:"update"`
}

// EndFrame carries the result of a finished game.
type EndFrame struct {
	Type        string   `json:"type"`
	Reason      string   `json:"reason"`
	Leaderboard []WPMRes `json:"leaderboard"`
}

// Why a game ended, so a client can explain itself rather than just stopping.
const (
	// EndFinished means every player ran out of time or finished the snippet.
	EndFinished = "finished"
	// EndAborted means the game was canceled before it could complete.
	EndAborted = "aborted"
)

// MatchFoundFrame tells a queued player a game is being prepared for them.
type MatchFoundFrame struct {
	Type       string `json:"type"`
	LobbyType  string `json:"lobby_type"`
	DurationMS int64  `json:"duration_ms"`
	Players    int    `json:"players"`
	Countdown  int    `json:"countdown_seconds"`
}

// QueueStatusFrame tells a queued player their position in the queue.
type QueueStatusFrame struct {
	Type      string `json:"type"`
	LobbyType string `json:"lobby_type"`
	Size      int    `json:"size"`
	Position  int    `json:"position"`
}

// ContestStartingFrame warns members that a contest begins shortly.
type ContestStartingFrame struct {
	Type    string `json:"type"`
	JobID   uint32 `json:"job_id"`
	Seconds int    `json:"seconds"`
}

// ErrorFrame reports a client-visible failure.
type ErrorFrame struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewErrorFrame builds an error frame.
func NewErrorFrame(msg string) ErrorFrame {
	return ErrorFrame{Type: FrameError, Message: msg}
}

// ClientFrame is anything a client sends.
//
// Keypress is embedded rather than nested so the highest frequency frame in
// the protocol stays flat on the wire. The extra zero-valued fields on a
// "ready" or "leave" frame are harmless.
type ClientFrame struct {
	Type string `json:"type"`
	Keypress
}

// UnmarshalClientFrame decodes an inbound message.
//
// A frame the client sent as a bare keypress object with no "type" is
// accepted as a keypress, because that was the shape the game loop accepted
// before the protocol was formalized and an in-flight client may still be
// using it.
func UnmarshalClientFrame(raw []byte) (ClientFrame, error) {
	var f ClientFrame
	if err := json.Unmarshal(raw, &f); err != nil {
		return ClientFrame{}, err
	}
	if f.Type == "" {
		f.Type = ClientKeypress
	}
	return f, nil
}
