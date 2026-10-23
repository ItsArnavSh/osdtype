package postgresql

import (
	"context"
	"time"

	"osdtyp/app/entity"
)

func (d *Database) CreateBugReport(ctx context.Context, report *entity.BugReport) error {
	return d.db.WithContext(ctx).Create(report).Error
}

func (d *Database) GetBugReportBySubmissionID(ctx context.Context, submissionID string) (*entity.BugReport, error) {
	var report entity.BugReport
	if err := d.db.WithContext(ctx).Where("submission_id = ?", submissionID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (d *Database) ListBugReportsForUser(ctx context.Context, userID uint32, limit int) ([]entity.BugReport, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var reports []entity.BugReport
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&reports).Error
	return reports, err
}

func (d *Database) CountRecentBugReports(ctx context.Context, ipHash, title string, since time.Time) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&entity.BugReport{}).
		Where("ip_hash = ? AND lower(title) = lower(?) AND created_at >= ?", ipHash, title, since).
		Count(&count).Error
	return count, err
}
