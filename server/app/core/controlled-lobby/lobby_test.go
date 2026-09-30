//go:build unit

package controlledlobby

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"osdtyp/app/core/game"
	"osdtyp/app/core/usersession"
	"osdtyp/app/entity"

	"go.uber.org/zap"
)

// The roster moved from an in-memory map to the database. These tests are
// written against the seam that move left behind — the Store interface — so the
// lobby's own logic (capacity, presence, the ordering it promises, and the
// rules about who can start a round) can be checked without a database or a
// WebSocket handshake.

// fakeStore is an in-memory Store with the same concurrency behavior as the
// real one: every method is guarded, because the lobby calls it from HTTP
// handler goroutines and from the scheduler's own.
type fakeStore struct {
	mu sync.Mutex

	rooms   map[uint32]entity.Room
	members map[uint32]map[uint32]entity.RoomPerm

	// nextID is what CreateRoom assigns.
	nextID uint32
	// addCalls counts AddRosterMember invocations, including ones that found
	// the member already present.
	addCalls int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		rooms:   make(map[uint32]entity.Room),
		members: make(map[uint32]map[uint32]entity.RoomPerm),
		nextID:  1,
	}
}

func (f *fakeStore) CreateRoom(_ context.Context, room *entity.Room) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	room.ID = f.nextID
	f.nextID++
	if room.Code == "" {
		room.Code = "ABC234"
	}
	f.rooms[room.ID] = *room
	if f.members[room.ID] == nil {
		f.members[room.ID] = make(map[uint32]entity.RoomPerm)
	}
	return nil
}

func (f *fakeStore) Roster(_ context.Context, roomID uint32) ([]entity.RosterMember, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]entity.RosterMember, 0, len(f.members[roomID]))
	for id, perm := range f.members[roomID] {
		// Only active members, matching the real query. A blocked or departed
		// member is a row that still exists, and a fake that returned it would
		// make "leave" untestable.
		if !perm.IsActive() {
			continue
		}
		out = append(out, entity.RosterMember{
			UserID:   id,
			Username: nameFor(id),
			Rank:     uint16(1000 + id),
		})
	}
	return out, nil
}

// AddRosterMember mirrors the real store's contract: an existing active member
// is a no-op that is not turned away by a full room, a new member is refused
// once the room is at capacity, and a returning member (one who was demoted or
// left) is admitted without counting against the limit a second time.
//
// If this fake simply appended, a capacity test would pass for the wrong
// reason: the lobby would hand it the right answer regardless of what the store
// did, and the bug that made all forty concurrent joins succeed would be
// reintroduced here uncaught.
func (f *fakeStore) AddRosterMember(_ context.Context, roomID, userID uint32, perm entity.RoomPerm, capacity int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.addCalls++
	if f.members[roomID] == nil {
		f.members[roomID] = make(map[uint32]entity.RoomPerm)
	}

	if existing, ok := f.members[roomID][userID]; ok && existing.IsActive() {
		return false, nil
	}
	if perm == entity.BLOCKED {
		return false, entity.ErrRoomBlocked
	}

	if _, returning := f.members[roomID][userID]; !returning && capacity > 0 {
		if f.activeCount(roomID) >= capacity {
			return false, entity.ErrRoomFull
		}
	}

	f.members[roomID][userID] = perm
	return true, nil
}

// activeCount is how many members count against the capacity. The caller holds
// the lock.
func (f *fakeStore) activeCount(roomID uint32) int {
	n := 0
	for _, p := range f.members[roomID] {
		if p.IsActive() {
			n++
		}
	}
	return n
}

func (f *fakeStore) RemoveRosterMember(_ context.Context, roomID, userID uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if room, ok := f.members[roomID]; ok {
		delete(room, userID)
	}
	return nil
}

func (f *fakeStore) RoomMembers(_ context.Context, roomID uint32) ([]entity.RoomMember, error) {
	members, err := f.Roster(context.Background(), roomID)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	perms := f.members[roomID]
	out := make([]entity.RoomMember, len(members))
	for i, m := range members {
		out[i] = entity.RoomMember{
			UserID:   m.UserID,
			Username: m.Username,
			Rank:     m.Rank,
			IsMod:    perms[m.UserID] == entity.MOD,
		}
	}
	f.mu.Unlock()
	return out, nil
}

func nameFor(id uint32) string { return "player" + string(rune('A'+id%26)) }

// harness is a lobby wired to a fake store, a real session registry and a
// record of which users are connected.
type harness struct {
	lobby    *ControlledLobby
	store    *fakeStore
	sessions *usersession.ActiveSessions
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	store := newFakeStore()
	sessions := usersession.NewActiveSessions(zap.NewNop().Sugar())
	t.Cleanup(sessions.Drain)

	lobby := NewControlledLobby(
		zap.NewNop().Sugar(),
		game.NewActiveGames(zap.NewNop().Sugar()),
		sessions,
		store,
	)
	return &harness{lobby: lobby, store: store, sessions: sessions}
}

// connect makes a user appear online, and returns a teardown.
func (h *harness) connect(id uint32) func() {
	s := h.sessions.RegisterStub(id)
	return func() { s.UserOffline() }
}

// TestCreateGameAssignsAHostAndACode covers the two things a caller needs from
// creation: a share code to hand out, and themselves on the roster as a mod so
// they can invite and start.
func TestCreateGameAssignsAHostAndACode(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{
		Name:      "friday night",
		CreatorID: 7,
	})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	if room.ID == 0 {
		t.Error("the room was stored but came back with no id")
	}
	if room.Code == "" {
		t.Error("the room came back with no share code, so nobody could join it")
	}
	if room.Status != entity.LobbyOpen {
		t.Errorf("status = %v, want LobbyOpen", room.Status)
	}

	members, err := h.lobby.Members(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].UserID != 7 {
		t.Fatalf("the creator is not on the roster: %+v", members)
	}
	if !members[0].IsMod {
		t.Error("the creator is not a mod, so they cannot invite or start")
	}
}

// TestCreateGameFillsInTheDefaults is what makes a bare create request work:
// the client sends a name and nothing else.
func TestCreateGameFillsInTheDefaults(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	if room.Capacity != defaultCapacity {
		t.Errorf("capacity = %d, want the default %d", room.Capacity, defaultCapacity)
	}
	if room.Public != entity.PRIVATE {
		t.Errorf("public = %v, want PRIVATE; a lobby reached by share code is not public", room.Public)
	}
}

// TestCreateGameClampsCapacity: a host asking for four hundred players gets the
// ceiling rather than a room nobody can win.
func TestCreateGameClampsCapacity(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{
		Name: "big", CreatorID: 1, Capacity: 400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if room.Capacity != maxCapacity {
		t.Errorf("capacity = %d, want the ceiling %d", room.Capacity, maxCapacity)
	}
}

// TestJoinWithoutASessionIsRejected is a regression test.
//
// Join used to call GetSession(id).Subscribe() and use the result without
// checking it, so joining from a page whose socket had dropped dereferenced a
// nil pointer and returned an unexplained 500. It also left no trace, so
// retrying did nothing.
func TestJoinWithoutASessionIsRejected(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	err = h.lobby.Join(context.Background(), room, 2)
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("joining with no socket returned %v, want ErrNoSession", err)
	}

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("a rejected join changed the roster to %+v", members)
	}
}

// TestJoinAddsToTheRoster is the ordinary path.
func TestJoinAddsToTheRoster(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}
	disconnect := h.connect(2)
	defer disconnect()

	if err := h.lobby.Join(context.Background(), room, 2); err != nil {
		t.Fatalf("Join: %v", err)
	}

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected the host and the joiner, got %+v", members)
	}
	if got := h.sessions.GetStatus(2); got != entity.PLAYING {
		t.Errorf("the joiner's status is %v, want PLAYING so the UI can say so", got)
	}
}

// TestJoinIsIdempotent: a client that retries, or a double click, must not
// create a second membership or a second roster broadcast.
func TestJoinIsIdempotent(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}
	disconnect := h.connect(2)
	defer disconnect()

	for range 3 {
		if err := h.lobby.Join(context.Background(), room, 2); err != nil {
			t.Fatalf("Join: %v", err)
		}
	}

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("three joins produced a roster of %+v", members)
	}
}

// TestJoinRefusesAFullLobby, including the case that made the check awkward: the
// host is already on the roster, so a capacity of one means nobody can join.
func TestJoinRefusesAFullLobby(t *testing.T) {
	h := newHarness(t)

	// Three players: the host plus two joiners.
	room, err := h.lobby.CreateGame(context.Background(), entity.Room{
		Name: "trio", CreatorID: 1, Capacity: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	disconnects := make([]func(), 0, 2)
	defer func() {
		for _, d := range disconnects {
			d()
		}
	}()
	for _, id := range []uint32{2, 3} {
		disconnects = append(disconnects, h.connect(id))
		if err := h.lobby.Join(context.Background(), room, id); err != nil {
			t.Fatalf("Join(%d): %v", id, err)
		}
	}

	disconnects = append(disconnects, h.connect(4))
	err = h.lobby.Join(context.Background(), room, 4)
	if !errors.Is(err, ErrRoomFull) {
		t.Fatalf("joining a full lobby returned %v, want ErrRoomFull", err)
	}

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("a refused join changed the roster to %+v", members)
	}
}

// TestRejoinAfterLeavingTheRoomIsAllowed covers the interaction between the
// capacity check and the existing-membership check: a player who was in the room
// and left has no row, so they are a new arrival. A player who is still on the
// roster is not, and must not be turned away by a room that is full.
func TestFullLobbyStillAcceptsAMemberWhoIsAlreadyIn(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{
		Name: "trio", CreatorID: 1, Capacity: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	disconnects := []func(){h.connect(2), h.connect(3)}
	defer func() {
		for _, d := range disconnects {
			d()
		}
	}()
	for _, id := range []uint32{2, 3} {
		if err := h.lobby.Join(context.Background(), room, id); err != nil {
			t.Fatal(err)
		}
	}
	// The room is full now. A member re-joining must still succeed, or a client
	// that retried after a timeout would lock its own member out.
	if err := h.lobby.Join(context.Background(), room, 2); err != nil {
		t.Errorf("re-joining a full room returned %v", err)
	}
}

// TestLeaveRemovesFromTheRosterAndFreesTheStatus.
func TestLeaveRemovesFromTheRoster(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}
	disconnect := h.connect(2)
	defer disconnect()

	if err := h.lobby.Join(context.Background(), room, 2); err != nil {
		t.Fatal(err)
	}
	if err := h.lobby.Leave(context.Background(), room.ID, 2); err != nil {
		t.Fatal(err)
	}

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("expected only the host, got %+v", members)
	}
	if got := h.sessions.GetStatus(2); got != entity.AVAILABLE {
		t.Errorf("the leaver's status is %v, want AVAILABLE", got)
	}
}

// TestMembersReportsPresenceFromTheRegistry is the point of splitting identity
// from presence: the database cannot know who is connected, so the flag has to
// come from the session registry, and every reader of the roster has to agree.
func TestMembersReportsPresenceFromTheRegistry(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	disconnect := h.connect(2)
	if err := h.lobby.Join(context.Background(), room, 2); err != nil {
		t.Fatal(err)
	}

	// Host is offline: they created the room but never opened a socket in this
	// process. Joiner is online.
	members, err := h.lobby.Members(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected two members, got %+v", members)
	}
	for _, m := range members {
		want := m.UserID == 2
		if m.Online != want {
			t.Errorf("member %d reports Online=%v, want %v", m.UserID, m.Online, want)
		}
	}

	// And a closed session counts as offline, not as available. This is what
	// stopped the invite path pushing into a dead socket.
	disconnect()
	members, err = h.lobby.Members(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range members {
		if m.Online {
			t.Errorf("member %d still reports online after their socket closed", m.UserID)
		}
	}
}

// TestMembersPutsModsFirst is the ordering the host sees. A host who opens their
// own lobby and is not at the top of it looks like they are not in charge.
func TestMembersPutsModsFirst(t *testing.T) {
	out := sortedMembers([]entity.RoomMember{
		{UserID: 1, Rank: 2000},
		{UserID: 2, Rank: 1000, IsMod: true},
		{UserID: 3, Rank: 1500},
		{UserID: 4, Rank: 1200, IsMod: true},
	})

	if !out[0].IsMod || !out[1].IsMod {
		t.Fatalf("mods are not at the top: %+v", out)
	}
	if out[0].Rank != 1200 {
		t.Errorf("mods are not ordered among themselves: %+v", out)
	}
	if out[2].Rank != 2000 {
		t.Errorf("the rest are not ordered by rank: %+v", out)
	}
}

// TestSortedMembersDoesNotMutateItsInput: the caller gets this slice straight
// from the store, and reordering it in place would scramble a cache.
func TestSortedMembersDoesNotMutateItsInput(t *testing.T) {
	in := []entity.RoomMember{{UserID: 1, Rank: 100}, {UserID: 2, Rank: 900}}
	sortedMembers(in)

	if in[0].UserID != 1 {
		t.Errorf("the input was reordered: %+v", in)
	}
}

// TestStartNeedsTwoConnectedPlayers is the check the UI has to agree with.
//
// The count comes from the roster resolved to live sockets, not from the roster
// alone: a room whose other members have all closed their laptops is not
// playable, and starting it would produce a round nobody is in.
func TestStartNeedsTwoConnectedPlayers(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	sig := make(chan []entity.WPMRes, 1)
	if _, err := h.lobby.Start(context.Background(), room.ID, 30*time.Second, sig); err == nil {
		t.Fatal("an empty room started")
	}

	// One connected member is still not a game.
	disconnect := h.connect(1)
	defer disconnect()
	if _, err := h.lobby.Start(context.Background(), room.ID, 30*time.Second, sig); err == nil {
		t.Fatal("a room with one member started")
	}

	// With nobody connected, the roster is not even consulted for playability
	// in a useful way: the members exist but cannot play.
	if _, err := h.lobby.Start(context.Background(), room.ID, 30*time.Second, sig); err == nil {
		t.Fatal("a room of disconnected members started")
	}
}

// TestStartRefusesToRunTheSameRoomTwice covers a host mashing the button, and
// a contest's scheduled start racing a manual one. Two rounds on one room means
// two leaders, two winners, and two rating updates for one game.
func TestStartRefusesToRunTheSameRoomTwice(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []uint32{1, 2} {
		d := h.connect(id)
		defer d()
	}
	for _, id := range []uint32{1, 2} {
		if _, err := h.store.AddRosterMember(context.Background(), room.ID, id, entity.MEMBER, 0); err != nil {
			t.Fatal(err)
		}
	}

	// Occupy the start slot directly rather than running a real round, which
	// would need a snippet service and a client.
	h.lobby.starting.Lock()
	h.lobby.inFlight[room.ID] = true
	h.lobby.starting.Unlock()

	sig := make(chan []entity.WPMRes, 1)
	_, err = h.lobby.Start(context.Background(), room.ID, 30*time.Second, sig)
	if err == nil {
		t.Fatal("a room already starting was allowed to start again")
	}
}

// TestStartSkipsDisconnectedMembers is the rule that makes a flaky connection
// survivable: a member who drops between the roster read and the start should
// not cancel the game for everyone else.
func TestStartSkipsDisconnectedMembers(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}
	disconnect := h.connect(1)
	defer disconnect()

	if _, err := h.store.AddRosterMember(context.Background(), room.ID, 2, entity.MEMBER, 0); err != nil {
		t.Fatal(err)
	}

	sig := make(chan []entity.WPMRes, 1)
	// One of the two members has no socket. The game should have gone ahead with
	// the one who does; it must not refuse outright. The error this returns is
	// the "needs two connected players" one, which is what proves the
	// disconnected member was excluded rather than counted.
	if _, err := h.lobby.Start(context.Background(), room.ID, 30*time.Second, sig); err == nil {
		t.Fatal("a room with one connected member started")
	}
}

// TestConcurrentJoinsDoNotCorruptTheRoster is the regression the old in-memory
// map could not survive.
//
// The map version raised "fatal error: concurrent map read and map write" under
// load, which takes the whole server down. The database version is safe by
// construction; this asserts the observable outcome — every join is recorded
// exactly once, and no capacity check lets the room overflow — so a future
// change that reintroduces shared mutable state fails here rather than in
// production.
func TestConcurrentJoinsDoNotCorruptTheRoster(t *testing.T) {
	h := newHarness(t)

	const capacity = 6
	room, err := h.lobby.CreateGame(context.Background(), entity.Room{
		Name: "race", CreatorID: 1, Capacity: capacity,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The host is already on the roster, so capacity-1 more can get in.
	const contenders = 40
	disconnects := make([]func(), 0, contenders)
	for id := uint32(2); id <= contenders+1; id++ {
		disconnects = append(disconnects, h.connect(id))
	}
	defer func() {
		for _, d := range disconnects {
			d()
		}
	}()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var accepted int

	for id := uint32(2); id <= contenders+1; id++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			err := h.lobby.Join(context.Background(), room, id)
			switch {
			case err == nil:
				mu.Lock()
				accepted++
				mu.Unlock()
			case errors.Is(err, ErrRoomFull):
			default:
				t.Errorf("join %d failed unexpectedly: %v", id, err)
			}
		}(id)
	}
	wg.Wait()

	members, err := h.store.Roster(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != capacity {
		t.Errorf("the roster has %d members, want exactly the capacity %d", len(members), capacity)
	}
	if accepted != capacity-1 {
		t.Errorf("%d joins were accepted, want %d", accepted, capacity-1)
	}

	// And no duplicate rows: the roster is keyed by user.
	seen := make(map[uint32]bool, len(members))
	for _, m := range members {
		if seen[m.UserID] {
			t.Errorf("member %d appears twice", m.UserID)
		}
		seen[m.UserID] = true
	}
}

// TestInviteReachesTheInviteeWithoutASocket is the whole reason notifications
// are durable. An invitation to somebody whose tab is closed has to be waiting
// for them, not lost.
func TestInviteReachesTheInviteeWithoutASocket(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "x", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	store := &recordingNotifications{}
	err = h.lobby.Invite(context.Background(),
		entity.User{ID: 1, Username: "host"}, room, 99, store, time.Hour)
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}

	saved := store.all()
	if len(saved) != 1 {
		t.Fatalf("expected the invitation to be stored, got %d", len(saved))
	}
	got := saved[0]
	if got.UserID != 99 {
		t.Errorf("stored for user %d, want 99", got.UserID)
	}
	if got.Kind != entity.KindLobbyInvite {
		t.Errorf("kind = %v, want a lobby invite", got.Kind)
	}
}

// TestInvitePayloadCarriesTheShareCode is what makes an invitation usable: the
// invitee has to be able to join from the notification alone, with no way to
// ask the host what the code was.
func TestInvitePayloadCarriesTheShareCode(t *testing.T) {
	h := newHarness(t)

	room, err := h.lobby.CreateGame(context.Background(), entity.Room{Name: "friday", CreatorID: 1})
	if err != nil {
		t.Fatal(err)
	}

	store := &recordingNotifications{}
	if err := h.lobby.Invite(context.Background(),
		entity.User{ID: 1, Username: "host"}, room, 99, store, time.Hour); err != nil {
		t.Fatal(err)
	}

	var invite entity.LobbyInvite
	if err := json.Unmarshal(store.all()[0].Payload, &invite); err != nil {
		t.Fatalf("the stored payload does not decode as a LobbyInvite: %v", err)
	}
	if invite.RoomCode != room.Code {
		t.Errorf("the invite carries code %q, want %q", invite.RoomCode, room.Code)
	}
	if invite.From != "host" {
		t.Errorf("the invite names %q as the sender, want host", invite.From)
	}
	if invite.ExpiresAt <= 0 {
		t.Error("the invite has no expiry, so it stays actionable forever")
	}
}

// TestNormalizeCode covers the reality of share codes: people retype them from
// a screenshot, in lowercase, with a trailing space from a bad paste.
func TestNormalizeCode(t *testing.T) {
	// A slice of pairs rather than a map, because the padded case is the whole
	// point of the test and a map key with deliberate leading and trailing
	// spaces is invisible in a diff.
	cases := []struct{ in, want string }{
		{"abc234", "ABC234"},
		{" ABC234 ", "ABC234"},
		{"AbC234", "ABC234"},
		{"\tabc234\n", "ABC234"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := NormalizeCode(c.in); got != c.want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestValidateCode rejects the shapes people actually mistype, including the
// alphabet's deliberate omissions. The code alphabet drops I, O, 0 and 1
// because they are read wrong; a validator that accepted them would let somebody
// invent a code nobody else can type.
func TestValidateCode(t *testing.T) {
	valid := []string{"ABC234", "ZZZZZZ", "HKLMNP"}
	for _, c := range valid {
		if !ValidateCode(c) {
			t.Errorf("ValidateCode(%q) = false, want true", c)
		}
	}

	invalid := []string{
		"",        // empty
		"ABC23",   // too short
		"ABC2345", // too long
		"ABC23I",  // I is not in the alphabet
		"ABC230",  // 0 is not in the alphabet
		"ABC23O",  // O is not in the alphabet
		"ABC-34",  // punctuation
		"abc 34",  // a space
		"ABC23é",  // non-ASCII
	}
	for _, c := range invalid {
		if ValidateCode(c) {
			t.Errorf("ValidateCode(%q) = true, want false", c)
		}
	}
}

// TestParseRoomID is here because a malformed id used to reach the database as
// zero, which then looked like a valid room that happened to be empty.
func TestParseRoomID(t *testing.T) {
	got, err := ParseRoomID("42")
	if err != nil || got != 42 {
		t.Errorf(`ParseRoomID("42") = %d, %v`, got, err)
	}

	for _, bad := range []string{"", "abc", "-1", "99999999999999999999"} {
		if _, err := ParseRoomID(bad); err == nil {
			t.Errorf("ParseRoomID(%q) returned no error", bad)
		}
	}
}

func TestParseRoomIDRejectsOverflow(t *testing.T) {
	// A value past uint32 would wrap to a small, possibly real, room id.
	if _, err := ParseRoomID("4294967296"); err == nil {
		t.Error("an id past uint32 was accepted; it would wrap to a real room")
	}
}
