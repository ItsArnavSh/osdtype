package postgresql

import (
	"context"
	"osdtyp/app/entity"
)

func (d *Database) AddUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Create(&user).Error
}
func (d *Database) GetUser(ctx context.Context, username string) (entity.User, error) {
	var userData entity.User
	result := d.db.WithContext(ctx).Where("username = ?", username).First(&userData)
	if result.Error != nil {
		return entity.User{}, result.Error
	}
	return userData, nil
}
func (d *Database) UserExists(ctx context.Context, username string) (bool, error) {
	result := d.db.WithContext(ctx).
		Model(&entity.User{}).
		Select("1").
		Where("username = ?", username).
		Limit(1).
		Find(nil)

	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
func (d *Database) ChangeRank(ctx context.Context, userid uint64, rank uint16) error {
	result := d.db.WithContext(ctx).Update("rank", rank).Where("id =?", userid)
	return result.Error
}
func (d *Database) GetRank(ctx context.Context, userid uint64) (uint16, error) {
	var user entity.User
	err := d.db.WithContext(ctx).Where("id= ?", userid).First(&user).Error
	if err != nil {
		return 0, err
	}
	return user.CurrentRank, nil
}
