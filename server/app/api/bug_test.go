package api

import (
	"fmt"
	"net/http"
	"testing"
)

func bugBody(title, submissionID string) map[string]any {
	return map[string]any{
		"title":         title,
		"category":      "bug",
		"severity":      "medium",
		"description":   "The ranked queue did not advance after the countdown finished.",
		"steps":         "1. Queue for 30s\n2. Wait for the bot",
		"expected":      "A game starts",
		"actual":        "The page stays on the queue",
		"page_url":      "https://example.test/play/ranked",
		"user_agent":    "test-browser",
		"app_version":   "test",
		"submission_id": submissionID,
	}
}

func TestBugReportGuestAndDuplicateSubmission(t *testing.T) {
	env := newAPIEnv(t)
	client := &apiClient{t: t, base: env.http.URL}

	body := bugBody("Guest report", "guest-report-1")
	resp, raw := client.do(http.MethodPost, "/bugs", body, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create bug status %d: %s", resp.StatusCode, raw)
	}
	created := decode[struct {
		Report struct {
			PublicID string `json:"public_id"`
			Status   string `json:"status"`
		} `json:"report"`
		Duplicate bool `json:"duplicate"`
	}](t, raw)
	if created.Report.PublicID == "" || created.Report.Status != "new" || created.Duplicate {
		t.Fatalf("unexpected create response: %s", raw)
	}

	resp, raw = client.do(http.MethodPost, "/bugs", body, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("duplicate bug status %d: %s", resp.StatusCode, raw)
	}
	duplicate := decode[struct {
		Report struct {
			PublicID string `json:"public_id"`
		} `json:"report"`
		Duplicate bool `json:"duplicate"`
	}](t, raw)
	if !duplicate.Duplicate || duplicate.Report.PublicID != created.Report.PublicID {
		t.Fatalf("duplicate did not reuse report: %s", raw)
	}

	resp, _ = client.do(http.MethodGet, "/bugs/mine", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("guest mine status %d", resp.StatusCode)
	}
}

func TestBugReportAuthenticatedListAndValidation(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")

	invalid := bugBody("x", "alice-invalid")
	invalid["description"] = "short"
	resp, raw := alice.do(http.MethodPost, "/bugs", invalid, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid bug status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = alice.do(http.MethodPost, "/bugs", bugBody("Ranked queue stuck", "alice-report-1"), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create bug status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = alice.do(http.MethodGet, "/bugs/mine", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mine status %d: %s", resp.StatusCode, raw)
	}
	list := decode[struct {
		Reports []struct {
			PublicID string `json:"public_id"`
			Title    string `json:"title"`
		} `json:"reports"`
	}](t, raw)
	if len(list.Reports) != 1 || list.Reports[0].Title != "Ranked queue stuck" {
		t.Fatalf("unexpected reports: %s", raw)
	}

	resp, raw = alice.do(http.MethodPost, "/bugs", bugBody("Ranked queue stuck", "alice-report-2"), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate title status %d: %s", resp.StatusCode, raw)
	}
}

func TestBugReportSubmissionIDCannotBeReusedByAnotherUser(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	body := bugBody("Alice report", "shared-submission-id")
	resp, raw := alice.do(http.MethodPost, "/bugs", body, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("alice create status %d: %s", resp.StatusCode, raw)
	}

	body["title"] = "Bob report"
	resp, raw = bob.do(http.MethodPost, "/bugs", body, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("bob reuse status %d: %s", resp.StatusCode, raw)
	}
}

func TestBugReportRateLimit(t *testing.T) {
	env := newAPIEnv(t)
	client := &apiClient{t: t, base: env.http.URL}
	for i := 0; i < 5; i++ {
		resp, raw := client.do(http.MethodPost, "/bugs", bugBody(fmt.Sprintf("Report %d", i), fmt.Sprintf("rate-%d", i)), nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("report %d status %d: %s", i, resp.StatusCode, raw)
		}
	}
	resp, raw := client.do(http.MethodPost, "/bugs", bugBody("Report six", "rate-6"), nil)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("rate limit status %d: %s", resp.StatusCode, raw)
	}
}
