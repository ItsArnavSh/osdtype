package api

import (
	"errors"
	"net/http"
	"strings"

	controlledlobby "osdtyp/app/core/controlled-lobby"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

// Private lobbies: create, look up by share code, invite, start.
//
// The shape is the Among Us one, because it is the one that works when people
// are talking to each other. The six character code is the primary key and is
// short enough to read aloud without ambiguity, and the password is a separate
// optional lock for a lobby you do not want anyone who has the code walking
// into. Both are needed: a code alone is guessable in a small deployment, and
// a password alone is not something anyone can say out loud.

// createLobby opens a private game lobby and returns its share code.
func (s *Server) createLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		var req services.LobbyRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			fail(ctx, http.StatusBadRequest, "could not read the lobby: "+err.Error())
			return
		}

		room, err := s.services.CreateLobby(ctx.Request.Context(), userID, req)
		if err != nil {
			// errors.Is, not ==: the service wraps this to name the mode it
			// could not parse, and == against a wrapped error is false, which
			// would answer 428 to a player who was online the whole time.
			if errors.Is(err, services.ErrNoLiveSession) {
				fail(ctx, http.StatusPreconditionRequired, err.Error())
				return
			}
			s.logger.Errorw("could not create a lobby", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not open the lobby")
			return
		}

		ctx.JSON(http.StatusCreated, lobbyResponse{
			Room:        room,
			Code:        room.Code,
			HasPassword: room.PasswordSet,
			// The join path as a code, so the interface can build a link
			// without having to know how the route is shaped.
			JoinPath: joinPath(room.Code),
		})
	})
}

// joinLobby puts the caller into a lobby, by share code.
//
// It is a POST rather than a GET because it changes state and because it takes
// a password, which does not belong in a URL: URLs end up in logs, in browser
// history, and in a Referer header.
func (s *Server) joinLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		var req struct {
			Code     string `json:"code"`
			Password string `json:"password"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			// Fall back to a query, so a bare link with ?code= still works
			// for a lobby with no password.
			req.Code = ctx.Query("code")
			req.Password = ctx.Query("password")
		}

		if strings.TrimSpace(req.Code) == "" {
			fail(ctx, http.StatusBadRequest, "a lobby code is required")
			return
		}

		room, err := s.services.JoinLobby(ctx.Request.Context(), userID, req.Code, req.Password)
		if err != nil {
			// errors.Is throughout: the store wraps the not-found and the
			// capacity refusals, so == never matches and every one of these
			// branches fell through to a 500.
			switch {
			case errors.Is(err, services.ErrNoLiveSession):
				fail(ctx, http.StatusPreconditionRequired, err.Error())
			case errors.Is(err, services.ErrRoomNotFound):
				fail(ctx, http.StatusNotFound, err.Error())
			case errors.Is(err, services.ErrLobbyPassword):
				// 403 rather than 401: the caller is authenticated, they just
				// are not allowed in yet, and a 401 would send them off to log
				// in again.
				fail(ctx, http.StatusForbidden, err.Error())
			case isLobbyFull(err):
				// 409: the request was fine, the room simply has no space, and
				// the client can show that and offer to be told when a slot frees.
				fail(ctx, http.StatusConflict, err.Error())
			default:
				s.logger.Errorw("could not join a lobby", "user_id", userID, "code", req.Code, "error", err)
				fail(ctx, http.StatusInternalServerError, "could not join the lobby")
			}
			return
		}

		view, err := s.services.LobbyView(ctx.Request.Context(), userID, room.ID)
		if err != nil {
			fail(ctx, http.StatusInternalServerError, "joined, but could not load the lobby")
			return
		}
		ctx.JSON(http.StatusOK, view)
	})
}

// getLobby returns a lobby's roster for one caller.
//
// It is what makes a share link a page rather than a redirect: the invitee sees
// who is already waiting before deciding to join.
func (s *Server) getLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		roomID, err := roomIDParam(ctx)
		if err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return
		}
		s.respondLobby(ctx, userID, roomID, ctx.Query("password"))
	})
}

// getLobbyByCode returns a lobby's roster by share code, which is the path an
// invitation link takes.
func (s *Server) getLobbyByCode(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		code := strings.TrimSpace(ctx.Param("code"))
		if code == "" {
			// A bare ?code= still works, so a link that was hand-written
			// against the old shape lands somewhere.
			code = strings.TrimSpace(ctx.Query("code"))
		}
		if code == "" {
			fail(ctx, http.StatusBadRequest, "a lobby code is required")
			return
		}
		view, err := s.services.LobbyByCodeView(ctx.Request.Context(), userID, code, ctx.Query("password"))
		if err != nil {
			switch {
			case errors.Is(err, services.ErrRoomNotFound):
				fail(ctx, http.StatusNotFound, err.Error())
			// 403 rather than 401: the caller is authenticated, they just are
			// not allowed in yet, and a 401 would send them off to log in again.
			case errors.Is(err, services.ErrLobbyPassword):
				fail(ctx, http.StatusForbidden, err.Error())
			default:
				s.logger.Errorw("could not load a lobby by code", "code", code, "error", err)
				fail(ctx, http.StatusInternalServerError, "could not load the lobby")
			}
			return
		}
		ctx.JSON(http.StatusOK, view)
	})
}

// respondLobby writes a lobby's roster.
//
// The password argument is not read: LobbyView takes the caller's identity, not
// a password, so the room's privacy is enforced by who is allowed to ask rather
// than by a field on the request. It is kept on the signature because both
// callers have one to hand, and deleting it is a call-site change for no
// behavioral gain. Silently ignoring an argument is worse than removing it, so
// it is named for what it is.
func (s *Server) respondLobby(ctx *gin.Context, userID, roomID uint32, _ string) {
	view, err := s.services.LobbyView(ctx.Request.Context(), userID, roomID)
	if err != nil {
		if errors.Is(err, services.ErrLobbyPassword) {
			fail(ctx, http.StatusForbidden, err.Error())
			return
		}
		fail(ctx, http.StatusNotFound, "no such lobby")
		return
	}
	ctx.JSON(http.StatusOK, view)
}

// leaveLobby removes the caller from a lobby.
func (s *Server) leaveLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		roomID, err := roomIDParam(ctx)
		if err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.services.LeaveLobby(ctx.Request.Context(), userID, roomID); err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"left": true})
	})
}

// startLobby runs a lobby as a game.
//
// The response comes back as soon as the round is launched. The game itself
// reaches the caller over the WebSocket, as a seed frame and a countdown, so
// holding the request open for the length of a round would just time out.
func (s *Server) startLobby(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		roomID, err := roomIDParam(ctx)
		if err != nil {
			fail(ctx, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.services.StartLobby(ctx.Request.Context(), userID, roomID); err != nil {
			switch {
			case errors.Is(err, services.ErrLobbyClosed):
				fail(ctx, http.StatusConflict, err.Error())
			case errors.Is(err, services.ErrNoLiveSession):
				fail(ctx, http.StatusPreconditionRequired, err.Error())
			default:
				s.logger.Warnw("could not start a lobby", "user_id", userID, "room_id", roomID, "error", err)
				fail(ctx, http.StatusForbidden, err.Error())
			}
			return
		}
		ctx.JSON(http.StatusAccepted, gin.H{"starting": true, "room_id": roomID})
	})
}

// lobbyResponse is what a create returns, shaped for a client that wants to
// show the code and the link without reassembling them.
type lobbyResponse struct {
	Room entity.Room `json:"room"`
	// Code is repeated outside the room so a client that renders
	// "share this code" does not have to know which field of the room holds it.
	Code string `json:"code"`
	// HasPassword says whether a joiner will be asked for one.
	HasPassword bool `json:"has_password"`
	// JoinPath is the route an invitation link points at.
	JoinPath string `json:"join_path"`
}

// joinPath is the client route for a share code.
func joinPath(code string) string { return "/lobby/" + strings.ToUpper(code) }

// isLobbyFull reports whether an error is the lobby's own "no space left".
//
// It is a predicate rather than an inline check so the handler can map a domain
// error to a status code without importing the lobby package for one sentinel.
//
// The two spellings of the sentinel are the same value: the store's is an alias
// for the entity one, and the lobby's is an alias for that. Both are named
// anyway, because a handler that reads "check both" is correct against any
// future change that un-aliases them, whereas one that reads "they are the same
// object" breaks the day they are not.
func isLobbyFull(err error) bool {
	return errors.Is(err, controlledlobby.ErrRoomFull) || errors.Is(err, postgresql.ErrRoomFull)
}
