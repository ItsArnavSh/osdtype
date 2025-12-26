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
	JobID    uint32
	Time     time.Time
}
