package services

import (
	"context"
	"time"

	"osdtyp/app/entity"
)

// The notification inbox behind the live connection.
//
// Opening the socket itself is not here: the upgrade needs the live
// *gin.Context, and routing that through a service whose only other argument
// is a context.Context meant the handler had two ways to start a session and
// the one in the service could never work.

// Notifications returns a caller's notifications, newest first.
func (s *ServiceLayer) Notifications(ctx context.Context, userID uint32, unreadOnly bool, limit int) ([]entity.Notification, error) {
	items, err := s.db.Notifications(ctx, userID, unreadOnly, limit)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		// Read notifications are older than the badge cares about, and a
		// player who has been away for a month should not have their inbox
		// trimmed on this read: the prune is cheap and runs on the same path.
		if err := s.db.PruneNotifications(ctx, userID, inboxRetention); err != nil {
			s.logger.Debugw("could not prune old notifications", "user_id", userID, "error", err)
		}
	}
	return items, nil
}

// MarkNotificationsRead clears a caller's unread badge.
func (s *ServiceLayer) MarkNotificationsRead(ctx context.Context, userID uint32) error {
	return s.db.MarkNotificationsRead(ctx, userID)
}

// inboxRetention is how long a read notification is kept before it is pruned.
//
// Thirty days is long enough that an invitation you ignored on Tuesday is
// still traceable, and short enough that the table stays small. Unread
// notifications are never pruned: they are the ones a person has not dealt
// with yet.
const inboxRetention = 30 * 24 * time.Hour

// Deliver sends a notification to a player, storing it if they are not
// connected.
//
// It is how the rest of the service layer pushes anything out of band: a
// friend request, a rank change, a contest about to start.
func (s *ServiceLayer) Deliver(ctx context.Context, userID uint32, kind entity.Kind, payload any) error {
	return s.core.Sessions.Deliver(ctx, s.db, userID, kind, payload)
}
