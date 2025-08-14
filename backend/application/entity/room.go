package entity

type RoomRole struct {
}

// Contest Role
// Define all the room structs here
type GameConf struct {
	Language string
	time     int
}

// Room structs
type Room struct {
	RoomName string `json:"room_name"`
}
type AddPlayer struct {
	PlayerID string `json:"player_id"`
	RoomID   string `json:"room_id"`
}

type ChangePermsStruct struct {
	PlayerID string `json:"player_id"`
	NewPerm  string `json:"NewPerm"`
	RoomID   string `json:"room_id"`
}
type RemovePlayer struct {
	PlayerID string `json:"player_id"`
	RoomID   string `json:"room_id"`
}
