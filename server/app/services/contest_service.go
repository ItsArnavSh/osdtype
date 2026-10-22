package services

import (
	"context"
	"encoding/json"
	"errors"
	"osdtyp/app/entity"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CreateContestRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartsAt    time.Time `json:"starts_at"`
	Duration    int       `json:"duration"`
	Language    string    `json:"language"`
}

type ContestView struct {
	ID          uint32               `json:"id"`
	RoomCode    string               `json:"room_code"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	StartsAt    time.Time            `json:"starts_at"`
	Duration    int                  `json:"duration"`
	Language    string               `json:"language"`
	Status      entity.ContestStatus `json:"status"`
	GameID      uint32               `json:"game_id"`
	Leaderboard []entity.ResultView  `json:"leaderboard,omitempty"`
}

func (s *ServiceLayer) CreateContest(ctx context.Context, uid uint32, code string, req CreateContestRequest) (ContestView, error) {
	room, _, err := s.requireRoomRole(ctx, uid, code, entity.RoleOwner)
	if err != nil {
		return ContestView{}, err
	}
	if room.Status == entity.RoomPlaying {
		return ContestView{}, errf(ErrConflict, "cannot schedule a contest during a game")
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len(req.Title) > 120 {
		return ContestView{}, errf(ErrBadRequest, "title must be 1-120 characters")
	}
	if req.StartsAt.IsZero() || req.StartsAt.Before(time.Now().Add(30*time.Second)) {
		return ContestView{}, errf(ErrBadRequest, "start time must be at least 30 seconds in the future")
	}
	if req.Duration == 0 {
		req.Duration = room.Duration
	}
	if _, ok := entity.LobbyTypeFromSeconds(req.Duration); !ok {
		return ContestView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300")
	}
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Language == "" {
		req.Language = room.Language
	}
	if !validLanguage(req.Language) || req.Language == "random" {
		return ContestView{}, errf(ErrBadRequest, "unknown language")
	}

	contest := entity.Contest{
		ID:          s.intGen.GenerateID(),
		RoomID:      room.ID,
		CreatedBy:   uid,
		Title:       req.Title,
		Description: req.Description,
		StartsAt:    req.StartsAt.UTC(),
		Duration:    req.Duration,
		Language:    req.Language,
		Status:      entity.ContestUpcoming,
	}
	if err := s.db.CreateContest(ctx, &contest); err != nil {
		return ContestView{}, err
	}
	if err := s.db.CreateTask(ctx, &entity.Task{
		Category: entity.CONTEST,
		JobID:    contest.ID,
		Time:     contest.StartsAt,
	}); err != nil {
		return ContestView{}, err
	}
	return s.contestView(ctx, contest, room.Code), nil
}

func (s *ServiceLayer) ListContests(ctx context.Context, code string, offset, limit int) ([]ContestView, error) {
	room, err := s.db.GetRoomByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, errf(ErrNotFound, "room not found")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	contests, err := s.db.ListContests(ctx, room.ID, offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ContestView, 0, len(contests))
	for _, c := range contests {
		out = append(out, s.contestView(ctx, c, room.Code))
	}
	return out, nil
}

func (s *ServiceLayer) GetContest(ctx context.Context, id uint32) (ContestView, error) {
	c, err := s.db.GetContest(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	room, err := s.db.GetRoomByID(ctx, c.RoomID)
	if err != nil {
		return ContestView{}, err
	}
	return s.contestView(ctx, c, room.Code), nil
}

func (s *ServiceLayer) contestView(ctx context.Context, c entity.Contest, code string) ContestView {
	out := ContestView{
		ID:          c.ID,
		RoomCode:    code,
		Title:       c.Title,
		Description: c.Description,
		StartsAt:    c.StartsAt,
		Duration:    c.Duration,
		Language:    c.Language,
		Status:      c.Status,
		GameID:      c.GameID,
	}
	if len(c.Leaderboard) > 0 {
		_ = json.Unmarshal(c.Leaderboard, &out.Leaderboard)
	}
	return out
}

// HandleContestTask is driven by the scheduler. It moves a contest from
// upcoming -> lobby -> started; the game finish hook closes it out.
func (s *ServiceLayer) HandleContestTask(ctx context.Context, task entity.Task) error {
	if task.Category != entity.CONTEST {
		return nil
	}
	contest, err := s.db.GetContest(ctx, task.JobID)
	if err != nil {
		return err
	}
	room, err := s.db.GetRoomByID(ctx, contest.RoomID)
	if err != nil {
		return err
	}
	if room.Status == entity.RoomClosed {
		contest.Status = entity.ContestEnded
		return s.db.SaveContest(ctx, &contest)
	}

	switch contest.Status {
	case entity.ContestUpcoming:
		now := time.Now().UTC()
		contest.Status = entity.ContestLobby
		contest.LobbyOpensAt = &now
		if err := s.db.SaveContest(ctx, &contest); err != nil {
			return err
		}
		s.broadcastRoom(ctx, room.ID, "room_update")
		return s.db.CreateTask(ctx, &entity.Task{
			Category: entity.CONTEST,
			JobID:    contest.ID,
			Time:     now.Add(time.Duration(s.lobbyMinutes) * time.Minute),
		})
	case entity.ContestLobby:
		if room.Status == entity.RoomPlaying {
			// The host already started a normal game; start the contest after
			// it finishes rather than overlapping clocks.
			return s.db.CreateTask(ctx, &entity.Task{
				Category: entity.CONTEST,
				JobID:    contest.ID,
				Time:     time.Now().Add(30 * time.Second),
			})
		}
		room.Duration = contest.Duration
		room.Language = contest.Language
		g, err := s.createRoomGame(ctx, room, entity.GameContest, contest.ID)
		if err != nil {
			if errors.Is(err, ErrConflict) {
				return s.db.CreateTask(ctx, &entity.Task{
					Category: entity.CONTEST,
					JobID:    contest.ID,
					Time:     time.Now().Add(30 * time.Second),
				})
			}
			return err
		}
		contest.Status = entity.ContestStarted
		contest.GameID = g.ID
		return s.db.SaveContest(ctx, &contest)
	}
	return nil
}
