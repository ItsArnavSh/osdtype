package services

import (
	"context"
	"encoding/json"
	"time"

	"osdtyp/app/entity"
)

// UserDataExport is a portable snapshot of the account data OSDTYP keeps.
// Replay events are included where the player chose to save them.
type UserDataExport struct {
	Version    int                   `json:"version"`
	ExportedAt time.Time             `json:"exported_at"`
	Account    entity.UserView       `json:"account"`
	CreatedAt  time.Time             `json:"created_at"`
	Settings   ProfileSettingsView   `json:"settings"`
	History    []entity.MatchHistory `json:"history"`
	Pinned     []entity.MatchHistory `json:"pinned"`
	Best       *entity.MatchHistory  `json:"best,omitempty"`
	Recordings []ExportRecording     `json:"recordings"`
	Retention  ExportRetention       `json:"retention"`
}

type ExportRecording struct {
	GameID uint32          `json:"game_id"`
	Events json.RawMessage `json:"events"`
}

type ExportRetention struct {
	UnpinnedRuns int  `json:"unpinned_runs_kept"`
	PinnedKept   bool `json:"pinned_runs_kept"`
	BestKept     bool `json:"best_run_kept"`
}

func (s *ServiceLayer) ExportUserData(ctx context.Context, uid uint32) (UserDataExport, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return UserDataExport{}, err
	}
	history, err := s.db.UserGameHistory(ctx, uid, 200)
	if err != nil {
		return UserDataExport{}, err
	}
	pinned, err := s.db.PinnedGameHistory(ctx, uid, 100)
	if err != nil {
		return UserDataExport{}, err
	}
	best, err := s.db.BestGameHistory(ctx, uid)
	if err != nil {
		return UserDataExport{}, err
	}
	recordings, err := s.db.ListUserRecordings(ctx, uid)
	if err != nil {
		return UserDataExport{}, err
	}
	exported := make([]ExportRecording, 0, len(recordings))
	for _, recording := range recordings {
		events := json.RawMessage(recording.Events)
		if !json.Valid(events) {
			// Corrupt legacy blobs are still represented in the export rather
			// than making the whole download fail.
			encoded, _ := json.Marshal(string(recording.Events))
			events = encoded
		}
		exported = append(exported, ExportRecording{GameID: recording.GameID, Events: events})
	}
	return UserDataExport{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Account:    entity.NewUserView(user),
		CreatedAt:  user.CreatedAt,
		Settings:   profileSettingsView(user),
		History:    history,
		Pinned:     pinned,
		Best:       best,
		Recordings: exported,
		Retention: ExportRetention{
			UnpinnedRuns: entity.MatchRetentionLimit,
			PinnedKept:   true,
			BestKept:     true,
		},
	}, nil
}

// DeleteRunHistory removes every unpinned run. Pinned runs are always kept.
func (s *ServiceLayer) DeleteRunHistory(ctx context.Context, uid uint32) (int, error) {
	return s.db.PurgeUnpinnedHistory(ctx, uid)
}
