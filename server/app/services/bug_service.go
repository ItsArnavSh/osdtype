package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/entity"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const (
	bugCategoryBug         = "bug"
	bugCategoryUI          = "ui"
	bugCategoryGameplay    = "gameplay"
	bugCategoryPerformance = "performance"
	bugCategoryAccount     = "account"
	bugCategoryContest     = "contest"
	bugCategoryOther       = "other"

	bugSeverityLow      = "low"
	bugSeverityMedium   = "medium"
	bugSeverityHigh     = "high"
	bugSeverityCritical = "critical"

	bugRateLimitPerHour = 5
	bugRateLimitPerDay  = 20
	bugDuplicateWindow  = 10 * time.Minute
)

var bugCategories = map[string]bool{
	bugCategoryBug:         true,
	bugCategoryUI:          true,
	bugCategoryGameplay:    true,
	bugCategoryPerformance: true,
	bugCategoryAccount:     true,
	bugCategoryContest:     true,
	bugCategoryOther:       true,
}

var bugSeverities = map[string]bool{
	bugSeverityLow:      true,
	bugSeverityMedium:   true,
	bugSeverityHigh:     true,
	bugSeverityCritical: true,
}

// BugReportInput is the JSON body accepted by POST /bugs.
type BugReportInput struct {
	Title        string `json:"title"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	Description  string `json:"description"`
	Steps        string `json:"steps"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	PageURL      string `json:"page_url"`
	UserAgent    string `json:"user_agent"`
	AppVersion   string `json:"app_version"`
	RequestID    string `json:"request_id"`
	ContactEmail string `json:"contact_email"`
	SubmissionID string `json:"submission_id"`
	// Website is a honeypot. It must stay empty.
	Website string `json:"website"`
}

func normalizeBugInput(in *BugReportInput) {
	in.Title = strings.TrimSpace(in.Title)
	in.Category = strings.ToLower(strings.TrimSpace(in.Category))
	in.Severity = strings.ToLower(strings.TrimSpace(in.Severity))
	in.Description = strings.TrimSpace(in.Description)
	in.Steps = strings.TrimSpace(in.Steps)
	in.Expected = strings.TrimSpace(in.Expected)
	in.Actual = strings.TrimSpace(in.Actual)
	in.PageURL = strings.TrimSpace(in.PageURL)
	in.UserAgent = strings.TrimSpace(in.UserAgent)
	in.AppVersion = strings.TrimSpace(in.AppVersion)
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.ContactEmail = strings.TrimSpace(in.ContactEmail)
	in.SubmissionID = strings.TrimSpace(in.SubmissionID)
	in.Website = strings.TrimSpace(in.Website)
}

func validateBugInput(in BugReportInput) error {
	if in.Website != "" {
		return errf(ErrBadRequest, "invalid submission")
	}
	if len(in.Title) < 3 || len(in.Title) > 160 {
		return errf(ErrBadRequest, "title must be between 3 and 160 characters")
	}
	if !bugCategories[in.Category] {
		return errf(ErrBadRequest, "unknown bug category")
	}
	if !bugSeverities[in.Severity] {
		return errf(ErrBadRequest, "unknown bug severity")
	}
	if len(in.Description) < 10 || len(in.Description) > 8000 {
		return errf(ErrBadRequest, "description must be between 10 and 8000 characters")
	}
	if len(in.Steps) > 4000 || len(in.Expected) > 4000 || len(in.Actual) > 4000 {
		return errf(ErrBadRequest, "steps, expected and actual must be at most 4000 characters each")
	}
	if len(in.PageURL) > 2048 || len(in.UserAgent) > 1024 || len(in.AppVersion) > 64 || len(in.RequestID) > 64 {
		return errf(ErrBadRequest, "one or more context fields are too long")
	}
	if len(in.ContactEmail) > 254 || (in.ContactEmail != "" && !strings.Contains(in.ContactEmail, "@")) {
		return errf(ErrBadRequest, "contact email is invalid")
	}
	if len(in.SubmissionID) > 64 {
		return errf(ErrBadRequest, "submission id is too long")
	}
	for _, r := range in.SubmissionID {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return errf(ErrBadRequest, "submission id contains invalid characters")
		}
	}
	return nil
}

func newPublicBugID() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("BUG-%d", time.Now().UTC().UnixNano()%1_000_000_000)
	}
	return "BUG-" + strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func newSubmissionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(b)
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip)))
	return hex.EncodeToString(sum[:])
}

func (s *ServiceLayer) bugRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	rdb := s.rdb()
	if rdb == nil || rdb.R == nil {
		return true, nil
	}
	n, err := rdb.R.Incr(ctx, "bugs:rate:"+key).Result()
	if err != nil {
		return true, err
	}
	if n == 1 {
		_ = rdb.R.Expire(ctx, "bugs:rate:"+key, window).Err()
	}
	return n <= limit, nil
}

// CreateBugReport stores a report and returns duplicate=true when the same
// client submission id was already accepted.
func (s *ServiceLayer) CreateBugReport(ctx context.Context, in BugReportInput, userID *uint32, ip string) (*entity.BugReport, bool, error) {
	normalizeBugInput(&in)
	if err := validateBugInput(in); err != nil {
		return nil, false, err
	}
	if in.SubmissionID == "" {
		in.SubmissionID = newSubmissionID()
	}

	ipHash := hashIP(ip)
	existing, err := s.db.GetBugReportBySubmissionID(ctx, in.SubmissionID)
	if err == nil && existing != nil {
		sameOwner := existing.IPHash == ipHash && existing.UserID == nil && userID == nil
		if existing.UserID != nil && userID != nil && *existing.UserID == *userID {
			sameOwner = true
		}
		if sameOwner {
			return existing, true, nil
		}
		return nil, false, errf(ErrConflict, "submission id already used")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if ok, err := s.bugRateLimit(ctx, "ip:"+ipHash, bugRateLimitPerHour, time.Hour); err != nil {
		s.logger.Warnw("bug report rate limit unavailable", "error", err)
	} else if !ok {
		return nil, false, errf(ErrRateLimited, "too many bug reports; try again later")
	}
	if userID != nil {
		if ok, err := s.bugRateLimit(ctx, "user:"+strconv.FormatUint(uint64(*userID), 10), bugRateLimitPerDay, 24*time.Hour); err != nil {
			s.logger.Warnw("bug report user rate limit unavailable", "error", err)
		} else if !ok {
			return nil, false, errf(ErrRateLimited, "too many bug reports; try again later")
		}
	}

	recent, err := s.db.CountRecentBugReports(ctx, ipHash, in.Title, time.Now().UTC().Add(-bugDuplicateWindow))
	if err == nil && recent > 0 {
		return nil, false, errf(ErrConflict, "a similar report was submitted recently")
	}
	if err != nil {
		s.logger.Warnw("bug report duplicate check failed", "error", err)
	}

	report := &entity.BugReport{
		PublicID:     newPublicBugID(),
		UserID:       userID,
		SubmissionID: in.SubmissionID,
		IPHash:       ipHash,
		Title:        in.Title,
		Category:     in.Category,
		Severity:     in.Severity,
		Description:  in.Description,
		Steps:        in.Steps,
		Expected:     in.Expected,
		Actual:       in.Actual,
		PageURL:      in.PageURL,
		UserAgent:    in.UserAgent,
		AppVersion:   in.AppVersion,
		RequestID:    in.RequestID,
		ContactEmail: in.ContactEmail,
		Status:       entity.BugStatusNew,
	}
	if err := s.db.CreateBugReport(ctx, report); err != nil {
		return nil, false, err
	}

	if webhook := strings.TrimSpace(viper.GetString("BugReport.webhook_url")); webhook != "" {
		go s.notifyBugWebhook(webhook, *report)
	}
	s.logger.Infow("bug report submitted", "bug_id", report.PublicID, "category", report.Category, "severity", report.Severity)
	return report, false, nil
}

func (s *ServiceLayer) ListMyBugReports(ctx context.Context, userID uint32, limit int) ([]entity.BugReport, error) {
	return s.db.ListBugReportsForUser(ctx, userID, limit)
}

func (s *ServiceLayer) notifyBugWebhook(webhook string, report entity.BugReport) {
	description := report.Description
	if len(description) > 500 {
		description = description[:500] + "…"
	}
	message := fmt.Sprintf("**%s** [%s] %s\n%s\n%s",
		report.PublicID, strings.ToUpper(report.Severity), report.Title, report.PageURL, description)
	payload := map[string]string{"content": message, "text": message}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		s.logger.Warnw("bug webhook request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		s.logger.Warnw("bug webhook delivery failed", "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warnw("bug webhook rejected", "status", resp.StatusCode)
	}
}
