package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/services"
)

func loginUserID(t *testing.T, client *apiClient) uint32 {
	t.Helper()
	resp, raw := client.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}
	state := decode[struct {
		User entity.UserView `json:"user"`
	}](t, raw)
	return state.User.ID
}

func createPractice(t *testing.T, client *apiClient, snippet string, events []entity.Keypress) entity.PracticeRunView {
	t.Helper()
	resp, raw := client.do(http.MethodPost, "/practice/runs", map[string]any{
		"language":     "go",
		"duration":     30,
		"duration_ms":  5000,
		"seed":         "42",
		"snippet":      snippet,
		"start_offset": 0,
		"final":        "package",
		"events":       events,
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create practice status %d: %s", resp.StatusCode, raw)
	}
	return decode[entity.PracticeRunView](t, raw)
}

func guestReplay(t *testing.T, env *apiEnv, gameID, userID uint32) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/replay/%d/%d", env.http.URL, gameID, userID))
	if err != nil {
		t.Fatalf("public replay: %v", err)
	}
	defer resp.Body.Close()
	var buf []byte
	if resp.StatusCode == http.StatusOK {
		var bundle entity.ReplayBundle
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&bundle); err != nil {
			t.Fatalf("decode public replay: %v", err)
		}
		buf, _ = json.Marshal(bundle)
	}
	return resp, buf
}

func TestAPIPracticeTemporarySaveAndPublicReplay(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	aliceID := loginUserID(t, alice)
	snippet := "package main\n\nfunc main() { println(\"hello\") }\n"
	events := []entity.Keypress{
		{Value: "p", Action: entity.KEYPRESS, TimeMS: 120},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 240},
		{Value: "c", Action: entity.KEYPRESS, TimeMS: 360},
		{Value: "k", Action: entity.KEYPRESS, TimeMS: 480},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 600},
		{Value: "g", Action: entity.KEYPRESS, TimeMS: 720},
		{Value: "e", Action: entity.KEYPRESS, TimeMS: 840},
	}
	runA := createPractice(t, alice, snippet, events)
	if !runA.Temporary || runA.URL == "" {
		t.Fatalf("expected temporary run with share url, got %+v", runA)
	}

	// Temporary runs are not history, but they are replayable by link.
	resp, raw := alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status %d: %s", resp.StatusCode, raw)
	}
	profile := decode[services.ProfileView](t, raw)
	if len(profile.History) != 0 {
		t.Fatalf("temporary practice leaked into history: %d entries", len(profile.History))
	}
	resp, raw = guestReplay(t, env, runA.ID, aliceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public replay before save status %d: %s", resp.StatusCode, raw)
	}
	bundle := decode[entity.ReplayBundle](t, raw)
	if bundle.Game.Mode != entity.GamePractice || !bundle.Game.Temporary {
		t.Fatalf("unexpected public game: %+v", bundle.Game)
	}
	if len(bundle.Runs) != 1 || len(bundle.Runs[0].Events) != len(events) {
		t.Fatalf("bad replay runs: %s", raw)
	}

	resp, raw = alice.do(http.MethodPost, fmt.Sprintf("/practice/runs/%d/save", runA.ID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save practice status %d: %s", resp.StatusCode, raw)
	}
	saved := decode[entity.PracticeRunView](t, raw)
	if saved.Temporary {
		t.Fatalf("save did not clear temporary: %s", raw)
	}

	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile after save status %d: %s", resp.StatusCode, raw)
	}
	profile = decode[services.ProfileView](t, raw)
	if len(profile.History) != 1 {
		t.Fatalf("saved practice missing from history: %s", raw)
	}
	if profile.History[0].Mode != entity.GamePractice || profile.History[0].UserID != aliceID {
		t.Fatalf("bad practice history row: %+v", profile.History[0])
	}

	// A newer temporary attempt replaces only the latest temporary run, not
	// the saved one, and still stays out of history until saved.
	runB := createPractice(t, alice, snippet, events)
	runC := createPractice(t, alice, snippet, events)
	resp, _ = guestReplay(t, env, runB.ID, aliceID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("replaced temporary replay status %d, want 404", resp.StatusCode)
	}
	resp, raw = guestReplay(t, env, runC.ID, aliceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("latest temporary replay status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status %d: %s", resp.StatusCode, raw)
	}
	profile = decode[services.ProfileView](t, raw)
	if len(profile.History) != 1 {
		t.Fatalf("temporary replacement changed history: %s", raw)
	}

	resp, raw = alice.do(http.MethodPost, fmt.Sprintf("/practice/runs/%d/save", runC.ID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save latest status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	profile = decode[services.ProfileView](t, raw)
	if len(profile.History) != 2 {
		t.Fatalf("expected two saved practice runs, got %s", raw)
	}
}

func TestAPIGameReplaysAreParticipantOnlyAndPubliclyShareable(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")
	charlie := env.client(t, "charlie")
	aliceID := loginUserID(t, alice)
	bobID := loginUserID(t, bob)

	now := time.Now().UTC()
	gameID := uint32(880000 + (now.UnixNano()/1000)%100000)
	game := entity.Game{
		ID:         gameID,
		Mode:       entity.GameRoom,
		Duration:   30,
		Language:   "go",
		Seed:       9,
		Snippet:    "package main\n\nfunc main() {}\n",
		Status:     entity.GameFinished,
		StartsAt:   &now,
		EndsAt:     &now,
		CreatedAt:  now,
		FinishedAt: &now,
	}
	players := []entity.GamePlayer{
		{UserID: aliceID, WPM: 100, Raw: 104, Accuracy: 0.96, Placement: 1, Finished: true, Acked: true},
		{UserID: bobID, WPM: 80, Raw: 82, Accuracy: 0.94, Placement: 2, Finished: true, Acked: true},
	}
	if err := env.db.CreateGame(context.Background(), &game, players); err != nil {
		t.Fatalf("create game: %v", err)
	}
	events, _ := json.Marshal([]entity.Keypress{
		{Value: "p", Action: entity.KEYPRESS, TimeMS: 100},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 200},
	})
	if err := env.db.SaveRecordings(context.Background(), []entity.Recording{
		{GameID: gameID, UserID: aliceID, Events: events, Final: "pa", CreatedAt: now},
	}); err != nil {
		t.Fatalf("save recordings: %v", err)
	}

	resp, raw := alice.do(http.MethodGet, fmt.Sprintf("/game/%d/replays", gameID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("game replays status %d: %s", resp.StatusCode, raw)
	}
	bundle := decode[entity.ReplayBundle](t, raw)
	if len(bundle.Runs) != 2 {
		t.Fatalf("expected both players in combined bundle: %s", raw)
	}
	var aliceEvents int
	for _, run := range bundle.Runs {
		if run.UserID == aliceID {
			aliceEvents = len(run.Events)
		}
	}
	if aliceEvents != 2 {
		t.Fatalf("alice events = %d, want 2", aliceEvents)
	}

	resp, raw = charlie.do(http.MethodGet, fmt.Sprintf("/game/%d/replays", gameID), nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-participant replays status %d, want 403: %s", resp.StatusCode, raw)
	}

	resp, raw = guestReplay(t, env, gameID, aliceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public game replay status %d: %s", resp.StatusCode, raw)
	}
	public := decode[entity.ReplayBundle](t, raw)
	if len(public.Runs) != 1 || public.Runs[0].UserID != aliceID {
		t.Fatalf("public replay did not isolate requested run: %s", raw)
	}
	if len(public.Game.Players) != 0 || len(public.Game.Leaderboard) != 0 || public.Game.MyResult != nil {
		t.Fatalf("public replay leaked other participant data: %s", raw)
	}
	resp, _ = guestReplay(t, env, gameID, bobID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public replay without recording status %d, want 404", resp.StatusCode)
	}
}

func TestAPIMutualFollowsBecomeFriends(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	for _, client := range []*apiClient{alice, bob} {
		target := "bob"
		if client == bob {
			target = "alice"
		}
		resp, raw := client.do(http.MethodPost, "/user/follow", map[string]string{"user": target}, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("follow %s status %d: %s", target, resp.StatusCode, raw)
		}
	}

	resp, raw := alice.do(http.MethodGet, "/user/friends", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("friends status %d: %s", resp.StatusCode, raw)
	}
	out := decode[struct {
		Friends []services.FriendView `json:"friends"`
	}](t, raw)
	if len(out.Friends) != 1 {
		t.Fatalf("expected one friend: %s", raw)
	}
	friend := out.Friends[0]
	if !friend.IsFriend || !friend.Follows || friend.User.Username != "bob" {
		t.Fatalf("bad mutual friend view: %+v", friend)
	}
}
