package entity

import "time"

type ContestStatus string

const (
	ContestUpcoming ContestStatus = "upcoming"
	ContestLobby    ContestStatus = "lobby"
	ContestStarted  ContestStatus = "started"
	ContestEnded    ContestStatus = "ended"
)

type Contest struct {
	ID           uint32        `gorm:"primaryKey;autoIncrement:false" json:"id"`
	RoomID       uint32        `gorm:"index;not null" json:"room_id"`
	CreatedBy    uint32        `gorm:"not null" json:"created_by"`
	Title        string        `gorm:"size:120;not null" json:"title"`
	Description  string        `gorm:"size:1000" json:"description"`
	StartsAt     time.Time     `json:"starts_at"`
	Duration     int           `gorm:"not null;default:90" json:"duration"`
	Language     string        `gorm:"size:16;not null;default:go" json:"language"`
	Status       ContestStatus `gorm:"size:16;not null;default:upcoming" json:"status"`
	LobbyOpensAt *time.Time    `json:"lobby_opens_at"`
	GameID       uint32        `json:"game_id"`
	Leaderboard  []byte        `gorm:"type:bytea" json:"-"`
	CreatedAt    time.Time     `json:"created_at"`
}
