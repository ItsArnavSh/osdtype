package entity

import (
	"time"

	"gorm.io/datatypes"
)

// User is an account.
//
// The model is deliberately thin: an account is an identity and a rating.
// Anything else a person accumulates, a run or a friendship, is derived from
// these two rather than cached on the row.
type User struct {
	ID        uint32 `gorm:"primaryKey;autoIncrement:false"`
	Username  string `gorm:"uniqueIndex"`
	AvatarURL string

	// CurrentRank is the Elo rating.
	//
	// It is a uint16 because the reachable range is 100 to 3000, which leaves
	// two orders of magnitude of headroom in a 16 bit column. The arithmetic
	// is always done in float so a rating cannot wrap during a calculation.
	CurrentRank uint16

	// GamesPlayed is how many rated results have been recorded. It is what
	// makes the K-factor decay, so a provisional player is not stuck: without
	// it a rating could never leave the starting value quickly.
	GamesPlayed int
}

// WinCount and LossCount are not stored: they are derived from runs rather
// than incremented in place, so they cannot drift out of step with the run
// table.

// Run is one completed game, single player or multiplayer.
//
// Recording runs is what turns the rating from "changed somehow" into something
// a player can see a history of, and it is what gives the leaderboard and the
// anticheat something to read.
type Run struct {
	ID     uint32 `gorm:"primaryKey;autoIncrement:false"`
	UserID uint32 `gorm:"index"`

	// Mode is SPRINT, STANDARD or MARATHON.
	Mode LobbyType

	// Solo is true for a practice run and false for a rated game.
	Solo bool `gorm:"index"`

	WPM      float32
	Raw      float32
	Accuracy float32
	Correct  int32
	Wrong    int32

	// DurationMS is how long the player had.
	DurationMS int64

	// RankBefore and RankAfter bracket the rating change, so a leaderboard row
	// can show "+18" without replaying every intermediate run.
	RankBefore uint16
	RankAfter  uint16

	// Language and Seed identify the snippet, so a run can be replayed and so
	// two runs over the same text can be compared.
	Language Language
	Seed     uint32

	// Passed is the anticheat verdict. A failed run is stored but does not
	// move the rating.
	Passed bool

	// CheatScore is the anticheat's numeric verdict, kept for diagnostics.
	CheatScore int

	PlayedAt time.Time `gorm:"index"`
}

// RankChange describes what one result did to a rating, so the interface can
// report "+18, Silver to Gold" without recomputing anything.
type RankChange struct {
	Before  uint16 `json:"before"`
	After   uint16 `json:"after"`
	Delta   int    `json:"delta"`
	Tier    string `json:"tier"`
	OldTier string `json:"old_tier"`
}

// Promoted reports whether the change crossed a tier boundary.
func (r RankChange) Promoted() bool { return r.Tier != r.OldTier && r.Delta > 0 }

// Demoted reports whether the change dropped a tier.
func (r RankChange) Demoted() bool { return r.Tier != r.OldTier && r.Delta < 0 }

// RunResult is what a run submission returns: the stored run, what it did to
// the rating, and whether the anticheat cleared it.
type RunResult struct {
	Run     Run        `json:"run"`
	Change  RankChange `json:"change"`
	Passed  bool       `json:"passed"`
	Message string     `json:"message"`
}

// LeaderboardEntry is one row of a leaderboard response, with the display
// decoration resolved on the server so every client agrees on tier names.
type LeaderboardEntry struct {
	Rank      int     `json:"rank"`
	UserID    uint32  `json:"user_id"`
	Username  string  `json:"username"`
	AvatarURL string  `json:"avatar_url"`
	Rating    uint16  `json:"rating"`
	Tier      string  `json:"tier"`
	WPM       float32 `json:"wpm"`
	Accuracy  float32 `json:"accuracy"`
	Games     int     `json:"games"`
	IsSelf    bool    `json:"is_self"`
}

// LeaderboardPage is a page of leaderboard rows plus the caller's own rank,
// which may well be outside the page they asked for.
type LeaderboardPage struct {
	Entries   []LeaderboardEntry `json:"entries"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	Total     int64              `json:"total"`
	SelfRank  int                `json:"self_rank"`
	SelfEntry *LeaderboardEntry  `json:"self_entry,omitempty"`
}

// FriendCard is a person in a friend list, with everything the interface
// needs to render a row including whether they are online now.
type FriendCard struct {
	UserID    uint32     `json:"user_id"`
	Username  string     `json:"username"`
	AvatarURL string     `json:"avatar_url"`
	Rating    uint16     `json:"rating"`
	Tier      string     `json:"tier"`
	Online    bool       `json:"online"`
	Status    UserStatus `json:"status"`
	// Friends distinguishes a mutual follow from a one-way one, so a row can
	// show "friends" versus "following".
	Friends bool `json:"friends"`
}

// UserSearchResult is a row of the player search.
type UserSearchResult struct {
	UserID    uint32 `json:"user_id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Rating    uint16 `json:"rating"`
	Tier      string `json:"tier"`
	Online    bool   `json:"online"`
}

// LanguageOption describes a language the snippet service can generate for,
// so a client can render the picker without hardcoding the list.
type LanguageOption struct {
	Value  int    `json:"value"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

// ModeOption describes a game mode, for the same reason.
type ModeOption struct {
	Value   int    `json:"value"`
	Name    string `json:"name"`
	Seconds int    `json:"seconds"`
}

// GraphPayload is a player's rating history, for the profile graph.
type GraphPayload struct {
	Points []GraphPoint `json:"points"`
}

// GraphPoint is one rating sample.
type GraphPoint struct {
	At     int64   `json:"at"`
	Rating uint16  `json:"rating"`
	WPM    float32 `json:"wpm"`
}

// RunPage is a page of a player's recent runs.
type RunPage struct {
	Runs  []Run `json:"runs"`
	Total int64 `json:"total"`
}

// Settings are the per-user preferences the interface writes.
type Settings struct {
	UserID uint32 `gorm:"primaryKey"`
	// Language is the user's preferred typing language, used to preselect the
	// picker and to pin solo runs to one language.
	Language Language
	// DefaultMode is the mode a ranked queue click starts in.
	DefaultMode LobbyType
	// Theme is "system", "light" or "dark".
	Theme string
	// PublicProfile controls whether the player appears on the leaderboard.
	PublicProfile bool
	// ShowLiveKeystrokes controls whether other players' input is rendered.
	ShowLiveKeystrokes bool
	Extra              datatypes.JSON `gorm:"-"`
}
