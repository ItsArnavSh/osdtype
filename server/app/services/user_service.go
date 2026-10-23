package services

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)

func (s *ServiceLayer) newUser(name string, rating int, isBot bool) entity.User {
	user := entity.User{
		ID:       s.intGen.GenerateID(),
		Username: name,
		IsBot:    isBot,
	}
	user.SetAllRatings(rating)
	return user
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

	user := s.newUser(username, entity.StartingElo, false)
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
	Positions  map[string]int64      `json:"positions"`
	History    []entity.MatchHistory `json:"history"`
	Pinned     []entity.MatchHistory `json:"pinned"`
	Best       *entity.MatchHistory  `json:"best"`
	TopWPM     float64               `json:"top_wpm"`
	Online     bool                  `json:"online"`
	IsFriend   bool                  `json:"is_friend"`
	IsFollowed bool                  `json:"is_followed"`
	FollowsYou bool                  `json:"follows_you"`
}

func (s *ServiceLayer) Profile(ctx context.Context, viewerID uint32, username string) (ProfileView, error) {
	user, err := s.GetUserFromName(ctx, username)
	if err != nil {
		return ProfileView{}, err
	}
	positions := make(map[string]int64, len(entity.RankedDurations))
	var bestPosition int64
	for _, duration := range entity.RankedDurations {
		pos, err := s.db.RankedPosition(ctx, user, duration)
		if err != nil {
			return ProfileView{}, err
		}
		positions[strconv.Itoa(duration)] = pos
		if pos > 0 && (bestPosition == 0 || pos < bestPosition) {
			bestPosition = pos
		}
	}
	history, err := s.db.UserGameHistory(ctx, user.ID, 50)
	if err != nil {
		return ProfileView{}, err
	}
	pinned, err := s.db.PinnedGameHistory(ctx, user.ID, 12)
	if err != nil {
		return ProfileView{}, err
	}
	best, err := s.db.BestGameHistory(ctx, user.ID)
	if err != nil {
		return ProfileView{}, err
	}
	topWPM := 0.0
	if best != nil {
		topWPM = best.WPM
	}
	online, _ := s.rdb().IsOnline(ctx, user.ID)
	out := ProfileView{
		User:      entity.NewUserView(user),
		Position:  bestPosition,
		Positions: positions,
		History:   history,
		Pinned:    pinned,
		Best:      best,
		TopWPM:    topWPM,
		Online:    online || user.IsBot,
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
		targetFollowing, _ := s.db.GetFollowing(ctx, user.ID)
		for _, id := range targetFollowing {
			if id == viewerID {
				out.FollowsYou = true
				break
			}
		}
	}
	return out, nil
}

type LeaderboardRow struct {
	Position    int64           `json:"position"`
	Rating      int             `json:"rating"`
	PeakRating  int             `json:"peak_rating"`
	GamesPlayed int             `json:"games_played"`
	Wins        int             `json:"wins"`
	User        entity.UserView `json:"user"`
}

type LeaderboardView struct {
	Duration    int              `json:"duration"`
	Leaderboard []LeaderboardRow `json:"leaderboard"`
	Nearby      []LeaderboardRow `json:"nearby"`
	MyPosition  int64            `json:"my_position"`
}

type LeaderboardSearchView struct {
	Duration int              `json:"duration"`
	Results  []LeaderboardRow `json:"results"`
}

const (
	// The public ladder always leads with the top five. Anything beyond that
	// belongs to the player's own ±5 window or an explicit search.
	leaderboardTopSize      = 5
	leaderboardNearbyRadius = 5
)

func leaderboardRow(position int64, u entity.User, duration int) LeaderboardRow {
	return LeaderboardRow{
		Position:    position,
		Rating:      u.RatingFor(duration),
		PeakRating:  u.PeakFor(duration),
		GamesPlayed: u.GamesFor(duration),
		Wins:        u.WinsFor(duration),
		User:        entity.NewUserView(u),
	}
}

func (s *ServiceLayer) Leaderboard(ctx context.Context, viewerID uint32, duration, offset, limit int) (LeaderboardView, error) {
	if _, ok := entity.LobbyTypeFromSeconds(duration); !ok {
		return LeaderboardView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300 seconds")
	}
	if limit <= 0 || limit > leaderboardTopSize {
		limit = leaderboardTopSize
	}
	if offset < 0 {
		offset = 0
	}
	users, err := s.db.Leaderboard(ctx, duration, offset, limit)
	if err != nil {
		return LeaderboardView{}, err
	}
	top := make([]LeaderboardRow, 0, len(users))
	for i, u := range users {
		top = append(top, leaderboardRow(int64(offset+i+1), u, duration))
	}
	out := LeaderboardView{Duration: duration, Leaderboard: top, Nearby: []LeaderboardRow{}}
	if viewerID == 0 {
		return out, nil
	}
	viewer, err := s.GetUser(ctx, viewerID)
	if err != nil {
		return LeaderboardView{}, err
	}
	position, err := s.db.RankedPosition(ctx, viewer, duration)
	if err != nil {
		return LeaderboardView{}, err
	}
	out.MyPosition = position
	if position == 0 {
		return out, nil
	}

	// Five players above and five below, clamped at the ends. At the top
	// of the ladder this is six rows (self plus five below), not a window
	// that shifts ten players downward.
	const radius = leaderboardNearbyRadius
	start := position - 1 - radius
	if start < 0 {
		start = 0
	}
	end := position + radius
	nearbyUsers, err := s.db.Leaderboard(ctx, duration, int(start), radius*2+1)
	if err != nil {
		return LeaderboardView{}, err
	}
	out.Nearby = make([]LeaderboardRow, 0, len(nearbyUsers))
	for i, u := range nearbyUsers {
		pos := int64(start) + int64(i) + 1
		if pos > end {
			break
		}
		out.Nearby = append(out.Nearby, leaderboardRow(pos, u, duration))
	}
	return out, nil
}

// SearchLeaderboard returns ranked players matching a username fragment. Each
// row carries its exact ladder position so the result is meaningful even when
// the player is nowhere near the requested clock's top five.
func (s *ServiceLayer) SearchLeaderboard(ctx context.Context, duration int, query string, limit int) (LeaderboardSearchView, error) {
	if _, ok := entity.LobbyTypeFromSeconds(duration); !ok {
		return LeaderboardSearchView{}, errf(ErrBadRequest, "duration must be 30, 90 or 300 seconds")
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	out := LeaderboardSearchView{Duration: duration, Results: []LeaderboardRow{}}
	if query == "" {
		return out, nil
	}
	users, err := s.db.SearchLeaderboard(ctx, duration, query, limit)
	if err != nil {
		return LeaderboardSearchView{}, err
	}
	for _, u := range users {
		position, err := s.db.RankedPosition(ctx, u, duration)
		if err != nil {
			return LeaderboardSearchView{}, err
		}
		if position == 0 {
			continue
		}
		out.Results = append(out.Results, leaderboardRow(position, u, duration))
	}
	return out, nil
}
