package entity

import "errors"

// The ranked ladder: what a rating means, what a run is, and what a submission
// looks like.
//
// The rating used to be a number on a user row that changed with no record of
// why. Nothing stored the game that moved it, so a player could not see a
// history, the leaderboard had nothing to page through, and the anticheat had
// no keystrokes to look at. Everything here exists to make a result
// observable.

// ErrUnknownMode is returned for a game mode name or value the server does not
// run.
//
// It lives here rather than in the service layer because both the service and
// the database raise it, and a handler that wants to answer 400 has to be able
// to recognize both without importing both.
var ErrUnknownMode = errors.New("unknown game mode")

// ErrUnknownLanguage is the same for a language the server does not have text
// for.
var ErrUnknownLanguage = errors.New("unknown language")

// Tier is a named band of the rating range.
type Tier struct {
	Name  string `json:"name"`
	Min   uint16 `json:"min"`
	Color string `json:"color"`
}

// QueueStatus is a reply to a join or leave, so the interface can render the
// queue state from the response alone rather than from a separate poll.
type QueueStatus struct {
	Type string `json:"type"`
	// LobbyType is the mode, by name.
	LobbyType string `json:"lobby_type"`
	// Position is the caller's place in the queue, one based.
	Position int `json:"position"`
	// Size is how many are waiting.
	Size int `json:"size"`
	// Rating and Tier are the caller's standing, because the queue is the
	// moment a player wants to know their number.
	Rating uint16 `json:"rating"`
	Tier   string `json:"tier"`
	// Queued says whether the caller is actually in the queue. A leave reply
	// has it false.
	Queued bool `json:"queued"`
}

// LeaderboardOptions narrows a leaderboard request.
type LeaderboardOptions struct {
	// Scope is "global", "friends" or "room".
	Scope string `json:"scope"`
	// RoomID scopes to one room's members. It is only read when Scope is
	// "room"; the friends scope always means the caller's own friends.
	RoomID uint32 `json:"room_id"`
	// Mode filters by game mode, by name. Empty means every mode.
	Mode string `json:"mode"`
	// SoloOnly restricts to practice runs.
	SoloOnly bool `json:"solo_only"`
	// Page is zero based.
	Page int `json:"page"`
	// PageSize is how many rows to return.
	PageSize int `json:"page_size"`
}

// RunSubmission is what a client sends when a solo run finishes.
//
// The client scores locally, because it is the only party that saw the whole
// run: the server would have to hold the keystroke stream for a round it did
// not observe. What it does verify is the timing log, which is what the
// anticheat reads, and the seed, which ties the result to a text the server
// can regenerate.
type RunSubmission struct {
	// Mode is the game mode, by name.
	Mode string `json:"mode"`
	// Language is the language code or number, as the seed frame named it.
	Language string `json:"language"`
	// Seed is the seed the server sent, as a decimal string. It identifies the
	// snippet, so a result cannot be claimed for a different one.
	Seed string `json:"seed"`

	WPM      float32 `json:"wpm"`
	Raw      float32 `json:"raw"`
	Accuracy float32 `json:"accuracy"`
	Correct  int32   `json:"correct"`
	Wrong    int32   `json:"wrong"`
	// DurationMS is how long the player had.
	DurationMS int64 `json:"duration_ms"`

	// TimestampsMS is when each keystroke happened, in order. It is the only
	// part of the submission the server can check for itself, and it is what
	// makes a run faked at a plausible speed detectable at all.
	TimestampsMS []int32 `json:"timestamps_ms"`
}

// MaxSubmissionTimestamps bounds how many timestamps a submission may carry.
//
// A round's keystroke count is bounded by the round: 1000 characters over at
// most five minutes is at most a few thousand events. A client that sends more
// than this is either broken or trying to make the anticheat do quadratic
// work on attacker-chosen input, and either way it is not a real run.
const MaxSubmissionTimestamps = 20_000

// Valid reports whether a submission is internally coherent.
//
// The score fields are sanity bounded rather than exact: the server cannot
// recompute WPM from the timestamps without knowing the snippet, but a score
// outside a physically possible range is a client that has lost track of its
// own state, and accepting it would let a bug inflate a rating.
func (r RunSubmission) Valid() bool {
	if r.DurationMS <= 0 {
		return false
	}
	if r.WPM < 0 || r.WPM > maxPlausibleWPM {
		return false
	}
	if r.Accuracy < 0 || r.Accuracy > 100 {
		return false
	}
	if r.Correct < 0 || r.Wrong < 0 {
		return false
	}
	if len(r.TimestampsMS) > MaxSubmissionTimestamps {
		return false
	}
	return true
}

// maxPlausibleWPM is beyond any human record for a sustained run. It is set
// generously so a fast typist is never rejected, but low enough that a runaway
// counter is caught.
const maxPlausibleWPM = 500
