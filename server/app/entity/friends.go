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
