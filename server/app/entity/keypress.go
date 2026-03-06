package entity

type action uint8

const (
	KEYPRESS action = iota
	BACKSPACE
)

type Keypress struct {
	Value  string `json:"value"`
	Action action `json:"action"`
	TimeMS int64  `json:"time_ms"`
}
