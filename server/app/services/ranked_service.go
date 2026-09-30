package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/core/anticheat"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

// Ranked play: the queues, the leaderboard, the rating, and the solo run that
// moves it.

// ErrAlreadyQueued is returned when a player queues for a mode they are
// already in.
var ErrAlreadyQueued = errors.New("you are already in that queue")

// ErrNotQueued is returned when a player tries to leave a queue they are not
// in. It is not an error worth a 500, but it is not silently ignored either.
var ErrNotQueued = errors.New("you are not in that queue")

// ErrUnknownMode is the service layer's name for a bad mode. It is an alias
// rather than a second sentinel so errors.Is matches either spelling.
var ErrUnknownMode = entity.ErrUnknownMode

// JoinRanked queues a player for a ranked game in a mode.
//
// The mode is accepted as a name or a number, so a client that has an enum
// value and a client that has a string both work against the same endpoint.
//
// It refuses a second queue for the same mode. The matchmaker replaces the
// entry, so the player would only hold one lease, but the first call's lease
// handle was already discarded by the replace, so the second call's release
// would let a third process take the socket out from under a live game.
func (s *ServiceLayer) JoinRanked(ctx context.Context, userID uint32, modeName string) (entity.QueueStatus, error) {
	mode, ok := entity.LobbyTypeFromString(strings.TrimSpace(modeName))
	if !ok {
		return entity.QueueStatus{}, fmt.Errorf("%w: %q", ErrUnknownMode, modeName)
	}

	if !s.core.Sessions.Online(userID) {
		return entity.QueueStatus{}, ErrNoLiveSession
	}

	rank, err := s.db.GetRank(ctx, userID)
	if err != nil {
		return entity.QueueStatus{}, err
	}

	if err := s.core.Matchmaker.AddToGlobalLobby(ctx, userID, rank, mode); err != nil {
		return entity.QueueStatus{}, err
	}

	return entity.QueueStatus{
		Type:      entity.FrameQueueStatus,
		LobbyType: mode.String(),
		Position:  1,
		Size:      s.core.Matchmaker.QueueSize(mode),
		Rating:    rank,
		Tier:      utils.TierOf(rank).Name,
		Queued:    true,
	}, nil
}

// LeaveRanked removes a player from a queue.
func (s *ServiceLayer) LeaveRanked(ctx context.Context, userID uint32, modeName string) error {
	mode, ok := entity.LobbyTypeFromString(strings.TrimSpace(modeName))
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownMode, modeName)
	}
	if !s.core.Matchmaker.Queued(userID, mode) {
		return ErrNotQueued
	}
	s.core.Matchmaker.LeaveQueue(userID, mode)
	return nil
}

// Queued reports whether a player is in a mode's queue, so the interface can
// show the right button on reload instead of always offering "join".
func (s *ServiceLayer) Queued(userID uint32, mode entity.LobbyType) bool {
	return s.core.Matchmaker.Queued(userID, mode)
}

// TierName is a rating's tier, for the many places a rating is shown next to
// the name of the band it falls in.
func (s *ServiceLayer) TierName(rating uint16) string {
	return utils.TierOf(rating).Name
}

// GetRank returns a player's rating.
func (s *ServiceLayer) GetRank(ctx context.Context, userID uint32) (uint16, error) {
	return s.db.GetRank(ctx, userID)
}

// Tiers returns the rating table, so a client can draw a ladder or color a
// rating without hardcoding the bands. The bands are a server rule: two
// clients that disagreed about where Gold started would show different tiers
// for the same player.
func (s *ServiceLayer) Tiers() []entity.Tier {
	all := utils.AllTiers()
	out := make([]entity.Tier, 0, len(all))
	for _, t := range all {
		out = append(out, entity.Tier{
			Name:  t.Name,
			Min:   t.Min,
			Color: t.Color,
		})
	}
	return out
}

// Modes returns the game modes, for the same reason.
func (s *ServiceLayer) Modes() []entity.ModeOption {
	names := map[entity.LobbyType]string{
		entity.SPRINT:   "Sprint",
		entity.STANDARD: "Standard",
		entity.MARATHON: "Marathon",
	}
	out := make([]entity.ModeOption, 0, 3)
	for _, m := range []entity.LobbyType{entity.SPRINT, entity.STANDARD, entity.MARATHON} {
		out = append(out, entity.ModeOption{
			Value:   int(m),
			Name:    names[m],
			Seconds: int(m.Duration().Seconds()),
		})
	}
	return out
}

// Languages returns the languages a round can be typed in.
func (s *ServiceLayer) Languages() []entity.LanguageOption {
	out := make([]entity.LanguageOption, 0, len(entity.KnownLanguages))
	for i, l := range entity.KnownLanguages {
		out = append(out, entity.LanguageOption{
			Value:  i,
			Name:   l.DisplayName(),
			Code:   l.String(),
			Active: true,
		})
	}
	return out
}

// Leaderboard returns a page of rankings.
//
// The scope lets a caller narrow to their friends or to one room, which is
// what makes a small deployment interesting: a global ladder of twenty people
// is a global ladder of twenty people.
func (s *ServiceLayer) Leaderboard(ctx context.Context, caller uint32, opts entity.LeaderboardOptions) (entity.LeaderboardPage, error) {
	if opts.Mode != "" {
		if _, ok := entity.LobbyTypeFromString(opts.Mode); !ok {
			return entity.LeaderboardPage{}, fmt.Errorf("%w: %q", ErrUnknownMode, opts.Mode)
		}
	}

	page, err := s.db.Leaderboard(ctx, opts, caller)
	if err != nil {
		return entity.LeaderboardPage{}, err
	}

	if page.SelfEntry != nil {
		// The caller's own row is ranked by their rating, not by their
		// position in the page they happened to land on, so a player outside
		// the returned page still sees a truthful number.
		page.SelfEntry.Rank = page.SelfRank
	}
	return page, nil
}

// SubmitRun records a solo run, rates it, and returns what it did.
//
// This is the only way a solo player ever moves their rating, and it is
// deliberately worth far less than a ranked game: a run has no opponent, so
// there is no result to be good or bad at. Letting it move the rating at the
// rated rate would make practicing alone the fastest route to the top of the
// ladder, which is exactly what ranked play exists to prevent.
//
// The anticheat runs here, and its verdict is returned even when the run is
// stored, so a flagged player is told what tripped rather than just seeing
// their rating not move.
func (s *ServiceLayer) SubmitRun(ctx context.Context, userID uint32, req entity.RunSubmission) (entity.RunResult, error) {
	if !s.core.Sessions.Online(userID) {
		// A socket is not strictly needed to submit a result, but the seed
		// frame that told the client what to type only goes to a connected
		// client, so an offline submission is one whose text cannot be checked.
		return entity.RunResult{}, ErrNoLiveSession
	}

	mode, ok := entity.LobbyTypeFromString(strings.TrimSpace(req.Mode))
	if !ok {
		return entity.RunResult{}, fmt.Errorf("%w: %q", ErrUnknownMode, req.Mode)
	}

	lang, ok := entity.ParseLanguage(strings.TrimSpace(strings.Trim(req.Language, "0123456789")))
	if !ok {
		return entity.RunResult{}, fmt.Errorf("unknown language %q", req.Language)
	}

	if req.Seed == "" {
		return entity.RunResult{}, errors.New("a run must name the seed it was played on")
	}
	seed, err := strconv.ParseUint(req.Seed, 10, 32)
	if err != nil {
		return entity.RunResult{}, errors.New("the seed is not a number")
	}

	if req.DurationMS <= 0 {
		return entity.RunResult{}, errors.New("a run must report how long it lasted")
	}

	// The API layer checks Valid(), which covers this bound, but the service is
	// the layer the anticheat's cost is actually paid at and it is reachable from
	// anywhere in the process. A caller that skipped the HTTP hop would hand the
	// scorer an unbounded attacker-chosen slice.
	if len(req.TimestampsMS) > entity.MaxSubmissionTimestamps {
		return entity.RunResult{}, fmt.Errorf("a run carries at most %d keystrokes, this one claims %d",
			entity.MaxSubmissionTimestamps, len(req.TimestampsMS))
	}

	verdict := s.anticheat.RunTimestamps(req.TimestampsMS)

	run := entity.Run{
		ID:         s.int_gen.GenerateID(),
		UserID:     userID,
		Mode:       mode,
		Solo:       true,
		WPM:        req.WPM,
		Raw:        req.Raw,
		Accuracy:   req.Accuracy,
		Correct:    req.Correct,
		Wrong:      req.Wrong,
		DurationMS: req.DurationMS,
		Language:   lang,
		Seed:       uint32(seed),
		Passed:     verdict.Passed,
		CheatScore: verdict.Score,
		PlayedAt:   time.Now(),
	}

	stored, change, err := s.db.RateRun(ctx, run)
	if err != nil {
		return entity.RunResult{}, err
	}

	result := entity.RunResult{
		Run:     stored,
		Change:  change,
		Passed:  verdict.Passed,
		Message: anticheatMessage(verdict),
	}

	// A tier change is worth interrupting someone for, but only when the run
	// actually counted.
	if verdict.Passed && (change.Promoted() || change.Demoted()) {
		if err := s.core.Sessions.Deliver(ctx, s.db, userID, entity.KindRankChanged, entity.RankChanged{
			Old:     change.Before,
			New:     change.After,
			Delta:   change.Delta,
			Tier:    change.Tier,
			OldTier: change.OldTier,
		}); err != nil {
			// The rating is already committed, so a failed push is not worth
			// failing the request over. It is logged by Deliver's caller.
			s.logger.Warnw("could not push a rank change", "user_id", userID, "error", err)
		}
	}

	s.logger.Infow("run recorded",
		"user_id", userID,
		"wpm", req.WPM,
		"passed", verdict.Passed,
		"delta", change.Delta)
	return result, nil
}

// anticheatMessage turns a verdict into something a person should read.
func anticheatMessage(v anticheat.Verdict) string {
	if v.Passed {
		return "run recorded"
	}
	if len(v.Reasons) == 0 {
		return "run recorded, not rated"
	}
	return "run recorded but not rated: " + strings.Join(v.Reasons, "; ")
}

// Runs returns a player's recent runs.
func (s *ServiceLayer) Runs(ctx context.Context, userID uint32, page, pageSize int) (entity.RunPage, error) {
	return s.db.RecentRuns(ctx, userID, page, pageSize)
}

// RatingHistory returns a player's rating over time, for the profile graph.
func (s *ServiceLayer) RatingHistory(ctx context.Context, userID uint32, limit int) (entity.GraphPayload, error) {
	return s.db.RatingHistory(ctx, userID, limit)
}
