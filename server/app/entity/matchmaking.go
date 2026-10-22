package entity

import "time"

type LobbyType int

const (
	SPRINT LobbyType = iota
	STANDARD
	MARATHON
)

func (l LobbyType) Seconds() int {
	switch l {
	case SPRINT:
		return 30
	case STANDARD:
		return 90
	case MARATHON:
		return 300
	}
	return 90
}

func (l LobbyType) Duration() time.Duration {
	return time.Duration(l.Seconds()) * time.Second
}

func LobbyTypeFromSeconds(s int) (LobbyType, bool) {
	switch s {
	case 30:
		return SPRINT, true
	case 90:
		return STANDARD, true
	case 300:
		return MARATHON, true
	}
	return 0, false
}
