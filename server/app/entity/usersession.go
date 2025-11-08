package entity

type UserStatus uint16

const (
	AVAILABLE UserStatus = iota
	PLAYING
	OFFLINE
)
