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
