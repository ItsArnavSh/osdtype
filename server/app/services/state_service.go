package services

import (
	"context"
	"errors"
	"strconv"
	"time"

	"osdtyp/app/core/matchmaker"
	"osdtyp/app/entity"

	"gorm.io/gorm"
)

type StateView struct {
	State      string                  `json:"state"`
	User       entity.UserView         `json:"user"`
	Queue      *matchmaker.QueueStatus `json:"queue,omitempty"`
	Room       *entity.RoomView        `json:"room,omitempty"`
	Game       *entity.GameView        `json:"game,omitempty"`
	Results    *entity.GameView        `json:"results,omitempty"`
	Invites    []entity.InviteView     `json:"invites"`
	ServerTime time.Time               `json:"server_time"`
}

// State is the single source of truth the browser calls after login and after
// every reconnect. It decides where the player belongs.
func (s *ServiceLayer) State(ctx context.Context, uid uint32) (StateView, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return StateView{}, err
	}
	invites, _ := s.pendingInvites(ctx, uid)
	out := StateView{
		User:       entity.NewUserView(user),
		Invites:    invites,
		ServerTime: time.Now().UTC(),
	}

	st := s.state().Get(ctx, uid)
	if st.Status == "" || st.Status == "idle" {
		st = s.state().Derive(ctx, uid)
	}

	switch st.Status {
	case "queue":
		if status, ok, err := s.mm().Status(ctx, uid, user.RatingFor(st.QueueDuration), st.QueueDuration); err == nil && ok {
			out.State = "queue"
			out.Queue = &status
			return out, nil
		}
		s.state().Clear(ctx, uid)
		out.State = "idle"
		return out, nil

	case "room":
		room, err := s.db.GetRoomByCode(ctx, st.RoomCode)
		if err != nil {
			s.state().Clear(ctx, uid)
			out.State = "idle"
			return out, nil
		}
		if _, err := s.db.GetMember(ctx, room.ID, uid); err != nil {
			s.state().Clear(ctx, uid)
			out.State = "idle"
			return out, nil
		}
		view := s.roomView(ctx, room, uid, true)
		out.State = "room"
		out.Room = &view
		return out, nil

	case "game":
		if view, err := s.GetGameView(ctx, uid, st.GameID); err == nil {
			out.State = "game"
			out.Game = &view
			return out, nil
		}
		if game, err := s.db.ActiveGameForUser(ctx, uid); err == nil {
			if view, err2 := s.gameViewFromDB(ctx, uid, game); err2 == nil {
				out.State = "game"
				out.Game = &view
				return out, nil
			}
		}
		s.state().Clear(ctx, uid)
		out.State = "idle"
		return out, nil

	case "results":
		game, err := s.db.GetGame(ctx, st.ResultsGameID)
		if err != nil {
			s.state().Clear(ctx, uid)
			out.State = "idle"
			return out, nil
		}
		view, err := s.gameViewFromDB(ctx, uid, game)
		if err != nil {
			s.state().Clear(ctx, uid)
			out.State = "idle"
			return out, nil
		}
		out.State = "results"
		out.Results = &view
		if st.RoomCode != "" {
			if room, err := s.db.GetRoomByCode(ctx, st.RoomCode); err == nil {
				member, err := s.db.GetMember(ctx, room.ID, uid)
				if err == nil && member.Role != "" && room.Status != entity.RoomClosed {
					rv := s.roomView(ctx, room, uid, true)
					out.Room = &rv
				}
			}
		}
		return out, nil
	}

	out.State = "idle"
	return out, nil
}

func (s *ServiceLayer) pendingInvites(ctx context.Context, uid uint32) ([]entity.InviteView, error) {
	invites, err := s.db.PendingInvites(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]entity.InviteView, 0, len(invites))
	for _, inv := range invites {
		view := entity.InviteView{
			ID:        inv.ID,
			Status:    inv.Status,
			CreatedAt: inv.CreatedAt,
			ExpiresAt: inv.ExpiresAt,
		}
		if inv.Room != nil {
			view.RoomCode = inv.Room.Code
			view.RoomName = inv.Room.Name
		}
		if inv.Inviter != nil {
			view.From = inv.Inviter.Username
			view.FromID = inv.Inviter.ID
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *ServiceLayer) roomView(ctx context.Context, room entity.Room, viewerID uint32, includeMembers bool) entity.RoomView {
	view := entity.RoomView{
		ID:          room.ID,
		Code:        room.Code,
		Name:        room.Name,
		Description: room.Description,
		IsPublic:    room.IsPublic,
		HasPassword: room.PasswordHash != "",
		MaxPlayers:  room.MaxPlayers,
		HostID:      room.HostID,
		Status:      room.Status,
		Duration:    room.Duration,
		Language:    room.Language,
		GameID:      room.GameID,
		InviteLink:  s.frontendURL + "/join/" + room.Code,
		CreatedAt:   room.CreatedAt,
	}
	members, err := s.db.ListMembers(ctx, room.ID)
	if err != nil {
		return view
	}
	view.MemberCount = len(members)
	if includeMembers {
		view.Members = make([]entity.MemberView, 0, len(members))
		for _, m := range members {
			if m.User == nil {
				continue
			}
			online, _ := s.rdb().IsOnline(ctx, m.UserID)
			view.Members = append(view.Members, entity.MemberView{
				UserID:    m.UserID,
				Username:  m.User.Username,
				AvatarURL: m.User.AvatarURL,
				Rating:    m.User.Rating,
				Role:      m.Role,
				JoinedAt:  m.JoinedAt,
				IsBot:     m.User.IsBot,
				Online:    online || m.User.IsBot,
			})
			if m.UserID == viewerID {
				view.MyRole = m.Role
			}
		}
	}
	return view
}

func (s *ServiceLayer) GetGameView(ctx context.Context, uid, gameID uint32) (entity.GameView, error) {
	if view, ok := s.g().Snapshot(uid); ok && view.ID == gameID {
		if view.RoomID != 0 {
			if room, err := s.db.GetRoomByID(ctx, view.RoomID); err == nil {
				view.RoomCode = room.Code
			}
		}
		s.enrichContestGameView(ctx, &view)
		return view, nil
	}
	game, err := s.db.GetGame(ctx, gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.GameView{}, errf(ErrNotFound, "game not found")
		}
		return entity.GameView{}, err
	}
	return s.gameViewFromDB(ctx, uid, game)
}

func (s *ServiceLayer) gameViewFromDB(ctx context.Context, viewerID uint32, game entity.Game) (entity.GameView, error) {
	return s.gameViewFromDBFor(ctx, viewerID, game, true)
}

// gameViewFromDBFor builds the shared view. requireParticipant is false for
// public replay pages, which intentionally let anyone holding the link inspect
// a recorded run without joining the game.
func (s *ServiceLayer) gameViewFromDBFor(ctx context.Context, viewerID uint32, game entity.Game, requireParticipant bool) (entity.GameView, error) {
	players, err := s.db.GetGamePlayers(ctx, game.ID)
	if err != nil {
		return entity.GameView{}, err
	}
	found := false
	view := entity.GameView{
		ID:             game.ID,
		Mode:           game.Mode,
		Status:         game.Status,
		RoomID:         game.RoomID,
		ContestID:      game.ContestID,
		ContestStageID: game.ContestStageID,
		Duration:       game.Duration,
		Language:       game.Language,
		Seed:           strconv.FormatUint(game.Seed, 10),
		Snippet:        game.Snippet,
		StartOffset:    game.StartOffset,
		Temporary:      game.Temporary,
		StartsAt:       game.StartsAt,
		EndsAt:         game.EndsAt,
		CreatedAt:      game.CreatedAt,
	}
	if game.RoomID != 0 {
		if room, err := s.db.GetRoomByID(ctx, game.RoomID); err == nil {
			view.RoomCode = room.Code
		}
	}

	results := make([]entity.ResultView, 0, len(players))
	for _, p := range players {
		username, avatar := "", ""
		if p.User != nil {
			username, avatar = p.User.Username, p.User.AvatarURL
		}
		view.Players = append(view.Players, entity.PlayerProgress{
			UserID:        p.UserID,
			Username:      username,
			AvatarURL:     avatar,
			CurrentPoints: p.CurrentPoints,
			WPM:           p.WPM,
			Accuracy:      p.Accuracy,
			Finished:      p.Finished,
			Flagged:       p.Flagged,
			IsBot:         p.IsBot,
		})
		result := entity.ResultView{
			UserID:       p.UserID,
			Username:     username,
			AvatarURL:    avatar,
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
		}
		results = append(results, result)
		if p.UserID == viewerID {
			found = true
			r := result
			view.MyResult = &r
		}
	}
	if requireParticipant && !found {
		return entity.GameView{}, errf(ErrForbidden, "you are not part of this game")
	}
	if game.Status == entity.GameFinished {
		sortResults(results)
		view.Leaderboard = results
	}
	s.enrichContestGameView(ctx, &view)
	return view, nil
}

func (s *ServiceLayer) enrichContestGameView(ctx context.Context, view *entity.GameView) {
	if view == nil || view.ContestID == 0 {
		return
	}
	contest, err := s.db.GetContest(ctx, view.ContestID)
	if err != nil {
		return
	}
	view.ContestCode = contest.Code
	if view.ContestStageID == 0 {
		return
	}
	stage, err := s.db.GetContestStage(ctx, view.ContestStageID)
	if err != nil {
		return
	}
	view.ContestStageName = stage.Name
	view.ContestStageIndex = stage.Position
}

func sortResults(results []entity.ResultView) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Rank < results[i].Rank {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func (s *ServiceLayer) AckResults(ctx context.Context, uid uint32) (StateView, error) {
	st := s.state().AckResults(ctx, uid)
	if st.Status == "room" {
		return s.State(ctx, uid)
	}
	return s.State(ctx, uid)
}
