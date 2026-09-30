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
	// JobID is minted by the caller so it can be written to the contest row and
	// the task row together. autoIncrement is turned off explicitly: GORM treats
	// an integer primary key as a serial by default, and an older schema in the
	// wild already has a nextval() default here, which would let the sequence
	// hand out an id the contest row does not know about.
	JobID uint32 `gorm:"primaryKey;autoIncrement:false"`
	Time  time.Time
}
