package services

import (
	"context"
	"encoding/json"
	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"sort"
	"time"

	"gorm.io/gorm"
)

func (s *ServiceLayer) LeaveGame(ctx context.Context, uid, gameID uint32) (StateView, error) {
	activeID, inGame := s.g().IsInGame(uid)
	if !inGame {
		if gameID != 0 {
			if g, err := s.db.GetGame(ctx, gameID); err == nil && g.Status == entity.GameFinished {
				return s.State(ctx, uid)
			}
		}
		return s.State(ctx, uid)
	}
	if gameID != 0 && gameID != activeID {
		return StateView{}, errf(ErrBadRequest, "that is not your current game")
	}
	_ = s.db.MarkPlayerLeft(ctx, activeID, uid)
	s.g().Leave(ctx, uid)

	// The game may have finished synchronously because this was the last
	// human. In that case the finish hook already wrote the results state.
	if st, ok, _ := s.rdb().GetState(ctx, uid); ok && st.Status == "results" {
		return s.State(ctx, uid)
	}
	if room, inRoom, _ := s.db.ActiveRoomForUser(ctx, uid); inRoom {
		s.state().SetRoom(ctx, uid, room.Code)
	} else {
		s.state().Clear(ctx, uid)
	}
	return s.State(ctx, uid)
}

func (s *ServiceLayer) onGameFinished(ctx context.Context, g entity.Game, players []entity.GamePlayer, recordings []entity.Recording, flags []entity.CheatFlag) ([]entity.ResultView, error) {
	ids := make([]uint32, 0, len(players))
	for _, p := range players {
		ids = append(ids, p.UserID)
	}
	users, err := s.db.ListUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	userMap := make(map[uint32]entity.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	ratings := make([]int, len(players))
	positions := make([]int, len(players))
	for i, p := range players {
		ratings[i] = p.RatingBefore
		positions[i] = p.Placement
		if positions[i] <= 0 {
			positions[i] = i + 1
		}
	}
	after := ratings
	if g.Mode == entity.GameRanked {
		after = utils.UpdateElo(ratings, positions)
	}

	for i := range players {
		if players[i].Flagged {
			// A flagged run must never move the rating, even if the stored
			// before-value is stale (for example after a reconnect).
			current := players[i].RatingBefore
			if user, ok := userMap[players[i].UserID]; ok {
				current = user.Rating
			}
			players[i].RatingBefore = current
			players[i].RatingAfter = current
			continue
		}
		players[i].RatingAfter = after[i]
	}

	if err := s.db.FinishGame(ctx, &g, players, flags); err != nil {
		return nil, err
	}
	if err := s.db.SaveRecordings(ctx, recordings); err != nil {
		s.logger.Warnw("save recordings", "error", err)
	}

	var histories []entity.RatingHistory
	now := time.Now().UTC()
	for i, p := range players {
		user, ok := userMap[p.UserID]
		if !ok {
			continue
		}
		delta := p.RatingAfter - p.RatingBefore
		if g.Mode != entity.GameRanked {
			delta = 0
		}
		user.Rating = p.RatingAfter
		if user.Rating > user.PeakRating {
			user.PeakRating = user.Rating
		}
		if !user.IsBot {
			user.GamesPlayed++
			if p.Placement == 1 && !p.Flagged && !p.Left {
				user.Wins++
			}
		}
		if err := s.db.SaveUser(ctx, user); err != nil {
			s.logger.Warnw("save user after game", "user_id", user.ID, "error", err)
		}
		if !user.IsBot {
			_ = s.rdb().SetRating(ctx, user.ID, user.Rating)
		}
		if g.Mode == entity.GameRanked {
			histories = append(histories, entity.RatingHistory{
				UserID:      user.ID,
				GameID:      g.ID,
				RatingDelta: delta,
				RatingAfter: user.Rating,
				Mode:        g.Mode,
				CreatedAt:   now,
			})
		}
		_ = i
	}
	if err := s.db.SaveRatings(ctx, histories); err != nil {
		s.logger.Warnw("save rating history", "error", err)
	}

	roomCode := ""
	if g.RoomID != 0 {
		if room, err := s.db.GetRoomByID(ctx, g.RoomID); err == nil {
			roomCode = room.Code
			if room.GameID == g.ID {
				_ = s.db.UpdateRoomFields(ctx, room.ID, map[string]any{
					"status":  entity.RoomLobby,
					"game_id": 0,
				})
			}
			s.broadcastRoom(ctx, room.ID, "room_update")
		}
	}
	if g.ContestID != 0 {
		if contest, err := s.db.GetContest(ctx, g.ContestID); err == nil {
			lb, _ := json.Marshal(resultViewsFor(g.ID, players, userMap))
			contest.Status = entity.ContestEnded
			contest.Leaderboard = lb
			contest.GameID = g.ID
			_ = s.db.SaveContest(ctx, &contest)
		}
	}

	views := resultViewsFor(g.ID, players, userMap)
	for _, p := range players {
		user, ok := userMap[p.UserID]
		if !ok || user.IsBot {
			continue
		}
		s.state().SetResults(ctx, p.UserID, g.ID, roomCode)
		if st, err := s.State(ctx, p.UserID); err == nil {
			s.hub().SendToUser(p.UserID, hub.Ev("state", st))
		}
	}
	return views, nil
}

func resultViewsFor(gameID uint32, players []entity.GamePlayer, users map[uint32]entity.User) []entity.ResultView {
	out := make([]entity.ResultView, 0, len(players))
	for _, p := range players {
		u := users[p.UserID]
		out = append(out, entity.ResultView{
			UserID:       p.UserID,
			Username:     u.Username,
			AvatarURL:    u.AvatarURL,
			Rank:         p.Placement,
			WPM:          p.WPM,
			Raw:          p.Raw,
			Accuracy:     p.Accuracy,
			Correct:      p.Correct,
			Wrong:        p.Wrong,
			RatingBefore: p.RatingBefore,
			RatingAfter:  p.RatingAfter,
			RatingDelta:  p.RatingAfter - p.RatingBefore,
			Flagged:      p.Flagged,
			Left:         p.Left,
			IsBot:        p.IsBot,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Left != out[j].Left {
			return !out[i].Left
		}
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].WPM > out[j].WPM
	})
	return out
}

func (s *ServiceLayer) RecentGame(ctx context.Context, uid, gameID uint32) (entity.GameView, error) {
	g, err := s.db.GetGame(ctx, gameID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return entity.GameView{}, errf(ErrNotFound, "game not found")
		}
		return entity.GameView{}, err
	}
	return s.gameViewFromDB(ctx, uid, g)
}
