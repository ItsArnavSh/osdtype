package entity

import (
	"time"

	"gorm.io/datatypes"
)

type ContestStatus int

const (
	UPCOMING ContestStatus = iota
	LOBBY
	STARTED
	ENDED
)

type Contest struct {
	ID          string `gorm:"primaryKey"`
	JobID       uint32
	RoomID      uint32
	Time        time.Time
	Data        string // The title, writeup etc
	Lang        Language
	Duration    LobbyType
	LobbyID     uint32 // Will be alloted by the scheduler
	Status      ContestStatus
	Leaderboard datatypes.JSON
}
