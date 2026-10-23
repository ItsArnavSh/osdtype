package entity

import "time"

const (
	BugStatusNew        = "new"
	BugStatusTriaged    = "triaged"
	BugStatusInProgress = "in_progress"
	BugStatusResolved   = "resolved"
	BugStatusClosed     = "closed"
	BugStatusWontFix    = "wontfix"
)

// BugReport is a user-submitted issue. The public view intentionally omits
// internal fields such as the submitter id, IP hash and admin notes.
type BugReport struct {
	ID uint32 `gorm:"primaryKey" json:"-"`

	PublicID     string  `gorm:"size:16;uniqueIndex;not null" json:"public_id"`
	UserID       *uint32 `gorm:"index" json:"-"`
	SubmissionID string  `gorm:"size:64;uniqueIndex" json:"-"`
	IPHash       string  `gorm:"size:64;index" json:"-"`

	Title       string `gorm:"size:200;not null" json:"title"`
	Category    string `gorm:"size:32;not null;index" json:"category"`
	Severity    string `gorm:"size:16;not null;index" json:"severity"`
	Description string `gorm:"type:text;not null" json:"description"`
	Steps       string `gorm:"type:text" json:"steps"`
	Expected    string `gorm:"type:text" json:"expected"`
	Actual      string `gorm:"type:text" json:"actual"`

	PageURL    string `gorm:"size:2048" json:"page_url"`
	UserAgent  string `gorm:"size:1024" json:"user_agent"`
	AppVersion string `gorm:"size:64" json:"app_version"`
	RequestID  string `gorm:"size:64;index" json:"request_id"`

	ContactEmail string `gorm:"size:254" json:"contact_email"`
	Status       string `gorm:"size:24;not null;default:new;index" json:"status"`
	AdminNotes   string `gorm:"type:text" json:"-"`

	CreatedAt  time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type BugReportView struct {
	PublicID     string     `json:"public_id"`
	Title        string     `json:"title"`
	Category     string     `json:"category"`
	Severity     string     `json:"severity"`
	Description  string     `json:"description"`
	Steps        string     `json:"steps"`
	Expected     string     `json:"expected"`
	Actual       string     `json:"actual"`
	PageURL      string     `json:"page_url"`
	UserAgent    string     `json:"user_agent"`
	AppVersion   string     `json:"app_version"`
	RequestID    string     `json:"request_id"`
	ContactEmail string     `json:"contact_email"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

func NewBugReportView(report BugReport) BugReportView {
	return BugReportView{
		PublicID:     report.PublicID,
		Title:        report.Title,
		Category:     report.Category,
		Severity:     report.Severity,
		Description:  report.Description,
		Steps:        report.Steps,
		Expected:     report.Expected,
		Actual:       report.Actual,
		PageURL:      report.PageURL,
		UserAgent:    report.UserAgent,
		AppVersion:   report.AppVersion,
		RequestID:    report.RequestID,
		ContactEmail: report.ContactEmail,
		Status:       report.Status,
		CreatedAt:    report.CreatedAt,
		UpdatedAt:    report.UpdatedAt,
		ResolvedAt:   report.ResolvedAt,
	}
}
