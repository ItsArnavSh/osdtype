package entity

type action uint8

const (
	KEYPRESS action = iota
	BACKSPACE
)

type Keypress struct {
	// GameID is optional. The game page includes it so a keypress queued
	// during a reconnect can never be applied to a different match.
	GameID uint32 `json:"game_id,omitempty"`
	Value  string `json:"value"`
	Action action `json:"action"`
	TimeMS int64  `json:"time_ms"`
}
