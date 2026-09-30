package postgresql

import (
	"context"
	"errors"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

func (d *Database) NewTask(task entity.Task) error {
	return d.db.Create(&task).Error
}

// PeekRecentTask returns the earliest scheduled task without removing it.
//
// The scheduler used to pop and delete, then wait: any nudge during that wait
// threw the popped task away and it never ran. Reading first and deleting only
// once the task has actually been handled keeps it durable across nudges.
func (d *Database) PeekRecentTask(ctx context.Context) (entity.Task, error) {
	var task entity.Task
	err := d.db.WithContext(ctx).Order("time ASC").First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Task{}, nil
		}
		return entity.Task{}, err
	}
	return task, nil
}

// DeleteTask removes a task once it has been handled.
func (d *Database) DeleteTask(ctx context.Context, jobID uint32) error {
	return d.db.WithContext(ctx).Where("job_id = ?", jobID).Delete(&entity.Task{}).Error
}

// PopRecentTask removes and returns the earliest task. It is kept for callers
// that genuinely want to claim a task; the scheduler uses PeekRecentTask and
// DeleteTask instead so an interrupted wait cannot lose the task.
func (d *Database) PopRecentTask() (entity.Task, error) {
	var task entity.Task

	err := d.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Order("time ASC").First(&task).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				task = entity.Task{} // zero value
				return nil
			}
			return err
		}

		return tx.Delete(&task).Error
	})

	if err != nil {
		return entity.Task{}, err
	}

	return task, nil
}
