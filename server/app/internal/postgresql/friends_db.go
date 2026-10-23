package postgresql

import (
	"context"
	"errors"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Database) setRelation(ctx context.Context, tx *gorm.DB, a, b uint32, rel entity.Relation) error {
	row := entity.Friends{A: a, B: b, Relation: rel, CreatedAt: tx.NowFunc()}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "a_id"}, {Name: "b_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"relation"}),
	}).Create(&row).Error
}

func (d *Database) FollowUser(ctx context.Context, id, otherid uint32) error {
	if id == otherid {
		return errors.New("cannot follow yourself")
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var direct entity.Friends
		directErr := tx.Where("a_id = ? AND b_id = ?", id, otherid).First(&direct).Error
		if directErr == nil {
			if direct.Relation == entity.FOLLOWS {
				var inverse entity.Friends
				if err := tx.Where("a_id = ? AND b_id = ?", otherid, id).First(&inverse).Error; err == nil && inverse.Relation == entity.FOLLOWS {
					if err := d.setRelation(ctx, tx, id, otherid, entity.FRIENDS); err != nil {
						return err
					}
					return d.setRelation(ctx, tx, otherid, id, entity.FRIENDS)
				}
			}
			return nil
		}
		if !errors.Is(directErr, gorm.ErrRecordNotFound) {
			return directErr
		}

		var inverse entity.Friends
		inverseErr := tx.Where("a_id = ? AND b_id = ?", otherid, id).First(&inverse).Error
		if inverseErr == nil && inverse.Relation == entity.FOLLOWS {
			if err := d.setRelation(ctx, tx, id, otherid, entity.FRIENDS); err != nil {
				return err
			}
			return d.setRelation(ctx, tx, otherid, id, entity.FRIENDS)
		}
		if inverseErr != nil && !errors.Is(inverseErr, gorm.ErrRecordNotFound) {
			return inverseErr
		}
		return d.setRelation(ctx, tx, id, otherid, entity.FOLLOWS)
	})
}

func (d *Database) UnfollowUser(ctx context.Context, id, otherid uint32) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var direct entity.Friends
		err := tx.Where("a_id = ? AND b_id = ?", id, otherid).First(&direct).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.Where("a_id = ? AND b_id = ?", id, otherid).Delete(&entity.Friends{}).Error; err != nil {
			return err
		}
		if direct.Relation == entity.FRIENDS {
			var inverse entity.Friends
			if err := tx.Where("a_id = ? AND b_id = ?", otherid, id).First(&inverse).Error; err == nil {
				return tx.Model(&entity.Friends{}).Where("a_id = ? AND b_id = ?", otherid, id).
					Update("relation", entity.FOLLOWS).Error
			}
		}
		return nil
	})
}

func (d *Database) GetFriends(ctx context.Context, id uint32) ([]uint32, error) {
	var rows []entity.Friends
	if err := d.db.WithContext(ctx).
		Where("(a_id = ? OR b_id = ?) AND relation = ?", id, id, entity.FRIENDS).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint32]struct{})
	out := make([]uint32, 0, len(rows))
	for _, row := range rows {
		other := row.B
		if row.B == id {
			other = row.A
		}
		if _, ok := seen[other]; ok {
			continue
		}
		seen[other] = struct{}{}
		out = append(out, other)
	}
	return out, nil
}

func (d *Database) GetFollowing(ctx context.Context, id uint32) ([]uint32, error) {
	var rows []entity.Friends
	if err := d.db.WithContext(ctx).Where("a_id = ?", id).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uint32, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.B)
	}
	return out, nil
}

func (d *Database) IsFriend(ctx context.Context, a, b uint32) (bool, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.Friends{}).
		Where("a_id = ? AND b_id = ? AND relation = ?", a, b, entity.FRIENDS).Count(&n).Error
	return n > 0, err
}

func (d *Database) IsFollowing(ctx context.Context, a, b uint32) (bool, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.Friends{}).
		Where("a_id = ? AND b_id = ?", a, b).Count(&n).Error
	return n > 0, err
}
