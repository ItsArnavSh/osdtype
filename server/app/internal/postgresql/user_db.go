package postgresql

import (
	"context"

	"osdtyp/app/entity"
)

func (d *Database) AddUser(ctx context.Context, user entity.User) error {
	return d.db.WithContext(ctx).Create(&user).Error
}

// GetUser reads one account by id.
//
// It takes a context, unlike the original, so the query is cancellable and so
// callers have somewhere to put the one they already have. A context-free read
// of the user table means a request whose client disconnected still runs to
// completion and holds a connection while it does; this is the hottest single-row
// read in the process, called on every authenticated request through the session
// registry and the lobby.
func (d *Database) GetUser(ctx context.Context, userid uint32) (entity.User, error) {
	var userData entity.User
	result := d.db.WithContext(ctx).Where("id = ?", userid).First(&userData)
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

// ChangeRank updates a user's rank.
//
// The column is current_rank, matching the CurrentRank field: the update
// previously named a "rank" column that does not exist, so every ranked match
// result was discarded.
