package postgresql

import (
	"context"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Database) CreateContest(ctx context.Context, contest *entity.Contest) error {
	return d.db.WithContext(ctx).Create(contest).Error
}

func (d *Database) SaveContest(ctx context.Context, contest *entity.Contest) error {
	return d.db.WithContext(ctx).Save(contest).Error
}

func (d *Database) GetContest(ctx context.Context, id uint32) (entity.Contest, error) {
	var contest entity.Contest
	if err := d.db.WithContext(ctx).First(&contest, "id = ?", id).Error; err != nil {
		return entity.Contest{}, err
	}
	return contest, nil
}

func (d *Database) ListContests(ctx context.Context, roomID uint32, offset, limit int) ([]entity.Contest, error) {
	var contests []entity.Contest
	err := d.db.WithContext(ctx).Where("room_id = ?", roomID).
		Order("starts_at DESC").Offset(offset).Limit(limit).Find(&contests).Error
	return contests, err
}

func (d *Database) CreateTask(ctx context.Context, task *entity.Task) error {
	return d.db.WithContext(ctx).Create(task).Error
}

// PopDueTask claims the next due task. The row is marked completed inside a
// transaction so two scheduler instances cannot run the same job.
func (d *Database) PopDueTask(ctx context.Context, now time.Time) (entity.Task, error) {
	var task entity.Task
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("completed = ? AND time <= ?", false, now).
			Order("time ASC").Limit(1).Find(&task)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&entity.Task{}).Where("id = ?", task.ID).Update("completed", true).Error
	})
	return task, err
}

func (d *Database) NextTaskTime(ctx context.Context) (time.Time, error) {
	var task entity.Task
	res := d.db.WithContext(ctx).Where("completed = ?", false).Order("time ASC").Limit(1).Find(&task)
	if res.Error != nil {
		return time.Time{}, res.Error
	}
	if res.RowsAffected == 0 {
		return time.Time{}, nil
	}
	return task.Time, nil
}
