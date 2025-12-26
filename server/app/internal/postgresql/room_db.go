package postgresql

import (
	"context"
	"osdtyp/app/entity"
)

func (d *Database) CreateRoom(ctx context.Context, room entity.Room) error {
	return d.db.WithContext(ctx).Create(&room).Error
}

func (d *Database) AddMember(ctx context.Context, room_user entity.Room_User) error {
	return d.db.WithContext(ctx).Create(&room_user).Error
}
func (d *Database) UpdateMembership(ctx context.Context, room_user entity.Room_User) error {
	return d.db.WithContext(ctx).
		Model(&entity.Room_User{}).
		Where("room_id=? AND user_id=?", room_user.RoomID, room_user.UserID).
		Update("perm", room_user.Perm).
		Error
}
func (d *Database) SeePerms(ctx context.Context, room_user entity.Room_User) (entity.Room_User, error) {
	result := d.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", room_user.RoomID, room_user.UserID).
		First(&room_user)
	if result.Error != nil {
		return entity.Room_User{}, result.Error
	}
	return room_user, nil
}

func (d *Database) PageList(ctx context.Context, user_id uint32, index, limit uint8) ([]entity.Room, error) {

	var rooms []entity.Room

	offset := int(index) * int(limit)

	err := d.db.WithContext(ctx).
		Where("user_id = ?", user_id).
		Order("id DESC").
		Offset(offset).
		Limit(int(limit)).
		Find(&rooms).Error

	if err != nil {
		return nil, err
	}

	return rooms, nil
}
