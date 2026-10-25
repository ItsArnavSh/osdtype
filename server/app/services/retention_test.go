package services

import (
	"context"
	"testing"
	"time"

	"osdtyp/app/entity"
)

func TestRetentionKeepsNewestPinnedAndBestRuns(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	const total = 35
	gameIDs := make([]uint32, total)
	for i := 0; i < total; i++ {
		finished := base.Add(time.Duration(i) * time.Minute)
		gameID := uint32(910000 + i)
		gameIDs[i] = gameID
		// The oldest run is deliberately the best one so it falls outside
		// the newest-30 window. Run #2 is pinned for the same reason.
		wpm := float64(40 + i)
		if i == 0 {
			wpm = 300
		}
		pinned := i == 1
		player := entity.GamePlayer{
			UserID:    alice.ID,
			WPM:       wpm,
			Raw:       wpm + 4,
			Accuracy:  0.98,
			Placement: 1,
			Finished:  true,
			Acked:     true,
			Pinned:    pinned,
		}
		if pinned {
			player.PinnedAt = &finished
		}
		game := entity.Game{
			ID:         gameID,
			Mode:       entity.GameRanked,
			Duration:   30,
			Language:   "go",
			Seed:       11,
			Snippet:    "package retained\n",
			Status:     entity.GameFinished,
			StartsAt:   &finished,
			EndsAt:     &finished,
			CreatedAt:  finished,
			FinishedAt: &finished,
		}
		if err := e.db.CreateGame(ctx, &game, []entity.GamePlayer{player}); err != nil {
			t.Fatalf("create game %d: %v", i, err)
		}
		if err := e.db.SaveRecordings(ctx, []entity.Recording{{
			GameID: gameID, UserID: alice.ID, Events: []byte("[]"), Final: "package", CreatedAt: finished,
		}}); err != nil {
			t.Fatalf("recording %d: %v", i, err)
		}
	}

	removed, err := e.db.PruneUserMatchHistory(ctx, alice.ID, entity.MatchRetentionLimit)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 3 {
		t.Fatalf("pruned %d runs, want 3", removed)
	}

	history, err := e.db.UserGameHistory(ctx, alice.ID, 100)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 32 {
		t.Fatalf("history has %d runs, want 32 (30 newest + best + pinned)", len(history))
	}
	seen := make(map[uint32]bool, len(history))
	for _, run := range history {
		seen[run.GameID] = true
	}
	if !seen[gameIDs[0]] {
		t.Fatal("best run was pruned")
	}
	if !seen[gameIDs[1]] {
		t.Fatal("pinned run was pruned")
	}
	for _, i := range []int{2, 3, 4} {
		if seen[gameIDs[i]] {
			t.Fatalf("old run %d survived retention", i)
		}
		if _, err := e.db.GetRecording(ctx, gameIDs[i], alice.ID); err == nil {
			t.Fatalf("old recording %d survived retention", i)
		}
	}

	// A second pass must be a no-op.
	removed, err = e.db.PruneUserMatchHistory(ctx, alice.ID, entity.MatchRetentionLimit)
	if err != nil || removed != 0 {
		t.Fatalf("second prune removed %d runs: %v", removed, err)
	}

	// The explicit cleanup keeps the same two keepsakes and drops the rest.
	if _, err := e.db.PurgeUnpinnedHistory(ctx, alice.ID); err != nil {
		t.Fatalf("purge: %v", err)
	}
	history, err = e.db.UserGameHistory(ctx, alice.ID, 100)
	if err != nil {
		t.Fatalf("history after purge: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history after purge has %d runs, want best + pinned", len(history))
	}
	if history[0].GameID != gameIDs[1] && history[0].GameID != gameIDs[0] {
		t.Fatalf("unexpected surviving run: %+v", history[0])
	}
	var recordings int64
	if err := e.gdb.Model(&entity.Recording{}).Where("user_id = ?", alice.ID).Count(&recordings).Error; err != nil {
		t.Fatalf("count recordings: %v", err)
	}
	if recordings != 2 {
		t.Fatalf("recordings after purge = %d, want 2", recordings)
	}
}
