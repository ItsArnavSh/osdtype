package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"osdtyp/app/entity"
)

func TestRoomPasswordJoinIdempotencyAndLimits(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	charlie := e.login(t, "charlie")

	room, err := e.svc.CreateRoom(ctx, alice.ID, CreateRoomRequest{
		Name: "Sprint room", Description: "fast", IsPublic: false, Password: "secret",
		MaxPlayers: 2, Duration: 30, Language: "go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !room.HasPassword || room.MyRole != entity.RoleOwner || room.MemberCount != 1 {
		t.Fatalf("bad room: %+v", room)
	}
	if room.InviteLink == "" {
		t.Fatal("missing invite link")
	}

	if _, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected password required, got %v", err)
	}
	if _, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, "nope"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected wrong password, got %v", err)
	}
	joined, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if joined.MemberCount != 2 || joined.MyRole != entity.RoleMember {
		t.Fatalf("bad join: %+v", joined)
	}
	again, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, "secret")
	if err != nil || again.MemberCount != 2 {
		t.Fatalf("join not idempotent: %v %+v", err, again)
	}
	if _, err := e.svc.JoinRoom(ctx, charlie.ID, room.Code, "secret"); !errors.Is(err, ErrFull) {
		t.Fatalf("expected full room, got %v", err)
	}

	if st := e.state(t, bob.ID); st.State != "room" || st.Room == nil || st.Room.Code != room.Code {
		t.Fatalf("bad bob state: %+v", st)
	}
	// Rebuild state from durable data, as after a Redis flush/restart.
	_ = e.rdb.ClearState(ctx, bob.ID)
	if st := e.state(t, bob.ID); st.State != "room" || st.Room == nil {
		t.Fatalf("state did not rebuild: %+v", st)
	}

	if err := e.svc.LeaveRoom(ctx, bob.ID, room.Code); err != nil {
		t.Fatal(err)
	}
	if st := e.state(t, bob.ID); st.State != "idle" {
		t.Fatalf("expected idle after leave: %+v", st)
	}

	invite, err := e.svc.InviteUser(ctx, alice.ID, room.Code, InviteRequest{Username: "charlie"})
	if err != nil {
		t.Fatal(err)
	}
	invites, err := e.svc.ListInvites(ctx, charlie.ID)
	if err != nil || len(invites) != 1 || invites[0].RoomCode != room.Code {
		t.Fatalf("invites: %v %+v", err, invites)
	}
	joined, err = e.svc.AcceptInvite(ctx, charlie.ID, invite.ID)
	if err != nil {
		t.Fatal(err)
	}
	if joined.MemberCount != 2 {
		t.Fatalf("invite join count = %d", joined.MemberCount)
	}
	if st := e.state(t, charlie.ID); st.State != "room" {
		t.Fatalf("expected room state after accept: %+v", st)
	}

	if err := e.svc.CloseRoom(ctx, alice.ID, room.Code); err != nil {
		t.Fatal(err)
	}
	if st := e.state(t, charlie.ID); st.State != "idle" {
		t.Fatalf("expected idle after close: %+v", st)
	}
}

func TestRoomPermissionsAndBots(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	charlie := e.login(t, "charlie")

	room, err := e.svc.CreateRoom(ctx, alice.ID, CreateRoomRequest{Name: "mods", IsPublic: true, MaxPlayers: 4, Duration: 90, Language: "go"})
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []entity.User{bob, charlie} {
		if _, err := e.svc.JoinRoom(ctx, u.ID, room.Code, ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.svc.KickUser(ctx, bob.ID, room.Code, alice.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member kicked owner: %v", err)
	}
	if err := e.svc.PromoteUser(ctx, alice.ID, room.Code, bob.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.KickUser(ctx, bob.ID, room.Code, charlie.ID); err != nil {
		t.Fatalf("mod kick: %v", err)
	}
	if err := e.svc.UnbanUser(ctx, bob.ID, room.Code, charlie.ID); err != nil {
		t.Fatalf("unban missing: %v", err)
	}
	if err := e.svc.BanUser(ctx, alice.ID, room.Code, charlie.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.JoinRoom(ctx, charlie.ID, room.Code, ""); !errors.Is(err, ErrBanned) {
		t.Fatalf("banned user joined: %v", err)
	}
	if err := e.svc.DemoteUser(ctx, alice.ID, room.Code, bob.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.AddBot(ctx, bob.ID, room.Code, entity.ACE); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-mod added bot: %v", err)
	}
	view, err := e.svc.AddBot(ctx, alice.ID, room.Code, entity.ACE)
	if err != nil {
		t.Fatal(err)
	}
	if view.MemberCount != 3 {
		t.Fatalf("bot not added: %+v", view)
	}
	if _, err := e.svc.AddBot(ctx, alice.ID, room.Code, entity.ACE); err != nil {
		t.Fatalf("duplicate bot add should be idempotent: %v", err)
	}
}

func TestCloseRoomAbandonsActiveGame(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	room, err := e.svc.CreateRoom(ctx, alice.ID, CreateRoomRequest{Name: "abandon", IsPublic: true, MaxPlayers: 4, Duration: 30, Language: "go"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, ""); err != nil {
		t.Fatal(err)
	}
	view, err := e.svc.StartRoomGame(ctx, alice.ID, room.Code, StartGameRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.CloseRoom(ctx, alice.ID, room.Code); err != nil {
		t.Fatal(err)
	}
	game, err := e.db.GetGame(ctx, view.ID)
	if err != nil || game.Status != entity.GameAbandoned {
		t.Fatalf("game not abandoned: %v %+v", err, game)
	}
	if _, err := e.db.ActiveGameForUser(ctx, alice.ID); err == nil {
		t.Fatal("abandoned game still counts as active")
	}
	if _, err := e.svc.GetRoom(ctx, bob.ID, room.Code); !errors.Is(err, ErrNotFound) {
		t.Fatalf("closed room should not be joinable/readable: %v", err)
	}
	if st := e.state(t, bob.ID); st.State != "idle" {
		t.Fatalf("expected idle after close: %+v", st)
	}
}

func TestRoomGameEndToEnd(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	room, err := e.svc.CreateRoom(ctx, alice.ID, CreateRoomRequest{Name: "race", IsPublic: true, MaxPlayers: 4, Duration: 30, Language: "go"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.JoinRoom(ctx, bob.ID, room.Code, ""); err != nil {
		t.Fatal(err)
	}
	view, err := e.svc.StartRoomGame(ctx, alice.ID, room.Code, StartGameRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if view.Snippet == "" || view.Seed == "" || view.StartsAt == nil {
		t.Fatalf("incomplete game view: %+v", view)
	}
	persisted, err := e.db.GetGamePlayers(ctx, view.ID)
	if err != nil || len(persisted) != 2 {
		t.Fatalf("game players not persisted against game %d: %v %+v", view.ID, err, persisted)
	}
	wait := time.Until(*view.StartsAt)
	if wait > 0 {
		time.Sleep(wait + 15*time.Millisecond)
	}
	game, err := e.svc.GetGameView(ctx, alice.ID, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	typed := nonWhitespace(game.Snippet[game.StartOffset:])
	if typed == "" {
		t.Fatal("empty typing target")
	}
	if ok := e.svc.g().HandleInput(alice.ID, entity.Keypress{Value: typed, Action: entity.KEYPRESS}); !ok {
		t.Fatal("alice input rejected")
	}
	if ok := e.svc.g().HandleInput(bob.ID, entity.Keypress{Value: typed, Action: entity.KEYPRESS}); !ok {
		t.Fatal("bob input rejected")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st := e.state(t, alice.ID)
		if st.State == "results" {
			if st.Results == nil || len(st.Results.Leaderboard) != 2 {
				t.Fatalf("bad results: %+v", st.Results)
			}
			finished, err := e.db.GetGame(ctx, view.ID)
			if err != nil || finished.Status != entity.GameFinished {
				t.Fatalf("game not finished: %v %+v", err, finished)
			}
			dbRoom, _ := e.db.GetRoomByCode(ctx, room.Code)
			if dbRoom.Status != entity.RoomLobby || dbRoom.GameID != 0 {
				t.Fatalf("room not reset: %+v", dbRoom)
			}
			// Results survive a Redis flush until the player acknowledges them.
			_ = e.rdb.ClearState(ctx, alice.ID)
			if rebuilt := e.state(t, alice.ID); rebuilt.State != "results" || rebuilt.Results == nil || rebuilt.Results.ID != view.ID {
				t.Fatalf("results were not rebuilt from durable data: %+v", rebuilt)
			}
			acked, err := e.svc.AckResults(ctx, alice.ID)
			if err != nil {
				t.Fatalf("ack results: %v", err)
			}
			if acked.State != "room" || acked.Room == nil || acked.Room.Code != room.Code {
				t.Fatalf("expected room after ack: %+v", acked)
			}
			_ = e.rdb.ClearState(ctx, alice.ID)
			if rebuilt := e.state(t, alice.ID); rebuilt.State != "room" || rebuilt.Room == nil {
				t.Fatalf("acknowledged results resurfaced after rebuild: %+v", rebuilt)
			}
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("game did not finish, state=%+v", e.state(t, alice.ID))
}

func TestFinishHookDoesNotResurfaceResultsForPlayerWhoLeft(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	g := entity.Game{
		ID: e.svc.intGen.GenerateID(), Mode: entity.GameRanked, Duration: 30,
		Language: "go", Seed: 42, Snippet: generatedSnippet(), Status: entity.GameCountdown,
		CreatedAt: time.Now().UTC(),
	}
	players := []entity.GamePlayer{
		{GameID: g.ID, UserID: alice.ID, RatingBefore: 1000, WPM: 20, Placement: 2, Left: true},
		{GameID: g.ID, UserID: bob.ID, RatingBefore: 1000, WPM: 50, Placement: 1},
	}
	if err := e.db.CreateGame(ctx, &g, players); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.onGameFinished(ctx, g, players, nil, nil); err != nil {
		t.Fatal(err)
	}

	if st := e.state(t, alice.ID); st.State == "results" {
		t.Fatalf("player who left was dragged back to results: %+v", st)
	}
	if st := e.state(t, bob.ID); st.State != "results" || st.Results == nil || st.Results.ID != g.ID {
		t.Fatalf("remaining player did not get results: %+v", st)
	}
}

func TestFinishHookAppliesEloAndFlags(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	makeGame := func(flagged bool) (entity.Game, []entity.GamePlayer) {
		g := entity.Game{
			ID: e.svc.intGen.GenerateID(), Mode: entity.GameRanked, Duration: 30,
			Language: "go", Seed: 42, Snippet: generatedSnippet(), Status: entity.GameCountdown,
			CreatedAt: time.Now().UTC(),
		}
		players := []entity.GamePlayer{
			{GameID: g.ID, UserID: alice.ID, RatingBefore: 1000, WPM: 90, Accuracy: 0.99, Placement: 1, Flagged: flagged},
			{GameID: g.ID, UserID: bob.ID, RatingBefore: 1000, WPM: 50, Accuracy: 0.92, Placement: 2},
		}
		if err := e.db.CreateGame(ctx, &g, players); err != nil {
			t.Fatal(err)
		}
		return g, players
	}

	g, players := makeGame(false)
	views, err := e.svc.onGameFinished(ctx, g, players, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].RatingDelta <= 0 || views[1].RatingDelta >= 0 {
		t.Fatalf("unexpected rating views: %+v", views)
	}
	a, _ := e.db.GetUser(alice.ID)
	b, _ := e.db.GetUser(bob.ID)
	if a.Rating <= 1000 || b.Rating >= 1000 {
		t.Fatalf("ratings not applied: alice=%d bob=%d", a.Rating, b.Rating)
	}
	if a.RatingFor(entity.RankedDuration30) <= 1000 || b.RatingFor(entity.RankedDuration30) >= 1000 {
		t.Fatalf("30s ladder not applied: alice=%d bob=%d",
			a.RatingFor(entity.RankedDuration30), b.RatingFor(entity.RankedDuration30))
	}
	if a.RatingFor(entity.RankedDuration90) != entity.StartingElo ||
		a.RatingFor(entity.RankedDuration300) != entity.StartingElo ||
		a.GamesPlayed30 != 1 || a.GamesPlayed90 != 0 || a.GamesPlayed300 != 0 {
		t.Fatalf("30s game leaked into another ladder: %+v", a)
	}
	if st := e.state(t, alice.ID); st.State != "results" {
		t.Fatalf("missing results state: %+v", st)
	}
	history, err := e.db.UserGameHistory(ctx, alice.ID, 5)
	if err != nil || len(history) != 1 || history[0].RatingDelta <= 0 {
		t.Fatalf("history: %v %+v", err, history)
	}

	// A flagged player gets no rating change.
	g2, players2 := makeGame(true)
	views, err = e.svc.onGameFinished(ctx, g2, players2, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if views[0].RatingDelta != 0 {
		t.Fatalf("flagged player gained rating: %+v", views[0])
	}
	a2, _ := e.db.GetUser(alice.ID)
	if a2.Rating != a.Rating {
		t.Fatalf("flagged player rating changed: %d -> %d", a.Rating, a2.Rating)
	}
}

func TestRankedGameMovesOnlyItsClock(t *testing.T) {
	e := newTestEnv(t, "")
	ctx := context.Background()
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	for _, uid := range []uint32{alice.ID, bob.ID} {
		u, err := e.db.GetUser(uid)
		if err != nil {
			t.Fatal(err)
		}
		u.SetRatingFor(entity.RankedDuration30, 500)
		u.SetRatingFor(entity.RankedDuration90, 700)
		u.SetRatingFor(entity.RankedDuration300, 900)
		if err := e.db.SaveUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}

	game := entity.Game{
		ID:       9911,
		Mode:     entity.GameRanked,
		Duration: entity.RankedDuration30,
		Language: "go",
		Seed:     1,
		Status:   entity.GameActive,
	}
	players := []entity.GamePlayer{
		{GameID: game.ID, UserID: alice.ID, RatingBefore: 500, Placement: 1, WPM: 80, Accuracy: 98},
		{GameID: game.ID, UserID: bob.ID, RatingBefore: 500, Placement: 2, WPM: 70, Accuracy: 98},
	}
	views, err := e.svc.onGameFinished(ctx, game, players, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("result views = %d, want 2", len(views))
	}

	gotAlice, _ := e.db.GetUser(alice.ID)
	gotBob, _ := e.db.GetUser(bob.ID)
	if gotAlice.RatingFor(entity.RankedDuration30) == 500 || gotBob.RatingFor(entity.RankedDuration30) == 500 {
		t.Fatalf("30s ladder did not move: alice=%d bob=%d",
			gotAlice.RatingFor(entity.RankedDuration30), gotBob.RatingFor(entity.RankedDuration30))
	}
	for _, u := range []entity.User{gotAlice, gotBob} {
		if u.RatingFor(entity.RankedDuration90) != 700 || u.RatingFor(entity.RankedDuration300) != 900 {
			t.Fatalf("30s race changed another ladder for %s: %d/%d",
				u.Username, u.RatingFor(entity.RankedDuration90), u.RatingFor(entity.RankedDuration300))
		}
		if u.GamesFor(entity.RankedDuration30) != 1 || u.GamesFor(entity.RankedDuration90) != 0 || u.GamesFor(entity.RankedDuration300) != 0 {
			t.Fatalf("per-clock record leaked for %s: %d/%d/%d", u.Username,
				u.GamesFor(entity.RankedDuration30), u.GamesFor(entity.RankedDuration90), u.GamesFor(entity.RankedDuration300))
		}
	}
}
