package hub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubKeepsMultipleConnectionsForUser(t *testing.T) {
	h := &Hub{clients: make(map[uint32]map[*Client]struct{})}
	first := &Client{UserID: 7, hub: h, send: make(chan Event, 1), done: make(chan struct{})}
	second := &Client{UserID: 7, hub: h, send: make(chan Event, 1), done: make(chan struct{})}
	h.clients[7] = map[*Client]struct{}{first: {}, second: {}}

	if !h.IsOnline(7) {
		t.Fatal("user should be online")
	}
	if got := h.OnlineCount(); got != 1 {
		t.Fatalf("OnlineCount = %d, want 1 user", got)
	}

	h.SendToUser(7, Ev("progress", nil))
	for i, c := range []*Client{first, second} {
		select {
		case ev := <-c.send:
			if ev.Type != "progress" {
				t.Fatalf("connection %d got %q", i, ev.Type)
			}
		default:
			t.Fatalf("connection %d did not receive event", i)
		}
	}

	if h.remove(first) {
		t.Fatal("removing one tab must not be the last connection")
	}
	if !h.IsOnline(7) {
		t.Fatal("user should still be online")
	}
	if !h.remove(second) {
		t.Fatal("removing the last tab should report last")
	}
	if h.IsOnline(7) {
		t.Fatal("user should be offline")
	}
	if got := h.OnlineCount(); got != 0 {
		t.Fatalf("OnlineCount = %d, want 0", got)
	}
}

// A second socket used to close the first one. Two tabs then repeatedly
// reconnected and kicked each other offline, dropping keystrokes during the
// race. This test drives real websocket connections through Serve.
func TestServeKeepsBothSocketsAndNotifiesDisconnectOnce(t *testing.T) {
	h := New(nil, nil)
	var connects, disconnects atomic.Int32
	h.SetHandlers(
		func(uint32, []byte) {},
		func(uint32) { connects.Add(1) },
		func(uint32) { disconnects.Add(1) },
	)

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Serve(7, conn)
	}))
	defer server.Close()

	dial := func() *websocket.Conn {
		t.Helper()
		url := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}
	readEvent := func(conn *websocket.Conn) Event {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var ev Event
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read: %v", err)
		}
		return ev
	}
	waitFor := func(label string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for %s", label)
	}
	connectionCount := func() int {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.clients[7])
	}

	first := dial()
	waitFor("first connection", func() bool { return connectionCount() == 1 })
	second := dial()
	waitFor("second connection", func() bool { return connectionCount() == 2 })
	if got := connects.Load(); got != 2 {
		t.Fatalf("connect callbacks = %d, want 2", got)
	}

	h.SendToUser(7, Ev("hello", "both"))
	if got := readEvent(first); got.Type != "hello" {
		t.Fatalf("first socket got %q", got.Type)
	}
	if got := readEvent(second); got.Type != "hello" {
		t.Fatalf("second socket got %q", got.Type)
	}

	_ = first.Close()
	waitFor("first socket removal", func() bool { return connectionCount() == 1 })
	if got := disconnects.Load(); got != 0 {
		t.Fatalf("disconnect callback fired while one socket remained: %d", got)
	}
	if !h.IsOnline(7) {
		t.Fatal("user should still be online")
	}

	h.SendToUser(7, Ev("still-here", nil))
	if got := readEvent(second); got.Type != "still-here" {
		t.Fatalf("remaining socket got %q", got.Type)
	}

	_ = second.Close()
	waitFor("last socket removal", func() bool { return !h.IsOnline(7) })
	waitFor("disconnect callback", func() bool { return disconnects.Load() == 1 })
}
