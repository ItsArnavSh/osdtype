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

// User is a registered player. The per-clock fields are the source of truth
// for ranked standing. Rating and PeakRating cache the highest values across
// the three clocks for compact display in the hub and social surfaces.
type User struct {
	ID        uint32 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Username  string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	AvatarURL string `json:"avatar_url"`

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
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	AvatarURL    string             `json:"avatar_url"`
	Rating       int                `json:"rating"`
	PeakRating   int                `json:"peak_rating"`
	GamesPlayed  int                `json:"games_played"`
	Wins         int                `json:"wins"`
	IsBot        bool               `json:"is_bot"`
	Ratings      map[string]EloView `json:"ratings"`
	BestDuration int                `json:"best_duration"`
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
		AvatarURL:    u.AvatarURL,
		Rating:       best,
		PeakRating:   max(u.PeakRating30, u.PeakRating90, u.PeakRating300),
		GamesPlayed:  u.GamesPlayed,
		Wins:         u.Wins,
		IsBot:        u.IsBot,
		Ratings:      ratings,
		BestDuration: bestDuration,
	}
}
