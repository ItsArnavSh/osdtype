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
func (d *Database) ListContests(room_id uint64, limit, index int) ([]entity.Contest, error) {
	var contests []entity.Contest
	err := d.db.Where("room_id = ?", room_id).Order("time DESC").Offset(index * limit).Limit(limit).Find(&contests).Error
	return contests, err
}
func (d *Database) GetContestData(job_id uint64) (entity.Contest, error) {
	var contest entity.Contest
	err := d.db.Where("job_id = ", job_id).First(&contest).Error
	return contest, err
}
