package services

import (
	"context"
	"encoding/json"
	"time"

	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
)

type inbound struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (s *ServiceLayer) onSocketConnect(uid uint32) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.rdb().SetOnline(ctx, uid, 70*time.Second)
	if st, err := s.State(ctx, uid); err == nil {
		s.hub().SendToUser(uid, hub.Ev("state", st))
	}
}

func (s *ServiceLayer) onSocketDisconnect(uid uint32) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.rdb().SetOffline(ctx, uid)
}

func (s *ServiceLayer) onSocketMessage(uid uint32, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.rdb().SetOnline(ctx, uid, 70*time.Second)

	var msg inbound
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Type == "" {
		// Backwards compatible raw keypress.
		var kp entity.Keypress
		if json.Unmarshal(payload, &kp) == nil {
			s.g().HandleInput(uid, kp)
		}
		return
	}

	switch msg.Type {
	case "keypress":
		var kp entity.Keypress
		if err := json.Unmarshal(msg.Data, &kp); err != nil {
			return
		}
		s.g().HandleInput(uid, kp)
	case "leave_game":
		var req struct {
			GameID uint32 `json:"game_id"`
		}
		_ = json.Unmarshal(msg.Data, &req)
		_, _ = s.LeaveGame(ctx, uid, req.GameID)
	case "queue_cancel":
		var req struct {
			Duration int `json:"duration"`
		}
		_ = json.Unmarshal(msg.Data, &req)
		if req.Duration == 0 {
			req.Duration = 90
		}
		_ = s.LeaveQueue(ctx, uid, req.Duration)
	case "state":
		if st, err := s.State(ctx, uid); err == nil {
			s.hub().SendToUser(uid, hub.Ev("state", st))
		}
	case "ping":
		s.hub().SendToUser(uid, hub.Ev("pong", map[string]any{"time": time.Now().UTC()}))
	}
}

func (s *ServiceLayer) touchPresence(ctx context.Context, uid uint32) {
	_ = s.rdb().SetOnline(ctx, uid, 70*time.Second)
}
