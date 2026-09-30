package entity

// RoomPerm is a user's standing in a room. It was unexported, which meant
// other packages could read Room_User.Perm but never write a meaningful value.
type RoomPerm int

const (
	MOD RoomPerm = iota
	MEMBER
	BLOCKED
	LEFT
)

type roomType int

const (
	PRIVATE roomType = iota // Invite-only
	PUBLIC                  // Can be joined simply
)

type Room struct {
	ID     uint32
	Name   string
	Desc   string
	Public roomType
}
type Room_User struct {
	RoomID uint32
	UserID uint32
	Perm   RoomPerm
}
