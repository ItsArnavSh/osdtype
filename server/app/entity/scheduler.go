package entity

import (
	"time"
)

type jobtyp int

const (
	CONTEST jobtyp = iota
)

type Task struct {
	Category jobtyp
	JobID    uint64
	MetaData string
	Time     time.Time
}
type Contest struct {
	RoomID   uint64
	Data     []byte //The title, writeup etc
	Lang     Language
	Duration LobbyType
}
