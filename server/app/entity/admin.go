package entity

import "time"

// ActivityHour records one distinct authenticated user in one UTC hour. The
// composite primary key makes the hourly upsert idempotent.
type ActivityHour struct {
	Hour      time.Time `gorm:"primaryKey;column:hour;not null" json:"hour"`
	UserID    uint32    `gorm:"primaryKey;column:user_id;not null" json:"user_id"`
	FirstSeen time.Time `gorm:"not null" json:"first_seen"`
}

func (ActivityHour) TableName() string { return "activity_hours" }

// HealthCheck is a minute-level self-check of the API's dependencies. Rows
// are written only while Postgres is reachable, so a full database outage is
// visible in the process logs and in the external uptime monitor instead.
type HealthCheck struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	CheckedAt  time.Time `gorm:"index;not null" json:"checked_at"`
	PostgresOK bool      `gorm:"not null" json:"postgres_ok"`
	RedisOK    bool      `gorm:"not null" json:"redis_ok"`
	LatencyMS  int64     `gorm:"not null;default:0" json:"latency_ms"`
	Error      string    `gorm:"column:error;size:512" json:"error,omitempty"`
}

// SystemEvent is a durable, low-volume operational event. Request logs stay
// in the in-memory ring buffer and Render's log stream; this table is for
// things an operator should still see after a restart.
type SystemEvent struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	Level     string    `gorm:"size:16;not null;index" json:"level"`
	Kind      string    `gorm:"size:64;not null;index" json:"kind"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	Metadata  string    `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

// AuditLog records privileged actions such as promoting an admin or changing
// a bug report's state.
type AuditLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	ActorID   uint32    `gorm:"index;not null" json:"actor_id"`
	Actor     string    `gorm:"size:64;not null" json:"actor"`
	Action    string    `gorm:"size:64;not null;index" json:"action"`
	Target    string    `gorm:"size:128" json:"target,omitempty"`
	Detail    string    `gorm:"type:text" json:"detail,omitempty"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

// AdminUserView is the operator-facing user row. It includes role state that
// is not part of the public profile projection.
type AdminUserView struct {
	ID          uint32    `json:"id"`
	Username    string    `json:"username"`
	AvatarURL   string    `json:"avatar_url"`
	Rating      int       `json:"rating"`
	GamesPlayed int       `json:"games_played"`
	IsAdmin     bool      `json:"is_admin"`
	IsPrimary   bool      `json:"is_primary"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminBugView adds submitter and triage fields to the public bug view.
type AdminBugView struct {
	BugReportView
	UserID     *uint32 `json:"user_id,omitempty"`
	Username   string  `json:"username,omitempty"`
	AdminNotes string  `json:"admin_notes"`
}

type ActivityHourCount struct {
	Hour  time.Time `json:"hour"`
	Count int64     `json:"count"`
}

// Admin filters stay in the entity package so the API, service and database
// layers share the same query shape without importing each other.
type BugAdminFilter struct {
	Status   string
	Severity string
	Category string
	Query    string
}

type UserAdminFilter struct {
	Query string
	Role  string
}

type SystemEventFilter struct {
	Level string
	Kind  string
	Query string
}

type AuditFilter struct {
	Actor  string
	Action string
	Query  string
}
