package services

import (
	"context"
	"encoding/json"
	"errors"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

// GameReplays returns every recorded run for the results screen. The caller
// must be a participant so private room games are not enumerable.
func (s *ServiceLayer) GameReplays(ctx context.Context, uid, gameID uint32) (entity.ReplayBundle, error) {
	game, err := s.db.GetGame(ctx, gameID)
	if err != nil {
		return entity.ReplayBundle{}, gameLookupError(err)
	}
	if game.Status != entity.GameFinished {
		return entity.ReplayBundle{}, errf(ErrConflict, "game is not finished")
	}
	players, err := s.db.GetGamePlayers(ctx, gameID)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	found := false
	for _, p := range players {
		if p.UserID == uid {
			found = true
			break
		}
	}
	if !found {
		return entity.ReplayBundle{}, errf(ErrForbidden, "you are not part of this game")
	}
	view, err := s.gameViewFromDBFor(ctx, uid, game, true)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	return s.buildReplayBundle(ctx, view, players, 0)
}

// PublicReplay serves a single run from a share link. It deliberately does
// not require authentication, but only exposes the requested player's events.
func (s *ServiceLayer) PublicReplay(ctx context.Context, gameID, userID uint32) (entity.ReplayBundle, error) {
	game, err := s.db.GetGame(ctx, gameID)
	if err != nil {
		return entity.ReplayBundle{}, gameLookupError(err)
	}
	if game.Status != entity.GameFinished {
		return entity.ReplayBundle{}, errf(ErrNotFound, "replay not found")
	}
	players, err := s.db.GetGamePlayers(ctx, gameID)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	found := false
	for _, p := range players {
		if p.UserID == userID {
			found = true
			break
		}
	}
	if !found {
		return entity.ReplayBundle{}, errf(ErrNotFound, "replay not found")
	}
	view, err := s.gameViewFromDBFor(ctx, 0, game, false)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	// A share link grants access to one recorded run, not the rest of the
	// lobby. Strip participant rows and standings before returning the view.
	view.Players = nil
	view.Leaderboard = nil
	view.MyResult = nil
	view.RoomCode = ""
	bundle, err := s.buildReplayBundle(ctx, view, players, userID)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	if len(bundle.Runs) == 0 || len(bundle.Runs[0].Events) == 0 {
		return entity.ReplayBundle{}, errf(ErrNotFound, "recording not found")
	}
	return bundle, nil
}

func (s *ServiceLayer) buildReplayBundle(ctx context.Context, view entity.GameView, players []entity.GamePlayer, onlyUserID uint32) (entity.ReplayBundle, error) {
	recordings, err := s.db.GetRecordings(ctx, view.ID)
	if err != nil {
		return entity.ReplayBundle{}, err
	}
	byUser := make(map[uint32]entity.Recording, len(recordings))
	for _, rec := range recordings {
		byUser[rec.UserID] = rec
	}
	runs := make([]entity.ReplayRun, 0, len(players))
	for _, p := range players {
		if onlyUserID != 0 && p.UserID != onlyUserID {
			continue
		}
		run := entity.ReplayRun{
			UserID:   p.UserID,
			WPM:      p.WPM,
			Raw:      p.Raw,
			Accuracy: p.Accuracy,
			Correct:  p.Correct,
			Wrong:    p.Wrong,
			IsBot:    p.IsBot,
			Events:   []entity.Keypress{},
		}
		if p.User != nil {
			run.Username = p.User.Username
			run.AvatarURL = p.User.AvatarURL
		}
		if rec, ok := byUser[p.UserID]; ok {
			run.Final = rec.Final
			if len(rec.Events) > 0 {
				if err := json.Unmarshal(rec.Events, &run.Events); err != nil {
					s.logger.Warnw("decode recording", "game_id", view.ID, "user_id", p.UserID, "error", err)
					run.Events = []entity.Keypress{}
				}
			}
		}
		runs = append(runs, run)
	}
	return entity.ReplayBundle{Game: view, Runs: runs}, nil
}

func gameLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errf(ErrNotFound, "game not found")
	}
	return err
}
