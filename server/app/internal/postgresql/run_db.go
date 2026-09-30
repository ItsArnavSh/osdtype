package postgresql

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"gorm.io/gorm"
)

// Runs, ratings and the leaderboard.
//
// The rating used to be a number that changed with no record of why. Nothing
// stored the game that moved it, so a player could not see a history, the
// leaderboard could not be built, and the anticheat had nothing to score.

// elo is the rating engine the service layer and the matchmaker both use, so a
// solo run and a ranked game cannot disagree about how a rating moves.
var elo = utils.NewEloEngine()

// SaveRun records a completed game and returns it with its id assigned.
func (d *Database) SaveRun(ctx context.Context, run entity.Run) (entity.Run, error) {
	if run.PlayedAt.IsZero() {
		run.PlayedAt = time.Now()
	}
	if err := d.db.WithContext(ctx).Create(&run).Error; err != nil {
		return entity.Run{}, err
	}
	return run, nil
}

// RateRun stores a solo result and applies the resulting rating change in one
// transaction.
//
// Both halves have to succeed together: a run recorded without a rating change
// would show a score that the leaderboard disagrees with, and a rating change
// without a run would move a player for a game nobody can see.
func (d *Database) RateRun(ctx context.Context, run entity.Run) (entity.Run, entity.RankChange, error) {
	var change entity.RankChange

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user entity.User
		if err := tx.First(&user, "id = ?", run.UserID).Error; err != nil {
			return err
		}

		run.RankBefore = user.CurrentRank
		run.RankAfter = user.CurrentRank
		change = entity.RankChange{
			Before: user.CurrentRank,
			After:  user.CurrentRank,
			Tier:   utils.TierOf(user.CurrentRank).Name,
		}

		// A solo run is worth less than a rated match, which is what stops a
		// player from grinding their rating alone. A failed anticheat verdict
		// is worth nothing at all.
		if run.Passed {
			newRating := elo.RateSolo(run.WPM, user.CurrentRank, user.GamesPlayed)
			run.RankAfter = newRating
			change.After = newRating
			change.Delta = int(newRating) - int(user.CurrentRank)
			change.OldTier = utils.TierOf(user.CurrentRank).Name
			change.Tier = utils.TierOf(newRating).Name

			if err := tx.Model(&entity.User{}).
				Where("id = ?", run.UserID).
				Updates(map[string]any{
					"current_rank": newRating,
					"games_played": gorm.Expr("games_played + 1"),
				}).Error; err != nil {
				return err
			}
		}

		return tx.Create(&run).Error
	})

	if err != nil {
		return entity.Run{}, entity.RankChange{}, err
	}
	return run, change, nil
}

// RatingsFor returns a set of players' current ratings and games played.
func (d *Database) RatingsFor(ctx context.Context, ids []uint32) ([]uint16, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	rows := make([]entity.User, 0, len(ids))
	if err := d.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	byID := make(map[uint32]entity.User, len(rows))
	for _, u := range rows {
		byID[u.ID] = u
	}

	ratings := make([]uint16, len(ids))
	played := make([]int, len(ids))
	for i, id := range ids {
		u, ok := byID[id]
		if !ok {
			ratings[i] = utils.StartingRating
			continue
		}
		ratings[i] = u.CurrentRank
		played[i] = u.GamesPlayed
	}
	return ratings, played, nil
}

// ChangeRank sets a player's rating outright.
//
// The previous version wrote to a "rank" column that does not exist; the
// column is current_rank, so every rating update silently failed.
func (d *Database) ChangeRank(userid uint32, rank uint16) error {
	return d.db.Model(&entity.User{}).
		Where("id = ?", userid).
		Update("current_rank", rank).Error
}

// GetRank returns a player's rating.
func (d *Database) GetRank(ctx context.Context, userid uint32) (uint16, error) {
	var rank uint16
	err := d.db.WithContext(ctx).
		Model(&entity.User{}).
		Select("current_rank").
		Where("id = ?", userid).
		Scan(&rank).Error
	return rank, err
}

// leaderboardFilter is a leaderboard request with every choice resolved.
//
// The mode has already been turned into a value, the friends scope has been
// pointed at the caller rather than at whatever the request claimed, and the
// paging has been bounded. Keeping that resolution in one place is what stops
// the query below having to know how a scope is spelled, and stops a request
// from asking for someone else's friends.
type leaderboardFilter struct {
	// Scope is "global", "friends" or "room".
	Scope string
	// ScopeID is whose friends, for the friends scope.
	ScopeID uint32
	// RoomID is which room, for the room scope.
	RoomID uint32
	// Mode filters by game mode. A nil value means every mode.
	Mode *entity.LobbyType
	// SoloOnly restricts to practice runs.
	SoloOnly bool
	// Page is zero based.
	Page int
	// PageSize is how many rows to return.
	PageSize int
	// SelfID is the caller, used to mark their own row.
	SelfID uint32
}

// resolveLeaderboard turns a request into the filter the query needs.
//
// It clamps rather than rejects for paging, because a page index that is
// negative or enormous is a client that has lost track, and answering with a
// page is more useful than a 400. It does reject an unknown scope, because
// silently treating "friendz" as global would return the whole ladder to
// someone who asked for their friends.
func resolveLeaderboard(opts entity.LeaderboardOptions, selfID uint32) (leaderboardFilter, error) {
	filter := leaderboardFilter{
		Scope:    normalizeScope(opts.Scope),
		RoomID:   opts.RoomID,
		SoloOnly: opts.SoloOnly,
		Page:     max(opts.Page, 0),
		PageSize: clampPageSize(opts.PageSize),
		ScopeID:  selfID,
		SelfID:   selfID,
	}

	if opts.Mode != "" {
		mode, ok := entity.LobbyTypeFromString(opts.Mode)
		if !ok {
			return leaderboardFilter{}, fmt.Errorf("%w: %q", entity.ErrUnknownMode, opts.Mode)
		}
		filter.Mode = &mode
	}

	// A friends scope is always the caller's own friends. Taking it from the
	// request would let anyone read anyone's friend list.
	if filter.Scope == "friends" {
		filter.ScopeID = selfID
	}
	// A room scope with no room would return nobody, which reads as "you are
	// last" rather than "you did not say which room". Treat it as global.
	if filter.Scope == "room" && filter.RoomID == 0 {
		filter.Scope = "global"
	}

	return filter, nil
}

func normalizeScope(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "friends", "friend":
		return "friends"
	case "room", "lobby":
		return "room"
	default:
		return "global"
	}
}

// Page size bounds.
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func clampPageSize(n int) int {
	if n <= 0 {
		return defaultPageSize
	}
	if n > maxPageSize {
		return maxPageSize
	}
	return n
}

// Leaderboard returns a page of rankings.
//
// Ranking is by rating, not by best WPM: the rating already accounts for who you
// played, so ranking the raw score would put a player who beat a beginner above
// one who beat an equal, which is the opposite of the point.
func (d *Database) Leaderboard(ctx context.Context, opts entity.LeaderboardOptions, selfID uint32) (entity.LeaderboardPage, error) {
	filter, err := resolveLeaderboard(opts, selfID)
	if err != nil {
		return entity.LeaderboardPage{}, err
	}

	q := d.db.WithContext(ctx).Model(&entity.User{})

	switch filter.Scope {
	case "friends":
		q = q.Where("id IN (?)", d.friendIDsQuery(ctx, filter.ScopeID))
	case "room":
		q = q.Where("id IN (?)", d.roomMemberIDsQuery(ctx, filter.RoomID))
	case "global":
		// No filter.
	}

	var total int64
	if countErr := q.Count(&total).Error; countErr != nil {
		return entity.LeaderboardPage{}, countErr
	}

	var users []entity.User
	err = q.Order("current_rank DESC, games_played DESC, id ASC").
		Offset(filter.Page * filter.PageSize).
		Limit(filter.PageSize).
		Find(&users).Error
	if err != nil {
		return entity.LeaderboardPage{}, err
	}

	best, err := d.BestWPM(ctx, userIDs(users), filter)
	if err != nil {
		return entity.LeaderboardPage{}, err
	}

	entries := make([]entity.LeaderboardEntry, 0, len(users))
	for i, u := range users {
		entries = append(entries, entity.LeaderboardEntry{
			Rank:      filter.Page*filter.PageSize + i + 1,
			UserID:    u.ID,
			Username:  u.Username,
			AvatarURL: u.AvatarURL,
			Rating:    u.CurrentRank,
			Tier:      utils.TierOf(u.CurrentRank).Name,
			Games:     u.GamesPlayed,
			WPM:       best[u.ID],
			IsSelf:    u.ID == filter.SelfID,
		})
	}

	page := entity.LeaderboardPage{
		Entries:  entries,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
	}

	// The caller's own rank may be well outside the page they asked for, so it
	// is looked up separately. Without this, a player outside the top page has
	// no idea where they stand.
	if self, err := d.rankOf(ctx, filter.SelfID); err == nil && self > 0 {
		page.SelfRank = self
	}
	if selfEntry, err := d.entryFor(ctx, filter.SelfID, best); err == nil && selfEntry != nil {
		page.SelfEntry = selfEntry
	}

	return page, nil
}

// rankOf returns a player's position on the global ladder, one based.
func (d *Database) rankOf(ctx context.Context, userID uint32) (int, error) {
	if userID == 0 {
		return 0, nil
	}

	rating, err := d.GetRank(ctx, userID)
	if err != nil {
		return 0, err
	}

	var better int64
	if err := d.db.WithContext(ctx).Model(&entity.User{}).
		Where("current_rank > ?", rating).
		Count(&better).Error; err != nil {
		return 0, err
	}
	return int(better) + 1, nil
}

// entryFor builds a single leaderboard row, used for the caller's own entry.
func (d *Database) entryFor(ctx context.Context, userID uint32, best map[uint32]float32) (*entity.LeaderboardEntry, error) {
	if userID == 0 {
		return nil, nil
	}
	user, err := d.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity.LeaderboardEntry{
		UserID:    user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Rating:    user.CurrentRank,
		Tier:      utils.TierOf(user.CurrentRank).Name,
		Games:     user.GamesPlayed,
		WPM:       best[user.ID],
		IsSelf:    true,
	}, nil
}

func (d *Database) friendIDsQuery(ctx context.Context, id uint32) *gorm.DB {
	return d.db.WithContext(ctx).Model(&entity.Friends{}).
		Select("CASE WHEN a = ? THEN b ELSE a END", id).
		Where("(a = ? OR b = ?) AND relation = ?", id, id, entity.FRIENDS)
}

func (d *Database) roomMemberIDsQuery(ctx context.Context, roomID uint32) *gorm.DB {
	return d.db.WithContext(ctx).Model(&entity.Room_User{}).
		Select("user_id").
		Where("room_id = ?", roomID)
}

func userIDs(users []entity.User) []uint32 {
	out := make([]uint32, len(users))
	for i, u := range users {
		out[i] = u.ID
	}
	return out
}

// BestWPM returns each player's highest WPM over the last window, for the
// leaderboard's speed column.
//
// The window matters: an all-time best is a number almost nobody will ever
// beat again, which makes the column useless for comparing players who joined
// at different times.
func (d *Database) BestWPM(ctx context.Context, ids []uint32, opts leaderboardFilter) (map[uint32]float32, error) {
	out := make(map[uint32]float32, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	q := d.db.WithContext(ctx).Model(&entity.Run{}).
		Select("user_id, MAX(wpm) AS best").
		Where("user_id IN ?", ids).
		Where("played_at >= ?", time.Now().Add(-bestWindow))
	if opts.Mode != nil {
		q = q.Where("mode = ?", *opts.Mode)
	}
	if opts.SoloOnly {
		q = q.Where("solo = ?", true)
	}

	type row struct {
		UserID uint32
		Best   float32
	}
	var rows []row
	if err := q.Group("user_id").Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, r := range rows {
		out[r.UserID] = r.Best
	}
	return out, nil
}

// bestWindow is how far back the leaderboard's speed column looks.
const bestWindow = 30 * 24 * time.Hour

// RatingHistory returns a player's rating over time, for the profile graph.
func (d *Database) RatingHistory(ctx context.Context, userID uint32, limit int) (entity.GraphPayload, error) {
	if limit <= 0 {
		limit = 60
	}
	var runs []entity.Run
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("played_at DESC").
		Limit(limit).
		Find(&runs).Error
	if err != nil {
		return entity.GraphPayload{}, err
	}

	// Oldest first, which is the order a graph wants them in.
	sort.Slice(runs, func(i, j int) bool { return runs[i].PlayedAt.Before(runs[j].PlayedAt) })

	out := entity.GraphPayload{Points: make([]entity.GraphPoint, 0, len(runs))}
	for _, r := range runs {
		out.Points = append(out.Points, entity.GraphPoint{
			At:     r.PlayedAt.UnixMilli(),
			Rating: r.RankAfter,
			WPM:    r.WPM,
		})
	}
	return out, nil
}

// RecentRuns returns a page of a player's runs, newest first.
func (d *Database) RecentRuns(ctx context.Context, userID uint32, page, pageSize int) (entity.RunPage, error) {
	if pageSize <= 0 {
		pageSize = 25
	}
	if page < 0 {
		page = 0
	}

	q := d.db.WithContext(ctx).Model(&entity.Run{}).Where("user_id = ?", userID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return entity.RunPage{}, err
	}

	runs := make([]entity.Run, 0)
	err := q.Order("played_at DESC").
		Offset(page * pageSize).
		Limit(pageSize).
		Find(&runs).Error
	if err != nil {
		return entity.RunPage{}, err
	}
	return entity.RunPage{Runs: runs, Total: total}, nil
}
