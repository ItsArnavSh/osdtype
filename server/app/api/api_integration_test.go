//go:build integration

// This lives in the external api_test package because it drives the server
// through testsupport, which imports the api package itself. Everything here
// goes through exported entry points only.
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"osdtyp/app/entity"
	"osdtyp/app/internal/testsupport"
)

// ------------------------------------------------------------------ helpers

func doRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("could not encode the body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestPing covers the unauthenticated liveness route.
func TestPing(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/ping", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, w.Body.String())
	}
	if body["reply"] != "pong" {
		t.Errorf("expected pong, got %q", body["reply"])
	}
}

// TestAuthMiddlewareRejectsAnonymous guards every authenticated route.
func TestAuthMiddlewareRejectsAnonymous(t *testing.T) {
	router := testsupport.NewRouter(t)

	protected := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/user/whoami"},
		{http.MethodGet, "/user/join-lobby?duration=30"},
		{http.MethodGet, "/room/list?index=0"},
		{http.MethodPost, "/room/create"},
		{http.MethodPost, "/room/promote"},
		{http.MethodGet, "/room/contest/list?room_id=1&index=0"},
	}

	for _, tc := range protected {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := doRequest(t, router, tc.method, tc.path, "", nil)
			if w.Code == http.StatusOK {
				t.Errorf("expected the route to reject anonymous access, got 200")
			}
		})
	}
}

func TestAuthMiddlewareRejectsGarbageToken(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/user/whoami", "not-a-real-token", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --------------------------------------------------------------------- users

// TestGetUserRequiresAQueryParam covers the ?user= lookup.
func TestGetUserRequiresAQueryParam(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/get-user", "", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestWhoamiReturnsTheAuthenticatedUser walks the whole path: JWT, the
// services layer, and the database.
func TestWhoamiReturnsTheAuthenticatedUser(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const userID uint32 = 8080
	if err := db.AddUser(ctx, entity.User{ID: userID, Username: "whoami-user", CurrentRank: 1234}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/user/whoami", testsupport.TokenForUser(t, userID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var got entity.User
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("could not decode the response: %v", err)
	}
	if got.Username != "whoami-user" {
		t.Errorf("expected whoami-user, got %q", got.Username)
	}
	if got.CurrentRank != 1234 {
		t.Errorf("expected rank 1234, got %d", got.CurrentRank)
	}
}

func TestWhoamiForUnknownUser(t *testing.T) {
	router := testsupport.NewRouter(t)

	// A well-formed token for a user that does not exist should fail at the
	// database, not panic.
	w := doRequest(t, router, http.MethodGet, "/user/whoami", testsupport.TokenForUser(t, 999999), nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// TestGetUserByName covers the public profile lookup.
func TestGetUserByName(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	if err := db.AddUser(ctx, entity.User{ID: 8081, Username: "findable"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/get-user?user=findable", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "findable") {
		t.Errorf("expected the username in the body, got %s", w.Body.String())
	}
}

func TestGetUserByNameNotFound(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/get-user?user=nobody-here", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --------------------------------------------------------------------- rooms

func TestCreateRoom(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const creator uint32 = 8100
	if err := db.AddUser(ctx, entity.User{ID: creator, Username: "room-creator"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	body := entity.Room{Name: "integration-room", Desc: "made by a test", Public: entity.PUBLIC}
	w := doRequest(t, router, http.MethodPost, "/room/create", testsupport.TokenForUser(t, creator), body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	// The creator must be enrolled as a moderator.
	var perm entity.Room_User
	if err := db.RawDB().WithContext(ctx).
		Where("user_id = ? AND perm = ?", creator, entity.MOD).
		First(&perm).Error; err != nil {
		t.Fatalf("the creator was not made a moderator: %v", err)
	}
	if perm.RoomID == 0 {
		t.Error("the membership has no room id")
	}
}

// TestCreateRoomWithBrokenJSON is a regression test.
//
// CreateRoom wrote an error response and then fell through without returning,
// so a malformed body produced two concatenated JSON objects and the handler
// still went on to call the service with a zero-valued room.
func TestCreateRoomWithBrokenJSON(t *testing.T) {
	router := testsupport.NewRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/room/create", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testsupport.TokenForUser(t, 8110))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	// A body that parses as exactly one JSON object, not two glued together.
	var probe map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &probe); err != nil {
		t.Fatalf("response body is not a single JSON object: %v (%s)", err, w.Body.String())
	}
	if _, ok := probe["error"]; !ok {
		t.Errorf("expected an error key, got %v", probe)
	}
}

// TestRoomListReturnsJSONArray is a regression test.
//
// The handler marshaled the rooms and then put the resulting []byte into a
// gin.H. encoding/json renders a []byte as a base64 string, so the client
// received {"rooms":"W3si..."} rather than an array of rooms.
func TestRoomListReturnsJSONArray(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const owner uint32 = 8200
	if err := db.AddUser(ctx, entity.User{ID: owner, Username: "lister"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}
	room := entity.Room{ID: 82001, Name: "listed-room"}
	if err := db.CreateRoom(ctx, room); err != nil {
		t.Fatalf("could not seed the room: %v", err)
	}
	if err := db.AddMember(ctx, entity.Room_User{RoomID: 82001, UserID: owner, Perm: entity.MOD}); err != nil {
		t.Fatalf("could not seed the membership: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/room/list?index=0", testsupport.TokenForUser(t, owner), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var payload struct {
		Rooms json.RawMessage `json:"rooms"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("could not decode the response: %v (%s)", err, w.Body.String())
	}

	// The field must be JSON, not a base64 string.
	var rooms []entity.Room
	if err := json.Unmarshal(payload.Rooms, &rooms); err != nil {
		t.Fatalf("the rooms field is not a JSON array: %v (%s)", err, payload.Rooms)
	}
	if len(rooms) != 1 || rooms[0].ID != 82001 {
		t.Errorf("expected the listed room, got %+v", rooms)
	}
}

func TestRoomListRejectsABadIndex(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/room/list?index=notanumber", testsupport.TokenForUser(t, 8300), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestRoomListIndexIsNotCappedAtAByte checks the pagination bound.
//
// The index was parsed with a bit size of 8, so any page past 255 was rejected
// and rooms past the 2560th were unreachable.
func TestRoomListIndexIsNotCappedAtAByte(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/room/list?index=1000", testsupport.TokenForUser(t, 8301), nil)
	if w.Code == http.StatusBadRequest {
		t.Fatalf("a page index of 1000 was rejected: %s", w.Body.String())
	}
}

// TestRoomMembershipPermissions exercises the promote/demote/block lifecycle
// and the authorisation rule that only moderators may act.
func TestRoomMembershipPermissions(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const mod, member uint32 = 8400, 8401
	for _, u := range []entity.User{{ID: mod, Username: "mod"}, {ID: member, Username: "member"}} {
		if err := db.AddUser(ctx, u); err != nil {
			t.Fatalf("could not seed user %d: %v", u.ID, err)
		}
	}
	if err := db.CreateRoom(ctx, entity.Room{ID: 84010, Name: "perm-room"}); err != nil {
		t.Fatalf("could not seed the room: %v", err)
	}
	for _, m := range []entity.Room_User{
		{RoomID: 84010, UserID: mod, Perm: entity.MOD},
		{RoomID: 84010, UserID: member, Perm: entity.MEMBER},
	} {
		if err := db.AddMember(ctx, m); err != nil {
			t.Fatalf("could not seed membership: %v", err)
		}
	}

	modToken := testsupport.TokenForUser(t, mod)

	// Promote: a moderator may.
	w := doRequest(t, router, http.MethodPost, "/room/promote", modToken,
		entity.Room_User{RoomID: 84010, UserID: member})
	if w.Code != http.StatusOK {
		t.Fatalf("expected the promote to succeed, got %d (%s)", w.Code, w.Body.String())
	}

	got, err := db.SeePerms(ctx, entity.Room_User{RoomID: 84010, UserID: member})
	if err != nil {
		t.Fatalf("could not read the membership: %v", err)
	}
	if got.Perm != entity.MOD {
		t.Errorf("expected the user to be a moderator, got %v", got.Perm)
	}

	// Demote: back down again.
	w = doRequest(t, router, http.MethodPost, "/room/demote", modToken,
		entity.Room_User{RoomID: 84010, UserID: member})
	if w.Code != http.StatusOK {
		t.Fatalf("expected the demote to succeed, got %d (%s)", w.Code, w.Body.String())
	}

	// Block.
	w = doRequest(t, router, http.MethodPost, "/room/block", modToken,
		entity.Room_User{RoomID: 84010, UserID: member})
	if w.Code != http.StatusOK {
		t.Fatalf("expected the block to succeed, got %d (%s)", w.Code, w.Body.String())
	}

	got, _ = db.SeePerms(ctx, entity.Room_User{RoomID: 84010, UserID: member})
	if got.Perm != entity.BLOCKED {
		t.Errorf("expected the user to be blocked, got %v", got.Perm)
	}

	// Unblock.
	w = doRequest(t, router, http.MethodPost, "/room/unblock", modToken,
		entity.Room_User{RoomID: 84010, UserID: member})
	if w.Code != http.StatusOK {
		t.Fatalf("expected the unblock to succeed, got %d (%s)", w.Code, w.Body.String())
	}

	got, _ = db.SeePerms(ctx, entity.Room_User{RoomID: 84010, UserID: member})
	if got.Perm != entity.MEMBER {
		t.Errorf("expected the user to be a plain member, got %v", got.Perm)
	}
}

// TestNonModeratorCannotPromote covers the authorisation check.
func TestNonModeratorCannotPromote(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const mod, plain, target uint32 = 8500, 8501, 8502
	for _, u := range []entity.User{
		{ID: mod, Username: "auth-mod"}, {ID: plain, Username: "auth-plain"}, {ID: target, Username: "auth-target"},
	} {
		if err := db.AddUser(ctx, u); err != nil {
			t.Fatalf("could not seed user %d: %v", u.ID, err)
		}
	}
	if err := db.CreateRoom(ctx, entity.Room{ID: 85010}); err != nil {
		t.Fatalf("could not seed the room: %v", err)
	}
	for _, m := range []entity.Room_User{
		{RoomID: 85010, UserID: mod, Perm: entity.MOD},
		{RoomID: 85010, UserID: plain, Perm: entity.MEMBER},
		{RoomID: 85010, UserID: target, Perm: entity.MEMBER},
	} {
		if err := db.AddMember(ctx, m); err != nil {
			t.Fatalf("could not seed membership: %v", err)
		}
	}

	w := doRequest(t, router, http.MethodPost, "/room/promote", testsupport.TokenForUser(t, plain),
		entity.Room_User{RoomID: 85010, UserID: target})
	if w.Code == http.StatusOK {
		t.Fatal("a plain member was allowed to promote someone")
	}

	// The target must be untouched.
	got, err := db.SeePerms(ctx, entity.Room_User{RoomID: 85010, UserID: target})
	if err != nil {
		t.Fatalf("could not read the membership: %v", err)
	}
	if got.Perm != entity.MEMBER {
		t.Errorf("the target's permission changed to %v", got.Perm)
	}
}

// TestUserCanRemoveThemselves documents the one case where a non-moderator may
// act on a membership.
func TestUserCanRemoveThemselves(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const mod, leaver uint32 = 8600, 8601
	for _, u := range []entity.User{{ID: mod, Username: "leave-mod"}, {ID: leaver, Username: "leaver"}} {
		if err := db.AddUser(ctx, u); err != nil {
			t.Fatalf("could not seed user %d: %v", u.ID, err)
		}
	}
	if err := db.CreateRoom(ctx, entity.Room{ID: 86010}); err != nil {
		t.Fatalf("could not seed the room: %v", err)
	}
	for _, m := range []entity.Room_User{
		{RoomID: 86010, UserID: mod, Perm: entity.MOD},
		{RoomID: 86010, UserID: leaver, Perm: entity.MEMBER},
	} {
		if err := db.AddMember(ctx, m); err != nil {
			t.Fatalf("could not seed membership: %v", err)
		}
	}

	w := doRequest(t, router, http.MethodPost, "/room/remove", testsupport.TokenForUser(t, leaver),
		entity.Room_User{RoomID: 86010, UserID: leaver})
	if w.Code != http.StatusOK {
		t.Fatalf("expected a member to be able to leave, got %d (%s)", w.Code, w.Body.String())
	}

	got, err := db.SeePerms(ctx, entity.Room_User{RoomID: 86010, UserID: leaver})
	if err != nil {
		t.Fatalf("could not read the membership: %v", err)
	}
	if got.Perm != entity.LEFT {
		t.Errorf("expected LEFT, got %v", got.Perm)
	}
}

// ------------------------------------------------------------------ contests

// TestCreateContestAndReadItBack is the end-to-end scheduler input path.
func TestCreateContestAndReadItBack(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const creator uint32 = 8700
	if err := db.AddUser(ctx, entity.User{ID: creator, Username: "contest-maker"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	w := doRequest(t, router, http.MethodPost, "/room/contest/create", testsupport.TokenForUser(t, creator), map[string]any{
		"ID":       "",
		"room_id":  87010,
		"Time":     "2099-01-01T10:00:00Z",
		"Data":     "Integration Contest",
		"lang":     int(entity.CPP),
		"duration": int(entity.STANDARD),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var stored entity.Contest
	if err := db.RawDB().WithContext(ctx).Where("data = ?", "Integration Contest").First(&stored).Error; err != nil {
		t.Fatalf("the contest was not persisted: %v", err)
	}
	if stored.Status != entity.UPCOMING {
		t.Errorf("expected UPCOMING, got %v", stored.Status)
	}
}

// TestCreatedContestIsLinkedToItsTask is a regression test.
//
// NewContest used to leave both the contest's JobID and the queued task's JobID
// at zero, so every contest shared one job id. The scheduler resolves a task to
// its contest by job id, so all jobs would have driven the first contest, and
// UpdateContest's "where job_id = ?" would have matched every row.
func TestCreatedContestIsLinkedToItsTask(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const creator uint32 = 8710
	if err := db.AddUser(ctx, entity.User{ID: creator, Username: "linked-maker"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	w := doRequest(t, router, http.MethodPost, "/room/contest/create", testsupport.TokenForUser(t, creator), map[string]any{
		"ID":       "",
		"room_id":  87110,
		"Time":     "2099-01-01T10:00:00Z",
		"Data":     "Linked Contest",
		"lang":     int(entity.CPP),
		"duration": int(entity.STANDARD),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var stored entity.Contest
	if err := db.RawDB().WithContext(ctx).Where("data = ?", "Linked Contest").First(&stored).Error; err != nil {
		t.Fatalf("the contest was not persisted: %v", err)
	}
	if stored.JobID == 0 {
		t.Fatal("the contest was stored with job id 0, so no task can ever find it")
	}

	var task entity.Task
	if err := db.RawDB().WithContext(ctx).Where("job_id = ?", stored.JobID).First(&task).Error; err != nil {
		t.Fatalf("no scheduled task references the contest's job id %d: %v", stored.JobID, err)
	}
	if task.Category != entity.CONTEST {
		t.Errorf("expected a CONTEST task, got %v", task.Category)
	}
}

// TestEachContestGetsItsOwnJobID covers the same defect from the other side:
// two contests must not collide on a job id.
func TestEachContestGetsItsOwnJobID(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const creator uint32 = 8720
	if err := db.AddUser(ctx, entity.User{ID: creator, Username: "collide-maker"}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	seen := make(map[uint32]string)
	for i, data := range []string{"Contest A", "Contest B", "Contest C"} {
		w := doRequest(t, router, http.MethodPost, "/room/contest/create", testsupport.TokenForUser(t, creator), map[string]any{
			"ID":       "",
			"room_id":  87210,
			"Time":     "2099-01-01T10:00:00Z",
			"Data":     data,
			"lang":     int(entity.CPP),
			"duration": int(entity.STANDARD),
		})
		if w.Code != http.StatusOK {
			t.Fatalf("contest %d: expected 200, got %d (%s)", i, w.Code, w.Body.String())
		}

		var stored entity.Contest
		if err := db.RawDB().WithContext(ctx).Where("data = ?", data).First(&stored).Error; err != nil {
			t.Fatalf("contest %d was not persisted: %v", i, err)
		}
		if other, clash := seen[stored.JobID]; clash {
			t.Fatalf("%q and %q share job id %d", other, data, stored.JobID)
		}
		seen[stored.JobID] = data
	}
}

// TestContestListReturnsJSONArray is a regression test for the same base64
// problem as the room list.
func TestContestListReturnsJSONArray(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	if err := db.NewContest(ctx, entity.Contest{ID: "list-1", JobID: 88001, RoomID: 88010, Data: "listed"}); err != nil {
		t.Fatalf("could not seed the contest: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/room/contest/list?room_id=88010&index=0", testsupport.TokenForUser(t, 8800), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var payload struct {
		Contests json.RawMessage `json:"contests"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("could not decode the response: %v", err)
	}

	var contests []entity.Contest
	if err := json.Unmarshal(payload.Contests, &contests); err != nil {
		t.Fatalf("the contests field is not a JSON array: %v (%s)", err, payload.Contests)
	}
	if len(contests) != 1 {
		t.Fatalf("expected 1 contest, got %d", len(contests))
	}
}

func TestContestListRequiresRoomID(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/room/contest/list?index=0", testsupport.TokenForUser(t, 8801), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestContestListRejectsABadRoomID(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/room/contest/list?room_id=abc&index=0", testsupport.TokenForUser(t, 8802), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestContestListPaginationIsStable is a regression test for the swapped
// arguments: the page size and the page index were passed in the wrong order,
// so page 1 asked for one contest starting at offset 50.
func TestContestListPaginationIsStable(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const roomID uint32 = 89010
	const total = 4
	for i := 0; i < total; i++ {
		c := entity.Contest{
			ID:     "page-" + strconv.Itoa(i),
			JobID:  uint32(89000 + i),
			RoomID: roomID,
			Time:   timeFixture(i),
		}
		if err := db.NewContest(ctx, c); err != nil {
			t.Fatalf("could not seed contest %d: %v", i, err)
		}
	}

	page0 := fetchContestIDs(t, router, roomID, 0)
	if len(page0) != total {
		t.Errorf("page 0: expected %d contests, got %d", total, len(page0))
	}

	// A second page one past the end must be empty, not a re-read of page 0.
	page1 := fetchContestIDs(t, router, roomID, 1)
	if len(page1) != 0 {
		t.Errorf("page 1: expected no contests, got %v", page1)
	}
}

func fetchContestIDs(t *testing.T, router http.Handler, roomID uint32, index int) []string {
	t.Helper()
	path := "/room/contest/list?room_id=" + strconv.FormatUint(uint64(roomID), 10) +
		"&index=" + strconv.Itoa(index)
	w := doRequest(t, router, http.MethodGet, path, testsupport.TokenForUser(t, 8910), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var payload struct {
		Contests json.RawMessage `json:"contests"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("could not decode the response: %v", err)
	}
	var contests []entity.Contest
	if err := json.Unmarshal(payload.Contests, &contests); err != nil {
		t.Fatalf("contests is not a JSON array: %v (%s)", err, payload.Contests)
	}
	ids := make([]string, 0, len(contests))
	for _, c := range contests {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestGetContestDataByJobID(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	if err := db.NewContest(ctx, entity.Contest{ID: "single", JobID: 90001, RoomID: 90010, Data: "solo"}); err != nil {
		t.Fatalf("could not seed the contest: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/room/contest/90001", testsupport.TokenForUser(t, 9000), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var payload struct {
		Contest json.RawMessage `json:"contest"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("could not decode the response: %v", err)
	}

	var contest entity.Contest
	if err := json.Unmarshal(payload.Contest, &contest); err != nil {
		t.Fatalf("the contest field is not JSON: %v (%s)", err, payload.Contest)
	}
	if contest.ID != "single" {
		t.Errorf("expected the single contest, got %q", contest.ID)
	}
}

func TestGetContestDataRejectsABadJobID(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/room/contest/not-a-number", testsupport.TokenForUser(t, 9001), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ------------------------------------------------------------------- ranked

// TestJoinLobbyValidatesDuration checks the duration allowlist.
func TestJoinLobbyValidatesDuration(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const uid uint32 = 9100
	if err := db.AddUser(ctx, entity.User{ID: uid, Username: "ranked", CurrentRank: 1000}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}
	token := testsupport.TokenForUser(t, uid)

	for _, bad := range []string{"", "45", "abc", "-30"} {
		w := doRequest(t, router, http.MethodGet, "/user/join-lobby?duration="+bad, token, nil)
		if w.Code == http.StatusOK {
			t.Errorf("duration %q was accepted", bad)
		}
	}
}

// TestJoinLobbyRejectsAUserWithNoSession covers the case where the websocket
// session the matchmaker needs has not been established.
func TestJoinLobbyRejectsAUserWithNoSession(t *testing.T) {
	db := testsupport.NewTestDB(t)
	testsupport.ResetTestData(t, db)
	router := testsupport.NewRouter(t)
	ctx := context.Background()

	const uid uint32 = 9110
	if err := db.AddUser(ctx, entity.User{ID: uid, Username: "sessionless", CurrentRank: 1000}); err != nil {
		t.Fatalf("could not seed the user: %v", err)
	}

	w := doRequest(t, router, http.MethodGet, "/user/join-lobby?duration=30", testsupport.TokenForUser(t, uid), nil)
	if w.Code == http.StatusOK {
		t.Fatal("expected the join to fail without a live session")
	}
}

// ----------------------------------------------------------------- fake auth

// TestFakeLoginIssuesACookie covers the development login route.
func TestFakeLoginIssuesACookie(t *testing.T) {
	router := testsupport.NewRouter(t)

	w := doRequest(t, router, http.MethodGet, "/login/github/fake?username=ci-user", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	var token string
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" && c.Value != "" {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("no token cookie was set")
	}
	_ = os.Getenv("JWTKEY")
}
