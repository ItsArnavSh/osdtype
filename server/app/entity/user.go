package entity

import "time"

// User is a registered player. Rating is the single source of truth for a
// player's competitive rank; the tier/division label is derived from it.
type User struct {
	ID          uint32    `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Username    string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	AvatarURL   string    `json:"avatar_url"`
	Rating      int       `gorm:"not null;default:1000" json:"rating"`
	PeakRating  int       `gorm:"not null;default:1000" json:"peak_rating"`
	GamesPlayed int       `gorm:"not null;default:0" json:"games_played"`
	Wins        int       `gorm:"not null;default:0" json:"wins"`
	IsBot       bool      `gorm:"not null;default:false" json:"is_bot"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RankInfo is the presentation form of a rating.
type RankInfo struct {
	Tier     string  `json:"tier"`
	Division int     `json:"division"`
	Label    string  `json:"label"`
	Rating   int     `json:"rating"`
	Next     int     `json:"next"`
	Progress float64 `json:"progress"`
}

type rankTier struct {
	name  string
	start int
}

// Tiers are 200 points wide and split into four 50 point divisions. New
// players start at 1000 (Diamond II) and climb towards Grandmaster.
var rankTiers = []rankTier{
	{"Bronze", 0},
	{"Silver", 200},
	{"Gold", 400},
	{"Platinum", 600},
	{"Diamond", 800},
	{"Master", 1200},
	{"Grandmaster", 1600},
}

// RatingFloor is the minimum rating a player can fall to.
const RatingFloor = 0

// RankFor converts a rating into a tier label.
func RankFor(rating int) RankInfo {
	if rating < RatingFloor {
		rating = RatingFloor
	}

	idx := 0
	for i, tier := range rankTiers {
		if rating >= tier.start {
			idx = i
		}
	}
	tier := rankTiers[idx]
	next := tier.start + 200
	if idx+1 < len(rankTiers) {
		next = rankTiers[idx+1].start
	} else {
		next = tier.start + 400
	}

	// Four divisions spread evenly across the tier. Grandmaster is the top
	// tier and deliberately has no division label.
	width := next - tier.start
	division := 4 - ((rating-tier.start)*4)/width
	if division < 1 {
		division = 1
	}
	if division > 4 {
		division = 4
	}
	label := tier.name + " " + roman(division)
	if idx == len(rankTiers)-1 {
		label = tier.name
		division = 0
	}

	progress := float64(rating-tier.start) / float64(width)
	if progress > 1 {
		progress = 1
	}
	return RankInfo{
		Tier:     tier.name,
		Division: division,
		Label:    label,
		Rating:   rating,
		Next:     next,
		Progress: progress,
	}
}

func roman(n int) string {
	switch n {
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	default:
		return "IV"
	}
}

// UserView is the public, JSON-friendly projection of a user.
type UserView struct {
	ID          uint32   `json:"id"`
	Username    string   `json:"username"`
	AvatarURL   string   `json:"avatar_url"`
	Rating      int      `json:"rating"`
	PeakRating  int      `json:"peak_rating"`
	GamesPlayed int      `json:"games_played"`
	Wins        int      `json:"wins"`
	IsBot       bool     `json:"is_bot"`
	Rank        RankInfo `json:"rank"`
}

func NewUserView(u User) UserView {
	return UserView{
		ID:          u.ID,
		Username:    u.Username,
		AvatarURL:   u.AvatarURL,
		Rating:      u.Rating,
		PeakRating:  u.PeakRating,
		GamesPlayed: u.GamesPlayed,
		Wins:        u.Wins,
		IsBot:       u.IsBot,
		Rank:        RankFor(u.Rating),
	}
}
