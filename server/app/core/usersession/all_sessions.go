package usersession

import (
	"context"
	"errors"
	"sync"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ActiveSessions is the registry of every user currently connected over a
// WebSocket.
//
// Sessions are created and torn down on HTTP handler goroutines while the
// matchmaker, the scheduler and the lobby reads them from their own workers,
// so the map is guarded. It used to be a plain exported field, which meant a
// connect or a disconnect racing a lookup could take the whole process down
// with "concurrent map read and map write".
type ActiveSessions struct {
	mu     sync.RWMutex
	users  map[uint32]*UserSession
	logger *zap.SugaredLogger
}

// NewActiveSessions builds an empty registry.
func NewActiveSessions(logger *zap.SugaredLogger) *ActiveSessions {
	return &ActiveSessions{
		users:  make(map[uint32]*UserSession),
		logger: logger,
	}
}

// Count reports how many users are currently connected.
func (a *ActiveSessions) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.users)
}

// store registers a session without going through a WebSocket upgrade. Used by
// NewUserSession and by tests, which cannot perform a real handshake.
func (a *ActiveSessions) store(id uint32, session *UserSession) {
	a.mu.Lock()
	a.users[id] = session
	count := len(a.users)
	a.mu.Unlock()

	a.logger.Infow("session opened", "user_id", id, "connected", count)
}

// GetSession returns a user's live session, or nil if they are not connected.
func (a *ActiveSessions) GetSession(id uint32) *UserSession {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.users[id]
}

// GetStatus reports whether a user is available, playing or offline.
//
// A user whose socket has died but whose teardown has not run yet still
// reports AVAILABLE, so the closed channel is checked too. Reporting a dead
// session as available is what let the invite path push into a closed socket.
func (a *ActiveSessions) GetStatus(id uint32) entity.UserStatus {
	user := a.GetSession(id)
	switch {
	case user == nil:
		return entity.OFFLINE
	case user.IsClosed():
		return entity.OFFLINE
	default:
		return user.Status()
	}
}

// Online reports whether a user has a usable connection right now. This is
// what a roster or a friend list needs: a member who is mid-game is online
// but not available to be invited.
func (a *ActiveSessions) Online(id uint32) bool {
	user := a.GetSession(id)
	return user != nil && !user.IsClosed()
}

// Subscribe takes a lease on a user's socket. It returns nil if the user is
// not connected, so a caller joining a lobby without a live session gets an
// ordinary error rather than a nil dereference.
func (a *ActiveSessions) Subscribe(id uint32) *Subscriber {
	user := a.GetSession(id)
	if user == nil || user.IsClosed() {
		return nil
	}
	return user.Subscribe()
}

// Notify pushes an already-marshaled frame to a user.
//
// It reports whether the frame was queued, so a caller that cares can record
// the notification for delivery later.
func (a *ActiveSessions) Notify(id uint32, raw []byte) error {
	user := a.GetSession(id)
	if user == nil {
		return ErrSessionClosed
	}
	return user.Notify(raw)
}

// NewUserSession upgrades the request to a WebSocket and registers it.
func (a *ActiveSessions) NewUserSession(g *gin.Context, id uint32) error {
	ws, err := utils.UpgradeToWebSocket(g)
	if err != nil {
		return err
	}
	session := NewUserSession(ws, a.RemoveSession, id, a.logger)
	a.store(id, session)
	return nil
}

// RegisterStub makes a user appear connected without a transport behind the
// session.
//
// It exists for the paths that are about presence rather than about the wire:
// resolving a lobby's roster to sockets, admitting a player to the matchmaker,
// an end-to-end test that has no business performing a WebSocket handshake. A
// stub session accepts every frame it is sent and never receives one, so
// anything that actually needs a client's keystrokes will simply time out —
// which is honest, rather than a stub that silently reports a round nobody
// played.
func (a *ActiveSessions) RegisterStub(id uint32) *UserSession {
	session := newUserSession(newNopSocket(), a.RemoveSession, id, a.logger)
	a.store(id, session)
	return session
}

// UpdateSession records a user's availability.
//
// A user can disconnect between the caller deciding they are online and this
// write landing, so a missing session is expected rather than a bug.
func (a *ActiveSessions) UpdateSession(id uint32, status entity.UserStatus) {
	session := a.GetSession(id)
	if session == nil {
		return
	}
	session.SetStatus(status)
}

// RemoveSession drops a user's session. It runs as the session's own
// disconnect hook.
func (a *ActiveSessions) RemoveSession(userID uint32) {
	a.mu.Lock()
	delete(a.users, userID)
	count := len(a.users)
	a.mu.Unlock()

	a.logger.Infow("session closed", "user_id", userID, "connected", count)
}

// ConnectedUsers returns the ids of everyone currently online.
func (a *ActiveSessions) ConnectedUsers() []uint32 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	ids := make([]uint32, 0, len(a.users))
	for id, s := range a.users {
		if !s.IsClosed() {
			ids = append(ids, id)
		}
	}
	return ids
}

// Drain tears down every session and empties the registry.
//
// It is for a reload. A session left registered after a shutdown would make
// every Online check in the next process lifetime answer against a dead
// socket, and the invite path would push into it until the write failed.
func (a *ActiveSessions) Drain() {
	a.mu.Lock()
	sessions := make([]*UserSession, 0, len(a.users))
	for id, s := range a.users {
		sessions = append(sessions, s)
		delete(a.users, id)
	}
	a.mu.Unlock()

	for _, s := range sessions {
		s.UserOffline()
	}
	a.logger.Infow("session registry drained", "count", len(sessions))
}

// Deliver persists a notification and pushes it to a live socket.
//
// Writing first and pushing second means a player who was online but whose
// socket dropped mid-delivery still finds it waiting on their next fetch. The
// push is best effort by design.
func (a *ActiveSessions) Deliver(ctx context.Context, store NotificationStore, userID uint32, kind entity.Kind, payload any) error {
	raw := entity.Encode(payload)

	if store != nil {
		if err := store.SaveNotification(ctx, entity.Notification{
			ID:        nextNotificationID(),
			UserID:    userID,
			Kind:      kind,
			Payload:   raw,
			CreatedAt: time.Now(),
		}); err != nil {
			return err
		}
	}

	if !a.Online(userID) {
		// Stored, will be picked up when they come back.
		return nil
	}

	frame := entity.Encode(entity.NotificationFrame{
		Type: entity.FrameNotification,
		Kind: kind,
		Body: raw,
		At:   time.Now().UnixMilli(),
	})
	if err := a.Notify(userID, frame); err != nil && !errors.Is(err, ErrSessionBusy) {
		return err
	}
	return nil
}

// NotificationStore is the slice of the database ActiveSessions needs to make
// notifications durable. Declaring it here rather than taking the concrete
// database keeps this package free of a dependency on the persistence layer,
// which would otherwise be an import cycle through core.
type NotificationStore interface {
	SaveNotification(ctx context.Context, n entity.Notification) error
}

// notifIDs mints notification ids. Notifications have to be addressable so
// they can be marked read, which rules out an autoincrement column here: the
// id is assigned in the service layer before the row is written.
var notifIDs = utils.NewGenerator()

func nextNotificationID() uint32 { return notifIDs.GenerateID() }
