//go:build unit

package controlledlobby

import (
	"sync"
	"testing"

	"go.uber.org/zap"

	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

func newLobby() *ControlledLobby {
	// The session registry is real but empty: no player in these tests needs a
	// live WebSocket, and the roster is populated directly.
	sessions := usersession.NewActiveSessions(zap.NewNop().Sugar())
	return &ControlledLobby{
		logger:    zap.NewNop().Sugar(),
		generator: utils.NewGenerator(),
		lobby:     make(map[uint32][]entity.PlayerItem),
		session:   &sessions,
	}
}

// TestConcurrentLobbyAccessIsSafe is a regression test.
//
// The lobby roster is a plain map. Joins arrive on HTTP handler goroutines while
// the scheduler reads and deletes the same entry from its own goroutine, so
// under load the runtime raised "fatal error: concurrent map read and map
// write" and took the whole server down. Run this with -race to see the race
// itself; without it, the unsynchronized version fails with a concurrent map
// write.
func TestConcurrentLobbyAccessIsSafe(t *testing.T) {
	lobby := newLobby()

	const lobbies = 8
	const joinsPerLobby = 16

	var wg sync.WaitGroup

	// Writers, standing in for the HTTP handlers that join a lobby.
	for l := uint32(1); l <= lobbies; l++ {
		for range joinsPerLobby {
			wg.Add(1)
			go func() {
				defer wg.Done()
				lobby.addPlayer(l, entity.PlayerItem{ID: 1})
			}()
		}
	}

	// Readers and deleters, standing in for the scheduler draining a lobby.
	for range lobbies * 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for l := uint32(1); l <= lobbies; l++ {
				lobby.LobbySize(l)
				lobby.RemoveLobby(l)
			}
		}()
	}

	wg.Wait()
}

// TestLobbySizeCountsEveryJoin checks the observable state rather than just
// "it did not crash", so a lock that was added in the wrong place still fails.
func TestLobbySizeCountsEveryJoin(t *testing.T) {
	lobby := newLobby()
	const lobbyID = 42

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lobby.addPlayer(lobbyID, entity.PlayerItem{ID: 1})
		}()
	}

	// Concurrent readers must never observe a torn count.
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if n := lobby.LobbySize(lobbyID); n < 0 || n > 32 {
				t.Errorf("LobbySize returned an impossible count: %d", n)
			}
		}()
	}

	wg.Wait()

	if got := lobby.LobbySize(lobbyID); got != 32 {
		t.Errorf("expected all 32 joins to be recorded, got %d", got)
	}
}

// TestJoinWithoutASessionIsRejected is a regression test.
//
// JoinControlledLobby used to call c.session.GetSession(id).Subscribe() without
// checking the result, so a lobby join for a user with no live WebSocket
// dereferenced a nil pointer and the handler returned an unexplained 500.
func TestJoinWithoutASessionIsRejected(t *testing.T) {
	lobby := newLobby()

	err := lobby.JoinControlledLobby(999, 7)
	if err == nil {
		t.Fatal("expected an error when the user has no live session")
	}

	if got := lobby.LobbySize(7); got != 0 {
		t.Errorf("a rejected join left %d players on the roster", got)
	}
}

// TestStartGameFromAnUnknownLobbyReportsAnError checks that the empty check
// still runs before the game is started, now that the roster is taken under the
// lock and released before the round.
func TestStartGameFromAnUnknownLobbyReportsAnError(t *testing.T) {
	lobby := newLobby()

	sig := make(chan []entity.WPMRes)
	if err := lobby.StartGameFromLobby(1234, 0, sig); err == nil {
		t.Fatal("expected an error for an unknown lobby, got nil")
	}
}
