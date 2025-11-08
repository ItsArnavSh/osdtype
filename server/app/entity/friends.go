package entity

type Relation int

const (
	FOLLOWS Relation = iota
	FRIENDS
)

type Friends struct {
	A        uint64 `gorm:"primaryKey"`
	B        uint64 `gorm:"primaryKey"`
	Relation Relation
}
type Invite struct {
	From    string `json:"from"`
	LobbyID uint64 `json:"lobby_id"`
}
