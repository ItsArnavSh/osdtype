package entity

import "time"

type Relation int

const (
	FOLLOWS Relation = iota
	FRIENDS
)

type Friends struct {
	A         uint32    `gorm:"primaryKey;column:a_id" json:"a_id"`
	B         uint32    `gorm:"primaryKey;column:b_id" json:"b_id"`
	Relation  Relation  `gorm:"not null;default:0" json:"relation"`
	CreatedAt time.Time `json:"created_at"`
}
