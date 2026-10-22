package entity

import "time"

type jobtyp string

const (
	CONTEST jobtyp = "contest"
)

type Task struct {
	ID        uint32    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	Category  jobtyp    `gorm:"size:16;not null" json:"category"`
	JobID     uint32    `gorm:"index;not null" json:"job_id"`
	Time      time.Time `gorm:"index;not null" json:"time"`
	Completed bool      `gorm:"not null;default:false" json:"completed"`
}
