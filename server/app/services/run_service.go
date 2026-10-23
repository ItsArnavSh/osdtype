package services

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxRunNameRunes        = 80
	maxRunDescriptionRunes = 500
)

// RunMetaRequest describes the player-editable presentation of one run.
// Pointer fields distinguish "leave unchanged" from clearing a value.
type RunMetaRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Pinned      *bool   `json:"pinned"`
}

func (s *ServiceLayer) UpdateRunMeta(ctx context.Context, uid, gameID uint32, req RunMetaRequest) error {
	updates := make(map[string]any)
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if utf8.RuneCountInString(name) > maxRunNameRunes {
			return errf(ErrBadRequest, "run name is too long")
		}
		updates["run_name"] = name
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(description) > maxRunDescriptionRunes {
			return errf(ErrBadRequest, "run description is too long")
		}
		updates["run_description"] = description
	}
	if req.Pinned != nil {
		updates["pinned"] = *req.Pinned
		if *req.Pinned {
			updates["pinned_at"] = time.Now().UTC()
		} else {
			updates["pinned_at"] = nil
		}
	}
	if len(updates) == 0 {
		return errf(ErrBadRequest, "nothing to update")
	}
	found, err := s.db.UpdateRunMeta(ctx, gameID, uid, updates)
	if err != nil {
		return err
	}
	if !found {
		return errf(ErrNotFound, "run not found")
	}
	return nil
}
