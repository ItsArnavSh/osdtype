package services

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)

func (s *ServiceLayer) newUser(name string, rating int, isBot bool) entity.User {
	return entity.User{
		ID:         s.intGen.GenerateID(),
		Username:   name,
		Rating:     rating,
		PeakRating: rating,
		IsBot:      isBot,
	}
}

// LoginUser creates or returns the account for an OAuth identity.
func (s *ServiceLayer) LoginUser(ctx context.Context, username, avatar string) (entity.User, error) {
	username = strings.TrimSpace(username)
	if !usernameRe.MatchString(username) {
		return entity.User{}, errf(ErrBadRequest, "invalid username")
	}

	if existing, err := s.db.GetUserFromName(ctx, username); err == nil {
		if strings.Contains(existing.AvatarURL, "github.com/identicons/") && avatar == "" {
			// Older dev accounts used GitHub's retired identicon endpoint.
			// Clear it so the UI renders its local initial placeholder.
			existing.AvatarURL = ""
			_ = s.db.SaveUser(ctx, existing)
		} else if existing.AvatarURL == "" && avatar != "" {
			existing.AvatarURL = avatar
			_ = s.db.SaveUser(ctx, existing)
		}
		_ = s.rdb().SetRating(ctx, existing.ID, existing.Rating)
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.User{}, err
	}

	user := s.newUser(username, 1000, false)
	user.AvatarURL = avatar
	if err := s.db.AddUser(ctx, user); err != nil {
		// A concurrent callback may have won the race.
		if existing, getErr := s.db.GetUserFromName(ctx, username); getErr == nil {
			return existing, nil
		}
		return entity.User{}, err
	}
	_ = s.rdb().SetRating(ctx, user.ID, user.Rating)
	return user, nil
}

func (s *ServiceLayer) GetUser(ctx context.Context, uid uint32) (entity.User, error) {
	user, err := s.db.GetUser(uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, errf(ErrNotFound, "user not found")
		}
		return entity.User{}, err
	}
	return user, nil
}

func (s *ServiceLayer) GetUserFromName(ctx context.Context, name string) (entity.User, error) {
	user, err := s.db.GetUserFromName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, errf(ErrNotFound, "user not found")
		}
		return entity.User{}, err
	}
	return user, nil
}

func (s *ServiceLayer) SearchUsers(ctx context.Context, name string, limit int) ([]entity.UserView, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	users, err := s.db.SearchPeople(ctx, name, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entity.UserView, 0, len(users))
	for _, u := range users {
		out = append(out, entity.NewUserView(u))
	}
	return out, nil
}

type ProfileView struct {
	User       entity.UserView       `json:"user"`
	Position   int64                 `json:"position"`
	History    []entity.MatchHistory `json:"history"`
	Online     bool                  `json:"online"`
	IsFriend   bool                  `json:"is_friend"`
	IsFollowed bool                  `json:"is_followed"`
}

func (s *ServiceLayer) Profile(ctx context.Context, viewerID uint32, username string) (ProfileView, error) {
	user, err := s.GetUserFromName(ctx, username)
	if err != nil {
		return ProfileView{}, err
	}
	pos, err := s.db.CountRatingAbove(ctx, user.Rating)
	if err != nil {
		return ProfileView{}, err
	}
	history, err := s.db.UserGameHistory(ctx, user.ID, 10)
	if err != nil {
		return ProfileView{}, err
	}
	online, _ := s.rdb().IsOnline(ctx, user.ID)
	out := ProfileView{
		User:     entity.NewUserView(user),
		Position: pos + 1,
		History:  history,
		Online:   online || user.IsBot,
	}
	if viewerID != 0 && viewerID != user.ID {
		out.IsFriend, _ = s.db.IsFriend(ctx, viewerID, user.ID)
		following, _ := s.db.GetFollowing(ctx, viewerID)
		for _, id := range following {
			if id == user.ID {
				out.IsFollowed = true
				break
			}
		}
	}
	return out, nil
}

type LeaderboardRow struct {
	Position int64           `json:"position"`
	User     entity.UserView `json:"user"`
}

func (s *ServiceLayer) Leaderboard(ctx context.Context, offset, limit int) ([]LeaderboardRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	users, err := s.db.Leaderboard(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]LeaderboardRow, 0, len(users))
	for i, u := range users {
		out = append(out, LeaderboardRow{Position: int64(offset + i + 1), User: entity.NewUserView(u)})
	}
	return out, nil
}
