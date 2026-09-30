package services

import (
	"context"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

// What happens when a private lobby's round ends.
//
// This is the piece that makes a friends' game mean something. The round
// itself is a game loop and produces a leaderboard; without this, the leaderboard
// went to the players and was then discarded, and nobody's rating or history
// moved for a game they genuinely played. It is the same accounting the
// scheduler does for a contest, for the same reason.

// finishLobby applies a finished lobby's leaderboard.
//
// It waits on the result channel rather than having Start block for it, so the
// HTTP start request returns as soon as the round is launched. The context is
// detached from the request: a start request's context is canceled the moment
// the response is written, and a canceled context here would abandon the round
// result and leave everyone's rating unchanged for a game that was played.
func (s *ServiceLayer) finishLobby(ctx context.Context, roomID uint32, sig chan []entity.WPMRes) {
	// WithoutCancel keeps the database work alive past the response, and the
	// timeout bounds it so a round that never reports cannot pin a goroutine
	// forever.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resultTimeout)
	defer cancel()

	leaderboard, got := resultOrTimeout(ctx, sig)
	if !got {
		s.logger.Errorw("lobby round did not report a result", "room_id", roomID)
		if err := s.db.SetRoomStatus(ctx, roomID, entity.LobbyOpen); err != nil {
			s.logger.Errorw("could not reopen the lobby", "room_id", roomID, "error", err)
		}
		return
	}

	if err := s.applyLeaderboard(ctx, roomID, leaderboard); err != nil {
		s.logger.Errorw("could not apply a lobby result", "room_id", roomID, "error", err)
	}

	if err := s.db.SetRoomStatus(ctx, roomID, entity.LobbyFinished); err != nil {
		s.logger.Errorw("could not mark the lobby finished", "room_id", roomID, "error", err)
	}
	s.logger.Infow("lobby finished", "room_id", roomID, "players", len(leaderboard))
}

// resultOrTimeout reads the leaderboard, reporting false if the wait ran out
// or the context was canceled.
func resultOrTimeout(ctx context.Context, sig chan []entity.WPMRes) ([]entity.WPMRes, bool) {
	select {
	case res := <-sig:
		return res, res != nil
	case <-ctx.Done():
		return nil, false
	}
}

// resultTimeout is how long to wait for a round's leaderboard before giving up
// on it. It is longer than the longest mode's round plus the countdown, with
// room for a slow start, so it only fires when a round is genuinely stuck.
const resultTimeout = 8 * time.Minute

// applyLeaderboard rates a finished round and records a run for every player.
//
// The two halves are in one transaction in the database layer because half a
// result is worse than none: a rating that moved without a stored run shows a
// score the player cannot find in their history, and a stored run without a
// rating move shows a score the ladder disagrees with.
func (s *ServiceLayer) applyLeaderboard(ctx context.Context, roomID uint32, leaderboard []entity.WPMRes) error {
	if len(leaderboard) < 2 {
		// A one-player "round" is a practice run, not a match. Recording it
		// would move a rating for a game with nobody to beat.
		s.logger.Infow("lobby finished with too few players to rate", "room_id", roomID, "players", len(leaderboard))
		return nil
	}

	ids := make([]uint32, len(leaderboard))
	for i, e := range leaderboard {
		ids[i] = e.ID
	}

	before, played, err := s.db.RatingsFor(ctx, ids)
	if err != nil {
		return err
	}

	after := s.elo.Rate(leaderboard, before, played)

	now := time.Now()
	for i, e := range leaderboard {
		delta := int(after[i]) - int(before[i])

		run := entity.Run{
			ID:         s.int_gen.GenerateID(),
			UserID:     e.ID,
			Mode:       s.roomMode(ctx, roomID),
			Solo:       false,
			WPM:        e.WPM,
			Raw:        e.RAW,
			Accuracy:   e.Accuracy,
			Correct:    e.Correct,
			Wrong:      e.Wrong,
			DurationMS: 0,
			RankBefore: before[i],
			RankAfter:  after[i],
			Passed:     true,
			PlayedAt:   now,
		}
		if _, _, err := s.db.RateRun(ctx, run); err != nil {
			return err
		}
		if err := s.db.ChangeRank(e.ID, after[i]); err != nil {
			return err
		}

		if delta != 0 {
			_ = s.core.Sessions.Deliver(ctx, s.db, e.ID, entity.KindRankChanged, entity.RankChanged{
				Old:     before[i],
				New:     after[i],
				Delta:   delta,
				Tier:    utils.TierOf(after[i]).Name,
				OldTier: utils.TierOf(before[i]).Name,
			})
		}
	}
	return nil
}

// roomMode reports a room's mode, defaulting to sprint if the room is gone: a
// result whose room has been deleted should still be recorded rather than
// dropped.
func (s *ServiceLayer) roomMode(ctx context.Context, roomID uint32) entity.LobbyType {
	room, err := s.db.RoomByID(ctx, roomID)
	if err != nil {
		return entity.SPRINT
	}
	return room.Duration
}
