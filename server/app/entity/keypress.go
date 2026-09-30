package entity

// Action describes what a keypress did. It was unexported, which meant code
// outside this package could read Keypress.Action but never construct one.
type Action uint8

const (
	KEYPRESS Action = iota
	BACKSPACE
)

type Keypress struct {
	Value  string `json:"value"`
	Action Action `json:"action"`
	TimeMS int64  `json:"time_ms"`
}
