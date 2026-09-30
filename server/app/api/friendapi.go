package api

import (
	"net/http"
	"strconv"
	"strings"

	"osdtyp/app/api/auth"
	"osdtyp/app/entity"

	"github.com/gin-gonic/gin"
)

// The social surface: finding people, listing friends, and inviting them to a
// game.
//
// The invite route used to be a GET that took a numeric invitee id, so a
// person could not invite a friend without first looking their id up, and
// there was no route at all for listing friends or searching. Both halves of
// the feature were unreachable.

// follow adds a person to the caller's friend list, by name.
func (s *Server) follow(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userid uint32) {
		name := strings.TrimSpace(ctx.Query("user"))
		if name == "" {
			fail(ctx, http.StatusBadRequest, "say who to follow with ?user=NAME")
			return
		}

		other, err := s.services.GetUserFromName(ctx.Request.Context(), name)
		if err != nil {
			// A name that does not exist and a name that does but is not
			// visible are the same answer to a stranger.
			fail(ctx, http.StatusNotFound, "no player by that name")
			return
		}

		if err := s.services.FollowUser(ctx.Request.Context(), userid, other.ID); err != nil {
			s.logger.Warnw("could not follow", "user_id", userid, "other", other.ID, "error", err)
			fail(ctx, http.StatusConflict, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"following": other.Username})
	})
}

// unfollow removes a person from the caller's friend list, by name.
func (s *Server) unfollow(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userid uint32) {
		name := strings.TrimSpace(ctx.Query("user"))
		if name == "" {
			fail(ctx, http.StatusBadRequest, "say who to unfollow with ?user=NAME")
			return
		}

		other, err := s.services.GetUserFromName(ctx.Request.Context(), name)
		if err != nil {
			fail(ctx, http.StatusNotFound, "no player by that name")
			return
		}

		if err := s.services.UnfollowUser(ctx.Request.Context(), userid, other.ID); err != nil {
			s.logger.Warnw("could not unfollow", "user_id", userid, "other", other.ID, "error", err)
			fail(ctx, http.StatusConflict, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"unfollowed": other.Username})
	})
}

// friends returns the caller's friends, with who is online.
func (s *Server) friends(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userid uint32) {
		cards, err := s.services.Friends(ctx.Request.Context(), userid)
		if err != nil {
			s.logger.Errorw("could not list friends", "user_id", userid, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not load your friends")
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"friends": cards})
	})
}

// searchPlayers finds players by name, for the picker that feeds an invite.
//
// It excludes the caller. Returning yourself in a list you then have to filter
// out in the interface is a small thing that reads as a bug, and the exclusion
// is a rule rather than a presentation concern.
func (s *Server) searchPlayers(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userid uint32) {
		results, err := s.services.SearchUsers(ctx.Request.Context(), ctx.Query("q"))
		if err != nil {
			s.logger.Errorw("could not search players", "error", err)
			fail(ctx, http.StatusInternalServerError, "search failed")
			return
		}

		out := make([]entity.UserSearchResult, 0, len(results))
		for _, r := range results {
			if r.UserID == userid {
				continue
			}
			out = append(out, r)
		}
		ctx.JSON(http.StatusOK, gin.H{"players": out})
	})
}

// invitePlayerToLobby invites someone to a lobby.
//
// The invitee is named rather than numbered. Every other invite in the product
// works by name, and a numeric id here meant the interface had to resolve a
// friend to an id first, which is a step that exists for no reason the person
// using it can see.
//
// The inviter must be in the lobby. Without that check any player could invite
// a friend to someone else's game, which is both a privacy problem and a way
// to fill a stranger's lobby.
func (s *Server) invitePlayerToLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, invitorID uint32) {
		roomID, err := roomIDParam(ctx)
		if err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return
		}

		name := strings.TrimSpace(ctx.Query("user"))
		if name == "" {
			fail(ctx, http.StatusBadRequest, "say who to invite with ?user=NAME")
			return
		}

		room, err := s.services.LobbyView(ctx.Request.Context(), invitorID, roomID)
		if err != nil {
			fail(ctx, http.StatusNotFound, "no such lobby")
			return
		}
		if !room.Joined {
			fail(ctx, http.StatusForbidden, "join the lobby before inviting people to it")
			return
		}

		invitee, err := s.services.GetUserFromName(ctx.Request.Context(), name)
		if err != nil {
			fail(ctx, http.StatusNotFound, "no player by that name")
			return
		}
		if invitee.ID == invitorID {
			fail(ctx, http.StatusBadRequest, "you are already in it")
			return
		}

		// A password-protected lobby is not invited into by name alone: the
		// invitee has to supply the password like anyone else. Sending the
		// code and the room in the notification is what makes the invite a
		// link rather than an instruction.
		if err := s.services.InviteToLobby(ctx.Request.Context(), invitorID, invitee.ID, roomID); err != nil {
			s.logger.Warnw("could not deliver an invitation", "inviter", invitorID, "invitee", invitee.ID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not send the invitation")
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"invited": invitee.Username,
			// Whether the invite landed as a push or is waiting in the
			// invitee's inbox, so the interface can say "sent" honestly
			// rather than implying they have already seen it.
			"delivered_now": s.core.Sessions.Online(invitee.ID),
		})
	})
}

// withUser runs a handler that needs an authenticated caller.
//
// The identity check, the 401 and the logging were copy-pasted into every
// handler. The two endpoints that skipped it were the ones that read the caller
// for moderation checks, so an anonymous request could promote itself to mod.
func (s *Server) withUser(g *gin.Context, fn func(*gin.Context, uint32)) {
	userID, err := auth.GetUserID(g)
	if err != nil {
		s.logger.Warnw("unauthenticated request", "path", g.FullPath(), "error", err)
		fail(g, http.StatusUnauthorized, "sign in first")
		return
	}
	fn(g, userID)
}

// fail writes a JSON error body.
//
// The shape is always {"error": "..."} so a client has one thing to read. The
// previous handlers used several spellings, and one put a []byte in a gin.H,
// which base64-encoded the message.
func fail(g *gin.Context, code int, message string) {
	g.AbortWithStatusJSON(code, gin.H{"error": message})
}

// roomIDParam reads a room id from the path, then the query, then the body.
//
// A single helper because the id arrives as ":id" for a path route and as
// "?room_id=" for a query one, and every handler was re-deriving which.
func roomIDParam(g *gin.Context) (uint32, error) {
	raw := g.Param("id")
	if raw == "" {
		raw = g.Query("room_id")
	}
	if raw == "" {
		raw = g.Query("roomid")
	}
	if raw == "" {
		return 0, errMissingRoomID
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, errBadRoomID
	}
	return uint32(v), nil
}

// ErrMissingRoomID and friends are the messages the room id helper produces.
// They are exported so the tests can assert on them without string matching a
// format.
var (
	errMissingRoomID = &apiError{"which lobby? pass its id"}
	errBadRoomID     = &apiError{"that lobby id is not a number"}
)

// apiError is an error with a message meant for a person rather than a log.
type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }
