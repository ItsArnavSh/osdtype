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

	// HostID is whoever scheduled the contest. It becomes the creator of the
	// contest's lobby room, so the host can start it and manage its roster.
	HostID uint32
}

// Title is the contest's display name.
//
// It reads the Data column, which is where the create endpoint puts the title.
// That column carries both the title and the writeup, so a contest whose data
// is blank falls back to a generated name rather than an empty room name.
func (c Contest) Title() string {
	if c.Data == "" {
		return "Contest"
	}
	// The data column is written as "<title>|<writeup>" by the create handler.
	for i := range c.Data {
		if c.Data[i] == '|' {
			return c.Data[:i]
		}
	}
	return c.Data
}

// Writeup is the prose part of the data column, if there is one.
func (c Contest) Writeup() string {
	for i := range c.Data {
		if c.Data[i] == '|' {
			return c.Data[i+1:]
		}
	}
	return ""
}
