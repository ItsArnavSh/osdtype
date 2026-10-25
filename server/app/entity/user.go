package entity

import (
	"strconv"
	"time"
)

// StartingElo is the rating every new player begins with on every ranked
// clock. Competitive play only changes these numbers; there are no tiers,
// divisions or rank progress.
const StartingElo = 400

// Ranked durations, in seconds. Each clock owns an independent Elo ladder.
const (
	RankedDuration30  = 30
	RankedDuration90  = 90
	RankedDuration300 = 300
)

var RankedDurations = []int{RankedDuration30, RankedDuration90, RankedDuration300}

// EloView is the public per-clock rating projection.
type EloView struct {
	Rating      int `json:"rating"`
	PeakRating  int `json:"peak_rating"`
	GamesPlayed int `json:"games_played"`
	Wins        int `json:"wins"`
}

// ProfileVisibility controls the public profile card.
const (
	ProfilePublic    = "public"
	ProfileAnonymous = "anonymous"
	ProfilePrivate   = "private"
)

// RunSharing controls which finished runs appear on a public profile.
const (
	RunSharingAll    = "all"
	RunSharingPinned = "pinned"
	RunSharingNone   = "none"
)

// MatchRetentionLimit is the number of unpinned runs kept per account.
// Pinned runs are never pruned automatically.
const MatchRetentionLimit = 30

// User is a registered player. The per-clock fields are the source of truth
// for ranked standing. Rating and PeakRating cache the highest values across
// the three clocks for compact display in the hub and social surfaces.
type User struct {
	ID        uint32 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Username  string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	AvatarURL string `json:"avatar_url"`

	// Account customization. DisplayName and Bio are public; the sharing
	// fields are only returned through the authenticated settings endpoint.
	DisplayName       string `gorm:"size:40" json:"display_name,omitempty"`
	Bio               string `gorm:"size:280" json:"bio,omitempty"`
	ProfileVisibility string `gorm:"size:16;not null;default:'public'" json:"-"`
	DoNotDisturb      bool   `gorm:"not null;default:false" json:"-"`
	HideOnline        bool   `gorm:"not null;default:false" json:"-"`
	ShareRuns         string `gorm:"size:16;not null;default:'all'" json:"-"`
	ShareRanked       bool   `gorm:"not null;default:true" json:"-"`
	ShareRooms        bool   `gorm:"not null;default:true" json:"-"`
	SharePractice     bool   `gorm:"not null;default:true" json:"-"`
	ShareContests     bool   `gorm:"not null;default:true" json:"-"`
	HidePrivateRooms  bool   `gorm:"not null;default:true" json:"-"`
	ShareReplays      bool   `gorm:"not null;default:true" json:"-"`
	SaveRunData       bool   `gorm:"not null;default:true" json:"-"`

	Rating     int `gorm:"not null;default:400" json:"rating"`
	PeakRating int `gorm:"not null;default:400" json:"peak_rating"`

	Rating30      int `gorm:"column:rating_30;not null;default:400" json:"rating_30"`
	PeakRating30  int `gorm:"column:peak_rating_30;not null;default:400" json:"peak_rating_30"`
	GamesPlayed30 int `gorm:"column:games_played_30;not null;default:0" json:"games_played_30"`
	Wins30        int `gorm:"column:wins_30;not null;default:0" json:"wins_30"`

	Rating90      int `gorm:"column:rating_90;not null;default:400" json:"rating_90"`
	PeakRating90  int `gorm:"column:peak_rating_90;not null;default:400" json:"peak_rating_90"`
	GamesPlayed90 int `gorm:"column:games_played_90;not null;default:0" json:"games_played_90"`
	Wins90        int `gorm:"column:wins_90;not null;default:0" json:"wins_90"`

	Rating300      int `gorm:"column:rating_300;not null;default:400" json:"rating_300"`
	PeakRating300  int `gorm:"column:peak_rating_300;not null;default:400" json:"peak_rating_300"`
	GamesPlayed300 int `gorm:"column:games_played_300;not null;default:0" json:"games_played_300"`
	Wins300        int `gorm:"column:wins_300;not null;default:0" json:"wins_300"`

	GamesPlayed int       `gorm:"not null;default:0" json:"games_played"`
	Wins        int       `gorm:"not null;default:0" json:"wins"`
	IsBot       bool      `gorm:"not null;default:false" json:"is_bot"`
	IsAdmin     bool      `gorm:"not null;default:false;index" json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DisplayNameOrUsername returns the name a profile should lead with.
func (u User) DisplayNameOrUsername() string {
	if name := u.DisplayName; name != "" {
		return name
	}
	return u.Username
}

// SetDefaultSharing enables the public defaults for a new account.
func (u *User) SetDefaultSharing() {
	u.ProfileVisibility = ProfilePublic
	u.DoNotDisturb = false
	u.HideOnline = false
	u.ShareRuns = RunSharingAll
	u.ShareRanked = true
	u.ShareRooms = true
	u.SharePractice = true
	u.ShareContests = true
	u.HidePrivateRooms = true
	u.ShareReplays = true
	u.SaveRunData = true
}

// Visibility normalises the stored profile visibility.
func (u User) Visibility() string {
	switch u.ProfileVisibility {
	case ProfileAnonymous, ProfilePrivate:
		return u.ProfileVisibility
	default:
		return ProfilePublic
	}
}

// IsAnonymous reports whether public profile cards should be masked.
func (u User) IsAnonymous() bool {
	return u.Visibility() == ProfileAnonymous
}

// SharingMode normalises the stored sharing value. Older rows created before
// the setting existed are treated as all runs.
func (u User) SharingMode() string {
	switch u.ShareRuns {
	case RunSharingPinned, RunSharingNone:
		return u.ShareRuns
	default:
		return RunSharingAll
	}
}

// SharedModes lists the game modes this account currently exposes.
func (u User) SharedModes() []GameMode {
	modes := make([]GameMode, 0, 4)
	if u.ShareRanked {
		modes = append(modes, GameRanked)
	}
	if u.ShareRooms {
		modes = append(modes, GameRoom)
	}
	if u.SharePractice {
		modes = append(modes, GamePractice)
	}
	if u.ShareContests {
		modes = append(modes, GameContest)
	}
	return modes
}

// SharesRun reports whether one run is visible to somebody other than its
// owner. Profile and replay access use the same check.
func (u User) SharesRun(mode GameMode, pinned, roomPrivate bool) bool {
	sharing := u.SharingMode()
	if sharing == RunSharingNone {
		return false
	}
	if sharing == RunSharingPinned && !pinned {
		return false
	}
	switch mode {
	case GameRanked:
		if !u.ShareRanked {
			return false
		}
	case GameRoom:
		if !u.ShareRooms {
			return false
		}
		if u.HidePrivateRooms && roomPrivate {
			return false
		}
	case GamePractice:
		if !u.SharePractice {
			return false
		}
	case GameContest:
		if !u.ShareContests {
			return false
		}
	default:
		return false
	}
	return true
}

// RatingFor returns the Elo for one ranked clock. Invalid durations fall back
// to the 90s ladder; all callers validate the duration first.
func (u User) RatingFor(duration int) int {
	switch duration {
	case RankedDuration30:
		return u.Rating30
	case RankedDuration300:
		return u.Rating300
	default:
		return u.Rating90
	}
}

// PeakFor returns the peak Elo for one ranked clock.
func (u User) PeakFor(duration int) int {
	switch duration {
	case RankedDuration30:
		return u.PeakRating30
	case RankedDuration300:
		return u.PeakRating300
	default:
		return u.PeakRating90
	}
}

// GamesFor returns the completed ranked games for one ranked clock.
func (u User) GamesFor(duration int) int {
	switch duration {
	case RankedDuration30:
		return u.GamesPlayed30
	case RankedDuration300:
		return u.GamesPlayed300
	default:
		return u.GamesPlayed90
	}
}

// WinsFor returns the ranked wins for one ranked clock.
func (u User) WinsFor(duration int) int {
	switch duration {
	case RankedDuration30:
		return u.Wins30
	case RankedDuration300:
		return u.Wins300
	default:
		return u.Wins90
	}
}

// SetRatingFor updates one ladder and its peak, then refreshes the cached
// highest-across-clocks values.
func (u *User) SetRatingFor(duration, rating int) {
	switch duration {
	case RankedDuration30:
		u.Rating30 = rating
		if rating > u.PeakRating30 {
			u.PeakRating30 = rating
		}
	case RankedDuration300:
		u.Rating300 = rating
		if rating > u.PeakRating300 {
			u.PeakRating300 = rating
		}
	default:
		u.Rating90 = rating
		if rating > u.PeakRating90 {
			u.PeakRating90 = rating
		}
	}
	u.RefreshHighestElo()
}

// AddGameStatsFor records one ranked game on the game's clock.
func (u *User) AddGameStatsFor(duration int, won bool) {
	u.GamesPlayed++
	if won {
		u.Wins++
	}
	switch duration {
	case RankedDuration30:
		u.GamesPlayed30++
		if won {
			u.Wins30++
		}
	case RankedDuration300:
		u.GamesPlayed300++
		if won {
			u.Wins300++
		}
	default:
		u.GamesPlayed90++
		if won {
			u.Wins90++
		}
	}
}

// SetAllRatings seeds every ladder, used for new accounts and bots.
func (u *User) SetAllRatings(rating int) {
	u.Rating30, u.PeakRating30 = rating, rating
	u.Rating90, u.PeakRating90 = rating, rating
	u.Rating300, u.PeakRating300 = rating, rating
	u.RefreshHighestElo()
}

// RefreshHighestElo recomputes the cached highest rating and peak.
func (u *User) RefreshHighestElo() {
	u.Rating = max(u.Rating30, u.Rating90, u.Rating300)
	u.PeakRating = max(u.PeakRating30, u.PeakRating90, u.PeakRating300)
}

// RatingView returns the public stats for one clock.
func (u User) RatingView(duration int) EloView {
	return EloView{
		Rating:      u.RatingFor(duration),
		PeakRating:  u.PeakFor(duration),
		GamesPlayed: u.GamesFor(duration),
		Wins:        u.WinsFor(duration),
	}
}

// UserView is the public, JSON-friendly projection of a user.
type UserView struct {
	ID           uint32             `json:"id"`
	Username     string             `json:"username"`
	DisplayName  string             `json:"display_name,omitempty"`
	Bio          string             `json:"bio,omitempty"`
	AvatarURL    string             `json:"avatar_url"`
	Rating       int                `json:"rating"`
	PeakRating   int                `json:"peak_rating"`
	GamesPlayed  int                `json:"games_played"`
	Wins         int                `json:"wins"`
	IsBot        bool               `json:"is_bot"`
	IsAdmin      bool               `json:"is_admin,omitempty"`
	Ratings      map[string]EloView `json:"ratings"`
	BestDuration int                `json:"best_duration"`
	// SaveRunData is only populated for the signed-in account. Public
	// projections leave it nil so sharing preferences do not leak.
	SaveRunData *bool `json:"save_run_data,omitempty"`
}

func NewUserView(u User) UserView {
	ratings := make(map[string]EloView, len(RankedDurations))
	best := 0
	bestDuration := RankedDuration90
	for _, duration := range RankedDurations {
		view := u.RatingView(duration)
		ratings[strconv.Itoa(duration)] = view
		if view.Rating > best || (view.Rating == best && duration == RankedDuration90) {
			best = view.Rating
			bestDuration = duration
		}
	}
	if best == 0 {
		best = StartingElo
	}
	return UserView{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Bio:          u.Bio,
		AvatarURL:    u.AvatarURL,
		Rating:       best,
		PeakRating:   max(u.PeakRating30, u.PeakRating90, u.PeakRating300),
		GamesPlayed:  u.GamesPlayed,
		Wins:         u.Wins,
		IsBot:        u.IsBot,
		IsAdmin:      u.IsAdmin,
		Ratings:      ratings,
		BestDuration: bestDuration,
	}
}
