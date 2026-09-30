package entity

import (
	"time"

	"github.com/google/btree"
)

// Socket is a revocable lease on a user's WebSocket.
//
// The interface lives here rather than in usersession because usersession
// already imports entity: declaring it the other way round would be an import
// cycle. It also means nothing outside usersession can depend on the concrete
// channel plumbing, which is what let a game handler hold a reference to a
// socket that had already been closed.
//
// Exactly one lease is active at a time, because a WebSocket has exactly one
// writer. Taking a lease revokes the previous one by closing its Recv
// channel, rather than blocking on a mutex that a caller might hold forever.
// That blocking was the deadlock: joining a lobby took the lock and only ever
// released it if something else happened to send an unsubscribe sentinel down
// the same channel, so a player in a lobby could never queue for ranked.
type Socket interface {
	// Send delivers a server-to-client frame. It never blocks and never
	// panics, including after the session has been torn down.
	Send(raw []byte)
	// Recv is the stream of client frames, closed when the lease is revoked
	// or the session goes away.
	Recv() <-chan []byte
	// Release gives up the lease. Safe to call more than once.
	Release()
}

// LobbyEntry is a queued player as the matchmaker sees it.
type LobbyEntry struct {
	ID       uint32
	Name     string
	Rank     uint16
	JoinedAt time.Time
	Session  Socket
}

// PlayerItem is the matchmaker's btree element.
//
// Less orders by rank and then by id. The id tiebreak matters: a btree
// requires a strict weak ordering, and ordering on rank alone would make two
// equal-rank players compare equal, which lets the tree drop entries.
type PlayerItem LobbyEntry

func (a PlayerItem) Less(b btree.Item) bool {
	if a.Rank == b.(PlayerItem).Rank {
		return a.ID < b.(PlayerItem).ID
	}
	return a.Rank < b.(PlayerItem).Rank
}

// LobbyType is a game mode, named for the duration a round lasts.
type LobbyType int

const (
	SPRINT LobbyType = iota
	STANDARD
	MARATHON
)

// Duration is how long one round of this mode lasts.
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

// String is the wire and database name for the mode.
func (l LobbyType) String() string {
	switch l {
	case SPRINT:
		return "sprint"
	case STANDARD:
		return "standard"
	case MARATHON:
		return "marathon"
	}
	return "unknown"
}

// LobbyTypeFromString resolves a mode name, reporting whether it was known.
func LobbyTypeFromString(s string) (LobbyType, bool) {
	switch s {
	case "sprint", "30":
		return SPRINT, true
	case "standard", "90":
		return STANDARD, true
	case "marathon", "300":
		return MARATHON, true
	}
	return SPRINT, false
}
