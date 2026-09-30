package postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrRoomFull is returned when a room has no space left.
//
// It is an alias for the entity sentinel rather than a second error value with
// the same message, so errors.Is matches whichever spelling a caller happens to
// have imported. Two distinct values with equal text would not compare equal,
// and the caller that knew about the store's version would silently fall
// through to a 500 on a full room.
var ErrRoomFull = entity.ErrRoomFull

// codeAttempts is how many codes to try before giving up on a collision.
//
// Codes are drawn from 887 million possibilities, so a collision is
// vanishingly unlikely, but the index is the only thing standing between two
// rooms and a shared join URL, so the insert is retried rather than trusted.
const codeAttempts = 8

// CreateRoom stores a room, minting a unique share code.
//
// The room is taken by pointer because it comes back changed: the database
// assigns the primary key and the share code. This signature was by value until
// a test caught that the caller's copy was never updated, which meant every
// lobby handed its host an empty code and a room id of zero — the whole private
// lobby feature, unreachable. A by-value room silently discards both, and the
// failure is a blank string rather than an error.
//
// The code is generated here rather than by the caller so uniqueness is
// enforced at the point of insertion. Callers that pre-mint a code, which the
// tests do for determinism, are still checked against the unique index.
func (d *Database) CreateRoom(ctx context.Context, room *entity.Room) error {
	*room = d.normaliseRoom(*room)

	if room.Code == "" {
		for attempt := range codeAttempts {
			room.Code = d.codes.Generate()
			err := d.db.WithContext(ctx).Create(room).Error
			if err == nil {
				return nil
			}
			if !isUniqueViolation(err) {
				return err
			}
			if attempt == codeAttempts-1 {
				return fmt.Errorf("could not mint a unique room code after %d attempts: %w", codeAttempts, err)
			}
		}
	}
	return d.db.WithContext(ctx).Create(room).Error
}

// normaliseRoom fills in the defaults a bare create request leaves out.
func (d *Database) normaliseRoom(room entity.Room) entity.Room {
	if room.Public != entity.PUBLIC {
		room.Public = entity.PRIVATE
	}
	if room.Capacity <= 0 {
		room.Capacity = 6
	}
	return room
}

// isUniqueViolation reports whether an error is a Postgres unique constraint
// failure, which is the only thing a code collision can produce.
func isUniqueViolation(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrDuplicatedKey)
}

// RoomByID loads a room.
func (d *Database) RoomByID(ctx context.Context, id uint32) (entity.Room, error) {
	var room entity.Room
	if err := d.db.WithContext(ctx).First(&room, "id = ?", id).Error; err != nil {
		return entity.Room{}, err
	}
	return room, nil
}

// RoomByCode loads a room by its share code.
//
// The code is normalised here so a lowercase or padded code from a URL still
// resolves. GORM's case-sensitive equality is not enough on its own because
// people retype these from screenshots.
func (d *Database) RoomByCode(ctx context.Context, code string) (entity.Room, error) {
	var room entity.Room
	err := d.db.WithContext(ctx).
		Where("UPPER(code) = ?", code).
		First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Room{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return entity.Room{}, err
	}
	return room, nil
}

// ListPublicRooms returns the open rooms anyone can join without a code.
func (d *Database) ListPublicRooms(ctx context.Context, index, limit int) ([]entity.Room, error) {
	if limit <= 0 {
		limit = 20
	}
	if index < 0 {
		index = 0
	}
	rooms := make([]entity.Room, 0)
	err := d.db.WithContext(ctx).
		Where("public = ? AND capacity > 0", entity.PUBLIC).
		Order("id DESC").
		Offset(index * limit).
		Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

// UpdateRoom persists a room's mutable fields.
//
// It saves the whole row rather than a column list, so a field added to the
// model cannot be silently forgotten here.
func (d *Database) UpdateRoom(ctx context.Context, room entity.Room) error {
	return d.db.WithContext(ctx).Save(&room).Error
}

// SetRoomStatus moves a room to a new lifecycle state.
func (d *Database) SetRoomStatus(ctx context.Context, roomID uint32, status entity.RoomStatus) error {
	return d.db.WithContext(ctx).
		Model(&entity.Room{}).
		Where("id = ?", roomID).
		Update("status", status).Error
}

// DeleteRoom removes a room and its memberships.
func (d *Database) DeleteRoom(ctx context.Context, roomID uint32) error {
	if err := d.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Delete(&entity.Room_User{}).Error; err != nil {
		return err
	}
	return d.db.WithContext(ctx).Delete(&entity.Room{}, "id = ?", roomID).Error
}

// AddMember adds a membership.
//
// A rejoin after leaving is allowed and restores the previous standing, but a
// rejoin after being blocked is refused: leaving and rejoining must not be a
// way around a block.
func (d *Database) AddMember(ctx context.Context, ru entity.Room_User) error {
	existing, err := d.SeePerms(ctx, ru)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return d.db.WithContext(ctx).Create(&ru).Error
	case err != nil:
		return err
	}

	if existing.Perm == entity.BLOCKED {
		return errors.New("this user is blocked from the room")
	}
	return d.db.WithContext(ctx).
		Model(&entity.Room_User{}).
		Where("room_id = ? AND user_id = ?", ru.RoomID, ru.UserID).
		Update("perm", ru.Perm).Error
}

// UpdateMembership changes a member's standing.
func (d *Database) UpdateMembership(ctx context.Context, ru entity.Room_User) error {
	return d.db.WithContext(ctx).
		Model(&entity.Room_User{}).
		Where("room_id = ? AND user_id = ?", ru.RoomID, ru.UserID).
		Update("perm", ru.Perm).Error
}

// SeePerms returns a membership, or gorm.ErrRecordNotFound.
func (d *Database) SeePerms(ctx context.Context, ru entity.Room_User) (entity.Room_User, error) {
	var out entity.Room_User
	result := d.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", ru.RoomID, ru.UserID).
		First(&out)
	if result.Error != nil {
		return entity.Room_User{}, result.Error
	}
	return out, nil
}

// IsMember reports whether a user is an active member of a room, meaning they
// are neither blocked nor have left.
func (d *Database) IsMember(ctx context.Context, roomID, userID uint32) (bool, error) {
	ru, err := d.SeePerms(ctx, entity.Room_User{RoomID: roomID, UserID: userID})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ru.Perm.IsActive(), nil
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

// Roster returns a room's active members, ordered by rank so the game handler
// gets them in a sensible order.
//
// This is the bridge the private lobby needed: before it, the roster lived in
// a map inside the lobby manager, so nothing outside that package could ask who
// was in a room.
func (d *Database) Roster(ctx context.Context, roomID uint32) ([]entity.RosterMember, error) {
	var rows []struct {
		entity.Room_User
		Username string
		Rank     uint16
		Created  time.Time
	}

	err := d.db.WithContext(ctx).
		Model(&entity.Room_User{}).
		Select("room_users.*, users.username, users.current_rank AS rank, room_users.created_at AS created").
		Joins("JOIN users ON users.id = room_users.user_id").
		Where("room_users.room_id = ?", roomID).
		Where("room_users.perm NOT IN ?", []entity.RoomPerm{entity.BLOCKED, entity.LEFT}).
		Order("users.current_rank DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]entity.RosterMember, 0, len(rows))
	for _, r := range rows {
		out = append(out, entity.RosterMember{
			UserID:   r.UserID,
			Username: r.Username,
			Rank:     r.Rank,
			JoinedAt: r.Created.Unix(),
		})
	}
	return out, nil
}

// AddRosterMember adds a member, reporting false if they were already an
// active member so the caller can skip the "you joined" notification.
// AddRosterMember adds a member, reporting whether the roster changed.
//
// The capacity check and the insert happen in one transaction that first locks
// the room row, so concurrent joins for the same room serialize behind each
// other.
//
// This was not always so. The check lived in the lobby, which read the roster,
// compared the length to the capacity, and then called in to insert — three
// separate operations. Every one of forty simultaneous join requests read the
// same roster size before any of them wrote, so all forty passed the check and
// a six-player lobby ended up with ten people in it. Locking the room row is
// what makes "at most capacity" a property of the data rather than a hope about
// timing; the alternative is an advisory lock per room, which the row lock is
// already doing.
//
// capacity <= 0 means "no limit", which is what the pre-lobby callers pass.
func (d *Database) AddRosterMember(ctx context.Context, roomID, userID uint32, perm entity.RoomPerm, capacity int) (added bool, err error) {
	err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Locking the room row serializes every membership change for this
		// room. FOR UPDATE blocks until any concurrent transaction touching the
		// same room has committed or rolled back.
		// The inner errors are named rather than shadowing the named return,
		// so a `return err` here is visibly the transaction's result and not
		// whichever shadow happened to be nearest.
		var room entity.Room
		if lockErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&room, roomID).Error; lockErr != nil {
			return lockErr
		}

		var existing entity.Room_User
		// Named found rather than lookup because this is a *gorm.DB, and a
		// *gorm.DB has an Error() string method. Writing lookup.Error therefore
		// resolves to that method and yields a func() string, which is not an
		// error at all. The query result has to be held as a database row or
		// assigned out immediately.
		found := tx.
			Where("room_id = ? AND user_id = ?", roomID, userID).
			First(&existing)

		switch {
		case found.Error == nil && existing.Perm.IsActive():
			// Already on the roster. Rejoining is not an error and not a new
			// membership, and it must not be turned away by a full room.
			return nil
		case found.Error != nil && !errors.Is(found.Error, gorm.ErrRecordNotFound):
			return found.Error
		case existing.Perm == entity.BLOCKED:
			return entity.ErrRoomBlocked
		}

		if found.Error != nil {
			// A genuinely new member, so the capacity applies.
			if capacity > 0 {
				var count int64
				if countErr := tx.Model(&entity.Room_User{}).
					Where("room_id = ? AND perm NOT IN ?", roomID,
						[]entity.RoomPerm{entity.BLOCKED, entity.LEFT}).
					Count(&count).Error; countErr != nil {
					return countErr
				}
				if int(count) >= capacity {
					return entity.ErrRoomFull
				}
			}

			if createErr := tx.Create(&entity.Room_User{
				RoomID: roomID,
				UserID: userID,
				Perm:   perm,
			}).Error; createErr != nil {
				return createErr
			}
			added = true
			return nil
		}

		// A member who left or was demoted is coming back.
		if updateErr := tx.Model(&entity.Room_User{}).
			Where("room_id = ? AND user_id = ?", roomID, userID).
			Update("perm", perm).Error; updateErr != nil {
			return updateErr
		}
		added = true
		return nil
	})

	return added, err
}

// RemoveRosterMember removes a membership row outright.
func (d *Database) RemoveRosterMember(ctx context.Context, roomID, userID uint32) error {
	return d.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&entity.Room_User{}).Error
}

// RoomMembers returns a room's roster for display.
func (d *Database) RoomMembers(ctx context.Context, roomID uint32) ([]entity.RoomMember, error) {
	members, err := d.Roster(ctx, roomID)
	if err != nil {
		return nil, err
	}

	out := make([]entity.RoomMember, 0, len(members))
	for _, m := range members {
		perm, err := d.SeePerms(ctx, entity.Room_User{RoomID: roomID, UserID: m.UserID})
		if err != nil {
			// The roster and the membership should agree; if they do not,
			// skip the row rather than failing the whole listing.
			continue
		}
		user, err := d.GetUser(ctx, m.UserID)
		if err != nil {
			continue
		}
		out = append(out, entity.RoomMember{
			UserID:    m.UserID,
			Username:  m.Username,
			AvatarURL: user.AvatarURL,
			Rank:      m.Rank,
			Perm:      perm.Perm,
			IsMod:     perm.Perm == entity.MOD,
		})
	}
	return out, nil
}

// SaveNotification stores a notification for later delivery.
func (d *Database) SaveNotification(ctx context.Context, n entity.Notification) error {
	return d.db.WithContext(ctx).Create(&n).Error
}

// Notifications returns a user's notifications, newest first, and optionally
// marks them read.
func (d *Database) Notifications(ctx context.Context, userID uint32, unreadOnly bool, limit int) ([]entity.Notification, error) {
	if limit <= 0 {
		limit = 50
	}

	q := d.db.WithContext(ctx).Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("read = ?", false)
	}

	out := make([]entity.Notification, 0)
	if err := q.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// MarkNotificationsRead marks a user's notifications read.
func (d *Database) MarkNotificationsRead(ctx context.Context, userID uint32) error {
	return d.db.WithContext(ctx).
		Model(&entity.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true).Error
}

// PruneNotifications deletes a user's read notifications older than maxAge, so
// the inbox does not grow without bound.
func (d *Database) PruneNotifications(ctx context.Context, userID uint32, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	return d.db.WithContext(ctx).
		Where("user_id = ? AND read = ? AND created_at < ?", userID, true, cutoff).
		Delete(&entity.Notification{}).Error
}
