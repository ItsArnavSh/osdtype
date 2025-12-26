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
	JobID       uint32
	RoomID      uint32
	Time        time.Time
	Data        []byte //The title, writeup etc
	Lang        Language
	Duration    LobbyType
	LobbyID     uint32 //Will be alloted by the scheduler
	Status      ContestStatus
	Leaderboard []WPMRes
}
