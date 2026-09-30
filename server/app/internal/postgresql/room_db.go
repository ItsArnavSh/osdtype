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

// PageList returns one page of the rooms a user belongs to.
//
// The membership lives in room_users, not on the rooms row: the previous
// version filtered the rooms table on a "user_id" column that does not exist,
// so the query always failed and the room list could never return anything.
func (d *Database) PageList(ctx context.Context, userID uint32, index, limit int) ([]entity.Room, error) {
	if limit <= 0 {
		limit = 10
	}
	if index < 0 {
		index = 0
	}

	rooms := make([]entity.Room, 0)

	err := d.db.WithContext(ctx).
		Model(&entity.Room{}).
		Joins("JOIN room_users ON room_users.room_id = rooms.id").
		Where("room_users.user_id = ?", userID).
		// A blocked or left membership is not an active membership.
		Where("room_users.perm NOT IN ?", []entity.RoomPerm{entity.BLOCKED, entity.LEFT}).
		Order("rooms.id DESC").
		Offset(index * limit).
		Limit(limit).
		Find(&rooms).Error

	if err != nil {
		return nil, err
	}
	return rooms, nil
}
