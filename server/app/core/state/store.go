package state

import (
	"context"

	"osdtyp/app/core/game"
	"osdtyp/app/entity"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/internal/redis"

	"go.uber.org/zap"
)

// Store persists the backend-owned user state. Redis is the fast path; the
// database is authoritative and is used to rebuild state after a restart.
type Store struct {
	logger *zap.SugaredLogger
	rdb    *redis.Client
	db     *postgresql.Database
	games  *game.Manager
}

func New(logger *zap.SugaredLogger, rdb *redis.Client, db *postgresql.Database, games *game.Manager) *Store {
	return &Store{logger: logger, rdb: rdb, db: db, games: games}
}

func (s *Store) Set(ctx context.Context, uid uint32, st redis.UserState) {
	if err := s.rdb.SetState(ctx, uid, st); err != nil {
		s.logger.Warnw("failed to store user state", "user_id", uid, "error", err)
	}
}

func (s *Store) Get(ctx context.Context, uid uint32) redis.UserState {
	st, ok, err := s.rdb.GetState(ctx, uid)
	if err == nil && ok && st.Status != "" {
		return st
	}
	return s.Derive(ctx, uid)
}

// Derive rebuilds state from durable data.
func (s *Store) Derive(ctx context.Context, uid uint32) redis.UserState {
	if id, inGame := s.games.IsInGame(uid); inGame {
		st := redis.UserState{Status: "game", GameID: id}
		if view, ok := s.games.Snapshot(uid); ok && view.RoomID != 0 {
			if room, err := s.db.GetRoomByID(ctx, view.RoomID); err == nil {
				st.RoomCode = room.Code
			}
		}
		s.Set(ctx, uid, st)
		return st
	}
	if g, err := s.db.ActiveGameForUser(ctx, uid); err == nil {
		st := redis.UserState{Status: "game", GameID: g.ID}
		if g.RoomID != 0 {
			if room, err := s.db.GetRoomByID(ctx, g.RoomID); err == nil {
				st.RoomCode = room.Code
			}
		}
		s.Set(ctx, uid, st)
		return st
	}
	if g, err := s.db.LatestUnacknowledgedGame(ctx, uid); err == nil {
		st := redis.UserState{Status: "results", ResultsGameID: g.ID}
		if g.RoomID != 0 {
			if room, err := s.db.GetRoomByID(ctx, g.RoomID); err == nil {
				st.RoomCode = room.Code
			}
		}
		s.Set(ctx, uid, st)
		return st
	}
	if room, ok, err := s.db.ActiveRoomForUser(ctx, uid); err == nil && ok {
		st := redis.UserState{Status: "room", RoomCode: room.Code}
		s.Set(ctx, uid, st)
		return st
	}
	st := redis.UserState{Status: "idle"}
	s.Set(ctx, uid, st)
	return st
}

func (s *Store) Clear(ctx context.Context, uid uint32) {
	s.Set(ctx, uid, redis.UserState{Status: "idle"})
}

func (s *Store) SetRoom(ctx context.Context, uid uint32, code string) {
	s.Set(ctx, uid, redis.UserState{Status: "room", RoomCode: code})
}

func (s *Store) SetGame(ctx context.Context, uid uint32, gameID uint32, roomCode string) {
	// Starting or resuming a game implicitly dismisses any older results that
	// were never acknowledged, so they cannot resurface after a Redis restart.
	_ = s.db.AckAllResults(ctx, uid)
	s.Set(ctx, uid, redis.UserState{Status: "game", GameID: gameID, RoomCode: roomCode})
}

func (s *Store) SetQueue(ctx context.Context, uid uint32, duration int) {
	s.Set(ctx, uid, redis.UserState{Status: "queue", QueueDuration: duration})
}

func (s *Store) SetResults(ctx context.Context, uid uint32, gameID uint32, roomCode string) {
	s.Set(ctx, uid, redis.UserState{Status: "results", ResultsGameID: gameID, RoomCode: roomCode})
}

// AckResults leaves the results screen. If the player is still in a room the
// state falls back to that room, otherwise to idle.
func (s *Store) AckResults(ctx context.Context, uid uint32) redis.UserState {
	st := s.Get(ctx, uid)
	if st.Status != "results" {
		return st
	}
	if st.ResultsGameID != 0 {
		if err := s.db.AckResults(ctx, st.ResultsGameID, uid); err != nil {
			s.logger.Warnw("failed to persist result acknowledgement", "user_id", uid, "game_id", st.ResultsGameID, "error", err)
		}
	}
	if st.RoomCode != "" {
		if room, err := s.db.GetRoomByCode(ctx, st.RoomCode); err == nil {
			if member, err := s.db.GetMember(ctx, room.ID, uid); err == nil && member.Role != "" && room.Status != entity.RoomClosed {
				out := redis.UserState{Status: "room", RoomCode: room.Code}
				s.Set(ctx, uid, out)
				return out
			}
		}
	}
	out := redis.UserState{Status: "idle"}
	s.Set(ctx, uid, out)
	return out
}

func (s *Store) Room(uid uint32) (redis.UserState, bool) {
	// convenience for callers that already hold a context-free path
	return redis.UserState{}, false
}
