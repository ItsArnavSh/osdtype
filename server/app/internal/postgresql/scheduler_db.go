package postgresql

import (
	"errors"
	"osdtyp/app/entity"

	"gorm.io/gorm"
)

func (d *Database) NewTask(task entity.Task) error {
	return d.db.Create(&task).Error
}

func (d *Database) PopRecentTask() (entity.Task, error) {
	var task entity.Task

	err := d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Order("time ASC").First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				task = entity.Task{} // ensure zero value
				return nil
			}
			return err
		}

		if err := tx.Delete(&task).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return entity.Task{}, err // real database error
	}

	// If no task was found, task will be zero value
	// You can distinguish this by checking task fields (e.g. ID == 0)
	return task, nil
}
