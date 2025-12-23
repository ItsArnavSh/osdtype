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
	return d.db.WithContext(ctx).Update("perm", room_user.Perm).Where("room_id=? and user_id=?", room_user.RoomID, room_user.UserID).Error
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
func (d *Database) RemovePlayer(ctx context.Context, room_user entity.Room_User) error {
	result := d.db.Where("room_id=? AND user_id=?", room_user.RoomID, room_user.UserID).Delete(ctx)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
