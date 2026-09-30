//go:build unit

package controlledlobby

import (
	"context"
	"sync"

	"osdtyp/app/entity"
)

// recordingNotifications is a NotificationStore that keeps what it was given, so
// a test can assert on what a player would find in their inbox.
type recordingNotifications struct {
	mu    sync.Mutex
	saved []entity.Notification
}

func (r *recordingNotifications) SaveNotification(_ context.Context, n entity.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, n)
	return nil
}

// all is a copy of what was saved, so a test can assert on it after the
// delivery goroutine it raced with has finished.
func (r *recordingNotifications) all() []entity.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]entity.Notification, len(r.saved))
	copy(out, r.saved)
	return out
}
