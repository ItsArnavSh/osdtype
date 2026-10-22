package postgresql

import (
	"context"
	"strings"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

func (d *Database) AddUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Create(&user).Error
}

func (d *Database) SaveUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Save(&user).Error
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

func (d *Database) Leaderboard(ctx context.Context, offset, limit int) ([]entity.User, error) {
	var users []entity.User
	err := d.db.WithContext(ctx).
		Where("is_bot = ? AND games_played > 0", false).
		Order("rating DESC, games_played DESC, username ASC").
		Offset(offset).Limit(limit).Find(&users).Error
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

func (d *Database) CountRatingAbove(ctx context.Context, rating int) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ? AND games_played > 0 AND rating > ?", false, rating).Count(&n).Error
	return n, err
}

func (d *Database) GetBotForRating(ctx context.Context, rating int) (entity.User, error) {
	var bots []entity.User
	if err := d.db.WithContext(ctx).Where("is_bot = ?", true).Find(&bots).Error; err != nil {
		return entity.User{}, err
	}
	if len(bots) == 0 {
		return entity.User{}, gorm.ErrRecordNotFound
	}
	best := bots[0]
	bestDiff := best.Rating - rating
	if bestDiff < 0 {
		bestDiff = -bestDiff
	}
	for _, b := range bots[1:] {
		diff := b.Rating - rating
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			best, bestDiff = b, diff
		}
	}
	return best, nil
}
