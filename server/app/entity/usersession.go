package entity

// UserStatus is what a connected player is doing, so the interface can show
// "in a game" next to a friend and stop inviting them into one.
type UserStatus uint16

const (
	// AVAILABLE is connected and not in a game, so an invitation is welcome.
	AVAILABLE UserStatus = iota
	// PLAYING is connected and in a game. They are online but not available.
	PLAYING
	// OFFLINE is not connected.
	OFFLINE
)

// String is the wire name for a status.
//
// It is used by the presence endpoints, so the name is part of the protocol:
// a client switching on it needs the same spellings this produces.
func (u UserStatus) String() string {
	switch u {
	case AVAILABLE:
		return "available"
	case PLAYING:
		return "playing"
	case OFFLINE:
		return "offline"
	}
	return "unknown"
}
