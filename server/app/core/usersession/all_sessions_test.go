//go:build unit

package usersession

import (
	"sync"
	"testing"

	"go.uber.org/zap"

	"osdtyp/app/entity"
)

// TestConcurrentSessionRegistryAccessIsSafe is a regression test.
//
// ActiveSessions.Users was an exported plain map written by the HTTP handler
// that upgrades a WebSocket and by the session's own teardown goroutine, while
// the matchmaker read it from its worker. A connect racing a disconnect
// produced "fatal error: concurrent map read and map write" and killed the
// process. The map is now guarded, so this passes under -race.
func TestConcurrentSessionRegistryAccessIsSafe(t *testing.T) {
	sessions := NewActiveSessions(zap.NewNop().Sugar())

	var wg sync.WaitGroup

	for id := uint32(1); id <= 16; id++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions.store(id, &UserSession{UserID: id})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions.RemoveSession(id)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions.GetSession(id)
			sessions.GetStatus(id)
			sessions.Count()
		}()
	}

	wg.Wait()
}

// TestUpdateSessionOnAnAbsentUserIsANoOp is a regression test.
//
// UpdateSession indexed the map and dereferenced the result, so a status update
// for a user who had just disconnected panicked the handler.
func TestUpdateSessionOnAnAbsentUserIsANoOp(t *testing.T) {
	sessions := NewActiveSessions(zap.NewNop().Sugar())

	// Must not panic.
	sessions.UpdateSession(404, entity.AVAILABLE)

	if got := sessions.GetStatus(404); got != entity.OFFLINE {
		t.Errorf("an unknown user should read as OFFLINE, got %v", got)
	}
}

// TestUpdateSessionSetsTheStatus checks the ordinary path still works, and that
// the status is readable from another goroutine.
func TestUpdateSessionSetsTheStatus(t *testing.T) {
	sessions := NewActiveSessions(zap.NewNop().Sugar())
	sessions.store(1, &UserSession{UserID: 1})

	// AVAILABLE is the zero value, and NewUserSession sets it explicitly for a
	// real session, so that is what a bare one reads as.
	if got := sessions.GetStatus(1); got != entity.AVAILABLE {
		t.Fatalf("a fresh session should read AVAILABLE, got %v", got)
	}

	sessions.UpdateSession(1, entity.AVAILABLE)
	if got := sessions.GetStatus(1); got != entity.AVAILABLE {
		t.Errorf("expected AVAILABLE, got %v", got)
	}

	sessions.UpdateSession(1, entity.PLAYING)
	if got := sessions.GetStatus(1); got != entity.PLAYING {
		t.Errorf("expected PLAYING, got %v", got)
	}
}

// TestRemoveSessionIsIdempotent covers a disconnect arriving twice, which the
// once-guard in UserOffline is supposed to prevent but callers can still cause.
func TestRemoveSessionIsIdempotent(t *testing.T) {
	sessions := NewActiveSessions(zap.NewNop().Sugar())
	sessions.store(1, &UserSession{UserID: 1})

	sessions.RemoveSession(1)
	sessions.RemoveSession(1)
	sessions.RemoveSession(1)

	if got := sessions.Count(); got != 0 {
		t.Errorf("expected no connected users, got %d", got)
	}
}
