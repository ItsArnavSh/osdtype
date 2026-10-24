package game

import (
	"testing"
	"time"

	"osdtyp/app/entity"
)

func TestHandleInputRejectsKeypressFromAnotherGame(t *testing.T) {
	now := time.Now().UTC()
	ends := now.Add(time.Minute)
	rt := &runtime{
		game: entity.Game{
			ID:          42,
			Duration:    90,
			Snippet:     "abcdef",
			StartOffset: 0,
			StartsAt:    &now,
			EndsAt:      &ends,
		},
		players: map[uint32]*playerState{
			1: {user: entity.User{ID: 1}},
		},
		order:   []uint32{1},
		started: now.Add(-time.Second),
		ends:    ends,
		done:    make(chan struct{}),
	}
	m := &Manager{
		games:    map[uint32]*runtime{42: rt},
		userGame: map[uint32]uint32{1: 42},
	}

	stale := entity.Keypress{GameID: 99, Value: "x", Action: entity.KEYPRESS, TimeMS: 120}
	if m.HandleInput(1, stale) {
		t.Fatal("stale keypress should have been rejected")
	}
	if got := string(rt.players[1].typed); got != "" {
		t.Fatalf("stale keypress mutated typed state: %q", got)
	}

	current := entity.Keypress{GameID: 42, Value: "a", Action: entity.KEYPRESS, TimeMS: 120}
	if !m.HandleInput(1, current) {
		t.Fatal("keypress for the active game was rejected")
	}
	if got := string(rt.players[1].typed); got != "a" {
		t.Fatalf("typed = %q, want a", got)
	}
	events := rt.players[1].events
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].TimeMS != 120 {
		t.Fatalf("replay time = %d, want client time 120", events[0].TimeMS)
	}
}
