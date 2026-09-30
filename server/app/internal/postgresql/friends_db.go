package postgresql

import (
	"context"
	"errors"
	"strings"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The friend graph.
//
// A relation is a directed edge with a composite primary key of (a, b), so a
// mutual pair is stored as two rows: (me, them, FRIENDS) and (them, me,
// FRIENDS). That is what makes the list query below a left join rather than a
// plain read, and it is why an unqualified "where a = me or b = me" returns
// every friend twice.

// FollowUser makes one player's friend list include another's.
//
// It is idempotent. The composite primary key means a second follow of the
// same person raised a duplicate key violation, which surfaced as a 500 for
// clicking a button twice. The insert is now an upsert so the state converges
// instead of failing.
//
// The relation is derived, not asserted: following someone who already follows
// you makes the friendship mutual, and following someone who does not is a
// one-way FOLLOWS. Both the insert and the conflict update write the same
// computed value.
//
// It used to write FRIENDS unconditionally, which quietly deleted the model's
// only state distinction. Every one-way follow was stored as mutual, so
// "mutual" meant "followed" in the interface, and the only way to tell an
// actual friend from a celebrity you follow was to know it out of band. The
// three friend integration tests caught it.
func (d *Database) FollowUser(ctx context.Context, id, otherid uint32) error {
	if id == otherid {
		return errors.New("cannot follow yourself")
	}
	if _, err := d.GetUser(ctx, otherid); err != nil {
		return err
	}

	// The reverse edge is what makes this mutual. Read it before the write, in
	// the same transaction as the write, so two people following each other at
	// the same moment both land on FRIENDS instead of one of them being stuck
	// on FOLLOWS forever.
	relation := entity.FOLLOWS
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reverse int64
		if err := tx.Model(&entity.Friends{}).
			Where("a = ? AND b = ?", otherid, id).
			Count(&reverse).Error; err != nil {
			return err
		}
		if reverse == 0 {
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "a"}, {Name: "b"}},
				DoUpdates: clause.Assignments(map[string]any{"relation": relation}),
			}).Create(&entity.Friends{
				A:        id,
				B:        otherid,
				Relation: relation,
			}).Error
		}

		// They already follow back, so this is mutual and both rows have to say
		// so.
		//
		// Upgrading only the direction being written leaves the pair
		// self-contradictory: the new row says FRIENDS and the older one still
		// says FOLLOWS. Nothing in the schema stops that, and the queries that
		// pick a winner — ListFriends' "who is the friend" CASE, and anything
		// that filters on relation — then read whichever row they happened to
		// touch first. So the friend list disagreed with itself about whether
		// two people were friends. The integration test that follows both ways
		// and then checks both edges is what caught it; the original one only
		// checked the list, which was still the right length either way.
		if err := tx.Model(&entity.Friends{}).
			Where("a = ? AND b = ?", otherid, id).
			Update("relation", entity.FRIENDS).Error; err != nil {
			return err
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "a"}, {Name: "b"}},
			DoUpdates: clause.Assignments(map[string]any{"relation": entity.FRIENDS}),
		}).Create(&entity.Friends{
			A:        id,
			B:        otherid,
			Relation: entity.FRIENDS,
		}).Error
	})
	return err
}

// UnfollowUser removes a friendship, leaving the other direction intact.
//
// Unfollowing someone you are mutual with leaves them following you: they did
// nothing wrong, and silently removing their follow as well would make the
// action mean something different depending on which side you pressed.
func (d *Database) UnfollowUser(ctx context.Context, id, otherid uint32) error {
	if id == otherid {
		return errors.New("cannot unfollow yourself")
	}

	return d.db.WithContext(ctx).
		Where("a = ? AND b = ?", id, otherid).
		Delete(&entity.Friends{}).Error
}

// ListFriends returns everything one person is connected to, with the standing
// of each connection.
//
// It is a single query with two correlated subqueries, not a read of the
// friendship table followed by a user lookup per row. The previous shape was
// an N+1: one query for the edges, then one more for every single friend, so
// a list of fifty cost fifty-one round trips and could still return rows
// missing the users that had been deleted.
//
// The DISTINCT on the other user's id is what collapses the two rows of a
// mutual pair into one.
func (d *Database) ListFriends(ctx context.Context, id uint32) ([]entity.Friendship, error) {
	type row struct {
		FriendID uint32
		Followed bool
		Follows  bool
		Username string
		Avatar   string
		Rank     uint16
	}

	var rows []row
	err := d.db.WithContext(ctx).
		Model(&entity.Friends{}).
		Select(`
			DISTINCT
			CASE WHEN f.a = ? THEN f.b ELSE f.a END AS friend_id,
			EXISTS (SELECT 1 FROM friends w WHERE w.a = ? AND w.b = CASE WHEN f.a = ? THEN f.b ELSE f.a END) AS followed,
			EXISTS (SELECT 1 FROM friends w WHERE w.a = CASE WHEN f.a = ? THEN f.b ELSE f.a END AND w.b = ?) AS follows,
			users.username AS username,
			users.avatar_url AS avatar,
			users.current_rank AS rank
		`, id, id, id, id, id).
		Joins("JOIN users ON users.id = CASE WHEN f.a = ? THEN f.b ELSE f.a END", id).
		Where("f.a = ? OR f.b = ?", id, id).
		Order("rank DESC, username ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]entity.Friendship, 0, len(rows))
	for _, r := range rows {
		if r.FriendID == id {
			continue
		}
		out = append(out, entity.Friendship{
			FriendID:  r.FriendID,
			Followed:  r.Followed,
			Follows:   r.Follows,
			Mutual:    r.Followed && r.Follows,
			Username:  r.Username,
			AvatarURL: r.Avatar,
			Rank:      r.Rank,
		})
	}
	return out, nil
}

// GetFriends returns just the mutual friends' ids.
//
// It is kept for the friend-scoped leaderboard. Unlike the old implementation
// it does not read the edges and then map them in Go: the CASE expression is
// the same one ListFriends uses, so both agree on which id is "the other
// person".
func (d *Database) GetFriends(ctx context.Context, id uint32) ([]uint32, error) {
	type row struct {
		FriendID uint32
	}
	var rows []row
	err := d.db.WithContext(ctx).
		Model(&entity.Friends{}).
		Select("DISTINCT CASE WHEN a = ? THEN b ELSE a END AS friend_id", id).
		Where("(a = ? OR b = ?) AND relation = ?", id, id, entity.FRIENDS).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]uint32, 0, len(rows))
	for _, r := range rows {
		if r.FriendID != id {
			out = append(out, r.FriendID)
		}
	}
	return out, nil
}

// AreFriends reports whether two people are mutual.
func (d *Database) AreFriends(ctx context.Context, a, b uint32) (bool, error) {
	var n int64
	err := d.db.WithContext(ctx).
		Model(&entity.Friends{}).
		Where("relation = ? AND ((a = ? AND b = ?) OR (a = ? AND b = ?))", entity.FRIENDS, a, b, b, a).
		Count(&n).Error
	return n > 0, err
}

// SearchPeople finds accounts whose name contains a string.
//
// The column is username, matching the Username field on User. The previous
// query filtered on "user_name", which does not exist, so every search failed.
//
// The match is case insensitive, which is what someone typing a friend's name
// expects, and the ordering puts an exact match first: searching for someone
// whose name you already know should not make you scroll past everyone who
// merely starts with the same two letters.
func (d *Database) SearchPeople(ctx context.Context, txt string, limit uint8) ([]entity.User, error) {
	if limit == 0 {
		limit = 20
	}

	pattern := "%" + txt + "%"
	exact := strings.ToLower(txt)

	users := make([]entity.User, 0, limit)
	err := d.db.WithContext(ctx).
		Where("username ILIKE ?", pattern).
		Order("CASE" +
			" WHEN LOWER(username) = " + quoteLiteral(exact) + " THEN 0" +
			" WHEN username ILIKE " + quoteLiteral(txt+"%") + " THEN 1" +
			" ELSE 2 END").
		Order("current_rank DESC").
		Order("username ASC").
		Limit(int(limit)).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// quoteLiteral makes a string safe to inline into a raw ORDER BY expression.
//
// The search term reaches here from a query parameter, and the ordering
// expression is a CASE over the username so an exact match can be sorted first.
// GORM's placeholders do not expand inside a Select or Order string, so the
// alternative would be an unparameterised interpolation; quoting with doubled
// single quotes is what keeps a term containing a quote from ending the
// literal and becoming SQL.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// UserIDExists reports whether an account with this id exists, for validating
// a follow. It takes an id rather than a name because the follow path already
// has the id and resolving it back to a name would be a second query.
func (d *Database) UserIDExists(ctx context.Context, id uint32) (bool, error) {
	var n int64
	if err := d.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
