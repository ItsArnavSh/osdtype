package entity

type room_perm int

const (
	MOD room_perm = iota
	MEMBER
	BLOCKED
)

type room_type int

const (
	PRIVATE room_type = iota //Invite-only
	PUBLIC                   //Can be joined simply
)

type Room struct {
	Name   string
	Desc   string
	Public room_type
}
type Room_User struct {
	RoomID uint64
	UserID uint64
	Perm   room_perm
}
