package entity

// ReplayRun is one player's recorded keypress stream inside a game. The
// browser derives the graph and the simulated playback from this data.
type ReplayRun struct {
	UserID    uint32     `json:"user_id"`
	Username  string     `json:"username"`
	AvatarURL string     `json:"avatar_url"`
	WPM       float64    `json:"wpm"`
	Raw       float64    `json:"raw"`
	Accuracy  float64    `json:"accuracy"`
	Correct   int        `json:"correct"`
	Wrong     int        `json:"wrong"`
	IsBot     bool       `json:"is_bot"`
	Final     string     `json:"final"`
	Events    []Keypress `json:"events"`
}

// ReplayBundle is everything a results page or a shared replay page needs.
// Public replay responses contain a single run; the protected results endpoint
// contains every recorded run so the combined graph can be drawn.
type ReplayBundle struct {
	Game GameView    `json:"game"`
	Runs []ReplayRun `json:"runs"`
}

type PracticeRunView struct {
	ID        uint32 `json:"id"`
	Temporary bool   `json:"temporary"`
	URL       string `json:"url"`
}
