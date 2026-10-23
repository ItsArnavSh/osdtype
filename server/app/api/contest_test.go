package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/services"
)

func TestContestAPICreateListEnrollAndCalendar(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "api-contest-alice")
	bob := env.client(t, "api-contest-bob")

	starts := time.Now().UTC().Add(3 * time.Minute)
	resp, raw := alice.do(http.MethodPost, "/contests", services.CreateContestRequest{
		Title:       "API event",
		Description: "# hello\n\nAn **api** contest.",
		BannerURL:   "https://example.com/banner.png",
		Accent:      "#ff5500",
		Visibility:  entity.ContestPublic,
		StartsAt:    starts,
		Duration:    30,
		Language:    "go",
		MaxPlayers:  4,
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create contest status %d: %s", resp.StatusCode, raw)
	}
	created := decode[services.ContestView](t, raw)
	if created.Code == "" || !created.IsHost || created.EntryCount != 1 {
		t.Fatalf("unexpected contest response: %s", raw)
	}

	// Public board and detail work without a session.
	publicResp, err := http.Get(env.http.URL + "/contests")
	if err != nil {
		t.Fatalf("get public contests: %v", err)
	}
	defer publicResp.Body.Close()
	if publicResp.StatusCode != http.StatusOK {
		t.Fatalf("public list status: %d", publicResp.StatusCode)
	}
	var list services.ContestListView
	if err := json.NewDecoder(publicResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode public list: %v", err)
	}
	if len(list.Contests) != 1 || list.Contests[0].Code != created.Code {
		t.Fatalf("created contest missing from public board: %+v", list)
	}

	detailResp, err := http.Get(env.http.URL + "/contests/" + created.Code)
	if err != nil {
		t.Fatalf("get contest detail: %v", err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusOK {
		t.Fatalf("contest detail status: %d", detailResp.StatusCode)
	}

	resp, raw = bob.do(http.MethodPost, "/contests/"+created.Code+"/enroll", map[string]string{}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("enroll status %d: %s", resp.StatusCode, raw)
	}
	joined := decode[services.ContestView](t, raw)
	if !joined.IsEnrolled || joined.EntryCount != 2 {
		t.Fatalf("enrollment failed: %s", raw)
	}

	calResp, err := http.Get(env.http.URL + "/contests/" + created.Code + "/calendar")
	if err != nil {
		t.Fatalf("calendar: %v", err)
	}
	defer calResp.Body.Close()
	if calResp.StatusCode != http.StatusOK {
		t.Fatalf("calendar status: %d", calResp.StatusCode)
	}
	if got := calResp.Header.Get("Content-Type"); got != "text/calendar; charset=utf-8" {
		t.Fatalf("calendar content type: %q", got)
	}

	// Scheduling in the past is a validation error, never an internal error.
	resp, raw = alice.do(http.MethodPost, "/contests", services.CreateContestRequest{
		Title:      "past event",
		Visibility: entity.ContestPublic,
		StartsAt:   time.Now().UTC().Add(-time.Minute),
		Duration:   30,
		Language:   "go",
		MaxPlayers: 4,
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("past schedule status %d: %s", resp.StatusCode, raw)
	}
}

func TestContestAPIPrivatePasswordGate(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "api-private-alice")
	bob := env.client(t, "api-private-bob")

	resp, raw := alice.do(http.MethodPost, "/contests", services.CreateContestRequest{
		Title:      "private api event",
		Visibility: entity.ContestPrivate,
		Password:   "secret",
		StartsAt:   time.Now().UTC().Add(3 * time.Minute),
		Duration:   90,
		Language:   "rust",
		MaxPlayers: 4,
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create private status %d: %s", resp.StatusCode, raw)
	}
	created := decode[services.ContestView](t, raw)
	if !created.RequiresPassword {
		t.Fatalf("private contest did not report a password gate: %+v", created)
	}

	resp, raw = bob.do(http.MethodPost, "/contests/"+created.Code+"/enroll", map[string]string{}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("passwordless enroll status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodPost, "/contests/"+created.Code+"/enroll", map[string]string{"password": "nope"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong password status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodPost, "/contests/"+created.Code+"/enroll", map[string]string{"password": "secret"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("private enroll status %d: %s", resp.StatusCode, raw)
	}
}
