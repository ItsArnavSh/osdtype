package entity

import "time"

type ContestStatus int

const (
	UPCOMING ContestStatus = iota
	LOBBY
	STARTED
	ENDED
)

type Contest struct {
	JobID    uint64
	RoomID   uint64
	Time     time.Time
	Data     []byte //The title, writeup etc
	Lang     Language
	Duration LobbyType
	LobbyID  uint64 //Will be alloted by the scheduler
	Status   ContestStatus
}
type ContestLeaderboard struct {
	UserID   uint64
	Position int
	Stats    WPMRes
}
