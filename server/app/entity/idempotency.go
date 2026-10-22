package entity

import "time"

// IdempotencyKey stores the response of a mutating request so retries never
// create duplicate rooms, invites or matches.
type IdempotencyKey struct {
	Key       string    `gorm:"primaryKey;size:80" json:"key"`
	UserID    uint32    `gorm:"index" json:"user_id"`
	Method    string    `gorm:"size:8" json:"method"`
	Path      string    `gorm:"size:200" json:"path"`
	Status    int       `json:"status"`
	Response  []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
