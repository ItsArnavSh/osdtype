package entity

import (
	"time"

	"github.com/google/btree"
	"github.com/gorilla/websocket"
)

type LobbyEntry struct {
	ID       uint64
	Rank     uint16
	JoinedAt time.Time
	Websock  *websocket.Conn
}
type PlayerItem LobbyEntry

func (a PlayerItem) Less(b btree.Item) bool {
	if a.Rank == b.(PlayerItem).Rank {
		return a.ID < b.(PlayerItem).ID
	}
	return a.Rank < b.(PlayerItem).Rank
}

type LobbyType int

const (
	SPRINT LobbyType = iota
	STANDARD
	MARATHON
)

func (l LobbyType) Duration() time.Duration {
	switch l {
	case SPRINT:
		return time.Second * 30
	case STANDARD:
		return time.Second * 90
	case MARATHON:
		return time.Second * 300
	}
	return time.Second * 60
}
