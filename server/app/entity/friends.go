package entity

type Relation int

const (
	FOLLOWS Relation = iota
	FRIENDS
)

type Friends struct {
	A        uint32 `gorm:"primaryKey"`
	B        uint32 `gorm:"primaryKey"`
	Relation Relation
}
type Invite struct {
	From    string `json:"from"`
	LobbyID uint32 `json:"lobby_id"`
}
