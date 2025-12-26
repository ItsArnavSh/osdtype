package postgresql

import (
	"context"
	"osdtyp/app/entity"
)

func (d *Database) AddUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Create(&user).Error
}
func (d *Database) GetUser(userid uint32) (entity.User, error) {
	var userData entity.User
	result := d.db.Where("id = ?", userid).First(&userData)
	if result.Error != nil {
		return entity.User{}, result.Error
	}
	return userData, nil
}
func (d *Database) GetUserFromName(ctx context.Context, username string) (entity.User, error) {
	var userData entity.User
	result := d.db.WithContext(ctx).Where("username = ?", username).First(&userData)
	if result.Error != nil {
		return entity.User{}, result.Error
	}
	return userData, nil
}

func (d *Database) UserExists(ctx context.Context, username string) (bool, error) {
	var count int64
	result := d.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("username = ?", username).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func (d *Database) ChangeRank(userid uint32, rank uint16) error {
	result := d.db.
		Model(&entity.User{}).
		Where("id = ?", userid).
		Update("rank", rank)
	return result.Error
}

func (d *Database) GetRank(ctx context.Context, userid uint32) (uint16, error) {
	var user entity.User
	err := d.db.WithContext(ctx).Where("id= ?", userid).First(&user).Error
	if err != nil {
		return 0, err
	}
	return user.CurrentRank, nil
}
