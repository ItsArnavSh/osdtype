package api

import (
	"errors"
	"net/http"
	"time"

	"osdtyp/app/api/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The WebSocket upgrade and the notification inbox.
//
// The upgrade is the only route in the product that does not return JSON: it
// hands the connection to the session and the request never completes normally.
// That is why it logs nothing after a successful upgrade, and why the handlers
// below it are careful not to write a body.

// joinsession upgrades the request to the user's WebSocket.
//
// A missing session is a precondition failure rather than a 500. Every real
// time feature needs one, so a client that reaches any of them without a socket
// should be told to open one, not shown a server error it cannot act on.
func (s *Server) joinsession(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		if err := s.core.Sessions.NewUserSession(ctx, userID); err != nil {
			s.logger.Warnw("could not upgrade to a websocket", "user_id", userID, "error", err)
			fail(ctx, http.StatusBadRequest, "could not open a connection: "+err.Error())
			return
		}
		// From here the connection belongs to the session. Writing to ctx
		// would corrupt the frames, so the handler simply returns.
		s.logger.Infow("websocket opened", "user_id", userID, "connected", s.core.Sessions.Count())
	})
}

// getNotifications returns the caller's unread notifications and marks them
// read.
//
// It is the reconnect path. A frame pushed over the socket is gone the moment
// the tab closes, so anything the client missed is waiting here. Marking read
// on read rather than on a separate call means the common case is one request,
// and a client that wants to keep them can ask with unread_only=false.
func (s *Server) getNotifications(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		unreadOnly := ctx.Query("unread") != "false"

		notifications, err := s.services.Notifications(ctx.Request.Context(), userID, unreadOnly, queryInt(ctx, "limit", 50))
		if err != nil {
			s.logger.Errorw("could not read notifications", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not load your notifications")
			return
		}

		if unreadOnly && len(notifications) > 0 {
			if err := s.services.MarkNotificationsRead(ctx.Request.Context(), userID); err != nil {
				// The notifications were read successfully; failing the
				// request over the bookkeeping would make the client show them
				// again on the next poll, which is worse than a duplicate.
				s.logger.Warnw("could not mark notifications read", "user_id", userID, "error", err)
			}
		}

		ctx.JSON(http.StatusOK, gin.H{
			"notifications": notifications,
			"unread":        unreadCount(notifications),
		})
	})
}

// getNotificationCount returns just the badge number, which a client polls on
// a slow timer and which should stay cheap.
func (s *Server) getNotificationCount(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		notifications, err := s.services.Notifications(ctx.Request.Context(), userID, true, notificationBadgeLimit)
		if err != nil {
			// The badge is decoration. A database blip should not turn into an
			// error toast on every poll.
			s.logger.Warnw("could not count notifications", "user_id", userID, "error", err)
			ctx.JSON(http.StatusOK, gin.H{"unread": 0})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"unread": unreadCount(notifications)})
	})
}

// markNotificationsRead clears the caller's unread badge without fetching
// anything.
//
// The badge is read on a timer by a client that is not looking at the inbox
// page, so it has to be clearable on its own. Without this route, the badge
// only cleared when somebody happened to open the inbox.
func (s *Server) markNotificationsRead(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		if err := s.services.MarkNotificationsRead(ctx.Request.Context(), userID); err != nil {
			s.logger.Warnw("could not clear the notification badge", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not clear your notifications")
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"unread": 0})
	})
}

// notificationBadgeLimit bounds the count query. Anything past this is shown
// as "99+", so there is no reason to fetch more.
const notificationBadgeLimit = 100

// unreadCount is the badge number, capped.
func unreadCount[T any](items []T) int {
	if len(items) > 99 {
		return 99
	}
	return len(items)
}

// sessionStatus reports the caller's live connection state.
//
// A client uses it on load to decide whether to open a socket, and a person
// debugging a connection uses it to see what the server thinks.
func (s *Server) sessionStatus(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		connected := s.core.Sessions.Online(userID)
		ctx.JSON(http.StatusOK, gin.H{
			"connected":    connected,
			"status":       s.core.Sessions.GetStatus(userID).String(),
			"users_online": s.core.Sessions.Count(),
			"server_time":  time.Now().UTC(),
		})
	})
}

// isOnline reports whether a named player has a live connection. It backs the
// presence dot next to a friend in an invite picker.
func (s *Server) isOnline(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, _ uint32) {
		name := ctx.Query("user")
		if name == "" {
			fail(ctx, http.StatusBadRequest, "say who with ?user=NAME")
			return
		}
		other, err := s.services.GetUserFromName(ctx.Request.Context(), name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fail(ctx, http.StatusNotFound, "no player by that name")
				return
			}
			fail(ctx, http.StatusInternalServerError, "could not look that player up")
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"user":   other.Username,
			"online": s.core.Sessions.Online(other.ID),
			"status": s.core.Sessions.GetStatus(other.ID).String(),
		})
	})
}

// optionalUserID returns the caller's id, or zero for an anonymous request.
//
// The leaderboard is readable without signing in, so it uses this rather than
// withUser. Zero is not a real id: ids come from a generator seeded with the
// current time, so a zero id cannot collide with a real account.
func optionalUserID(ctx *gin.Context) uint32 {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return 0
	}
	return userID
}
