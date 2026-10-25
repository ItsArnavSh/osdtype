package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var adminBugStatuses = map[string]bool{
	entity.BugStatusNew:        true,
	entity.BugStatusTriaged:    true,
	entity.BugStatusInProgress: true,
	entity.BugStatusResolved:   true,
	entity.BugStatusClosed:     true,
	entity.BugStatusWontFix:    true,
}

func (s *ServiceLayer) PrimaryAdminUsername() string {
	return strings.TrimSpace(viper.GetString("Admin.primary_username"))
}

func (s *ServiceLayer) IsPrimaryAdmin(user entity.User) bool {
	primary := s.PrimaryAdminUsername()
	return primary != "" && !user.IsBot && strings.EqualFold(strings.TrimSpace(user.Username), primary)
}

func (s *ServiceLayer) IsAdmin(ctx context.Context, uid uint32) (bool, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return false, err
	}
	return s.IsPrimaryAdmin(user) || user.IsAdmin, nil
}

// UserView adds the computed primary-admin flag to the public projection.
// It is used for authenticated self views so the client can respect the
// account's save-data preference before sending a practice run.
func (s *ServiceLayer) UserView(user entity.User) entity.UserView {
	view := entity.NewUserView(user)
	view.IsAdmin = view.IsAdmin || s.IsPrimaryAdmin(user)
	saveRunData := user.SaveRunData
	view.SaveRunData = &saveRunData
	return view
}

// RecordActivity writes one distinct user for the current UTC hour. Redis
// throttles the write to once per user per hour; Postgres keeps the durable
// hourly count.
func (s *ServiceLayer) RecordActivity(ctx context.Context, uid uint32, at time.Time) error {
	hour := at.UTC().Truncate(time.Hour)
	if rdb := s.rdb(); rdb != nil && rdb.R != nil {
		key := fmt.Sprintf("activity:seen:%s:%d", hour.Format("20060102T15"), uid)
		ttl := time.Until(hour.Add(2 * time.Hour))
		if ttl <= 0 {
			ttl = 2 * time.Hour
		}
		first, err := rdb.R.SetNX(ctx, key, 1, ttl).Result()
		if err == nil && !first {
			return nil
		}
		if err != nil {
			s.logger.Warnw("activity throttle unavailable", "error", err)
		}
	}
	return s.db.RecordActivity(ctx, hour, uid, at.UTC())
}

type AdminOverview struct {
	Now            time.Time            `json:"now"`
	PrimaryAdmin   string               `json:"primary_admin"`
	Me             entity.AdminUserView `json:"me"`
	ActiveHour     int64                `json:"active_hour"`
	Active24h      int64                `json:"active_24h"`
	OnlineNow      int                  `json:"online_now"`
	QueueSize      int                  `json:"queue_size"`
	UsersTotal     int64                `json:"users_total"`
	GamesTotal     int64                `json:"games_total"`
	BugsOpen       int64                `json:"bugs_open"`
	BugsTotal      int64                `json:"bugs_total"`
	Uptime24h      *float64             `json:"uptime_24h"`
	Checks24h      int64                `json:"checks_24h"`
	FailedChecks24 int64                `json:"failed_checks_24h"`
}

func (s *ServiceLayer) adminUserView(user entity.User) entity.AdminUserView {
	return entity.AdminUserView{
		ID:          user.ID,
		Username:    user.Username,
		AvatarURL:   user.AvatarURL,
		Rating:      user.Rating,
		GamesPlayed: user.GamesPlayed,
		IsAdmin:     user.IsAdmin || s.IsPrimaryAdmin(user),
		IsPrimary:   s.IsPrimaryAdmin(user),
		CreatedAt:   user.CreatedAt,
	}
}

func (s *ServiceLayer) Overview(ctx context.Context, actor entity.User) (AdminOverview, error) {
	now := time.Now().UTC()
	hourStart := now.Truncate(time.Hour)
	dayStart := now.Add(-24 * time.Hour)

	activeHour, err := s.db.DistinctActiveUsers(ctx, hourStart)
	if err != nil {
		return AdminOverview{}, err
	}
	active24, err := s.db.DistinctActiveUsers(ctx, dayStart)
	if err != nil {
		return AdminOverview{}, err
	}
	usersTotal, err := s.db.CountUsers(ctx, false)
	if err != nil {
		return AdminOverview{}, err
	}
	gamesTotal, err := s.db.CountGames(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	bugsOpen, err := s.db.CountBugReports(ctx, []string{
		entity.BugStatusNew, entity.BugStatusTriaged, entity.BugStatusInProgress,
	})
	if err != nil {
		return AdminOverview{}, err
	}
	bugsTotal, err := s.db.CountBugReports(ctx, nil)
	if err != nil {
		return AdminOverview{}, err
	}
	stats := summarizeHealth(s.recentHealthChecks(ctx, dayStart, 2000))
	var uptime *float64
	if stats.Total > 0 {
		value := float64(stats.OK) / float64(stats.Total) * 100
		uptime = &value
	}
	return AdminOverview{
		Now:            now,
		PrimaryAdmin:   s.PrimaryAdminUsername(),
		Me:             s.adminUserView(actor),
		ActiveHour:     activeHour,
		Active24h:      active24,
		OnlineNow:      s.core.Hub.OnlineCount(),
		QueueSize:      s.core.Matchmaker.QueueSize(ctx),
		UsersTotal:     usersTotal,
		GamesTotal:     gamesTotal,
		BugsOpen:       bugsOpen,
		BugsTotal:      bugsTotal,
		Uptime24h:      uptime,
		Checks24h:      stats.Total,
		FailedChecks24: stats.Failed,
	}, nil
}

func (s *ServiceLayer) ListBugsAdmin(ctx context.Context, filter entity.BugAdminFilter, limit, offset int) ([]entity.AdminBugView, int64, error) {
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Status != "" && !adminBugStatuses[filter.Status] {
		return nil, 0, errf(ErrBadRequest, "unknown bug status")
	}
	reports, total, err := s.db.ListBugReportsAdmin(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint32, 0, len(reports))
	for _, report := range reports {
		if report.UserID != nil {
			ids = append(ids, *report.UserID)
		}
	}
	users, _ := s.db.ListUsersByIDs(ctx, ids)
	names := make(map[uint32]string, len(users))
	for _, user := range users {
		names[user.ID] = user.Username
	}
	out := make([]entity.AdminBugView, 0, len(reports))
	for _, report := range reports {
		view := entity.AdminBugView{
			BugReportView: entity.NewBugReportView(report),
			UserID:        report.UserID,
			AdminNotes:    report.AdminNotes,
		}
		if report.UserID != nil {
			view.Username = names[*report.UserID]
		}
		out = append(out, view)
	}
	return out, total, nil
}

func (s *ServiceLayer) UpdateBugStatus(ctx context.Context, actor entity.User, publicID, status, notes string) (entity.BugReport, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if !adminBugStatuses[status] {
		return entity.BugReport{}, errf(ErrBadRequest, "unknown bug status")
	}
	report, err := s.db.GetBugReportByPublicID(ctx, strings.TrimSpace(publicID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.BugReport{}, errf(ErrNotFound, "bug report not found")
		}
		return entity.BugReport{}, err
	}
	fields := map[string]any{"status": status, "admin_notes": strings.TrimSpace(notes)}
	if status == entity.BugStatusResolved || status == entity.BugStatusClosed || status == entity.BugStatusWontFix {
		fields["resolved_at"] = time.Now().UTC()
	} else {
		fields["resolved_at"] = nil
	}
	if err := s.db.UpdateBugReport(ctx, report.PublicID, fields); err != nil {
		return entity.BugReport{}, err
	}
	s.RecordAudit(ctx, actor, "bug.status", report.PublicID, status)
	updated, err := s.db.GetBugReportByPublicID(ctx, report.PublicID)
	if err != nil {
		return entity.BugReport{}, err
	}
	return updated, nil
}

func (s *ServiceLayer) ListUsersAdmin(ctx context.Context, filter entity.UserAdminFilter, limit, offset int) ([]entity.AdminUserView, int64, error) {
	users, total, err := s.db.ListUsersAdmin(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.AdminUserView, 0, len(users))
	for _, user := range users {
		out = append(out, s.adminUserView(user))
	}
	return out, total, nil
}

func (s *ServiceLayer) SetUserAdmin(ctx context.Context, actor entity.User, targetID uint32, admin bool) error {
	if !s.IsPrimaryAdmin(actor) {
		return errf(ErrForbidden, "only the primary admin can manage admins")
	}
	target, err := s.GetUser(ctx, targetID)
	if err != nil {
		return err
	}
	if target.IsBot {
		return errf(ErrBadRequest, "bot accounts cannot be admins")
	}
	if s.IsPrimaryAdmin(target) {
		return errf(ErrBadRequest, "the primary admin cannot be demoted")
	}
	ok, err := s.db.SetUserAdmin(ctx, target.ID, admin)
	if err != nil {
		return err
	}
	if !ok {
		return errf(ErrNotFound, "user not found")
	}
	action := "admin.revoke"
	if admin {
		action = "admin.grant"
	}
	s.RecordAudit(ctx, actor, action, target.Username, fmt.Sprintf("target_id=%d", target.ID))
	return nil
}

func (s *ServiceLayer) Activity(ctx context.Context, hours int) ([]entity.ActivityHourCount, error) {
	if hours <= 0 || hours > 168 {
		hours = 24
	}
	now := time.Now().UTC().Truncate(time.Hour)
	rows, err := s.db.ActivityByHour(ctx, now.Add(-time.Duration(hours-1)*time.Hour))
	if err != nil {
		return nil, err
	}
	counts := make(map[int64]int64, len(rows))
	for _, row := range rows {
		counts[row.Hour.UTC().Unix()] = row.Count
	}
	out := make([]entity.ActivityHourCount, 0, hours)
	for i := hours - 1; i >= 0; i-- {
		hour := now.Add(-time.Duration(i) * time.Hour)
		out = append(out, entity.ActivityHourCount{Hour: hour, Count: counts[hour.Unix()]})
	}
	return out, nil
}

type AdminHealthView struct {
	Current        map[string]bool      `json:"current"`
	Uptime24h      *float64             `json:"uptime_24h"`
	Checks24h      int64                `json:"checks_24h"`
	FailedChecks24 int64                `json:"failed_checks_24h"`
	Checks         []entity.HealthCheck `json:"checks"`
}

func (s *ServiceLayer) Health(ctx context.Context, limit, offset int) (AdminHealthView, error) {
	now := time.Now().UTC()
	checkCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	postgresOK := s.db != nil && s.db.PingContext(checkCtx) == nil
	redisOK := false
	if rdb := s.rdb(); rdb != nil {
		redisOK = rdb.Ping(checkCtx) == nil
	}
	checks := s.recentHealthChecks(ctx, now.Add(-24*time.Hour), 2000)
	stats := summarizeHealth(checks)
	var uptime *float64
	if stats.Total > 0 {
		value := float64(stats.OK) / float64(stats.Total) * 100
		uptime = &value
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	page := []entity.HealthCheck{}
	if offset < len(checks) {
		end := offset + limit
		if end > len(checks) {
			end = len(checks)
		}
		page = checks[offset:end]
	}
	return AdminHealthView{
		Current:        map[string]bool{"postgres": postgresOK, "redis": redisOK},
		Uptime24h:      uptime,
		Checks24h:      stats.Total,
		FailedChecks24: stats.Failed,
		Checks:         page,
	}, nil
}

func (s *ServiceLayer) Logs(ctx context.Context, filter utils.LogFilter, limit, offset int) ([]utils.LogEntry, int) {
	if user := strings.TrimSpace(filter.User); user != "" {
		if id, err := strconv.ParseUint(user, 10, 32); err == nil {
			filter.UserID = uint32(id)
		} else if account, err := s.GetUserFromName(ctx, user); err == nil {
			filter.UserID = account.ID
		}
	}
	return utils.Logs().Search(filter, limit, offset)
}

func (s *ServiceLayer) Events(ctx context.Context, filter entity.SystemEventFilter, limit, offset int) ([]entity.SystemEvent, int64, error) {
	return s.db.ListSystemEvents(ctx, filter, limit, offset)
}

func (s *ServiceLayer) Audits(ctx context.Context, filter entity.AuditFilter, limit, offset int) ([]entity.AuditLog, int64, error) {
	return s.db.ListAuditLogs(ctx, filter, limit, offset)
}

func (s *ServiceLayer) RecordEvent(ctx context.Context, level, kind, message string, metadata map[string]any) {
	event := &entity.SystemEvent{
		Level:     level,
		Kind:      kind,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}
	if len(metadata) > 0 {
		if raw, err := json.Marshal(metadata); err == nil {
			event.Metadata = string(raw)
		}
	}
	if err := s.db.CreateSystemEvent(ctx, event); err != nil {
		s.logger.Warnw("could not record system event", "kind", kind, "error", err)
	}
}

func (s *ServiceLayer) RecordAudit(ctx context.Context, actor entity.User, action, target, detail string) {
	audit := &entity.AuditLog{
		ActorID:   actor.ID,
		Actor:     actor.Username,
		Action:    action,
		Target:    target,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.db.CreateAuditLog(ctx, audit); err != nil {
		s.logger.Warnw("could not record audit log", "action", action, "error", err)
	}
}

func (s *ServiceLayer) StartMonitoring() {
	s.monitorMu.Lock()
	if s.monitorCancel != nil {
		s.monitorMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.monitorCancel = cancel
	s.monitorMu.Unlock()

	interval := time.Duration(viper.GetInt("Admin.health_interval_seconds")) * time.Second
	if interval < 15*time.Second {
		interval = 60 * time.Second
	}
	s.monitorWG.Add(1)
	go func() {
		defer s.monitorWG.Done()
		s.runHealthCheck(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runHealthCheck(ctx)
			}
		}
	}()

	// Run retention immediately and then periodically. This also cleans up
	// accounts that have not played since the limit was introduced.
	s.monitorWG.Add(1)
	go func() {
		defer s.monitorWG.Done()
		s.pruneAllMatchHistories(ctx)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pruneAllMatchHistories(ctx)
			}
		}
	}()
}

func (s *ServiceLayer) StopMonitoring() {
	s.monitorMu.Lock()
	cancel := s.monitorCancel
	s.monitorCancel = nil
	s.monitorMu.Unlock()
	if cancel != nil {
		cancel()
		s.monitorWG.Wait()
	}
}

func (s *ServiceLayer) runHealthCheck(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	start := time.Now()
	postgresOK := s.db != nil && s.db.PingContext(ctx) == nil
	redisOK := false
	if rdb := s.rdb(); rdb != nil {
		redisOK = rdb.Ping(ctx) == nil
	}
	check := &entity.HealthCheck{
		CheckedAt:  time.Now().UTC(),
		PostgresOK: postgresOK,
		RedisOK:    redisOK,
		LatencyMS:  time.Since(start).Milliseconds(),
	}
	if !postgresOK {
		check.Error = "postgres unavailable"
	}
	if !redisOK {
		if check.Error != "" {
			check.Error += "; "
		}
		check.Error += "redis unavailable"
	}
	s.storeHealthCheck(ctx, check)
	s.recordHealthTransition(ctx, postgresOK, redisOK)
}

const healthRedisKey = "health:checks"

// storeHealthCheck keeps the last seven days in Redis so a Postgres outage is
// still visible after the database comes back. Postgres remains the backup
// history when Redis is unavailable.
func (s *ServiceLayer) storeHealthCheck(ctx context.Context, check *entity.HealthCheck) {
	if rdb := s.rdb(); rdb != nil && rdb.R != nil {
		raw, err := json.Marshal(check)
		if err == nil {
			pipe := rdb.R.TxPipeline()
			pipe.LPush(ctx, healthRedisKey, raw)
			pipe.LTrim(ctx, healthRedisKey, 0, 10079)
			pipe.Expire(ctx, healthRedisKey, 7*24*time.Hour)
			if _, err := pipe.Exec(ctx); err != nil {
				s.logger.Warnw("could not record health check in redis", "error", err)
			}
		}
	}
	if check.PostgresOK {
		if err := s.db.CreateHealthCheck(ctx, check); err != nil {
			s.logger.Warnw("could not record health check", "error", err)
		}
	}
	if !check.PostgresOK || !check.RedisOK {
		s.logger.Warnw("health check failed", "postgres", check.PostgresOK, "redis", check.RedisOK)
	}
}

func (s *ServiceLayer) recentHealthChecks(ctx context.Context, since time.Time, limit int) []entity.HealthCheck {
	if limit <= 0 || limit > 10000 {
		limit = 2000
	}
	if rdb := s.rdb(); rdb != nil && rdb.R != nil {
		rawChecks, err := rdb.R.LRange(ctx, healthRedisKey, 0, int64(limit-1)).Result()
		if err == nil && len(rawChecks) > 0 {
			checks := make([]entity.HealthCheck, 0, len(rawChecks))
			for _, raw := range rawChecks {
				var check entity.HealthCheck
				if err := json.Unmarshal([]byte(raw), &check); err != nil {
					continue
				}
				if check.CheckedAt.UTC().Before(since.UTC()) {
					continue
				}
				checks = append(checks, check)
			}
			if len(checks) > 0 {
				return checks
			}
		}
	}
	checks, err := s.db.ListHealthChecks(ctx, since, limit)
	if err != nil {
		s.logger.Warnw("could not read health checks", "error", err)
		return nil
	}
	return checks
}

type healthSummary struct {
	Total  int64
	OK     int64
	Failed int64
}

func summarizeHealth(checks []entity.HealthCheck) healthSummary {
	var stats healthSummary
	for _, check := range checks {
		stats.Total++
		if check.PostgresOK && check.RedisOK {
			stats.OK++
		} else {
			stats.Failed++
		}
	}
	return stats
}

func (s *ServiceLayer) recordHealthTransition(ctx context.Context, postgresOK, redisOK bool) {
	s.healthMu.Lock()
	changed := !s.healthInitialized || s.lastPostgres != postgresOK || s.lastRedis != redisOK
	s.healthInitialized = true
	s.lastPostgres = postgresOK
	s.lastRedis = redisOK
	s.healthMu.Unlock()
	if !changed {
		return
	}
	level := "info"
	message := "dependencies healthy"
	if !postgresOK || !redisOK {
		level = "warn"
		message = "dependency health changed"
	}
	s.RecordEvent(ctx, level, "health", message, map[string]any{
		"postgres": postgresOK,
		"redis":    redisOK,
	})
}
