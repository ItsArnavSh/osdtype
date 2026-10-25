package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"osdtyp/app/entity"
	"osdtyp/app/services"
)

func TestAPIProfileSettingsVisibilityAndAnonymous(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	resp, raw := alice.do(http.MethodPatch, "/user/settings", map[string]any{
		"display_name":       "Alice A",
		"bio":                "types in the dark",
		"profile_visibility": "anonymous",
		"do_not_disturb":     true,
		"hide_online":        true,
		"share_runs":         "pinned",
		"share_ranked":       true,
		"share_rooms":        false,
		"share_practice":     true,
		"share_contests":     false,
		"hide_private_rooms": true,
		"share_replays":      false,
		"save_run_data":      true,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update settings status %d: %s", resp.StatusCode, raw)
	}
	settings := decode[services.ProfileSettingsView](t, raw)
	if settings.DisplayName != "Alice A" || settings.Bio != "types in the dark" {
		t.Fatalf("identity not saved: %+v", settings)
	}
	if settings.ProfileVisibility != entity.ProfileAnonymous || !settings.DoNotDisturb || !settings.HideOnline {
		t.Fatalf("privacy switches not saved: %+v", settings)
	}
	if settings.ShareRuns != entity.RunSharingPinned || settings.ShareRooms || settings.ShareContests {
		t.Fatalf("run switches not saved: %+v", settings)
	}
	if settings.RetentionLimit != entity.MatchRetentionLimit || !settings.RetentionKeepsBest {
		t.Fatalf("retention metadata missing: %+v", settings)
	}

	// Another player sees a masked profile card, not the saved identity.
	resp, raw = bob.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous profile status %d: %s", resp.StatusCode, raw)
	}
	public := decode[services.ProfileView](t, raw)
	if !public.Anonymous || public.User.Username != "anonymous" || public.User.DisplayName != "anonymous" {
		t.Fatalf("profile was not anonymised: %+v", public)
	}
	if public.User.Bio != "" || public.User.AvatarURL != "" {
		t.Fatalf("anonymous profile leaked identity fields: %+v", public.User)
	}
	if public.ReplaysVisible {
		t.Fatal("replays should be hidden when share_replays is off")
	}

	// The owner still sees their real identity and can see their own replays.
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("own profile status %d: %s", resp.StatusCode, raw)
	}
	own := decode[services.ProfileView](t, raw)
	if own.Anonymous || own.User.Username != "alice" || own.User.DisplayName != "Alice A" {
		t.Fatalf("owner profile was masked: %+v", own)
	}
	if !own.ReplaysVisible {
		t.Fatal("owner should always be able to open their own replays")
	}

	// Private profiles are indistinguishable from a missing handle.
	if resp, raw = alice.do(http.MethodPatch, "/user/settings", map[string]string{
		"profile_visibility": "private",
	}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("private setting status %d: %s", resp.StatusCode, raw)
	}
	if resp, _ = bob.do(http.MethodGet, "/user/alice/profile", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("private profile status %d, want 404", resp.StatusCode)
	}
	if resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("owner private profile status %d: %s", resp.StatusCode, raw)
	}
}

func TestAPIDoNotDisturbBlocksRoomInvites(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	resp, raw := bob.do(http.MethodPatch, "/user/settings", map[string]bool{
		"do_not_disturb": true,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dnd setting status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodPost, "/room/create", map[string]any{
		"name": "quiet room", "is_public": true, "max_players": 4, "duration": 30, "language": "go",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create room status %d: %s", resp.StatusCode, raw)
	}
	room := decode[entity.RoomView](t, raw)
	resp, raw = alice.do(http.MethodPost, "/room/"+room.Code+"/invite", map[string]string{"username": "bob"}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("dnd invite status %d, want 403: %s", resp.StatusCode, raw)
	}
	if !strings.Contains(string(raw), "do not disturb") {
		t.Fatalf("unexpected dnd error: %s", raw)
	}
}

func TestAPINoSaveRunDataWritesNoPracticeRun(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	aliceID := loginUserID(t, alice)

	resp, raw := alice.do(http.MethodPatch, "/user/settings", map[string]bool{
		"save_run_data": false,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("no-save setting status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}
	state := decode[struct {
		User entity.UserView `json:"user"`
	}](t, raw)
	if state.User.SaveRunData == nil || *state.User.SaveRunData {
		t.Fatalf("self state did not expose save_run_data=false: %+v", state.User)
	}

	snippet := "package main\n"
	events := []entity.Keypress{
		{Value: "p", Action: entity.KEYPRESS, TimeMS: 100},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 200},
	}
	run := createPractice(t, alice, snippet, events)
	if run.ID != 0 || run.URL != "" {
		t.Fatalf("no-save practice was persisted: %+v", run)
	}

	// Neither a shared nor an owner replay link should resolve.
	resp, raw = guestReplay(t, env, run.ID, aliceID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("no-save public replay status %d, want 404: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/replay/0/"+uintString(aliceID), nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("owner no-save replay status %d, want 404: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status %d: %s", resp.StatusCode, raw)
	}
	profile := decode[services.ProfileView](t, raw)
	if len(profile.History) != 0 || len(profile.Pinned) != 0 {
		t.Fatalf("no-save run leaked into profile: %+v", profile)
	}
}

func TestAPIReplaySharingSwitchesAndExport(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	aliceID := loginUserID(t, alice)

	run := createPractice(t, alice, "package export\n", []entity.Keypress{
		{Value: "p", Action: entity.KEYPRESS, TimeMS: 100},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 200},
	})
	if resp, raw := alice.do(http.MethodPost, "/practice/runs/"+uintString(run.ID)+"/save", nil, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("save practice status %d: %s", resp.StatusCode, raw)
	}

	// Sharing off blocks direct replay links, even though the run is saved.
	if resp, raw := alice.do(http.MethodPatch, "/user/settings", map[string]any{
		"share_replays": false,
	}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("share off status %d: %s", resp.StatusCode, raw)
	}
	if resp, _ := guestReplay(t, env, run.ID, aliceID); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("shared replay off status %d, want 404", resp.StatusCode)
	}
	if resp, raw := alice.do(http.MethodPatch, "/user/settings", map[string]any{
		"share_replays": true,
		"share_runs":    "none",
	}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("share runs none status %d: %s", resp.StatusCode, raw)
	}
	if resp, _ := guestReplay(t, env, run.ID, aliceID); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("no-share replay status %d, want 404", resp.StatusCode)
	}
	if resp, raw := alice.do(http.MethodPatch, "/user/settings", map[string]any{
		"share_runs": "all",
	}, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("share runs all status %d: %s", resp.StatusCode, raw)
	}
	if resp, raw := guestReplay(t, env, run.ID, aliceID); resp.StatusCode != http.StatusOK {
		t.Fatalf("shared replay status %d: %s", resp.StatusCode, raw)
	}

	resp, raw := alice.do(http.MethodGet, "/user/export", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export status %d: %s", resp.StatusCode, raw)
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("export is not an attachment: %q", resp.Header.Get("Content-Disposition"))
	}
	var exported services.UserDataExport
	if err := json.Unmarshal(raw, &exported); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if exported.Account.Username != "alice" || exported.Settings.Username != "alice" {
		t.Fatalf("export missing account: %+v", exported)
	}
	if len(exported.History) != 1 || len(exported.Recordings) != 1 {
		t.Fatalf("export did not include run data: %+v", exported)
	}
	if exported.Retention.UnpinnedRuns != entity.MatchRetentionLimit || !exported.Retention.BestKept {
		t.Fatalf("export retention metadata missing: %+v", exported.Retention)
	}

	resp, raw = alice.do(http.MethodDelete, "/user/runs", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete history status %d: %s", resp.StatusCode, raw)
	}
	deleted := decode[struct {
		Deleted    int  `json:"deleted"`
		PinnedKept bool `json:"pinned_kept"`
		BestKept   bool `json:"best_kept"`
	}](t, raw)
	if !deleted.PinnedKept || !deleted.BestKept {
		t.Fatalf("delete history response lost keepsakes: %+v", deleted)
	}
}
