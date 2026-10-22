package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"osdtyp/app/entity"

	"github.com/gorilla/websocket"
)

func TestAPIStateRoomIdempotencyAndInviteFlow(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")
	charlie := env.client(t, "charlie")

	resp, raw := alice.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("state status %d: %s", resp.StatusCode, raw)
	}
	st := decode[map[string]any](t, raw)
	if st["state"] != "idle" {
		t.Fatalf("initial state: %s", raw)
	}

	roomBody := map[string]any{
		"name":        "api room",
		"description": "integration",
		"is_public":   true,
		"max_players": 4,
		"duration":    30,
		"language":    "go",
	}
	headers := map[string]string{"Idempotency-Key": "create-room-1"}
	resp, first := alice.do(http.MethodPost, "/room/create", roomBody, headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create room %d: %s", resp.StatusCode, first)
	}
	room := decode[entity.RoomView](t, first)
	if room.Code == "" || room.MyRole != entity.RoleOwner {
		t.Fatalf("bad room: %s", first)
	}
	resp, second := alice.do(http.MethodPost, "/room/create", roomBody, headers)
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("Idempotent-Replay") != "true" {
		t.Fatalf("replay status %d headers %v", resp.StatusCode, resp.Header)
	}
	if string(first) != string(second) {
		t.Fatalf("replay body differs:\n%s\n%s", first, second)
	}

	resp, raw = bob.do(http.MethodPost, "/room/"+room.Code+"/join", map[string]string{}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join room %d: %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodGet, "/state", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob state %d: %s", resp.StatusCode, raw)
	}
	st = decode[map[string]any](t, raw)
	if st["state"] != "room" {
		t.Fatalf("bob not in room: %s", raw)
	}

	// Charlie receives an invite over the websocket while idle.
	header := http.Header{}
	header.Set("Cookie", charlie.cookie.String())
	wsURL := "ws" + strings.TrimPrefix(env.http.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	resp, raw = alice.do(http.MethodPost, "/room/"+room.Code+"/invite", map[string]string{"username": "charlie"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("invite %d: %s", resp.StatusCode, raw)
	}
	var event struct {
		Type string            `json:"type"`
		Data entity.InviteView `json:"data"`
	}
	// The socket sends an initial state snapshot on connect; wait for the
	// invite event itself.
	found := false
	for i := 0; i < 8; i++ {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read invite: %v", err)
		}
		if event.Type == "invite" {
			found = true
			break
		}
	}
	if !found || event.Data.RoomCode != room.Code || event.Data.From != "alice" {
		t.Fatalf("unexpected event: %+v", event)
	}

	resp, raw = charlie.do(http.MethodGet, "/invites", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("invites %d: %s", resp.StatusCode, raw)
	}
	var invites struct {
		Invites []entity.InviteView `json:"invites"`
	}
	if err := json.Unmarshal(raw, &invites); err != nil || len(invites.Invites) != 1 {
		t.Fatalf("invites: %v %s", err, raw)
	}
	resp, raw = charlie.do(http.MethodPost, "/invites/"+uintString(invites.Invites[0].ID)+"/accept", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("accept invite %d: %s", resp.StatusCode, raw)
	}
	resp, raw = charlie.do(http.MethodGet, "/state", nil, nil)
	st = decode[map[string]any](t, raw)
	if st["state"] != "room" {
		t.Fatalf("charlie not in room: %s", raw)
	}

	resp, raw = bob.do(http.MethodPost, "/room/"+room.Code+"/leave", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("leave %d: %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodGet, "/state", nil, nil)
	st = decode[map[string]any](t, raw)
	if st["state"] != "idle" {
		t.Fatalf("bob still in room: %s", raw)
	}
}

func TestAPIPasswordRoomAndErrors(t *testing.T) {
	env := newAPIEnv(t)
	alice := env.client(t, "alice")
	bob := env.client(t, "bob")

	body := map[string]any{
		"name": "locked", "is_public": false, "password": "hunter2",
		"max_players": 2, "duration": 30, "language": "python",
	}
	resp, raw := alice.do(http.MethodPost, "/room/create", body, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d: %s", resp.StatusCode, raw)
	}
	room := decode[entity.RoomView](t, raw)

	resp, raw = bob.do(http.MethodPost, "/room/"+room.Code+"/join", map[string]string{}, nil)
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(raw), "password_required") {
		t.Fatalf("expected password_required: %d %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodPost, "/room/"+room.Code+"/join", map[string]string{"password": "wrong"}, nil)
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(raw), "wrong_password") {
		t.Fatalf("expected wrong_password: %d %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodPost, "/room/"+room.Code+"/join", map[string]string{"password": "hunter2"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join %d: %s", resp.StatusCode, raw)
	}
	resp, raw = bob.do(http.MethodPost, "/room/NOPE99/join", map[string]string{}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing room status %d: %s", resp.StatusCode, raw)
	}
	resp, _ = alice.do(http.MethodGet, "/room/"+room.Code+"/link", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("link status %d", resp.StatusCode)
	}
}

func uintString(v uint32) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}
