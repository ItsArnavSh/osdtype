package postgresql

import (
	"osdtyp/app/entity"

	"gorm.io/gorm"
)

func (d *Database) NewTask(task entity.Task) error {
	return d.db.Create(&task).Error
}

func (d *Database) PopRecentTask() (entity.Task, error) {
	var task entity.Task

	err := d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Order("time ASC").
			First(&task).Error; err != nil {
			return err
		}

		if err := tx.Delete(&task).Error; err != nil {
			return err
		}

		return nil
	})

	return task, err
}
