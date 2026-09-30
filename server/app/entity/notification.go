package entity

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// Notifications are the durable half of the push channel.
//
// A frame pushed over the WebSocket is lost the moment a player closes the
// tab, so every notification is also written here. The socket delivers the
// live copy; this table is what the client reads on reconnect to catch up.

// Kind identifies what a notification is about.
type Kind string

const (
	// KindLobbyInvite is an invitation to a private lobby.
	KindLobbyInvite Kind = "lobby_invite"
	// KindFriendAdded fires when a mutual follow becomes a friendship.
	KindFriendAdded Kind = "friend_added"
	// KindMatchFound fires when a ranked queue produces a game.
	KindMatchFound Kind = "match_found"
	// KindContestStarting fires shortly before a contest lobby runs.
	KindContestStarting Kind = "contest_starting"
	// KindRankChanged fires when a rating moves enough to be worth reporting.
	KindRankChanged Kind = "rank_changed"
)

// Notification is one undelivered message for one user.
type Notification struct {
	ID        uint32 `gorm:"primaryKey;autoIncrement:false"`
	UserID    uint32 `gorm:"index"`
	Kind      Kind
	Payload   datatypes.JSON
	Read      bool `gorm:"index"`
	CreatedAt time.Time
}

// LobbyInvite is the payload of a KindLobbyInvite notification.
type LobbyInvite struct {
	From      string `json:"from"`
	FromID    uint32 `json:"from_id"`
	RoomID    uint32 `json:"room_id"`
	RoomCode  string `json:"room_code"`
	RoomName  string `json:"room_name"`
	ExpiresAt int64  `json:"expires_at"`
}

// RankChanged is the payload of a KindRankChanged notification.
type RankChanged struct {
	Old     uint16 `json:"old"`
	New     uint16 `json:"new"`
	Delta   int    `json:"delta"`
	Tier    string `json:"tier"`
	OldTier string `json:"old_tier"`
}

// NotificationFrame is the envelope pushed over the socket.
//
// The payload stays as raw JSON so a notification frame can be decoded by a
// client that knows its kind without the server having to model every
// payload shape in one struct.
type NotificationFrame struct {
	Type string          `json:"type"`
	Kind Kind            `json:"kind"`
	ID   uint32          `json:"id"`
	Body json.RawMessage `json:"body"`
	At   int64           `json:"at"`
}
