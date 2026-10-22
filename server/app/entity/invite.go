package entity

import "time"

type InviteStatus string

const (
	InvitePending  InviteStatus = "pending"
	InviteAccepted InviteStatus = "accepted"
	InviteDeclined InviteStatus = "declined"
	InviteExpired  InviteStatus = "expired"
)

type RoomInvite struct {
	ID        uint32       `gorm:"primaryKey;autoIncrement:false" json:"id"`
	RoomID    uint32       `gorm:"index;not null" json:"room_id"`
	InviterID uint32       `gorm:"not null" json:"inviter_id"`
	InviteeID uint32       `gorm:"index;not null" json:"invitee_id"`
	Status    InviteStatus `gorm:"size:16;not null;default:pending" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`

	Room    *Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Inviter *User `gorm:"foreignKey:InviterID" json:"inviter,omitempty"`
}

type InviteView struct {
	ID        uint32       `json:"id"`
	RoomCode  string       `json:"room_code"`
	RoomName  string       `json:"room_name"`
	From      string       `json:"from"`
	FromID    uint32       `json:"from_id"`
	Status    InviteStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}
