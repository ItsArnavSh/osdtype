//go:build unit

package entity

import (
	"testing"
	"time"
)

func TestLanguageStringCoversEveryLanguage(t *testing.T) {
	langs := []Language{C, GO, CPP, JAVA, RUST, TYPESCRIPT}
	for _, l := range langs {
		if l.String() == "" {
			t.Errorf("language %d has no name", int(l))
		}
	}
	if got := CPP.String(); got != "cpp" {
		t.Errorf("expected cpp, got %q", got)
	}
	if got := RUST.String(); got != "rs" {
		t.Errorf("expected rs, got %q", got)
	}
	if got := TYPESCRIPT.String(); got != "ts" {
		t.Errorf("expected ts, got %q", got)
	}
}

// TestLanguageStringOutOfRange is a regression test.
//
// String() indexes a fixed-size array with the language value, so any value
// outside 0..5 panicked with an index out of range. The value arrives from the
// database and from a websocket seed, so a bad row could take the process down.
// Out-of-range values now return an empty string.
func TestLanguageStringOutOfRange(t *testing.T) {
	for _, l := range []Language{-1, 6, 99, 1000} {
		got := l.String()
		if got != "" {
			t.Errorf("language %d returned %q, expected an empty string", int(l), got)
		}
	}
}

func TestLobbyTypeDurations(t *testing.T) {
	cases := map[LobbyType]time.Duration{
		SPRINT:   30 * time.Second,
		STANDARD: 90 * time.Second,
		MARATHON: 300 * time.Second,
	}
	for typ, want := range cases {
		if got := typ.Duration(); got != want {
			t.Errorf("lobby %d: expected %v, got %v", int(typ), want, got)
		}
	}
}

// TestLobbyTypeDefaultDuration checks the fallback for an unknown value.
func TestLobbyTypeDefaultDuration(t *testing.T) {
	if got := LobbyType(42).Duration(); got != time.Minute {
		t.Errorf("expected the 60s fallback, got %v", got)
	}
}

// TestPlayerItemOrdering pins the btree ordering the matchmaker relies on:
// by rank, then by id so the order is total and two equal ranks cannot
// compare equal.
func TestPlayerItemOrdering(t *testing.T) {
	a := PlayerItem{ID: 1, Rank: 1000}
	b := PlayerItem{ID: 2, Rank: 1000}
	c := PlayerItem{ID: 3, Rank: 2000}

	if !a.Less(b) {
		t.Error("equal ranks should order by id")
	}
	if b.Less(a) {
		t.Error("ordering is not antisymmetric")
	}
	if a.Less(c) != true {
		t.Error("a lower rank should sort first")
	}
	if c.Less(a) {
		t.Error("a higher rank should not sort first")
	}
}

func TestPlayerItemLessIsStrictTotalOrder(t *testing.T) {
	// The btree requires Less to be a strict weak ordering. A player must
	// never be less than itself.
	p := PlayerItem{ID: 7, Rank: 1500}
	if p.Less(p) {
		t.Error("an item must not be less than itself")
	}
}
