package postgresql

import (
	"context"
	"fmt"
	"strings"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

// eloColumns maps a ranked duration to its whitelisted database columns.
func eloColumns(duration int) (rating, games string, ok bool) {
	switch duration {
	case entity.RankedDuration30:
		return "rating_30", "games_played_30", true
	case entity.RankedDuration90:
		return "rating_90", "games_played_90", true
	case entity.RankedDuration300:
		return "rating_300", "games_played_300", true
	}
	return "", "", false
}

func (d *Database) AddUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Create(&user).Error
}

func (d *Database) SaveUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Save(&user).Error
}

// UpdateUserFields applies account-setting changes without rewriting cached
// ratings or game statistics from a stale user struct.
func (d *Database) UpdateUserFields(ctx context.Context, uid uint32, fields map[string]any) error {
	return d.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", uid).Updates(fields).Error
}

func (d *Database) GetUser(userid uint32) (entity.User, error) {
	var user entity.User
	if err := d.db.First(&user, "id = ?", userid).Error; err != nil {
		return entity.User{}, err
	}
	return user, nil
}

func (d *Database) GetUserFromName(ctx context.Context, username string) (entity.User, error) {
	var user entity.User
	if err := d.db.WithContext(ctx).Where("LOWER(username) = ?", strings.ToLower(username)).First(&user).Error; err != nil {
		return entity.User{}, err
	}
	return user, nil
}

func (d *Database) UserExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&entity.User{}).
		Where("LOWER(username) = ?", strings.ToLower(username)).Count(&count).Error
	return count > 0, err
}

func (d *Database) SearchPeople(ctx context.Context, txt string, limit int) ([]entity.User, error) {
	var users []entity.User
	err := d.db.WithContext(ctx).
		Where("LOWER(username) LIKE ? AND is_bot = ?", "%"+strings.ToLower(txt)+"%", false).
		Order("rating DESC, username ASC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (d *Database) ListUsersByIDs(ctx context.Context, ids []uint32) ([]entity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []entity.User
	err := d.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (d *Database) Leaderboard(ctx context.Context, duration, offset, limit int) ([]entity.User, error) {
	ratingCol, gamesCol, ok := eloColumns(duration)
	if !ok {
		return nil, fmt.Errorf("invalid ranked duration %d", duration)
	}
	var users []entity.User
	err := d.db.WithContext(ctx).
		Where(fmt.Sprintf("is_bot = ? AND %s > 0", gamesCol), false).
		Order(fmt.Sprintf("%s DESC, %s DESC, username ASC, id ASC", ratingCol, gamesCol)).
		Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// SearchLeaderboard returns ranked players whose username matches the query.
// Callers attach the exact ladder position so search results are useful even
// when they fall outside the top five.
func (d *Database) SearchLeaderboard(ctx context.Context, duration int, query string, limit int) ([]entity.User, error) {
	ratingCol, gamesCol, ok := eloColumns(duration)
	if !ok {
		return nil, fmt.Errorf("invalid ranked duration %d", duration)
	}
	var users []entity.User
	err := d.db.WithContext(ctx).
		Where(fmt.Sprintf("is_bot = ? AND %s > 0 AND LOWER(username) LIKE ?", gamesCol),
			false, "%"+strings.ToLower(strings.TrimSpace(query))+"%").
		Order(fmt.Sprintf("%s DESC, %s DESC, username ASC, id ASC", ratingCol, gamesCol)).
		Limit(limit).Find(&users).Error
	return users, err
}

func (d *Database) UpdateRating(ctx context.Context, uid uint32, rating int) error {
	user, err := d.GetUser(uid)
	if err != nil {
		return err
	}
	user.Rating = rating
	if rating > user.PeakRating {
		user.PeakRating = rating
	}
	return d.SaveUser(ctx, user)
}

func (d *Database) RecordGameStats(ctx context.Context, uid uint32, won bool) error {
	updates := map[string]any{"games_played": gorm.Expr("games_played + 1")}
	if won {
		updates["wins"] = gorm.Expr("wins + 1")
	}
	return d.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", uid).Updates(updates).Error
}

// RankedPosition returns the player's 1-based position on one clock's ladder
// using the same deterministic order as Leaderboard. Unranked players (bots
// and people without a completed game on that clock) return 0.
func (d *Database) RankedPosition(ctx context.Context, user entity.User, duration int) (int64, error) {
	ratingCol, gamesCol, ok := eloColumns(duration)
	if !ok {
		return 0, fmt.Errorf("invalid ranked duration %d", duration)
	}
	if user.IsBot || user.GamesFor(duration) <= 0 {
		return 0, nil
	}
	rating := user.RatingFor(duration)
	games := user.GamesFor(duration)
	var before int64
	query := fmt.Sprintf(`is_bot = ? AND %s > 0 AND (
			%s > ? OR
			(%s = ? AND %s > ?) OR
			(%s = ? AND %s = ? AND username < ?) OR
			(%s = ? AND %s = ? AND username = ? AND id < ?)
		)`, gamesCol, ratingCol, ratingCol, gamesCol, ratingCol, gamesCol, ratingCol, gamesCol)
	err := d.db.WithContext(ctx).Model(&entity.User{}).
		Where(query,
			false,
			rating,
			rating, games,
			rating, games, user.Username,
			rating, games, user.Username, user.ID,
		).Count(&before).Error
	if err != nil {
		return 0, err
	}
	return before + 1, nil
}

func (d *Database) GetBotForRating(ctx context.Context, rating, duration int) (entity.User, error) {
	var bots []entity.User
	if err := d.db.WithContext(ctx).Where("is_bot = ?", true).Find(&bots).Error; err != nil {
		return entity.User{}, err
	}
	if len(bots) == 0 {
		return entity.User{}, gorm.ErrRecordNotFound
	}
	best := bots[0]
	bestDiff := best.RatingFor(duration) - rating
	if bestDiff < 0 {
		bestDiff = -bestDiff
	}
	for _, b := range bots[1:] {
		diff := b.RatingFor(duration) - rating
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			best, bestDiff = b, diff
		}
	}
	return best, nil
}
