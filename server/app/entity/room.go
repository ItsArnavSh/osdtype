package entity

type room_perm int

const (
	MOD room_perm = iota
	MEMBER
	BLOCKED
	LEFT
)

type room_type int

const (
	PRIVATE room_type = iota //Invite-only
	PUBLIC                   //Can be joined simply
)

type Room struct {
	ID     uint32
	Name   string
	Desc   string
	Public room_type
}
type Room_User struct {
	RoomID uint32
	UserID uint32
	Perm   room_perm
}
