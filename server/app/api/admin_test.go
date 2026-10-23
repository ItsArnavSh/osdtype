package api

import (
	"net/http"
	"strconv"
	"testing"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/spf13/viper"
)

func TestAdminSurfaceIsHiddenAndPrimaryCanGrantAccess(t *testing.T) {
	env := newAPIEnv(t)
	guest := &apiClient{t: t, base: env.http.URL}
	alice := env.client(t, "alice")

	resp, _ := guest.do(http.MethodGet, "/admin/overview", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("guest admin status %d, want 404", resp.StatusCode)
	}
	resp, _ = alice.do(http.MethodGet, "/admin/overview", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("non-admin status %d, want 404", resp.StatusCode)
	}

	viper.Set("Admin.primary_username", "root")
	root := env.client(t, "root")
	resp, raw := root.do(http.MethodGet, "/admin/overview", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("primary admin status %d: %s", resp.StatusCode, raw)
	}
	overview := decode[struct {
		PrimaryAdmin string `json:"primary_admin"`
		ActiveHour   int64  `json:"active_hour"`
		Me           struct {
			Username  string `json:"username"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"me"`
	}](t, raw)
	if overview.PrimaryAdmin != "root" || overview.Me.Username != "root" || !overview.Me.IsPrimary {
		t.Fatalf("unexpected overview: %s", raw)
	}
	if overview.ActiveHour < 1 {
		t.Fatalf("activity middleware did not record the admin request: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/users", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin users status %d: %s", resp.StatusCode, raw)
	}
	users := decode[struct {
		Users []entity.AdminUserView `json:"users"`
	}](t, raw)
	var aliceID uint32
	for _, user := range users.Users {
		if user.Username == "alice" {
			aliceID = user.ID
		}
	}
	if aliceID == 0 {
		t.Fatalf("alice missing from admin users: %s", raw)
	}

	resp, raw = root.do(http.MethodPost, "/admin/users/"+itoa(aliceID)+"/admin", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("grant status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/admin/overview", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("granted admin status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = alice.do(http.MethodPost, "/admin/users/"+itoa(aliceID)+"/admin", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-primary grant status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = root.do(http.MethodDelete, "/admin/users/"+itoa(aliceID)+"/admin", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revoke status %d: %s", resp.StatusCode, raw)
	}
	resp, _ = alice.do(http.MethodGet, "/admin/overview", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("revoked admin status %d, want 404", resp.StatusCode)
	}
}

func TestAdminCanTriageBugReports(t *testing.T) {
	env := newAPIEnv(t)
	viper.Set("Admin.primary_username", "root")
	alice := env.client(t, "alice")
	root := env.client(t, "root")

	resp, raw := alice.do(http.MethodPost, "/bugs", bugBody("Admin triage target", "admin-triage-1"), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create bug status %d: %s", resp.StatusCode, raw)
	}
	created := decode[struct {
		Report struct {
			PublicID string `json:"public_id"`
		} `json:"report"`
	}](t, raw)

	resp, raw = root.do(http.MethodGet, "/admin/bugs?status=new", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin bugs status %d: %s", resp.StatusCode, raw)
	}
	list := decode[struct {
		Bugs []entity.AdminBugView `json:"bugs"`
	}](t, raw)
	if len(list.Bugs) != 1 || list.Bugs[0].PublicID != created.Report.PublicID {
		t.Fatalf("unexpected admin bugs: %s", raw)
	}
	if list.Bugs[0].Username != "alice" {
		t.Fatalf("missing submitter username: %s", raw)
	}

	resp, raw = root.do(http.MethodPatch, "/admin/bugs/"+created.Report.PublicID, map[string]any{
		"status":      entity.BugStatusResolved,
		"admin_notes": "fixed on the queue page",
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update bug status %d: %s", resp.StatusCode, raw)
	}
	updated := decode[struct {
		Bug entity.AdminBugView `json:"bug"`
	}](t, raw)
	if updated.Bug.Status != entity.BugStatusResolved || updated.Bug.AdminNotes == "" {
		t.Fatalf("unexpected updated bug: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/audits", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("audits status %d: %s", resp.StatusCode, raw)
	}
	audits := decode[struct {
		Audits []entity.AuditLog `json:"audits"`
	}](t, raw)
	if len(audits.Audits) == 0 || audits.Audits[0].Action != "bug.status" {
		t.Fatalf("missing bug triage audit: %s", raw)
	}
}

func TestAdminActivityHealthAndLogs(t *testing.T) {
	env := newAPIEnv(t)
	viper.Set("Admin.primary_username", "root")
	root := env.client(t, "root")

	// Make one authenticated request so the activity middleware has a row.
	resp, raw := root.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/activity?hours=3", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activity status %d: %s", resp.StatusCode, raw)
	}
	activity := decode[struct {
		Hours []entity.ActivityHourCount `json:"hours"`
	}](t, raw)
	if len(activity.Hours) != 3 {
		t.Fatalf("unexpected activity rows: %s", raw)
	}
	var total int64
	for _, row := range activity.Hours {
		total += row.Count
	}
	if total < 1 {
		t.Fatalf("activity was not recorded: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/health", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status %d: %s", resp.StatusCode, raw)
	}
	health := decode[struct {
		Current map[string]bool `json:"current"`
	}](t, raw)
	if !health.Current["postgres"] || !health.Current["redis"] {
		t.Fatalf("unexpected current health: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/logs", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logs status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/events", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("events status %d: %s", resp.StatusCode, raw)
	}
}

func TestAdminListsArePaginatedAndSearchable(t *testing.T) {
	env := newAPIEnv(t)
	viper.Set("Admin.primary_username", "root")
	alice := env.client(t, "alice")
	root := env.client(t, "root")

	for i, title := range []string{"Alpha overflow", "Beta queue", "Gamma render"} {
		resp, raw := alice.do(http.MethodPost, "/bugs", bugBody(title, "page-bug-"+strconv.Itoa(i)), nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create bug status %d: %s", resp.StatusCode, raw)
		}
	}

	resp, raw := root.do(http.MethodGet, "/admin/bugs?limit=2&offset=0", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bugs page one status %d: %s", resp.StatusCode, raw)
	}
	first := decode[struct {
		Bugs []entity.AdminBugView `json:"bugs"`
		Page struct {
			Total   int64 `json:"total"`
			Limit   int   `json:"limit"`
			Offset  int   `json:"offset"`
			Next    *int  `json:"next"`
			HasMore bool  `json:"has_more"`
		} `json:"page"`
	}](t, raw)
	if len(first.Bugs) != 2 || first.Page.Total != 3 || first.Page.Limit != 2 ||
		first.Page.Offset != 0 || first.Page.Next == nil || *first.Page.Next != 2 || !first.Page.HasMore {
		t.Fatalf("unexpected first bug page: %s", raw)
	}
	resp, raw = root.do(http.MethodGet, "/admin/bugs?limit=2&offset=2", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bugs page two status %d: %s", resp.StatusCode, raw)
	}
	second := decode[struct {
		Bugs []entity.AdminBugView `json:"bugs"`
		Page struct {
			Next    *int `json:"next"`
			HasMore bool `json:"has_more"`
		} `json:"page"`
	}](t, raw)
	if len(second.Bugs) != 1 || second.Page.Next != nil || second.Page.HasMore {
		t.Fatalf("unexpected second bug page: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/bugs?q=beta", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bug search status %d: %s", resp.StatusCode, raw)
	}
	search := decode[struct {
		Bugs []entity.AdminBugView `json:"bugs"`
		Page struct {
			Total int64 `json:"total"`
		} `json:"page"`
	}](t, raw)
	if len(search.Bugs) != 1 || search.Page.Total != 1 || search.Bugs[0].Title != "Beta queue" {
		t.Fatalf("unexpected bug search: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/users?limit=1&offset=0", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users page status %d: %s", resp.StatusCode, raw)
	}
	users := decode[struct {
		Users []entity.AdminUserView `json:"users"`
		Page  struct {
			Total   int64 `json:"total"`
			Next    *int  `json:"next"`
			HasMore bool  `json:"has_more"`
		} `json:"page"`
	}](t, raw)
	if len(users.Users) != 1 || users.Page.Total < 2 || users.Page.Next == nil || !users.Page.HasMore {
		t.Fatalf("unexpected user page: %s", raw)
	}
	resp, raw = root.do(http.MethodGet, "/admin/users?q=alice", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("user search status %d: %s", resp.StatusCode, raw)
	}
	foundUsers := decode[struct {
		Users []entity.AdminUserView `json:"users"`
	}](t, raw)
	if len(foundUsers.Users) != 1 || foundUsers.Users[0].Username != "alice" {
		t.Fatalf("unexpected user search: %s", raw)
	}
}

func TestAdminLogsSearchByUserAndLevel(t *testing.T) {
	env := newAPIEnv(t)
	viper.Set("Admin.primary_username", "root")
	alice := env.client(t, "alice")
	root := env.client(t, "root")

	resp, raw := alice.do(http.MethodGet, "/auth/me", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status %d: %s", resp.StatusCode, raw)
	}
	me := decode[struct {
		User entity.UserView `json:"user"`
	}](t, raw)

	marker := "admin-log-search-marker"
	utils.Logs().Add(utils.LogEntry{
		Level: "info", Message: marker + " alice typed",
		Fields: map[string]any{"user_id": me.User.ID, "username": "alice"},
	})
	utils.Logs().Add(utils.LogEntry{
		Level: "error", Message: marker + " alice failed",
		Fields: map[string]any{"user_id": me.User.ID, "username": "alice"},
	})
	utils.Logs().Add(utils.LogEntry{
		Level: "error", Message: marker + " someone else",
		Fields: map[string]any{"user_id": 999999, "username": "other"},
	})

	resp, raw = root.do(http.MethodGet, "/admin/logs?q="+marker+"&user=alice&limit=1&offset=0", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("log search status %d: %s", resp.StatusCode, raw)
	}
	logs := decode[struct {
		Logs []utils.LogEntry `json:"logs"`
		Page struct {
			Total   int64 `json:"total"`
			Next    *int  `json:"next"`
			HasMore bool  `json:"has_more"`
		} `json:"page"`
	}](t, raw)
	if len(logs.Logs) != 1 || logs.Page.Total != 2 || logs.Page.Next == nil || !logs.Page.HasMore {
		t.Fatalf("unexpected log search page: %s", raw)
	}

	resp, raw = root.do(http.MethodGet, "/admin/logs?q="+marker+"&user=alice&level=error", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("log level filter status %d: %s", resp.StatusCode, raw)
	}
	filtered := decode[struct {
		Logs []utils.LogEntry `json:"logs"`
		Page struct {
			Total int64 `json:"total"`
		} `json:"page"`
	}](t, raw)
	if len(filtered.Logs) != 1 || filtered.Page.Total != 1 || filtered.Logs[0].Level != "error" {
		t.Fatalf("unexpected log level filter: %s", raw)
	}
}

func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
