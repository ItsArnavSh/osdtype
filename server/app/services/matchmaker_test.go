package services

import (
	"context"
	"testing"
	"time"
)

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(15 * time.Millisecond)
	}
	return false
}

func TestMatchmakerPairsQueuedHumans(t *testing.T) {
	e := newTestEnvWith(t, "", func() {
		// Keep the bot fallback out of the way for this test.
		viperSetBotAfter(60)
	})
	e.core.Start()
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	if _, err := e.svc.JoinQueue(ctx, alice.ID, 30); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.JoinQueue(ctx, bob.ID, 30); err != nil {
		t.Fatal(err)
	}
	ok := waitFor(t, 3*time.Second, func() bool {
		_, a := e.svc.g().IsInGame(alice.ID)
		_, b := e.svc.g().IsInGame(bob.ID)
		return a && b
	})
	if !ok {
		t.Fatalf("players were not matched; states: %+v %+v", e.state(t, alice.ID), e.state(t, bob.ID))
	}
	aid, _ := e.svc.g().IsInGame(alice.ID)
	bid, _ := e.svc.g().IsInGame(bob.ID)
	if aid != bid {
		t.Fatalf("different games: %d %d", aid, bid)
	}
	if st := e.state(t, alice.ID); st.State != "game" || st.Game == nil || st.Game.ID != aid {
		t.Fatalf("bad state: %+v", st)
	}
	items, err := e.rdb.QueueItems(ctx, 30)
	if err != nil || len(items) != 0 {
		t.Fatalf("queue not empty: %v %+v", err, items)
	}
	e.svc.g().Abandon(aid)
}

func TestMatchmakerBotFallback(t *testing.T) {
	e := newTestEnvWith(t, "", func() {
		viperSetBotAfter(1)
	})
	e.core.Start()
	ctx := context.Background()
	alice := e.login(t, "alice")

	if _, err := e.svc.JoinQueue(ctx, alice.ID, 30); err != nil {
		t.Fatal(err)
	}
	ok := waitFor(t, 5*time.Second, func() bool {
		_, in := e.svc.g().IsInGame(alice.ID)
		return in
	})
	if !ok {
		t.Fatalf("bot fallback did not start a game: %+v", e.state(t, alice.ID))
	}
	aid, _ := e.svc.g().IsInGame(alice.ID)
	view, err := e.svc.GetGameView(ctx, alice.ID, aid)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Players) != 2 {
		t.Fatalf("expected human + bot, got %+v", view.Players)
	}
	bots := 0
	for _, p := range view.Players {
		if p.IsBot {
			bots++
		}
	}
	if bots != 1 {
		t.Fatalf("expected one bot: %+v", view.Players)
	}
	e.svc.g().Abandon(aid)
}
