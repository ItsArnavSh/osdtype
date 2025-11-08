package postgresql

import (
	"context"
	"osdtyp/app/entity"
)

func (d *Database) FollowUser(ctx context.Context, id, otherid uint64) error {
	// Check if inverse relation exists (other->me)
	var relation entity.Friends
	result := d.db.WithContext(ctx).Where("B = ? AND A = ?", id, otherid).First(&relation)

	if result.Error != nil {
		// No inverse relation exists, create a new follow (me->other)
		return d.db.WithContext(ctx).Create(&entity.Friends{
			A:        id,
			B:        otherid,
			Relation: entity.FOLLOWS,
		}).Error
	}

	// Inverse exists, check its state
	switch relation.Relation {
	case entity.FRIENDS:
		// Already friends, nothing to do
		return nil
	case entity.FOLLOWS:
		// Other follows me, so upgrade to FRIENDS
		return d.db.WithContext(ctx).Model(&entity.Friends{}).
			Where("B = ? AND A = ?", id, otherid).
			Update("relation", entity.FRIENDS).Error
	}

	return nil
}

func (d *Database) UnfollowUser(ctx context.Context, id, otherid uint64) error {
	// First, check if there's a direct relation (me->other)
	var directRelation entity.Friends
	directResult := d.db.WithContext(ctx).Where("A = ? AND B = ?", id, otherid).First(&directRelation)

	// Check if there's an inverse relation (other->me)
	var inverseRelation entity.Friends
	inverseResult := d.db.WithContext(ctx).Where("B = ? AND A = ?", id, otherid).First(&inverseRelation)

	// Case 1: Direct relation exists (A->B)
	if directResult.Error == nil {
		switch directRelation.Relation {
		case entity.FOLLOWS:
			// Simple follow, just delete it
			return d.db.WithContext(ctx).Where("A = ? AND B = ?", id, otherid).
				Delete(&entity.Friends{}).Error

		case entity.FRIENDS:
			// We're friends. Need to downgrade to B->A FOLLOWS
			// Delete the current A B FRIENDS record
			if err := d.db.WithContext(ctx).Where("A = ? AND B = ?", id, otherid).
				Delete(&entity.Friends{}).Error; err != nil {
				return err
			}
			// Create B->A FOLLOWS (other now follows me)
			return d.db.WithContext(ctx).Create(&entity.Friends{
				A:        otherid,
				B:        id,
				Relation: entity.FOLLOWS,
			}).Error
		}
	}

	// Case 2: Only inverse relation exists (B->A)
	if inverseResult.Error == nil && inverseRelation.Relation == entity.FRIENDS {
		// This means the record is stored as B A FRIENDS
		// Downgrade it to B A FOLLOWS
		return d.db.WithContext(ctx).Model(&entity.Friends{}).
			Where("B = ? AND A = ?", id, otherid).
			Update("relation", entity.FOLLOWS).Error
	}
	// Case 3: No relation exists, nothing to do
	return nil
}

func (d *Database) GetFriends(ctx context.Context, id uint64) ([]uint64, error) {
	var friends []uint64

	// Find all FRIENDS relations where the user is either A or B
	var relations []entity.Friends
	err := d.db.WithContext(ctx).
		Where("(A = ? OR B = ?) AND relation = ?", id, id, entity.FRIENDS).
		Find(&relations).Error

	if err != nil {
		return nil, err
	}

	// Extract the friend IDs (the other person in each relation)
	for _, rel := range relations {
		if rel.A == id {
			friends = append(friends, rel.B)
		} else {
			friends = append(friends, rel.A)
		}
	}

	return friends, nil
}
