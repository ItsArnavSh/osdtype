package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	gamecore "osdtyp/app/core/game"
	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"gorm.io/gorm"
)

type stagePerformance struct {
	user     entity.User
	wpm      float64
	raw      float64
	accuracy float64
	attempts int
	gameID   uint32
	flagged  bool
	left     bool
}

func stageEntryEligible(entry entity.ContestEntry, position int) bool {
	if entry.EliminatedAtStage > 0 && entry.EliminatedAtStage < position {
		return false
	}
	if entry.Eliminated && entry.EliminatedAtStage == 0 {
		return false
	}
	if entry.CurrentStage < position && entry.EliminatedAtStage != position {
		return false
	}
	return true
}

func stageEligibleEntries(entries []entity.ContestEntry, position int) []entity.ContestEntry {
	out := make([]entity.ContestEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.User != nil && entry.User.IsBot {
			continue
		}
		if stageEntryEligible(entry, position) {
			out = append(out, entry)
		}
	}
	return out
}

func projectedStageEnd(stage entity.ContestStage, startsAt time.Time) time.Time {
	if stage.EndsAt != nil && !stage.EndsAt.IsZero() {
		return stage.EndsAt.UTC()
	}
	switch stage.Format {
	case entity.ContestStageQualifier, entity.ContestStageHotLaps:
		return startsAt.Add(time.Duration(stage.WindowMinutes) * time.Minute)
	default:
		return startsAt.Add(time.Duration(stage.Duration)*time.Second + 20*time.Second)
	}
}

func projectStageStart(c entity.Contest, stages []entity.ContestStage, index int) time.Time {
	if index == 0 {
		if stages[0].StartsAt.IsZero() {
			return c.StartsAt
		}
		return stages[0].StartsAt
	}
	if !stages[index].StartsAt.IsZero() {
		return stages[index].StartsAt
	}
	previous := stages[index-1]
	previousStart := projectStageStart(c, stages, index-1)
	return projectedStageEnd(previous, previousStart).Add(time.Duration(stages[index].DelayMinutes) * time.Minute)
}

func (s *ServiceLayer) contestStageViews(ctx context.Context, c entity.Contest, stages []entity.ContestStage, entries []entity.ContestEntry, viewerID uint32, includeStandings bool) ([]ContestStageView, error) {
	out := make([]ContestStageView, 0, len(stages))
	for i, stage := range stages {
		startsAt := projectStageStart(c, stages, i)
		view := ContestStageView{
			ID:                 stage.ID,
			Position:           stage.Position,
			Name:               stage.Name,
			Format:             stage.Format,
			Status:             stage.Status,
			StartsAt:           startsAt,
			EndsAt:             stage.EndsAt,
			DelayMinutes:       stage.DelayMinutes,
			WindowMinutes:      stage.WindowMinutes,
			Duration:           stage.Duration,
			LanguageMode:       stage.LanguageMode,
			Language:           stage.Language,
			MaxTries:           stage.MaxTries,
			MinWPM:             stage.MinWPM,
			AdvanceType:        stage.AdvanceType,
			AdvanceValue:       stage.AdvanceValue,
			MaxPlayers:         stage.MaxPlayers,
			ViewerAttemptsLeft: -1,
		}
		eligible := stageEligibleEntries(entries, stage.Position)
		view.EntryCount = len(eligible)
		if includeStandings {
			perfs, err := s.collectStagePerformances(ctx, c.ID, stage)
			if err != nil {
				return nil, err
			}
			standings := s.buildStageStandings(stage, stages, entries, perfs, eligible)
			view.Standings = standings
			view.QualifiedCount = qualifiedPerformanceCount(stage, perfs)
			view.AdvanceCount = stageAdvanceCount(stage, view.QualifiedCount)
			view.TopRuns, view.RunCount, err = s.collectStageRuns(ctx, c.ID, stage)
			if err != nil {
				return nil, err
			}
			if viewerID != 0 {
				if stage.ID != 0 {
					if g, err := s.db.ActiveContestStageGame(ctx, stage.ID, viewerID); err == nil {
						view.ViewerGameID = g.ID
					}
				}
				for _, entry := range entries {
					if entry.UserID != viewerID {
						continue
					}
					if perf, ok := perfs[viewerID]; ok {
						view.ViewerAttempts = perf.attempts
						view.ViewerBestWPM = perf.wpm
					}
					if stage.MaxTries > 0 {
						left := stage.MaxTries - view.ViewerAttempts
						if left < 0 {
							left = 0
						}
						view.ViewerAttemptsLeft = left
					}
					if stage.Status == entity.ContestStageActive &&
						(stage.Format == entity.ContestStageQualifier || stage.Format == entity.ContestStageHotLaps) &&
						stageEntryEligible(entry, stage.Position) && !entry.Eliminated {
						view.ViewerCanAttempt = s.canStartContestAttempt(ctx, c, stage, entry, view)
					}
					break
				}
			}
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *ServiceLayer) collectStagePerformances(ctx context.Context, contestID uint32, stage entity.ContestStage) (map[uint32]stagePerformance, error) {
	games, err := s.db.ListContestStageGames(ctx, contestID, stage.ID)
	if err != nil {
		return nil, err
	}
	perfs := make(map[uint32]stagePerformance)
	for _, game := range games {
		players, err := s.db.GetGamePlayers(ctx, game.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range players {
			if p.User == nil {
				continue
			}
			current, ok := perfs[p.UserID]
			if !ok {
				current = stagePerformance{user: *p.User}
			}
			current.attempts++
			// Open stages keep every attempt; the best score is the one that
			// counts. Live heats have exactly one game per player.
			better := !ok || p.WPM > current.wpm ||
				(p.WPM == current.wpm && p.Accuracy > current.accuracy)
			if better && (game.Status == entity.GameFinished || p.Finished) {
				current.wpm = p.WPM
				current.raw = p.Raw
				current.accuracy = p.Accuracy
				current.gameID = game.ID
			}
			if game.Status == entity.GameFinished || p.Finished {
				current.flagged = current.flagged || p.Flagged
				current.left = current.left || p.Left
			}
			perfs[p.UserID] = current
		}
	}
	return perfs, nil
}

// collectStageRuns exposes the best individual attempts for the public event
// dashboard. Standings collapse every player to one best run; this keeps the
// attempt board visible so hosts and spectators can see how a score was built.
func (s *ServiceLayer) collectStageRuns(ctx context.Context, contestID uint32, stage entity.ContestStage) ([]ContestStageRun, int, error) {
	games, err := s.db.ListContestStageGames(ctx, contestID, stage.ID)
	if err != nil {
		return nil, 0, err
	}
	type run struct {
		user    entity.User
		wpm     float64
		raw     float64
		acc     float64
		gameID  uint32
		flagged bool
		left    bool
		at      time.Time
	}
	runs := make([]run, 0)
	runCount := 0
	for _, game := range games {
		players, err := s.db.GetGamePlayers(ctx, game.ID)
		if err != nil {
			return nil, 0, err
		}
		for _, p := range players {
			if p.User == nil {
				continue
			}
			runCount++
			recorded := game.Status == entity.GameFinished || p.Finished || p.WPM > 0 || p.Left
			if !recorded {
				continue
			}
			runs = append(runs, run{
				user:    *p.User,
				wpm:     p.WPM,
				raw:     p.Raw,
				acc:     p.Accuracy,
				gameID:  game.ID,
				flagged: p.Flagged,
				left:    p.Left,
				at:      game.CreatedAt,
			})
		}
	}
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].wpm != runs[j].wpm {
			return runs[i].wpm > runs[j].wpm
		}
		if runs[i].acc != runs[j].acc {
			return runs[i].acc > runs[j].acc
		}
		if !runs[i].at.Equal(runs[j].at) {
			return runs[i].at.Before(runs[j].at)
		}
		return runs[i].user.Username < runs[j].user.Username
	})
	if len(runs) > 8 {
		runs = runs[:8]
	}
	out := make([]ContestStageRun, 0, len(runs))
	for i, run := range runs {
		out = append(out, ContestStageRun{
			Rank:     i + 1,
			User:     entity.NewUserView(run.user),
			WPM:      run.wpm,
			Raw:      run.raw,
			Accuracy: run.acc,
			GameID:   run.gameID,
			Flagged:  run.flagged,
			Left:     run.left,
		})
	}
	return out, runCount, nil
}

func qualifiedPerformanceCount(stage entity.ContestStage, perfs map[uint32]stagePerformance) int {
	count := 0
	for _, perf := range perfs {
		if perf.flagged || perf.left {
			continue
		}
		if stage.Format == entity.ContestStageQualifier && perf.wpm < float64(stage.MinWPM) {
			continue
		}
		if perf.wpm <= 0 {
			continue
		}
		count++
	}
	return count
}

func stageAdvanceCount(stage entity.ContestStage, qualified int) int {
	if qualified <= 0 {
		return 0
	}
	switch stage.AdvanceType {
	case entity.ContestAdvanceCount:
		if stage.AdvanceValue < qualified {
			return stage.AdvanceValue
		}
		return qualified
	case entity.ContestAdvancePercent:
		n := int(math.Ceil(float64(qualified) * float64(stage.AdvanceValue) / 100))
		if n < 1 {
			n = 1
		}
		if n > qualified {
			n = qualified
		}
		return n
	default:
		return qualified
	}
}

func (s *ServiceLayer) buildStageStandings(stage entity.ContestStage, stages []entity.ContestStage, entries []entity.ContestEntry, perfs map[uint32]stagePerformance, eligible []entity.ContestEntry) []ContestStageStanding {
	entryByUser := make(map[uint32]entity.ContestEntry, len(entries))
	for _, entry := range entries {
		entryByUser[entry.UserID] = entry
	}
	standings := make([]ContestStageStanding, 0, len(eligible))
	for _, entry := range eligible {
		perf, ok := perfs[entry.UserID]
		if !ok || perf.attempts == 0 {
			continue
		}
		standings = append(standings, ContestStageStanding{
			User:     entity.NewUserView(perf.user),
			WPM:      perf.wpm,
			Raw:      perf.raw,
			Accuracy: perf.accuracy,
			Attempts: perf.attempts,
			GameID:   perf.gameID,
			Flagged:  perf.flagged,
			Left:     perf.left,
		})
	}
	sort.SliceStable(standings, func(i, j int) bool {
		if standings[i].WPM != standings[j].WPM {
			return standings[i].WPM > standings[j].WPM
		}
		if standings[i].Accuracy != standings[j].Accuracy {
			return standings[i].Accuracy > standings[j].Accuracy
		}
		return standings[i].User.Username < standings[j].User.Username
	})
	for i := range standings {
		standings[i].Rank = i + 1
		entry := entryByUser[standings[i].User.ID]
		standings[i].Eliminated = entry.Eliminated && entry.EliminatedAtStage == stage.Position
		if stage.Status == entity.ContestStageCompleted && stage.Position < len(stages) {
			standings[i].Advanced = entry.CurrentStage > stage.Position
		}
	}
	return standings
}

func (s *ServiceLayer) canStartContestAttempt(ctx context.Context, c entity.Contest, stage entity.ContestStage, entry entity.ContestEntry, view ContestStageView) bool {
	if c.Status != entity.ContestStarted || stage.Status != entity.ContestStageActive {
		return false
	}
	if stage.Format != entity.ContestStageQualifier && stage.Format != entity.ContestStageHotLaps {
		return false
	}
	if entry.Eliminated || entry.CurrentStage != stage.Position {
		return false
	}
	if view.ViewerGameID != 0 {
		return false
	}
	if view.ViewerAttemptsLeft == 0 {
		return false
	}
	if stage.EndsAt == nil {
		return false
	}
	now := time.Now().UTC()
	if !now.Add(time.Duration(stage.Duration)*time.Second + s.countdown + 2*time.Second).Before(stage.EndsAt.UTC()) {
		return false
	}
	if _, inGame := s.g().IsInGame(entry.UserID); inGame {
		return false
	}
	return true
}

func (s *ServiceLayer) HandleContestStageTask(ctx context.Context, task entity.Task) error {
	if task.Category != entity.CONTEST_STAGE {
		return nil
	}
	stage, err := s.db.GetContestStage(ctx, task.JobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	contest, err := s.db.GetContest(ctx, stage.ContestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if contest.Status == entity.ContestCancelled || contest.Status == entity.ContestEnded {
		return nil
	}
	now := time.Now().UTC()
	switch stage.Status {
	case entity.ContestStagePending:
		if stage.StartsAt.After(now.Add(2 * time.Second)) {
			return s.db.CreateTask(ctx, &entity.Task{
				Category: entity.CONTEST_STAGE,
				JobID:    stage.ID,
				Time:     stage.StartsAt,
			})
		}
		return s.startContestStage(ctx, contest, stage)
	case entity.ContestStageActive:
		return s.finishContestStage(ctx, contest, stage)
	default:
		return nil
	}
}

func (s *ServiceLayer) startContestStage(ctx context.Context, contest entity.Contest, stage entity.ContestStage) error {
	entries, err := s.db.ListContestEntries(ctx, contest.ID)
	if err != nil {
		return err
	}
	eligible := stageEligibleEntries(entries, stage.Position)
	if stage.Position == 1 && len(eligible) < 2 {
		return s.cancelContestForStart(ctx, contest, "at least two enrolled players are needed to start")
	}
	if len(eligible) == 0 {
		return s.cancelContestForStart(ctx, contest, "no players advanced to this stage")
	}

	now := time.Now().UTC()
	stage.Status = entity.ContestStageActive
	stage.StartedAt = &now
	stage.UpdatedAt = now
	stage.StartsAt = now

	if stage.Format == entity.ContestStageQualifier || stage.Format == entity.ContestStageHotLaps {
		if stage.Format == entity.ContestStageHotLaps && stage.Seed == 0 {
			stage.Seed = randomSeed()
		}
		ends := now.Add(time.Duration(stage.WindowMinutes) * time.Minute)
		stage.EndsAt = &ends
	} else {
		stage.Seed = randomSeed()
		// The live race starts after the countdown. Keep the stage open a
		// little longer so the finish hook always lands before aggregation.
		ends := now.Add(s.countdown + time.Duration(stage.Duration)*time.Second + 10*time.Second)
		stage.EndsAt = &ends
	}

	contest.Status = entity.ContestStarted
	contest.UpdatedAt = now
	if err := s.db.SaveContest(ctx, &contest); err != nil {
		return err
	}
	if err := s.db.SaveContestStage(ctx, &stage); err != nil {
		return err
	}

	if stage.Format == entity.ContestStageMultiplayer || stage.Format == entity.ContestStageRanked {
		if err := s.startContestLiveHeats(ctx, contest, stage, eligible); err != nil {
			s.logger.Errorw("contest live stage could not start", "contest", contest.Code, "stage", stage.Position, "error", err)
			return s.cancelContestForStart(ctx, contest, "the live stage could not start")
		}
	}
	if stage.EndsAt != nil {
		if err := s.db.CreateTask(ctx, &entity.Task{
			Category: entity.CONTEST_STAGE,
			JobID:    stage.ID,
			Time:     stage.EndsAt.UTC().Add(2 * time.Second),
		}); err != nil {
			return err
		}
	}

	payload := map[string]any{
		"contest_code":   contest.Code,
		"title":          contest.Title,
		"stage_position": stage.Position,
		"stage_name":     stage.Name,
		"stage_format":   stage.Format,
	}
	if stage.Position == 1 {
		payload["game_id"] = contest.GameID
		s.notifyContest(ctx, contest, "contest_started", payload)
	} else {
		s.notifyContest(ctx, contest, "contest_stage_started", payload)
	}
	return nil
}

func (s *ServiceLayer) startContestLiveHeats(ctx context.Context, contest entity.Contest, stage entity.ContestStage, entries []entity.ContestEntry) error {
	if len(entries) == 0 {
		return nil
	}
	language := stage.Language
	if !validLanguage(language) || language == "random" {
		return errf(ErrBadRequest, "stage language is invalid")
	}
	seed := stage.Seed
	if seed == 0 {
		seed = randomSeed()
	}
	genCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	snippet, _ := s.codegen.GenerateSnippet(genCtx, language, seed)
	cancel()
	if snippet == "" {
		return errors.New("snippet generator returned an empty snippet")
	}
	startOffset := utils.SnippetStartOffset(seed, snippet)

	heatCount := int(math.Ceil(float64(len(entries)) / float64(max(2, stage.MaxPlayers))))
	if heatCount < 1 {
		heatCount = 1
	}
	heatSize := int(math.Ceil(float64(len(entries)) / float64(heatCount)))
	now := time.Now().UTC()
	starts := now.Add(s.countdown)
	ends := starts.Add(time.Duration(stage.Duration) * time.Second)
	lastGameID := uint32(0)
	for heat := 0; heat < heatCount; heat++ {
		from := heat * heatSize
		to := min(from+heatSize, len(entries))
		if from >= to {
			continue
		}
		heatEntries := entries[from:to]
		users := make(map[uint32]entity.User, len(heatEntries))
		players := make([]entity.GamePlayer, 0, len(heatEntries))
		for _, entry := range heatEntries {
			if entry.User == nil {
				continue
			}
			users[entry.UserID] = *entry.User
			players = append(players, entity.GamePlayer{
				UserID:       entry.UserID,
				RatingBefore: entry.User.RatingFor(stage.Duration),
				IsBot:        entry.User.IsBot,
			})
		}
		if len(players) == 0 {
			continue
		}
		g := entity.Game{
			ID:             s.intGen.GenerateID(),
			Mode:           entity.GameContest,
			ContestID:      contest.ID,
			ContestStageID: stage.ID,
			Duration:       stage.Duration,
			Language:       language,
			Seed:           seed,
			Snippet:        snippet,
			StartOffset:    startOffset,
			Status:         entity.GameCountdown,
			StartsAt:       &starts,
			EndsAt:         &ends,
			CreatedAt:      now,
		}
		if err := s.db.CreateGame(ctx, &g, players); err != nil {
			return err
		}
		for uid := range users {
			s.state().SetGame(ctx, uid, g.ID, "")
		}
		opts := gamecore.StartOptions{Game: g, Players: players, Users: users}
		if err := s.g().Start(ctx, opts); err != nil {
			_ = s.db.UpdateGameFields(ctx, g.ID, map[string]any{"status": entity.GameAbandoned})
			return err
		}
		lastGameID = g.ID
	}
	if lastGameID != 0 {
		contest.GameID = lastGameID
		_ = s.db.SaveContest(ctx, &contest)
	}
	return nil
}

func (s *ServiceLayer) finishContestStage(ctx context.Context, contest entity.Contest, stage entity.ContestStage) error {
	stages, err := s.db.ListContestStages(ctx, contest.ID)
	if err != nil {
		return err
	}
	entries, err := s.db.ListContestEntries(ctx, contest.ID)
	if err != nil {
		return err
	}
	perfs, err := s.collectStagePerformances(ctx, contest.ID, stage)
	if err != nil {
		return err
	}
	standings := s.buildStageStandings(stage, stages, entries, perfs, stageEligibleEntries(entries, stage.Position))
	now := time.Now().UTC()
	stage.Status = entity.ContestStageCompleted
	stage.EndsAt = &now
	stage.UpdatedAt = now
	if err := s.db.SaveContestStage(ctx, &stage); err != nil {
		return err
	}

	final := stage.Position == len(stages)
	if final {
		leaderboard := standingsToResults(standings)
		raw, _ := json.Marshal(leaderboard)
		contest.Status = entity.ContestEnded
		contest.Leaderboard = raw
		contest.UpdatedAt = now
		if lastGame := lastStageGameID(perfs); lastGame != 0 {
			contest.GameID = lastGame
		}
		if err := s.db.SaveContest(ctx, &contest); err != nil {
			return err
		}
		s.notifyContest(ctx, contest, "contest_ended", map[string]any{
			"contest_code":   contest.Code,
			"title":          contest.Title,
			"stage_position": stage.Position,
		})
		return nil
	}

	qualified := make([]ContestStageStanding, 0, len(standings))
	for _, standing := range standings {
		if standing.Flagged || standing.Left || standing.WPM <= 0 {
			continue
		}
		if stage.Format == entity.ContestStageQualifier && standing.WPM < float64(stage.MinWPM) {
			continue
		}
		qualified = append(qualified, standing)
	}
	advanceCount := stageAdvanceCount(stage, len(qualified))
	if advanceCount > len(qualified) {
		advanceCount = len(qualified)
	}
	advance := make(map[uint32]bool, advanceCount)
	for i := 0; i < advanceCount; i++ {
		advance[qualified[i].User.ID] = true
	}
	eligible := stageEligibleEntries(entries, stage.Position)
	nextPosition := stage.Position + 1
	for i := range entries {
		entry := entries[i]
		wasEligible := false
		for _, candidate := range eligible {
			if candidate.UserID == entry.UserID {
				wasEligible = true
				break
			}
		}
		if !wasEligible {
			continue
		}
		if advance[entry.UserID] {
			entries[i].CurrentStage = nextPosition
			entries[i].Eliminated = false
			entries[i].EliminatedAtStage = 0
		} else {
			entries[i].Eliminated = true
			entries[i].EliminatedAtStage = stage.Position
		}
		if err := s.db.SaveContestEntry(ctx, &entries[i]); err != nil {
			return err
		}
	}
	contest.UpdatedAt = now
	if advanceCount == 0 {
		contest.Status = entity.ContestEnded
		contest.Leaderboard, _ = json.Marshal([]entity.ResultView{})
		if err := s.db.SaveContest(ctx, &contest); err != nil {
			return err
		}
		s.notifyContest(ctx, contest, "contest_ended", map[string]any{
			"contest_code": contest.Code,
			"title":        contest.Title,
			"reason":       "no one advanced",
		})
		return nil
	}
	if err := s.db.SaveContest(ctx, &contest); err != nil {
		return err
	}
	next := stages[nextPosition-1]
	next.StartsAt = now.Add(time.Duration(next.DelayMinutes) * time.Minute)
	next.UpdatedAt = now
	if err := s.db.SaveContestStage(ctx, &next); err != nil {
		return err
	}
	if err := s.db.CreateTask(ctx, &entity.Task{
		Category: entity.CONTEST_STAGE,
		JobID:    next.ID,
		Time:     next.StartsAt,
	}); err != nil {
		return err
	}
	s.notifyContest(ctx, contest, "contest_stage_completed", map[string]any{
		"contest_code":   contest.Code,
		"title":          contest.Title,
		"stage_position": stage.Position,
		"advanced":       advanceCount,
		"next_stage":     next.Position,
		"next_starts_at": next.StartsAt,
	})
	return nil
}

func lastStageGameID(perfs map[uint32]stagePerformance) uint32 {
	var id uint32
	for _, perf := range perfs {
		if perf.gameID > id {
			id = perf.gameID
		}
	}
	return id
}

func standingsToResults(standings []ContestStageStanding) []entity.ResultView {
	out := make([]entity.ResultView, 0, len(standings))
	for _, standing := range standings {
		out = append(out, entity.ResultView{
			UserID:    standing.User.ID,
			Username:  standing.User.Username,
			AvatarURL: standing.User.AvatarURL,
			Rank:      standing.Rank,
			WPM:       standing.WPM,
			Raw:       standing.Raw,
			Accuracy:  standing.Accuracy,
			Flagged:   standing.Flagged,
			Left:      standing.Left,
		})
	}
	return out
}

func (s *ServiceLayer) StartContestAttempt(ctx context.Context, uid uint32, code string, position int, req ContestAttemptRequest) (entity.GameView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.GameView{}, errf(ErrNotFound, "contest not found")
		}
		return entity.GameView{}, err
	}
	if c.Status != entity.ContestStarted {
		return entity.GameView{}, errf(ErrConflict, "the contest is not accepting attempts")
	}
	stages, err := s.db.ListContestStages(ctx, c.ID)
	if err != nil {
		return entity.GameView{}, err
	}
	var stage *entity.ContestStage
	for i := range stages {
		if stages[i].Position == position {
			stage = &stages[i]
			break
		}
	}
	if stage == nil {
		return entity.GameView{}, errf(ErrNotFound, "stage not found")
	}
	if stage.Status != entity.ContestStageActive {
		return entity.GameView{}, errf(ErrConflict, "this stage is not open")
	}
	if stage.Format != entity.ContestStageQualifier && stage.Format != entity.ContestStageHotLaps {
		return entity.GameView{}, errf(ErrBadRequest, "this stage starts as a live race")
	}
	entries, err := s.db.ListContestEntries(ctx, c.ID)
	if err != nil {
		return entity.GameView{}, err
	}
	var entry *entity.ContestEntry
	for i := range entries {
		if entries[i].UserID == uid {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		return entity.GameView{}, errf(ErrForbidden, "enroll in this contest first")
	}
	if entry.Eliminated || entry.CurrentStage != stage.Position {
		return entity.GameView{}, errf(ErrForbidden, "you are not eligible for this stage")
	}
	if _, inGame := s.g().IsInGame(uid); inGame {
		return entity.GameView{}, errf(ErrConflict, "finish or leave your current run first")
	}
	if _, err := s.db.ActiveContestStageGame(ctx, stage.ID, uid); err == nil {
		return entity.GameView{}, errf(ErrConflict, "finish or leave your current attempt first")
	}
	attempts, err := s.db.CountContestStageGames(ctx, stage.ID, uid)
	if err != nil {
		return entity.GameView{}, err
	}
	if stage.MaxTries > 0 && int(attempts) >= stage.MaxTries {
		return entity.GameView{}, errf(ErrConflict, "you have used all of your attempts for this stage")
	}
	now := time.Now().UTC()
	if stage.EndsAt == nil || !now.Add(s.countdown+time.Duration(stage.Duration)*time.Second+2*time.Second).Before(stage.EndsAt.UTC()) {
		return entity.GameView{}, errf(ErrConflict, "there is not enough time left in this stage")
	}
	language := stage.Language
	if stage.LanguageMode == entity.ContestLanguageChoice {
		requested := strings.ToLower(strings.TrimSpace(req.Language))
		if requested != "" {
			language = requested
		}
	}
	if !validLanguage(language) || language == "random" {
		return entity.GameView{}, errf(ErrBadRequest, "choose a valid language for this attempt")
	}
	seed := stage.Seed
	if stage.Format == entity.ContestStageQualifier || seed == 0 {
		seed = randomSeed()
	}
	genCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	snippet, _ := s.codegen.GenerateSnippet(genCtx, language, seed)
	cancel()
	if snippet == "" {
		return entity.GameView{}, errors.New("could not generate a snippet for this attempt")
	}
	user, err := s.db.GetUser(uid)
	if err != nil {
		return entity.GameView{}, err
	}
	g := entity.Game{
		ID:             s.intGen.GenerateID(),
		Mode:           entity.GameContest,
		ContestID:      c.ID,
		ContestStageID: stage.ID,
		Duration:       stage.Duration,
		Language:       language,
		Seed:           seed,
		Snippet:        snippet,
		StartOffset:    utils.SnippetStartOffset(seed, snippet),
		Status:         entity.GameCountdown,
		StartsAt:       timePtr(now.Add(s.countdown)),
		EndsAt:         timePtr(now.Add(s.countdown + time.Duration(stage.Duration)*time.Second)),
		CreatedAt:      now,
	}
	players := []entity.GamePlayer{{
		UserID:       uid,
		RatingBefore: user.RatingFor(stage.Duration),
		IsBot:        false,
	}}
	if err := s.db.CreateGame(ctx, &g, players); err != nil {
		return entity.GameView{}, err
	}
	s.state().SetGame(ctx, uid, g.ID, "")
	opts := gamecore.StartOptions{
		Game:    g,
		Players: players,
		Users:   map[uint32]entity.User{uid: user},
	}
	if err := s.g().Start(ctx, opts); err != nil {
		_ = s.db.UpdateGameFields(ctx, g.ID, map[string]any{"status": entity.GameAbandoned})
		return entity.GameView{}, err
	}
	return s.GetGameView(ctx, uid, g.ID)
}

func timePtr(t time.Time) *time.Time { return &t }

func (s *ServiceLayer) cancelContestStages(ctx context.Context, contestID uint32) {
	stages, err := s.db.ListContestStages(ctx, contestID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for i := range stages {
		if stages[i].Status == entity.ContestStageCompleted {
			continue
		}
		stages[i].Status = entity.ContestStageCancelled
		stages[i].UpdatedAt = now
		_ = s.db.SaveContestStage(ctx, &stages[i])
	}
}
