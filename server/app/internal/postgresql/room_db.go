package postgresql

import (
	"context"
	"errors"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Database) CreateRoom(ctx context.Context, room *entity.Room) error {
	return d.db.WithContext(ctx).Create(room).Error
}

func (d *Database) GetRoomByCode(ctx context.Context, code string) (entity.Room, error) {
	var room entity.Room
	if err := d.db.WithContext(ctx).Where("code = ?", code).First(&room).Error; err != nil {
		return entity.Room{}, err
	}
	return room, nil
}

func (d *Database) GetRoomByID(ctx context.Context, id uint32) (entity.Room, error) {
	var room entity.Room
	if err := d.db.WithContext(ctx).First(&room, "id = ?", id).Error; err != nil {
		return entity.Room{}, err
	}
	return room, nil
}

func (d *Database) SaveRoom(ctx context.Context, room *entity.Room) error {
	return d.db.WithContext(ctx).Save(room).Error
}

func (d *Database) UpdateRoomFields(ctx context.Context, id uint32, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC()
	return d.db.WithContext(ctx).Model(&entity.Room{}).Where("id = ?", id).Updates(fields).Error
}

func (d *Database) ListPublicRooms(ctx context.Context, offset, limit int) ([]entity.Room, error) {
	var rooms []entity.Room
	err := d.db.WithContext(ctx).
		Where("is_public = ? AND status <> ?", true, entity.RoomClosed).
		Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&rooms).Error
	return rooms, err
}

func (d *Database) ListRoomsForUser(ctx context.Context, uid uint32, offset, limit int) ([]entity.Room, error) {
	var rooms []entity.Room
	err := d.db.WithContext(ctx).Table("rooms").
		Joins("JOIN room_members ON room_members.room_id = rooms.id").
		Where("room_members.user_id = ? AND rooms.status <> ?", uid, entity.RoomClosed).
		Order("rooms.updated_at DESC").Offset(offset).Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

func (d *Database) AddMember(ctx context.Context, member *entity.RoomMember) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(member).Error
}

func (d *Database) GetMember(ctx context.Context, roomID, uid uint32) (entity.RoomMember, error) {
	var member entity.RoomMember
	if err := d.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, uid).First(&member).Error; err != nil {
		return entity.RoomMember{}, err
	}
	return member, nil
}

func (d *Database) ListMembers(ctx context.Context, roomID uint32) ([]entity.RoomMember, error) {
	var members []entity.RoomMember
	err := d.db.WithContext(ctx).Preload("User").Where("room_id = ?", roomID).Order("joined_at ASC").Find(&members).Error
	return members, err
}

func (d *Database) CountMembers(ctx context.Context, roomID uint32) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.RoomMember{}).Where("room_id = ?", roomID).Count(&n).Error
	return n, err
}

func (d *Database) UpdateMemberRole(ctx context.Context, roomID, uid uint32, role entity.RoomRole) error {
	return d.db.WithContext(ctx).Model(&entity.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, uid).Update("role", role).Error
}

func (d *Database) RemoveMember(ctx context.Context, roomID, uid uint32) error {
	return d.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, uid).Delete(&entity.RoomMember{}).Error
}

func (d *Database) AddBan(ctx context.Context, ban *entity.RoomBan) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(ban).Error
}

func (d *Database) RemoveBan(ctx context.Context, roomID, uid uint32) error {
	return d.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, uid).Delete(&entity.RoomBan{}).Error
}

func (d *Database) IsBanned(ctx context.Context, roomID, uid uint32) (bool, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.RoomBan{}).Where("room_id = ? AND user_id = ?", roomID, uid).Count(&n).Error
	return n > 0, err
}

func (d *Database) CreateInvite(ctx context.Context, invite *entity.RoomInvite) error {
	return d.db.WithContext(ctx).Create(invite).Error
}

func (d *Database) GetInvite(ctx context.Context, id uint32) (entity.RoomInvite, error) {
	var inv entity.RoomInvite
	err := d.db.WithContext(ctx).Preload("Room").Preload("Inviter").First(&inv, "id = ?", id).Error
	return inv, err
}

func (d *Database) PendingInvites(ctx context.Context, uid uint32) ([]entity.RoomInvite, error) {
	var invites []entity.RoomInvite
	err := d.db.WithContext(ctx).Preload("Room").Preload("Inviter").
		Where("invitee_id = ? AND status = ? AND expires_at > ?", uid, entity.InvitePending, time.Now().UTC()).
		Order("created_at DESC").Find(&invites).Error
	return invites, err
}

func (d *Database) PendingInviteFor(ctx context.Context, roomID, uid uint32) (entity.RoomInvite, error) {
	var inv entity.RoomInvite
	err := d.db.WithContext(ctx).Where("room_id = ? AND invitee_id = ? AND status = ?", roomID, uid, entity.InvitePending).
		Order("created_at DESC").First(&inv).Error
	return inv, err
}

func (d *Database) UpdateInviteStatus(ctx context.Context, id uint32, status entity.InviteStatus) error {
	return d.db.WithContext(ctx).Model(&entity.RoomInvite{}).Where("id = ?", id).Update("status", status).Error
}

func (d *Database) TransferHost(ctx context.Context, roomID, uid uint32) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Room{}).Where("id = ?", roomID).Update("host_id", uid).Error; err != nil {
			return err
		}
		return tx.Model(&entity.RoomMember{}).Where("room_id = ? AND user_id = ?", roomID, uid).
			Update("role", entity.RoleOwner).Error
	})
}

func (d *Database) CloseRoom(ctx context.Context, roomID uint32) error {
	return d.db.WithContext(ctx).Model(&entity.Room{}).Where("id = ?", roomID).
		Updates(map[string]any{"status": entity.RoomClosed, "game_id": 0}).Error
}

func (d *Database) DeleteMemberIfExists(ctx context.Context, roomID, uid uint32) error {
	err := d.RemoveMember(ctx, roomID, uid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func (d *Database) ActiveRoomForUser(ctx context.Context, uid uint32) (entity.Room, bool, error) {
	var room entity.Room
	err := d.db.WithContext(ctx).Table("rooms").
		Joins("JOIN room_members ON room_members.room_id = rooms.id").
		Where("room_members.user_id = ? AND rooms.status <> ?", uid, entity.RoomClosed).
		Order("rooms.updated_at DESC").First(&room).Error
	if err == gorm.ErrRecordNotFound {
		return entity.Room{}, false, nil
	}
	if err != nil {
		return entity.Room{}, false, err
	}
	return room, true, nil
}
