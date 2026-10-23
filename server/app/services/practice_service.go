package services

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"osdtyp/app/entity"
	"osdtyp/app/utils"
)

// PracticeRunRequest mirrors the client-side typing session. The snippet is
// included because practice is generated locally by the WASM codegen.
type PracticeRunRequest struct {
	Language    string            `json:"language"`
	Duration    int               `json:"duration"`
	DurationMS  int64             `json:"duration_ms"`
	Seed        string            `json:"seed"`
	Snippet     string            `json:"snippet"`
	StartOffset int               `json:"start_offset"`
	Final       string            `json:"final"`
	Events      []entity.Keypress `json:"events"`
}

func (s *ServiceLayer) CreatePracticeRun(ctx context.Context, uid uint32, req PracticeRunRequest) (entity.PracticeRunView, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return entity.PracticeRunView{}, err
	}
	if req.Duration <= 0 || req.Duration > 3600 {
		return entity.PracticeRunView{}, errf(ErrBadRequest, "invalid practice duration")
	}
	if len(req.Snippet) == 0 || len(req.Snippet) > 200_000 {
		return entity.PracticeRunView{}, errf(ErrBadRequest, "invalid practice snippet")
	}
	if len(req.Events) > 50_000 {
		return entity.PracticeRunView{}, errf(ErrBadRequest, "too many keystrokes")
	}
	if req.StartOffset < 0 || req.StartOffset > len(req.Snippet) {
		return entity.PracticeRunView{}, errf(ErrBadRequest, "invalid start offset")
	}

	events := append([]entity.Keypress(nil), req.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].TimeMS < events[j].TimeMS })
	final := req.Final
	if final == "" {
		final = typedFromEvents(events)
	}

	durationMS := req.DurationMS
	maxMS := int64(req.Duration) * 1000
	if durationMS <= 0 || durationMS > maxMS {
		durationMS = maxMS
	}
	if durationMS < 1000 {
		durationMS = 1000
	}

	score := utils.Calculate_WPM(entity.WPM{
		OriginalSnippet: req.Snippet[req.StartOffset:],
		UserSnippet:     final,
		DurationMS:      durationMS,
	})

	seed, _ := strconv.ParseUint(strings.TrimSpace(req.Seed), 10, 64)
	now := time.Now().UTC()
	started := now.Add(-time.Duration(durationMS) * time.Millisecond)
	language := strings.TrimSpace(req.Language)
	if language == "" || len(language) > 16 {
		language = "go"
	}
	game := entity.Game{
		ID:          s.intGen.GenerateID(),
		Mode:        entity.GamePractice,
		Duration:    req.Duration,
		Language:    language,
		Seed:        seed,
		Snippet:     req.Snippet,
		StartOffset: req.StartOffset,
		Status:      entity.GameFinished,
		Temporary:   true,
		StartsAt:    &started,
		EndsAt:      &now,
		CreatedAt:   now,
		FinishedAt:  &now,
	}
	player := entity.GamePlayer{
		GameID:        game.ID,
		UserID:        uid,
		RatingBefore:  user.Rating,
		RatingAfter:   user.Rating,
		WPM:           score.WPM,
		Raw:           score.RAW,
		Accuracy:      score.Accuracy,
		Correct:       score.Correct,
		Wrong:         score.Wrong,
		CurrentPoints: req.StartOffset,
		Placement:     1,
		Finished:      true,
		Acked:         true,
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return entity.PracticeRunView{}, err
	}
	recording := entity.Recording{
		GameID:    game.ID,
		UserID:    uid,
		Events:    raw,
		Final:     final,
		CreatedAt: now,
	}
	if err := s.db.CreatePracticeRun(ctx, &game, &player, &recording); err != nil {
		return entity.PracticeRunView{}, err
	}
	return entity.PracticeRunView{
		ID:        game.ID,
		Temporary: true,
		URL:       s.practiceURL(game.ID, uid),
	}, nil
}

func (s *ServiceLayer) SavePracticeRun(ctx context.Context, uid, gameID uint32) (entity.PracticeRunView, error) {
	saved, err := s.db.SavePracticeRun(ctx, gameID, uid)
	if err != nil {
		return entity.PracticeRunView{}, err
	}
	if !saved {
		return entity.PracticeRunView{}, errf(ErrNotFound, "practice run not found")
	}
	return entity.PracticeRunView{
		ID:        gameID,
		Temporary: false,
		URL:       s.practiceURL(gameID, uid),
	}, nil
}

func (s *ServiceLayer) practiceURL(gameID, userID uint32) string {
	return s.frontendURL + "/replay/" + strconv.FormatUint(uint64(gameID), 10) + "/" + strconv.FormatUint(uint64(userID), 10)
}

func typedFromEvents(events []entity.Keypress) string {
	typed := make([]rune, 0, len(events))
	for _, ev := range events {
		if ev.Action == entity.BACKSPACE {
			if n := len([]rune(ev.Value)); n > 0 && len(typed) >= n {
				typed = typed[:len(typed)-n]
			} else if len(typed) > 0 {
				typed = typed[:len(typed)-1]
			}
			continue
		}
		typed = append(typed, []rune(ev.Value)...)
	}
	return string(typed)
}
