package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"osdtyp/app/entity"
	"osdtyp/app/services"

	"github.com/gin-gonic/gin"
)

// Ranked play, and the solo path that feeds it.
//
// Everything here was either unreachable or unranked before. The queue existed
// and worked, but nothing told a client what happened next, so a player who
// queued watched a spinner forever while the matchmaker ran a round whose seed
// and leaderboard went to sockets that had no protocol to receive them.

// queueRanked puts the caller in a ranked queue.
//
// The mode is a query parameter rather than part of the path so a client can
// switch modes without building a different URL, and so the same handler serves
// the "quick play" button and an explicit choice. It also still accepts the
// old ?duration=30|90|300 spelling, because a link from an old client or a
// bookmark should keep working.
func (s *Server) queueRanked(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		mode := rankedModeOf(ctx)

		status, err := s.services.JoinRanked(ctx.Request.Context(), userID, mode)
		if err != nil {
			s.respondQueueError(ctx, userID, err)
			return
		}
		ctx.JSON(http.StatusOK, status)
	})
}

// unqueueRanked takes the caller out of a ranked queue.
//
// A client has to be able to change its mind, or the queue becomes a trap: a
// player who picked the wrong mode has to reload the page to escape it, and
// reload does not dequeue.
func (s *Server) unqueueRanked(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		mode := rankedModeOf(ctx)

		if err := s.services.LeaveRanked(ctx.Request.Context(), userID, mode); err != nil {
			s.respondQueueError(ctx, userID, err)
			return
		}
		ctx.JSON(http.StatusOK, entity.QueueStatus{
			Type:      entity.FrameQueueStatus,
			LobbyType: mode,
			Queued:    false,
		})
	})
}

// rankedModeOf reads a mode name off a request, accepting both spellings and
// defaulting to sprint.
func rankedModeOf(ctx *gin.Context) string {
	if mode := strings.TrimSpace(ctx.Query("mode")); mode != "" {
		return mode
	}
	if mode := modeFromSeconds(ctx.Query("duration")); mode != "" {
		return mode
	}
	return entity.SPRINT.String()
}

// respondQueueError maps a queue error to a status code and a phrase a person
// can read.
func (s *Server) respondQueueError(ctx *gin.Context, userID uint32, err error) {
	switch {
	case errors.Is(err, services.ErrNoLiveSession):
		// 428 Precondition Required is the honest code: the request is well
		// formed, it just needs something the caller has not done yet.
		fail(ctx, http.StatusPreconditionRequired, err.Error())
	case errors.Is(err, services.ErrNotQueued):
		fail(ctx, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrUnknownMode):
		fail(ctx, http.StatusBadRequest, err.Error())
	default:
		s.logger.Errorw("queue request failed", "user_id", userID, "error", err)
		fail(ctx, http.StatusInternalServerError, "could not change the queue")
	}
}

// queueStatus reports where the caller stands, so a page load can restore the
// right button instead of always offering "join".
//
// Without it a client that reloads while queued shows the join button again,
// and clicking it replaced the queue entry rather than joining to the game
// that was already about to start.
func (s *Server) queueStatus(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		modes := s.core.Matchmaker.QueuedModes(userID)
		rank, err := s.services.GetRank(ctx.Request.Context(), userID)
		if err != nil {
			s.logger.Warnw("could not read a rating", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not read your rating")
			return
		}

		queued := make([]string, 0, len(modes))
		for _, m := range modes {
			queued = append(queued, m.String())
		}
		ctx.JSON(http.StatusOK, gin.H{
			"queued_modes": queued,
			"rating":       rank,
			"tier":         s.services.TierName(rank),
			"connected":    s.core.Sessions.Online(userID),
		})
	})
}

// leaderboard returns a page of rankings.
//
// It is readable without a session, so an anonymous caller gets a board with
// nobody's IsSelf marked. That is deliberate: a ladder is the one page a
// stranger should be able to see, and requiring a login to look at it just
// teaches people not to care.
func (s *Server) leaderboard(g *gin.Context) {
	opts := entity.LeaderboardOptions{
		Scope:    g.DefaultQuery("scope", "global"),
		Mode:     g.Query("mode"),
		Page:     queryInt(g, "page", 0),
		PageSize: queryInt(g, "page_size", 50),
		SoloOnly: g.Query("solo") == "true",
	}
	if raw := g.Query("room_id"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			fail(g, http.StatusBadRequest, "room_id is not a number")
			return
		}
		opts.RoomID = uint32(v)
	}

	page, err := s.services.Leaderboard(g.Request.Context(), optionalUserID(g), opts)
	if err != nil {
		if errors.Is(err, entity.ErrUnknownMode) {
			fail(g, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Warnw("leaderboard request failed", "error", err)
		fail(g, http.StatusInternalServerError, "could not load the leaderboard")
		return
	}
	g.JSON(http.StatusOK, page)
}

// submitRun records a solo run and returns what it did to the rating.
//
// This is what makes a practice run worth anything. Before it, a solo game
// ended on the client and was thrown away: nothing was stored, nothing was
// rated, and the rating could only ever move as a side effect of a match that
// most players never got into.
func (s *Server) submitRun(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		var req entity.RunSubmission
		if err := ctx.ShouldBindJSON(&req); err != nil {
			fail(ctx, http.StatusBadRequest, "could not read the run: "+err.Error())
			return
		}
		if !req.Valid() {
			fail(ctx, http.StatusBadRequest, "that run does not add up")
			return
		}

		result, err := s.services.SubmitRun(ctx.Request.Context(), userID, req)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrNoLiveSession):
				fail(ctx, http.StatusPreconditionRequired, err.Error())
			case errors.Is(err, services.ErrUnknownMode), errors.Is(err, entity.ErrUnknownMode):
				fail(ctx, http.StatusBadRequest, err.Error())
			default:
				s.logger.Errorw("could not record a run", "user_id", userID, "error", err)
				fail(ctx, http.StatusInternalServerError, "could not record the run")
			}
			return
		}
		ctx.JSON(http.StatusOK, result)
	})
}

// getRuns returns a player's recent runs.
func (s *Server) getRuns(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		page, err := s.services.Runs(ctx.Request.Context(), userID,
			queryInt(ctx, "page", 0), queryInt(ctx, "page_size", 25))
		if err != nil {
			s.logger.Warnw("could not load runs", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not load your runs")
			return
		}
		ctx.JSON(http.StatusOK, page)
	})
}

// getRankHistory returns a player's rating over time, for the profile graph.
func (s *Server) getRankHistory(g *gin.Context) {
	s.withUser(g, func(ctx *gin.Context, userID uint32) {
		history, err := s.services.RatingHistory(ctx.Request.Context(), userID, queryInt(ctx, "limit", 60))
		if err != nil {
			s.logger.Warnw("could not load a rating history", "user_id", userID, "error", err)
			fail(ctx, http.StatusInternalServerError, "could not load your history")
			return
		}
		ctx.JSON(http.StatusOK, history)
	})
}

// getOptions returns the game's vocabulary: modes, languages and rating tiers.
//
// It exists so the interface never hardcodes a list the server decides. The
// tier bands in particular are a rule, not a preference: two clients that
// disagreed about where Gold started would show different names for the same
// player, and neither would be obviously wrong.
func (s *Server) getOptions(g *gin.Context) {
	g.JSON(http.StatusOK, gin.H{
		"modes":     s.services.Modes(),
		"languages": s.services.Languages(),
		"tiers":     s.services.Tiers(),
	})
}

// modeFromSeconds maps the original duration-in-seconds parameter onto a mode
// name, so a link written before the modes were named still queues somebody.
func modeFromSeconds(raw string) string {
	switch strings.TrimSpace(raw) {
	case "30":
		return entity.SPRINT.String()
	case "90":
		return entity.STANDARD.String()
	case "300":
		return entity.MARATHON.String()
	}
	return ""
}

// queryInt reads a numeric query parameter, falling back when it is absent or
// not a number.
//
// A client that sends page=abc gets the first page rather than a 400: it is
// almost always a hand-typed link, and the response is still useful.
func queryInt(ctx *gin.Context, name string, fallback int) int {
	raw := ctx.Query(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
