package postgresql

import (
	"osdtyp/app/entity"
)

func (d *Database) NewTask(task entity.Task) error {
	return d.db.Create(&task).Error
}
func (d *Database) GetRecentTask() (entity.Task, error) {
	var task entity.Task
	err := d.db.Order("time ASC").First(&task).Error
	return task, err
}
