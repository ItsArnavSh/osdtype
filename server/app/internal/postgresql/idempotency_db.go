package postgresql

import (
	"context"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm/clause"
)

func (d *Database) GetIdempotency(ctx context.Context, key string) (entity.IdempotencyKey, error) {
	var row entity.IdempotencyKey
	err := d.db.WithContext(ctx).Where("key = ? AND created_at > ?", key, time.Now().Add(-24*time.Hour)).First(&row).Error
	return row, err
}

func (d *Database) SaveIdempotency(ctx context.Context, row *entity.IdempotencyKey) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
}

func (d *Database) PurgeIdempotency(ctx context.Context, before time.Time) error {
	return d.db.WithContext(ctx).Where("created_at < ?", before).Delete(&entity.IdempotencyKey{}).Error
}
