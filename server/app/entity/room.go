package entity

import "errors"

// Sentinels for the two ways joining a room can be refused.
//
// They live here because the transaction that raises them is in the
// persistence layer and the interface that answers 400 lives in the api
// layer, and neither of those should have to import the other to recognize the
// same refusal. Callers match with errors.Is.
var (
	// ErrRoomFull means the room is at capacity.
	ErrRoomFull = errors.New("room is full")
	// ErrRoomBlocked means this player has been removed from the room and may
	// not come back.
	ErrRoomBlocked = errors.New("you are blocked from this room")
)

// A Room is both a social space and a private game lobby.
//
// It started as only the first, with a name, a description and a membership
// table. A share-code lobby is the same shape: someone creates it, others join
// it, and the host decides when it runs. Keeping one model means a friends'
// game and a contest are the same object with a different creator.
//
// The share code is a stable public identifier, deliberately separate from ID.
// ID is a 32 bit snowflake of roughly ten digits; a code is six characters a
// person can read off a screen and type back in.
type Room struct {
	ID   uint32
	Name string
	Desc string

	// Public decides whether a room can be joined without a code. PRIVATE is
	// the default for a game lobby.
	Public RoomVisibility

	// Code is the share code. Unique across rooms, so it can be the join
	// path in a URL.
	Code string `gorm:"uniqueIndex;size:16"`

	// PasswordHash is a bcrypt hash, empty when the room is open. It is
	// deliberately not tagged gorm:"-": the column has to exist for the hash to
	// survive a restart, or every lobby with a password would silently reopen
	// as soon as the process reloaded. It is never serialized, which the
	// json:"-" tag guarantees: a response body is built from exported fields,
	// and a leaked hash is enough to mount an offline dictionary attack.
	PasswordHash string `gorm:"size:72" json:"-"`

	// PasswordSet mirrors whether PasswordHash is non-empty, so a response can
	// tell a client to prompt without carrying the hash itself.
	PasswordSet bool

	// Capacity is how many players the room holds.
	Capacity int

	// Duration is the game mode: SPRINT, STANDARD or MARATHON.
	Duration LobbyType

	// Status is where the room is in its lifecycle, which is what lets the
	// interface show "waiting for players" versus "running".
	Status RoomStatus

	// CreatorID is who is allowed to start the game.
	CreatorID uint32
}

// RoomVisibility decides whether a room can be found and joined freely.
//
// It was an unexported type behind exported constants, which made it
// impossible for the service layer to write a value of it: entity.PRIVATE was
// a constant, so there was no way to put a visibility on a room built
// elsewhere. Naming the type is what makes the field assignable.
type RoomVisibility int

const (
	// PRIVATE requires a share code, and a password if one is set.
	PRIVATE RoomVisibility = iota
	// PUBLIC can be joined by anyone who finds it.
	PUBLIC
)

// String is the wire and database name for a room's privacy.
func (r RoomVisibility) String() string {
	if r == PUBLIC {
		return "public"
	}
	return "private"
}

// IsPublic reports whether a room is freely joinable.
func (r Room) IsPublic() bool { return r.Public == PUBLIC }

// RoomStatus is a room's lifecycle state.
type RoomStatus int

const (
	// LobbyOpen means the room is accepting players.
	LobbyOpen RoomStatus = iota
	// LobbyStarting means a game has been launched from this room.
	LobbyStarting
	// LobbyRunning means a game is in progress.
	LobbyRunning
	// LobbyFinished means the last game in this room has ended.
	LobbyFinished
)

// String is the wire name for a status.
func (s RoomStatus) String() string {
	switch s {
	case LobbyStarting:
		return "starting"
	case LobbyRunning:
		return "running"
	case LobbyFinished:
		return "finished"
	}
	return "open"
}

// RoomPerm is a user's standing in a room.
//
// It was unexported, which meant other packages could read Room_User.Perm but
// never write a meaningful value, so every membership was written as the zero
// value.
type RoomPerm int

const (
	// MOD can manage the room, invite people and start games.
	MOD RoomPerm = iota
	// MEMBER is an ordinary participant.
	MEMBER
	// BLOCKED cannot rejoin.
	BLOCKED
	// LEFT left voluntarily and may return.
	LEFT
)

// String is the wire name for a permission level.
func (p RoomPerm) String() string {
	switch p {
	case MOD:
		return "mod"
	case MEMBER:
		return "member"
	case BLOCKED:
		return "blocked"
	case LEFT:
		return "left"
	}
	return "unknown"
}

// IsActive reports whether a permission level means "in the room".
func (p RoomPerm) IsActive() bool { return p == MOD || p == MEMBER }

// Room_User is a membership row.
type Room_User struct {
	RoomID uint32 `gorm:"primaryKey"`
	UserID uint32 `gorm:"primaryKey"`
	Perm   RoomPerm
}

// RoomMember is a roster entry as the interface needs it: who the player is,
// whether they are connected right now, and what their rating is.
type RoomMember struct {
	UserID    uint32   `json:"user_id"`
	Username  string   `json:"username"`
	AvatarURL string   `json:"avatar_url"`
	Rank      uint16   `json:"rank"`
	Perm      RoomPerm `json:"perm"`
	IsMod     bool     `json:"is_mod"`
	Online    bool     `json:"online"`
	// Ready is whether a member has said they are ready to start. The
	// interface shows it as a tick next to the name.
	Ready bool `json:"ready"`
}

// RosterMember is the minimal shape needed to build a game from a room.
type RosterMember struct {
	UserID   uint32
	Username string
	Rank     uint16
	JoinedAt int64
}

// RosterFrame is pushed whenever a room's membership changes, so every member
// sees the list update without polling.
type RosterFrame struct {
	Type    string       `json:"type"`
	RoomID  uint32       `json:"room_id"`
	Members []RoomMember `json:"members"`
}
