package postgresql

import (
	"context"

	"osdtyp/app/entity"
)

func (d *Database) NewContest(ctx context.Context, contest entity.Contest) error {
	return d.db.WithContext(ctx).Create(&contest).Error
}
func (d *Database) UpdateContest(contest entity.Contest) error {
	return d.db.Where("job_id = ?", contest.JobID).UpdateColumns(contest).Error
}

// ListContests returns one page of contests for a room.
//
// The parameters are named the way the only caller uses them, index first then
// limit. They were previously declared (limit, index) while the caller passed
// (index, 50), so the page size became the page number and the offset was
// computed from 50.
func (d *Database) ListContests(roomID uint32, index, limit int) ([]entity.Contest, error) {
	var contests []entity.Contest
	if limit <= 0 {
		limit = 50
	}
	if index < 0 {
		index = 0
	}
	err := d.db.Where("room_id = ?", roomID).
		Order("time DESC").
		Offset(index * limit).
		Limit(limit).
		Find(&contests).Error
	return contests, err
}
func (d *Database) GetContestData(jobID uint32) (entity.Contest, error) {
	var contest entity.Contest
	// The placeholder is required: writing "job_id = " without it sent gorm a
	// truncated fragment and every call failed with a SQL syntax error.
	err := d.db.Where("job_id = ?", jobID).First(&contest).Error
	return contest, err
}
