package game

import (
	"context"
	"encoding/json"
	"math/rand"
	"osdtyp/app/core/anticheat"
	"osdtyp/app/core/hub"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type playerState struct {
	user         entity.User
	ratingBefore int
	isBot        bool
	botCfg       entity.BotConfig
	typed        []rune
	events       []entity.Keypress
	timestamps   []int32
	finished     bool
	left         bool
	finishedAt   time.Time
	leftAt       time.Time
}

type runtime struct {
	m        *Manager
	game     entity.Game
	players  map[uint32]*playerState
	order    []uint32
	started  time.Time
	ends     time.Time
	done     chan struct{}
	doneOnce sync.Once

	mu       sync.Mutex
	finished bool
}

func newRuntime(m *Manager, opts StartOptions) *runtime {
	rt := &runtime{
		m:       m,
		game:    opts.Game,
		players: make(map[uint32]*playerState),
		order:   make([]uint32, 0, len(opts.Players)),
		done:    make(chan struct{}),
		started: opts.Game.StartsAt.UTC(),
		ends:    opts.Game.EndsAt.UTC(),
	}
	for _, gp := range opts.Players {
		user, ok := opts.Users[gp.UserID]
		if !ok {
			continue
		}
		ps := &playerState{
			user:         user,
			ratingBefore: user.Rating,
			isBot:        user.IsBot || gp.IsBot,
			botCfg:       opts.Bots[gp.UserID],
		}
		rt.players[gp.UserID] = ps
		rt.order = append(rt.order, gp.UserID)
	}
	return rt
}

func (r *runtime) run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	endTimer := time.NewTimer(time.Until(r.ends))
	defer endTimer.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.broadcast(hub.Ev("progress", r.progress()))
			r.mu.Lock()
			done := r.finished || r.allHumansDoneLocked()
			r.mu.Unlock()
			if done {
				r.finish(ctx)
				return
			}
		case <-endTimer.C:
			r.finish(ctx)
			return
		}
	}
}

func (r *runtime) applyInput(userID uint32, kp entity.Keypress) {
	now := time.Now().UTC()
	if now.Before(r.started) {
		return
	}

	r.mu.Lock()
	ps := r.players[userID]
	if ps == nil || r.finished || ps.finished || ps.left {
		r.mu.Unlock()
		return
	}

	if kp.Action == entity.BACKSPACE {
		if len(kp.Value) > 0 && strings.HasSuffix(string(ps.typed), kp.Value) {
			ps.typed = ps.typed[:len(ps.typed)-utf8.RuneCountInString(kp.Value)]
		} else if len(ps.typed) > 0 {
			ps.typed = ps.typed[:len(ps.typed)-1]
		}
	} else {
		ps.typed = append(ps.typed, []rune(kp.Value)...)
	}

	delta := int32(now.Sub(r.started).Milliseconds())
	if delta < 0 {
		delta = 0
	}
	if !ps.isBot {
		ps.events = append(ps.events, entity.Keypress{Value: kp.Value, Action: kp.Action, TimeMS: int64(delta)})
		if len(ps.timestamps) < 20000 {
			ps.timestamps = append(ps.timestamps, delta)
		}
	}

	if _, more := nextChar(r.game.Snippet, r.game.StartOffset, len([]rune(string(ps.typed)))); !more {
		ps.finished = true
		ps.finishedAt = now
	}
	allDone := r.allHumansDoneLocked()
	r.mu.Unlock()

	if now.After(r.ends) || allDone {
		r.finish(context.Background())
	}
}

func (r *runtime) allHumansDoneLocked() bool {
	humans := 0
	for _, ps := range r.players {
		if ps.isBot {
			continue
		}
		humans++
		if !ps.finished && !ps.left {
			return false
		}
	}
	return humans > 0
}

func (r *runtime) leave(userID uint32) {
	now := time.Now().UTC()
	r.mu.Lock()
	ps := r.players[userID]
	if ps != nil && !ps.left {
		ps.left = true
		ps.leftAt = now
		ps.finished = true
	}
	allDone := r.allHumansDoneLocked()
	r.mu.Unlock()
	if allDone {
		r.finish(context.Background())
	}
}

func (r *runtime) finish(ctx context.Context) {
	r.mu.Lock()
	if r.finished {
		r.mu.Unlock()
		return
	}
	r.finished = true
	game := r.game
	playerIDs := append([]uint32(nil), r.order...)

	var gps []entity.GamePlayer
	var recordings []entity.Recording
	var flags []entity.CheatFlag
	now := time.Now().UTC()

	for _, uid := range r.order {
		ps := r.players[uid]
		duration := r.game.Duration * 1000
		if ps.left && !ps.leftAt.IsZero() {
			duration = int(ps.leftAt.Sub(r.started).Milliseconds())
		} else if ps.finished && !ps.finishedAt.IsZero() {
			duration = int(ps.finishedAt.Sub(r.started).Milliseconds())
		}
		if duration < 1000 {
			duration = 1000
		}

		typed := string(ps.typed)
		original := ""
		if r.game.StartOffset >= 0 && r.game.StartOffset <= len(r.game.Snippet) {
			original = r.game.Snippet[r.game.StartOffset:]
		}
		score := utils.Calculate_WPM(entity.WPM{
			OriginalSnippet: original,
			UserSnippet:     typed,
			DurationMS:      int64(duration),
		})
		report := anticheat.Analyze(ps.timestamps, ps.events, len([]rune(typed)), int64(duration), ps.isBot)

		gp := entity.GamePlayer{
			GameID:        r.game.ID,
			UserID:        uid,
			RatingBefore:  ps.ratingBefore,
			RatingAfter:   ps.ratingBefore,
			WPM:           score.WPM,
			Raw:           score.RAW,
			Accuracy:      score.Accuracy,
			Correct:       score.Correct,
			Wrong:         score.Wrong,
			CurrentPoints: progressIndex(r.game.Snippet, r.game.StartOffset, len(ps.typed)),
			Finished:      ps.finished && !ps.left,
			Left:          ps.left,
			Flagged:       report.Flagged,
			IsBot:         ps.isBot,
		}
		gps = append(gps, gp)

		if !ps.isBot {
			raw, _ := json.Marshal(ps.events)
			recordings = append(recordings, entity.Recording{
				GameID:    r.game.ID,
				UserID:    uid,
				Events:    raw,
				Final:     typed,
				CreatedAt: now,
			})
		}
		for _, reason := range report.Reasons {
			flags = append(flags, entity.CheatFlag{
				GameID: r.game.ID, UserID: uid, Reason: reason, Risk: report.Risk, CreatedAt: now,
			})
		}
	}

	sort.SliceStable(gps, func(i, j int) bool {
		if gps[i].Left != gps[j].Left {
			return !gps[i].Left
		}
		if gps[i].WPM != gps[j].WPM {
			return gps[i].WPM > gps[j].WPM
		}
		return gps[i].Accuracy > gps[j].Accuracy
	})
	for i := range gps {
		gps[i].Placement = i + 1
	}
	r.mu.Unlock()

	views := make([]entity.ResultView, 0, len(gps))
	for _, gp := range gps {
		username := ""
		avatar := ""
		if ps := r.players[gp.UserID]; ps != nil {
			username = ps.user.Username
			avatar = ps.user.AvatarURL
		}
		views = append(views, entity.ResultView{
			UserID:       gp.UserID,
			Username:     username,
			AvatarURL:    avatar,
			Rank:         gp.Placement,
			WPM:          gp.WPM,
			Raw:          gp.Raw,
			Accuracy:     gp.Accuracy,
			Correct:      gp.Correct,
			Wrong:        gp.Wrong,
			RatingBefore: gp.RatingBefore,
			RatingAfter:  gp.RatingAfter,
			Flagged:      gp.Flagged,
			Left:         gp.Left,
			IsBot:        gp.IsBot,
		})
	}

	if r.m.onFinish != nil {
		final, err := r.m.onFinish(ctx, game, gps, recordings, flags)
		if err != nil {
			r.m.logger.Errorw("finish hook failed", "game_id", game.ID, "error", err)
		} else if len(final) > 0 {
			views = final
		}
	}

	r.broadcast(hub.Ev("leaderboard", views))
	r.closeDone()
	r.m.remove(game.ID)
	_ = playerIDs
}

func (r *runtime) originalLocked() string {
	if r.game.StartOffset < 0 || r.game.StartOffset > len(r.game.Snippet) {
		return r.game.Snippet
	}
	return r.game.Snippet[r.game.StartOffset:]
}

func (r *runtime) progress() []entity.PlayerProgress {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]entity.PlayerProgress, 0, len(r.order))
	for _, uid := range r.order {
		ps := r.players[uid]
		if ps == nil {
			continue
		}
		elapsed := now.Sub(r.started).Seconds()
		if elapsed < 1 {
			elapsed = 1
		}
		typedRunes := []rune(string(ps.typed))
		originalRunes := []rune(r.originalLocked())
		correct := 0
		compareLen := min(len(typedRunes), len(originalRunes))
		for i := 0; i < compareLen; i++ {
			if typedRunes[i] == originalRunes[i] {
				correct++
			}
		}
		wpm := (float64(len(typedRunes)) / 5.0) / (elapsed / 60.0)
		acc := 0.0
		if len(typedRunes) > 0 {
			acc = float64(correct) / float64(len(typedRunes))
		}
		out = append(out, entity.PlayerProgress{
			UserID:        uid,
			Username:      ps.user.Username,
			AvatarURL:     ps.user.AvatarURL,
			CurrentPoints: progressIndex(r.game.Snippet, r.game.StartOffset, len(ps.typed)),
			WPM:           wpm,
			Accuracy:      acc,
			Finished:      ps.finished,
			Flagged:       false,
			IsBot:         ps.isBot,
		})
	}
	return out
}

func (r *runtime) view(userID uint32) entity.GameView {
	r.mu.Lock()
	defer r.mu.Unlock()
	view := entity.GameView{
		ID:          r.game.ID,
		Mode:        r.game.Mode,
		Status:      r.game.Status,
		RoomID:      r.game.RoomID,
		ContestID:   r.game.ContestID,
		Duration:    r.game.Duration,
		Language:    r.game.Language,
		Seed:        strconv.FormatUint(r.game.Seed, 10),
		Snippet:     r.game.Snippet,
		StartOffset: r.game.StartOffset,
		StartsAt:    r.game.StartsAt,
		EndsAt:      r.game.EndsAt,
		CreatedAt:   r.game.CreatedAt,
		Players:     r.progressLocked(),
	}
	return view
}

func (r *runtime) progressLocked() []entity.PlayerProgress {
	now := time.Now().UTC()
	out := make([]entity.PlayerProgress, 0, len(r.order))
	for _, uid := range r.order {
		ps := r.players[uid]
		if ps == nil {
			continue
		}
		elapsed := now.Sub(r.started).Seconds()
		if elapsed < 1 {
			elapsed = 1
		}
		typedRunes := []rune(string(ps.typed))
		origRunes := []rune(r.originalLocked())
		correct := 0
		compareLen := min(len(typedRunes), len(origRunes))
		for i := 0; i < compareLen; i++ {
			if typedRunes[i] == origRunes[i] {
				correct++
			}
		}
		wpm := (float64(len(typedRunes)) / 5.0) / (elapsed / 60.0)
		acc := 0.0
		if len(typedRunes) > 0 {
			acc = float64(correct) / float64(len(typedRunes))
		}
		out = append(out, entity.PlayerProgress{
			UserID:        uid,
			Username:      ps.user.Username,
			AvatarURL:     ps.user.AvatarURL,
			CurrentPoints: progressIndex(r.game.Snippet, r.game.StartOffset, len(ps.typed)),
			WPM:           wpm,
			Accuracy:      acc,
			Finished:      ps.finished,
			IsBot:         ps.isBot,
		})
	}
	return out
}

func (r *runtime) closeDone() {
	r.doneOnce.Do(func() { close(r.done) })
}

func (r *runtime) broadcast(ev hub.Event) {
	ids := make([]uint32, 0, len(r.order))
	for _, uid := range r.order {
		ps := r.players[uid]
		if ps != nil && !ps.isBot {
			ids = append(ids, uid)
		}
	}
	r.m.hub.SendToUsers(ids, ev)
}

func (r *runtime) runBot(ctx context.Context, uid uint32, cfg entity.BotConfig) {
	r.mu.Lock()
	ps := r.players[uid]
	r.mu.Unlock()
	if ps == nil {
		return
	}
	wpm := cfg.BotWPM()
	if wpm <= 0 {
		wpm = 80
	}
	acc := cfg.Accuracy
	if acc <= 0 || acc > 100 {
		acc = 90
	}
	charsPerSecond := wpm * 5 / 60
	rng := rand.New(rand.NewSource(int64(r.game.Seed) ^ int64(uid)))

	wait := time.Until(r.started)
	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-r.done:
			return
		}
	}

	ticker := time.NewTicker(60 * time.Millisecond)
	defer ticker.Stop()
	accum := 0.0
	backspacePending := false

	for {
		select {
		case <-r.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			ps := r.players[uid]
			if ps == nil || ps.finished || ps.left || r.finished {
				r.mu.Unlock()
				return
			}
			typedLen := len(ps.typed)
			r.mu.Unlock()

			if time.Now().UTC().After(r.ends) {
				return
			}

			if backspacePending {
				r.applyInput(uid, entity.Keypress{Action: entity.BACKSPACE, Value: ""})
				backspacePending = false
				continue
			}

			accum += charsPerSecond * 0.06
			if accum < 1 {
				continue
			}
			accum -= 1

			next, ok := nextChar(r.game.Snippet, r.game.StartOffset, typedLen)
			if !ok {
				r.mu.Lock()
				ps.finished = true
				ps.finishedAt = time.Now().UTC()
				r.mu.Unlock()
				return
			}
			if rng.Intn(100) >= acc {
				// Mistype once, then erase it on the next tick.
				wrong := rune('x')
				if next == wrong {
					wrong = rune('z')
				}
				r.applyInput(uid, entity.Keypress{Action: entity.KEYPRESS, Value: string(wrong)})
				backspacePending = true
				continue
			}
			r.applyInput(uid, entity.Keypress{Action: entity.KEYPRESS, Value: string(next)})
		}
	}
}

// progressIndex maps a number of typed characters to the matching byte index
// in the display snippet, skipping whitespace exactly like the browser does.
func progressIndex(snippet string, start, typedLen int) int {
	if start < 0 {
		start = 0
	}
	if start > len(snippet) {
		return len(snippet)
	}
	if typedLen <= 0 {
		return start
	}
	idx := start
	count := 0
	for idx < len(snippet) {
		r, size := utf8.DecodeRuneInString(snippet[idx:])
		idx += size
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		count++
		if count >= typedLen {
			return idx
		}
	}
	return len(snippet)
}

func nextChar(snippet string, start, typedLen int) (rune, bool) {
	idx := progressIndex(snippet, start, typedLen)
	for idx < len(snippet) {
		r, size := utf8.DecodeRuneInString(snippet[idx:])
		idx += size
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		return r, true
	}
	return 0, false
}
