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
	Time     time.Time
}
