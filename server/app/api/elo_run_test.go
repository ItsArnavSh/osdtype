package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"osdtyp/app/entity"
	"osdtyp/app/services"
)

func TestAPIEloStartsAtFourHundredAndHasNoTierFields(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")

	resp, raw := alice.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}
	state := decode[struct {
		User map[string]any `json:"user"`
	}](t, raw)
	if got := state.User["rating"]; got != float64(entity.StartingElo) {
		t.Fatalf("starting rating = %v, want %d", got, entity.StartingElo)
	}
	if got := state.User["peak_rating"]; got != float64(entity.StartingElo) {
		t.Fatalf("starting peak = %v, want %d", got, entity.StartingElo)
	}
	ratings, ok := state.User["ratings"].(map[string]any)
	if !ok {
		t.Fatalf("per-clock ratings missing: %s", raw)
	}
	for _, duration := range entity.RankedDurations {
		clock, ok := ratings[fmt.Sprint(duration)].(map[string]any)
		if !ok {
			t.Fatalf("missing %ds rating: %s", duration, raw)
		}
		if got := clock["rating"]; got != float64(entity.StartingElo) {
			t.Fatalf("%ds starting rating = %v, want %d", duration, got, entity.StartingElo)
		}
	}
	if got := state.User["best_duration"]; got != float64(90) {
		t.Fatalf("best_duration = %v, want 90 for equal ladders", got)
	}
	if _, ok := state.User["rank"]; ok {
		t.Fatalf("tier rank still present in user view: %s", raw)
	}
	if _, ok := state.User["next"]; ok {
		t.Fatalf("rank progression still present in user view: %s", raw)
	}
}

func TestAPILeaderboardReturnsTopAndExactNearbyWindow(t *testing.T) {
	env := newAPIEnv(t)
	const playerCount = 10
	clients := make([]*apiClient, 0, playerCount)
	ids := make([]uint32, 0, playerCount)
	for i := range playerCount {
		client := env.client(t, fmt.Sprintf("player%02d", i))
		clients = append(clients, client)
		ids = append(ids, loginUserID(t, client))
	}

	// Give everyone a completed ranked record with a unique Elo so the
	// ordering, exact position and nearby window are deterministic.
	for i, id := range ids {
		user, err := env.db.GetUser(id)
		if err != nil {
			t.Fatalf("get user %d: %v", id, err)
		}
		user.SetRatingFor(entity.RankedDuration90, 1000-i*10)
		user.GamesPlayed90 = 10 - i
		user.Wins90 = 3
		// Stagger games played to prove the tiebreaker is deterministic too.
		user.GamesPlayed = 10 - i
		user.Wins = 3
		user.RefreshHighestElo()
		if err := env.db.SaveUser(context.Background(), user); err != nil {
			t.Fatalf("save user %d: %v", id, err)
		}
	}

	// The viewer is rank 5. The public ladder is capped at the top five and
	// their own window covers positions 1..10.
	viewer := clients[4]
	resp, raw := viewer.do(http.MethodGet, "/ranked/leaderboard?limit=50", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", resp.StatusCode, raw)
	}
	view := decode[services.LeaderboardView](t, raw)
	if view.MyPosition != 5 {
		t.Fatalf("my_position = %d, want 5: %s", view.MyPosition, raw)
	}
	if len(view.Leaderboard) != 5 {
		t.Fatalf("leaderboard has %d rows, want top 5", len(view.Leaderboard))
	}
	for i, row := range view.Leaderboard {
		if row.Position != int64(i+1) {
			t.Fatalf("top row %d position = %d", i, row.Position)
		}
	}
	if view.Duration != 90 {
		t.Fatalf("leaderboard duration = %d, want 90", view.Duration)
	}
	if view.Leaderboard[0].User.ID != ids[0] || view.Leaderboard[0].Rating != 1000 {
		t.Fatalf("top row is not the highest-rated player: %+v", view.Leaderboard[0])
	}
	if len(view.Nearby) != playerCount {
		t.Fatalf("nearby rows = %d, want %d: %s", len(view.Nearby), playerCount, raw)
	}
	if view.Nearby[0].Position != 1 || view.Nearby[len(view.Nearby)-1].Position != 10 {
		t.Fatalf("nearby window = %d..%d, want 1..10", view.Nearby[0].Position, view.Nearby[len(view.Nearby)-1].Position)
	}
	if view.Nearby[4].User.ID != ids[4] {
		t.Fatalf("nearby center is %d, want viewer %d", view.Nearby[4].User.ID, ids[4])
	}

	// At the top edge the window is clamped rather than wrapping.
	topViewer := clients[0]
	resp, raw = topViewer.do(http.MethodGet, "/ranked/leaderboard", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("top viewer leaderboard status %d: %s", resp.StatusCode, raw)
	}
	edge := decode[services.LeaderboardView](t, raw)
	if edge.MyPosition != 1 {
		t.Fatalf("top viewer position = %d, want 1", edge.MyPosition)
	}
	if len(edge.Nearby) != 6 || edge.Nearby[0].Position != 1 || edge.Nearby[5].Position != 6 {
		t.Fatalf("top edge window = %+v, want positions 1..6", edge.Nearby)
	}

	// Search is clock-scoped and reports exact ladder positions.
	resp, raw = viewer.do(http.MethodGet, "/ranked/leaderboard/search?duration=90&q=player07", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leaderboard search status %d: %s", resp.StatusCode, raw)
	}
	found := decode[services.LeaderboardSearchView](t, raw)
	if len(found.Results) != 1 || found.Results[0].Position != 8 || found.Results[0].Rating != 930 {
		t.Fatalf("search result = %+v: %s", found.Results, raw)
	}

	resp, raw = viewer.do(http.MethodGet, "/ranked/leaderboard/search?duration=45&q=player", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid search duration status %d, want 400: %s", resp.StatusCode, raw)
	}
}

func TestAPILeaderboardsAreIndependentPerClock(t *testing.T) {
	env := newAPIEnv(t)
	aliceClient := env.client(t, "alice")
	bobClient := env.client(t, "bob")
	aliceID := loginUserID(t, aliceClient)
	bobID := loginUserID(t, bobClient)

	alice, err := env.db.GetUser(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	alice.SetRatingFor(entity.RankedDuration30, 900)
	alice.GamesPlayed30, alice.Wins30 = 4, 3
	alice.SetRatingFor(entity.RankedDuration90, 500)
	alice.GamesPlayed90 = 2
	alice.GamesPlayed, alice.Wins = 6, 3
	if err := env.db.SaveUser(context.Background(), alice); err != nil {
		t.Fatal(err)
	}

	bob, err := env.db.GetUser(bobID)
	if err != nil {
		t.Fatal(err)
	}
	bob.SetRatingFor(entity.RankedDuration30, 700)
	bob.GamesPlayed30 = 1
	bob.SetRatingFor(entity.RankedDuration90, 800)
	bob.GamesPlayed90 = 1
	bob.GamesPlayed = 2
	if err := env.db.SaveUser(context.Background(), bob); err != nil {
		t.Fatal(err)
	}

	// Alice's highest clock is 30s, which the hub-facing user view exposes.
	resp, raw := aliceClient.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}
	state := decode[struct {
		User struct {
			Rating       int            `json:"rating"`
			BestDuration int            `json:"best_duration"`
			Ratings      map[string]any `json:"ratings"`
		} `json:"user"`
	}](t, raw)
	if state.User.Rating != 900 || state.User.BestDuration != 30 {
		t.Fatalf("highest Elo/hub clock = %d/%d, want 900/30", state.User.Rating, state.User.BestDuration)
	}

	check := func(duration, wantTop int, wantTopID uint32, wantPos int64, wantRows int) {
		t.Helper()
		resp, raw := bobClient.do(http.MethodGet,
			fmt.Sprintf("/ranked/leaderboard?duration=%d&limit=50", duration), nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%ds leaderboard status %d: %s", duration, resp.StatusCode, raw)
		}
		view := decode[services.LeaderboardView](t, raw)
		if view.Duration != duration || len(view.Leaderboard) != wantRows {
			t.Fatalf("%ds leaderboard = duration %d, rows %d, want %d: %s",
				duration, view.Duration, len(view.Leaderboard), wantRows, raw)
		}
		if wantRows > 0 && (view.Leaderboard[0].Rating != wantTop || view.Leaderboard[0].User.ID != wantTopID) {
			t.Fatalf("%ds top row = %+v, want rating %d user %d", duration, view.Leaderboard[0], wantTop, wantTopID)
		}
		if view.MyPosition != wantPos {
			t.Fatalf("%ds my_position = %d, want %d: %s", duration, view.MyPosition, wantPos, raw)
		}
	}

	// 30s is Alice's best clock. 90s belongs to Bob. Nobody has played 300s.
	check(30, 900, aliceID, 2, 2)
	check(90, 800, bobID, 1, 2)
	check(300, 0, 0, 0, 0)

	resp, raw = bobClient.do(http.MethodGet, "/ranked/leaderboard?duration=45", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid duration status %d, want 400: %s", resp.StatusCode, raw)
	}
}

func TestAPILeaderboardEmptyAndUnrankedViewer(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")

	resp, raw := alice.do(http.MethodGet, "/ranked/leaderboard", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leaderboard status %d: %s", resp.StatusCode, raw)
	}
	view := decode[services.LeaderboardView](t, raw)
	if len(view.Leaderboard) != 0 || len(view.Nearby) != 0 || view.MyPosition != 0 {
		t.Fatalf("fresh ladder should be empty and viewer unranked: %s", raw)
	}
}

func TestAPIRunMetadataPinsEditsAndValidatesOwnership(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	snippet := "package main\n\nfunc main() { println(\"run\") }\n"
	events := []entity.Keypress{
		{Value: "p", Action: entity.KEYPRESS, TimeMS: 100},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 200},
		{Value: "c", Action: entity.KEYPRESS, TimeMS: 300},
		{Value: "k", Action: entity.KEYPRESS, TimeMS: 400},
		{Value: "a", Action: entity.KEYPRESS, TimeMS: 500},
		{Value: "g", Action: entity.KEYPRESS, TimeMS: 600},
		{Value: "e", Action: entity.KEYPRESS, TimeMS: 700},
	}
	run := createPractice(t, alice, snippet, events)
	resp, raw := alice.do(http.MethodPost, fmt.Sprintf("/practice/runs/%d/save", run.ID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save practice status %d: %s", resp.StatusCode, raw)
	}

	path := fmt.Sprintf("/game/%d/run", run.ID)
	resp, raw = alice.do(http.MethodPatch, path, map[string]any{
		"name":        "focus sprint",
		"description": "clean keys, no corrections",
		"pinned":      true,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update run status %d: %s", resp.StatusCode, raw)
	}

	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status %d: %s", resp.StatusCode, raw)
	}
	profile := decode[services.ProfileView](t, raw)
	if len(profile.History) != 1 {
		t.Fatalf("history = %d, want 1: %s", len(profile.History), raw)
	}
	got := profile.History[0]
	if got.RunName != "focus sprint" || got.RunDescription != "clean keys, no corrections" || !got.Pinned || got.PinnedAt == nil {
		t.Fatalf("run metadata not persisted: %+v", got)
	}
	if len(profile.Pinned) != 1 || profile.Pinned[0].GameID != run.ID {
		t.Fatalf("pinned list missing run: %s", raw)
	}
	if profile.Best == nil || profile.Best.GameID != run.ID || profile.TopWPM <= 0 {
		t.Fatalf("best/top-wpm missing: %s", raw)
	}

	// Editing one field must not clobber the other.
	resp, raw = alice.do(http.MethodPatch, path, map[string]any{"description": "edited description"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit description status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	profile = decode[services.ProfileView](t, raw)
	if profile.History[0].RunName != "focus sprint" || profile.History[0].RunDescription != "edited description" {
		t.Fatalf("partial edit clobbered metadata: %+v", profile.History[0])
	}

	resp, raw = alice.do(http.MethodPatch, path, map[string]any{"pinned": false}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unpin status %d: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodGet, "/user/alice/profile", nil, nil)
	profile = decode[services.ProfileView](t, raw)
	if profile.History[0].Pinned || len(profile.Pinned) != 0 {
		t.Fatalf("run still pinned after unpin: %s", raw)
	}

	// Another player cannot edit this run, even if they know the game ID.
	resp, raw = bob.do(http.MethodPatch, path, map[string]any{"name": "hijack"}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("non-owner update status %d, want 404: %s", resp.StatusCode, raw)
	}

	// Temporary practice runs are not profile runs and cannot be edited.
	temporary := createPractice(t, alice, snippet, events)
	resp, raw = alice.do(http.MethodPatch, fmt.Sprintf("/game/%d/run", temporary.ID), map[string]any{"name": "temp"}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("temporary run update status %d, want 404: %s", resp.StatusCode, raw)
	}

	// Validation is measured in runes, not bytes, and rejects oversized input.
	resp, raw = alice.do(http.MethodPatch, path, map[string]any{"name": strings.Repeat("x", 81)}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized name status %d, want 400: %s", resp.StatusCode, raw)
	}
	resp, raw = alice.do(http.MethodPatch, path, map[string]any{"description": strings.Repeat("x", 501)}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized description status %d, want 400: %s", resp.StatusCode, raw)
	}
}
