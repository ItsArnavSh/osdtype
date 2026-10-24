package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"osdtyp/app/internal/redis"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Event is the single envelope used by every server -> client message.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

func Ev(typ string, data any) Event { return Event{Type: typ, Data: data} }

type Client struct {
	UserID uint32
	conn   *websocket.Conn
	hub    *Hub
	send   chan Event
	done   chan struct{}
	once   sync.Once
}

type Hub struct {
	logger     *zap.SugaredLogger
	rdb        *redis.Client
	instanceID string

	mu      sync.RWMutex
	clients map[uint32]map[*Client]struct{}

	onMessage    func(userID uint32, payload []byte)
	onConnect    func(userID uint32)
	onDisconnect func(userID uint32)
}

const eventChannel = "ws:events"

type wireEvent struct {
	Origin  string   `json:"origin"`
	UserIDs []uint32 `json:"user_ids"`
	Event   Event    `json:"event"`
}

func New(logger *zap.SugaredLogger, rdb *redis.Client) *Hub {
	host, _ := os.Hostname()
	h := &Hub{
		logger:     logger,
		rdb:        rdb,
		instanceID: fmt.Sprintf("%s-%d", host, time.Now().UnixNano()),
		clients:    make(map[uint32]map[*Client]struct{}),
	}
	if rdb != nil {
		go h.subscribe()
	}
	return h
}

func (h *Hub) SetHandlers(onMessage func(uint32, []byte), onConnect func(uint32), onDisconnect func(uint32)) {
	h.onMessage = onMessage
	h.onConnect = onConnect
	h.onDisconnect = onDisconnect
}

// Serve takes ownership of an upgraded connection and blocks until the read
// loop ends (callers can run it in the request goroutine).
func (h *Hub) Serve(userID uint32, conn *websocket.Conn) {
	client := &Client{
		UserID: userID,
		conn:   conn,
		hub:    h,
		send:   make(chan Event, 128),
		done:   make(chan struct{}),
	}

	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	// A user may legitimately have more than one tab or device open. Keep
	// every connection and fan events out to all of them instead of closing
	// the previous socket. Closing the old socket made two tabs continually
	// reconnect and knock each other offline, which also dropped keystrokes.
	h.clients[userID][client] = struct{}{}
	h.mu.Unlock()

	go client.writeLoop()
	if h.onConnect != nil {
		h.onConnect(userID)
	}
	client.readLoop()
}

func (h *Hub) SendToUser(userID uint32, ev Event) {
	h.SendToUsers([]uint32{userID}, ev)
}

func (h *Hub) SendToUsers(userIDs []uint32, ev Event) {
	if len(userIDs) == 0 {
		return
	}
	h.deliverLocal(userIDs, ev)
	if h.rdb != nil {
		payload, err := json.Marshal(wireEvent{Origin: h.instanceID, UserIDs: userIDs, Event: ev})
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := h.rdb.R.Publish(ctx, eventChannel, payload).Err(); err != nil {
			h.logger.Warnw("publish ws event failed", "error", err)
		}
	}
}

func (h *Hub) IsOnline(userID uint32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) deliverLocal(userIDs []uint32, ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, id := range userIDs {
		for c := range h.clients[id] {
			c.enqueue(ev)
		}
	}
}

func (h *Hub) subscribe() {
	ctx := context.Background()
	sub := h.rdb.R.Subscribe(ctx, eventChannel)
	defer sub.Close()
	for msg := range sub.Channel() {
		var wire wireEvent
		if err := json.Unmarshal([]byte(msg.Payload), &wire); err != nil {
			continue
		}
		if wire.Origin == h.instanceID {
			continue
		}
		h.deliverLocal(wire.UserIDs, wire.Event)
	}
}

func (h *Hub) remove(c *Client) bool {
	h.mu.Lock()
	conns := h.clients[c.UserID]
	if _, ok := conns[c]; !ok {
		h.mu.Unlock()
		return false
	}
	delete(conns, c)
	last := len(conns) == 0
	if last {
		delete(h.clients, c.UserID)
	}
	h.mu.Unlock()
	return last
}

func (c *Client) enqueue(ev Event) {
	select {
	case c.send <- ev:
	case <-c.done:
	default:
		c.hub.logger.Warnw("dropping ws message for slow client", "user_id", c.UserID, "type", ev.Type)
	}
}

func (c *Client) writeLoop() {
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case ev := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(ev); err != nil {
				c.close(true)
				return
			}
		case <-ping.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close(true)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) readLoop() {
	c.conn.SetReadLimit(1 << 20)
	c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			c.close(true)
			return
		}
		if c.hub.onMessage != nil {
			c.hub.onMessage(c.UserID, payload)
		}
	}
}

// close is idempotent. notify controls whether the disconnect callback runs.
// The callback only runs when the user's last connection is gone, so closing
// one of several tabs does not mark the account offline.
func (c *Client) close(notify bool) {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close()
		last := c.hub.remove(c)
		if notify && last && c.hub.onDisconnect != nil {
			c.hub.onDisconnect(c.UserID)
		}
	})
}
