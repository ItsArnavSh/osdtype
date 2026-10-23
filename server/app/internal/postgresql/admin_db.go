package postgresql

import (
	"context"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Database) RecordActivity(ctx context.Context, hour time.Time, uid uint32, at time.Time) error {
	row := entity.ActivityHour{
		Hour:      hour.UTC(),
		UserID:    uid,
		FirstSeen: at.UTC(),
	}
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (d *Database) ActivityByHour(ctx context.Context, since time.Time) ([]entity.ActivityHourCount, error) {
	var rows []entity.ActivityHourCount
	err := d.db.WithContext(ctx).
		Model(&entity.ActivityHour{}).
		Select("hour, count(*) AS count").
		Where("hour >= ?", since.UTC()).
		Group("hour").
		Order("hour ASC").
		Scan(&rows).Error
	return rows, err
}

func (d *Database) DistinctActiveUsers(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&entity.ActivityHour{}).
		Where("hour >= ?", since.UTC()).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}

func (d *Database) CountUsers(ctx context.Context, includeBots bool) (int64, error) {
	var count int64
	q := d.db.WithContext(ctx).Model(&entity.User{})
	if !includeBots {
		q = q.Where("is_bot = ?", false)
	}
	err := q.Count(&count).Error
	return count, err
}

func (d *Database) CountGames(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&entity.Game{}).Count(&count).Error
	return count, err
}

func (d *Database) CountBugReports(ctx context.Context, statuses []string) (int64, error) {
	var count int64
	q := d.db.WithContext(ctx).Model(&entity.BugReport{})
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	err := q.Count(&count).Error
	return count, err
}

func (d *Database) GetBugReportByPublicID(ctx context.Context, publicID string) (entity.BugReport, error) {
	var report entity.BugReport
	err := d.db.WithContext(ctx).Where("public_id = ?", publicID).First(&report).Error
	return report, err
}

func (d *Database) bugAdminQuery(ctx context.Context, filter entity.BugAdminFilter) *gorm.DB {
	q := d.db.WithContext(ctx).Model(&entity.BugReport{})
	if status := strings.TrimSpace(strings.ToLower(filter.Status)); status != "" {
		q = q.Where("status = ?", status)
	}
	if severity := strings.TrimSpace(strings.ToLower(filter.Severity)); severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if category := strings.TrimSpace(strings.ToLower(filter.Category)); category != "" {
		q = q.Where("category = ?", category)
	}
	if text := strings.TrimSpace(strings.ToLower(filter.Query)); text != "" {
		like := "%" + text + "%"
		q = q.Where(
			"LOWER(public_id) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			like, like, like,
		)
	}
	return q
}

func (d *Database) ListBugReportsAdmin(ctx context.Context, filter entity.BugAdminFilter, limit, offset int) ([]entity.BugReport, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := d.bugAdminQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reports []entity.BugReport
	err := d.bugAdminQuery(ctx, filter).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reports).Error
	return reports, total, err
}

func (d *Database) UpdateBugReport(ctx context.Context, publicID string, fields map[string]any) error {
	return d.db.WithContext(ctx).
		Model(&entity.BugReport{}).
		Where("public_id = ?", publicID).
		Updates(fields).Error
}

func (d *Database) userAdminQuery(ctx context.Context, filter entity.UserAdminFilter) *gorm.DB {
	q := d.db.WithContext(ctx).Model(&entity.User{}).Where("is_bot = ?", false)
	if text := strings.TrimSpace(filter.Query); text != "" {
		like := "%" + strings.ToLower(text) + "%"
		if id, err := strconv.ParseUint(text, 10, 32); err == nil {
			q = q.Where("id = ? OR LOWER(username) LIKE ?", uint32(id), like)
		} else {
			q = q.Where("LOWER(username) LIKE ?", like)
		}
	}
	switch strings.TrimSpace(strings.ToLower(filter.Role)) {
	case "admin":
		q = q.Where("is_admin = ?", true)
	case "player":
		q = q.Where("is_admin = ?", false)
	}
	return q
}

func (d *Database) ListUsersAdmin(ctx context.Context, filter entity.UserAdminFilter, limit, offset int) ([]entity.User, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := d.userAdminQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []entity.User
	err := d.userAdminQuery(ctx, filter).
		Order("is_admin DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	return users, total, err
}

func (d *Database) SetUserAdmin(ctx context.Context, uid uint32, admin bool) (bool, error) {
	result := d.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND is_bot = ?", uid, false).
		Update("is_admin", admin)
	return result.RowsAffected > 0, result.Error
}

func (d *Database) CreateHealthCheck(ctx context.Context, check *entity.HealthCheck) error {
	return d.db.WithContext(ctx).Create(check).Error
}

func (d *Database) ListHealthChecks(ctx context.Context, since time.Time, limit int) ([]entity.HealthCheck, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	var checks []entity.HealthCheck
	err := d.db.WithContext(ctx).
		Where("checked_at >= ?", since.UTC()).
		Order("checked_at DESC").
		Limit(limit).
		Find(&checks).Error
	return checks, err
}

type HealthCheckStats struct {
	Total  int64
	OK     int64
	Failed int64
}

func (d *Database) HealthCheckStats(ctx context.Context, since time.Time) (HealthCheckStats, error) {
	var stats HealthCheckStats
	err := d.db.WithContext(ctx).
		Model(&entity.HealthCheck{}).
		Select(`count(*) AS total,
			COALESCE(SUM(CASE WHEN postgres_ok AND redis_ok THEN 1 ELSE 0 END), 0) AS ok,
			COALESCE(SUM(CASE WHEN postgres_ok AND redis_ok THEN 0 ELSE 1 END), 0) AS failed`).
		Where("checked_at >= ?", since.UTC()).
		Scan(&stats).Error
	return stats, err
}

func (d *Database) CreateSystemEvent(ctx context.Context, event *entity.SystemEvent) error {
	return d.db.WithContext(ctx).Create(event).Error
}

func (d *Database) systemEventQuery(ctx context.Context, filter entity.SystemEventFilter) *gorm.DB {
	q := d.db.WithContext(ctx).Model(&entity.SystemEvent{})
	if level := strings.TrimSpace(strings.ToLower(filter.Level)); level != "" {
		q = q.Where("LOWER(level) = ?", level)
	}
	if kind := strings.TrimSpace(strings.ToLower(filter.Kind)); kind != "" {
		q = q.Where("LOWER(kind) = ?", kind)
	}
	if text := strings.TrimSpace(strings.ToLower(filter.Query)); text != "" {
		like := "%" + text + "%"
		q = q.Where("LOWER(message) LIKE ? OR LOWER(metadata) LIKE ?", like, like)
	}
	return q
}

func (d *Database) ListSystemEvents(ctx context.Context, filter entity.SystemEventFilter, limit, offset int) ([]entity.SystemEvent, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := d.systemEventQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var events []entity.SystemEvent
	err := d.systemEventQuery(ctx, filter).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error
	return events, total, err
}

func (d *Database) CreateAuditLog(ctx context.Context, audit *entity.AuditLog) error {
	return d.db.WithContext(ctx).Create(audit).Error
}

func (d *Database) auditQuery(ctx context.Context, filter entity.AuditFilter) *gorm.DB {
	q := d.db.WithContext(ctx).Model(&entity.AuditLog{})
	if actor := strings.TrimSpace(strings.ToLower(filter.Actor)); actor != "" {
		q = q.Where("LOWER(actor) LIKE ?", "%"+actor+"%")
	}
	if action := strings.TrimSpace(strings.ToLower(filter.Action)); action != "" {
		q = q.Where("LOWER(action) = ?", action)
	}
	if text := strings.TrimSpace(strings.ToLower(filter.Query)); text != "" {
		like := "%" + text + "%"
		q = q.Where(
			"LOWER(actor) LIKE ? OR LOWER(target) LIKE ? OR LOWER(detail) LIKE ?",
			like, like, like,
		)
	}
	return q
}

func (d *Database) ListAuditLogs(ctx context.Context, filter entity.AuditFilter, limit, offset int) ([]entity.AuditLog, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := d.auditQuery(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var audits []entity.AuditLog
	err := d.auditQuery(ctx, filter).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&audits).Error
	return audits, total, err
}
