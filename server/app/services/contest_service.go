package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"osdtyp/app/core/hub"
	"osdtyp/app/entity"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const contestAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type ContestStageInput struct {
	Name          string                     `json:"name"`
	Format        entity.ContestStageFormat  `json:"format"`
	DelayMinutes  int                        `json:"delay_minutes"`
	WindowMinutes int                        `json:"window_minutes"`
	Duration      int                        `json:"duration"`
	LanguageMode  entity.ContestLanguageMode `json:"language_mode"`
	Language      string                     `json:"language"`
	MaxTries      int                        `json:"max_tries"`
	MinWPM        int                        `json:"min_wpm"`
	AdvanceType   entity.ContestAdvanceType  `json:"advance_type"`
	AdvanceValue  int                        `json:"advance_value"`
	MaxPlayers    int                        `json:"max_players"`
}

type CreateContestRequest struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	BannerURL   string                   `json:"banner_url"`
	Accent      string                   `json:"accent"`
	CustomCSS   string                   `json:"custom_css"`
	Visibility  entity.ContestVisibility `json:"visibility"`
	Password    string                   `json:"password"`
	StartsAt    time.Time                `json:"starts_at"`
	// The top-level duration, language and capacity remain useful for cards and
	// legacy clients. When stages are supplied, the first stage becomes the
	// displayed summary format.
	Duration   int                 `json:"duration"`
	Language   string              `json:"language"`
	MaxPlayers int                 `json:"max_players"`
	Stages     []ContestStageInput `json:"stages"`
}

type UpdateContestRequest struct {
	Title       *string                   `json:"title"`
	Description *string                   `json:"description"`
	BannerURL   *string                   `json:"banner_url"`
	Accent      *string                   `json:"accent"`
	CustomCSS   *string                   `json:"custom_css"`
	Visibility  *entity.ContestVisibility `json:"visibility"`
	// Password is only meaningful for private contests. Sending an empty
	// string removes the requirement; omitting it leaves the current lock.
	Password   *string    `json:"password"`
	StartsAt   *time.Time `json:"starts_at"`
	Duration   *int       `json:"duration"`
	Language   *string    `json:"language"`
	MaxPlayers *int       `json:"max_players"`
	// Stages replaces the whole funnel when present. An empty array is
	// rejected; every contest keeps at least one stage.
	Stages []ContestStageInput `json:"stages"`
}

type ContestParticipant struct {
	User     entity.UserView `json:"user"`
	JoinedAt time.Time       `json:"joined_at"`
}

type ContestStageStanding struct {
	Rank       int             `json:"rank"`
	User       entity.UserView `json:"user"`
	WPM        float64         `json:"wpm"`
	Raw        float64         `json:"raw"`
	Accuracy   float64         `json:"accuracy"`
	Attempts   int             `json:"attempts"`
	GameID     uint32          `json:"game_id"`
	Advanced   bool            `json:"advanced"`
	Eliminated bool            `json:"eliminated"`
	Flagged    bool            `json:"flagged"`
	Left       bool            `json:"left"`
}

// ContestStageRun is one submitted attempt, not a player's best score. Public
// event pages use it to show a top-runs board beside the per-player standings.
type ContestStageRun struct {
	Rank     int             `json:"rank"`
	User     entity.UserView `json:"user"`
	WPM      float64         `json:"wpm"`
	Raw      float64         `json:"raw"`
	Accuracy float64         `json:"accuracy"`
	GameID   uint32          `json:"game_id"`
	Flagged  bool            `json:"flagged"`
	Left     bool            `json:"left"`
}

type ContestStageView struct {
	ID                 uint32                     `json:"id"`
	Position           int                        `json:"position"`
	Name               string                     `json:"name"`
	Format             entity.ContestStageFormat  `json:"format"`
	Status             entity.ContestStageStatus  `json:"status"`
	StartsAt           time.Time                  `json:"starts_at"`
	EndsAt             *time.Time                 `json:"ends_at"`
	DelayMinutes       int                        `json:"delay_minutes"`
	WindowMinutes      int                        `json:"window_minutes"`
	Duration           int                        `json:"duration"`
	LanguageMode       entity.ContestLanguageMode `json:"language_mode"`
	Language           string                     `json:"language"`
	MaxTries           int                        `json:"max_tries"`
	MinWPM             int                        `json:"min_wpm"`
	AdvanceType        entity.ContestAdvanceType  `json:"advance_type"`
	AdvanceValue       int                        `json:"advance_value"`
	MaxPlayers         int                        `json:"max_players"`
	EntryCount         int                        `json:"entry_count"`
	QualifiedCount     int                        `json:"qualified_count"`
	AdvanceCount       int                        `json:"advance_count"`
	RunCount           int                        `json:"run_count"`
	ViewerAttempts     int                        `json:"viewer_attempts"`
	ViewerBestWPM      float64                    `json:"viewer_best_wpm"`
	ViewerGameID       uint32                     `json:"viewer_game_id"`
	ViewerCanAttempt   bool                       `json:"viewer_can_attempt"`
	ViewerAttemptsLeft int                        `json:"viewer_attempts_left"`
	Standings          []ContestStageStanding     `json:"standings,omitempty"`
	TopRuns            []ContestStageRun          `json:"top_runs,omitempty"`
}

type ContestView struct {
	ID               uint32                   `json:"id"`
	Code             string                   `json:"code"`
	Title            string                   `json:"title"`
	Description      string                   `json:"description"`
	BannerURL        string                   `json:"banner_url"`
	Accent           string                   `json:"accent"`
	CustomCSS        string                   `json:"custom_css"`
	Visibility       entity.ContestVisibility `json:"visibility"`
	StartsAt         time.Time                `json:"starts_at"`
	Duration         int                      `json:"duration"`
	Language         string                   `json:"language"`
	Status           entity.ContestStatus     `json:"status"`
	MaxPlayers       int                      `json:"max_players"`
	EntryCount       int                      `json:"entry_count"`
	Host             entity.UserView          `json:"host"`
	IsHost           bool                     `json:"is_host"`
	IsEnrolled       bool                     `json:"is_enrolled"`
	RequiresPassword bool                     `json:"requires_password"`
	ShareLink        string                   `json:"share_link"`
	GameID           uint32                   `json:"game_id"`
	StageCount       int                      `json:"stage_count"`
	CurrentStage     int                      `json:"current_stage"`
	Stages           []ContestStageView       `json:"stages"`
	Participants     []ContestParticipant     `json:"participants,omitempty"`
	Leaderboard      []entity.ResultView      `json:"leaderboard,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
}

type ContestAttemptRequest struct {
	Language string `json:"language"`
}

type ContestListView struct {
	Contests   []ContestView `json:"contests"`
	Trending   []ContestView `json:"trending"`
	Mine       []ContestView `json:"mine"`
	ServerTime time.Time     `json:"server_time"`
}

func newContestCode() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("C%07d", time.Now().UnixNano()%10000000)
	}
	for i, b := range buf {
		buf[i] = contestAlphabet[int(b)%len(contestAlphabet)]
	}
	return string(buf[:])
}

func validateContestBanner(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) > 500 {
		return errf(ErrBadRequest, "banner URL is too long")
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errf(ErrBadRequest, "banner must be an http(s) image URL")
	}
	return nil
}

func validateContestCSS(css string) error {
	if len(css) > 20000 {
		return errf(ErrBadRequest, "custom CSS is too long")
	}
	lower := strings.ToLower(css)
	for _, blocked := range []string{"<", ">", "@import", "javascript:", "expression(", "behavior:", "url("} {
		if strings.Contains(lower, blocked) {
			return errf(ErrBadRequest, "custom CSS contains an unsupported value")
		}
	}
	return nil
}

func normalizeVisibility(v entity.ContestVisibility) (entity.ContestVisibility, error) {
	if v == "" {
		return entity.ContestPublic, nil
	}
	if v != entity.ContestPublic && v != entity.ContestPrivate {
		return "", errf(ErrBadRequest, "visibility must be public or private")
	}
	return v, nil
}

func validateContestFields(title, description, banner, accent, css string, startsAt time.Time, duration int, language string, maxPlayers int, requireFuture bool) error {
	if strings.TrimSpace(title) == "" || len(title) > 120 {
		return errf(ErrBadRequest, "title must be 1-120 characters")
	}
	if len(description) > 20000 {
		return errf(ErrBadRequest, "description is too long")
	}
	if err := validateContestBanner(banner); err != nil {
		return err
	}
	if accent != "" && (len(accent) != 7 || accent[0] != '#') {
		return errf(ErrBadRequest, "accent must be a hex colour such as #00a19c")
	}
	if accent != "" {
		for _, r := range accent[1:] {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return errf(ErrBadRequest, "accent must be a hex colour such as #00a19c")
			}
		}
	}
	if err := validateContestCSS(css); err != nil {
		return err
	}
	if _, ok := entity.LobbyTypeFromSeconds(duration); !ok {
		return errf(ErrBadRequest, "duration must be 30, 90 or 300 seconds")
	}
	language = strings.ToLower(strings.TrimSpace(language))
	if !validLanguage(language) || language == "random" || language == "" {
		return errf(ErrBadRequest, "language must be one of the bundled languages")
	}
	if maxPlayers < 2 || maxPlayers > 512 {
		return errf(ErrBadRequest, "contest capacity must be between 2 and 512")
	}
	if requireFuture && (startsAt.IsZero() || startsAt.Before(time.Now().UTC().Add(30*time.Second))) {
		return errf(ErrBadRequest, "start time must be at least 30 seconds in the future")
	}
	return nil
}

func stageFormatName(format entity.ContestStageFormat) string {
	switch format {
	case entity.ContestStageQualifier:
		return "Qualifier"
	case entity.ContestStageHotLaps:
		return "HotLaps"
	case entity.ContestStagePractice:
		return "Practice"
	case entity.ContestStageRanked:
		return "Ranked final"
	default:
		return "Live race"
	}
}

func (s *ServiceLayer) normalizeContestStages(inputs []ContestStageInput, startsAt time.Time, duration int, language string, maxPlayers int) ([]entity.ContestStage, error) {
	if len(inputs) == 0 {
		inputs = []ContestStageInput{{
			Format:      entity.ContestStageMultiplayer,
			Duration:    duration,
			Language:    language,
			AdvanceType: entity.ContestAdvanceAll,
			MaxPlayers:  maxPlayers,
		}}
	}
	if len(inputs) > 10 {
		return nil, errf(ErrBadRequest, "a contest can have at most 10 stages")
	}
	now := time.Now().UTC()
	stages := make([]entity.ContestStage, 0, len(inputs))
	for i, in := range inputs {
		position := i + 1
		in.Format = entity.ContestStageFormat(strings.ToLower(strings.TrimSpace(string(in.Format))))
		switch in.Format {
		case entity.ContestStageQualifier, entity.ContestStageHotLaps, entity.ContestStagePractice, entity.ContestStageMultiplayer, entity.ContestStageRanked:
		default:
			return nil, errf(ErrBadRequest, "stage %d has an unsupported game type", position)
		}
		in.Name = strings.TrimSpace(in.Name)
		if in.Name == "" {
			in.Name = fmt.Sprintf("%s · stage %d", stageFormatName(in.Format), position)
		}
		if len(in.Name) > 80 {
			return nil, errf(ErrBadRequest, "stage names must be 80 characters or fewer")
		}
		if in.Duration == 0 {
			in.Duration = duration
		}
		if _, ok := entity.LobbyTypeFromSeconds(in.Duration); !ok {
			return nil, errf(ErrBadRequest, "stage %d duration must be 30, 90 or 300 seconds", position)
		}
		in.Language = strings.ToLower(strings.TrimSpace(in.Language))
		if in.Language == "" {
			in.Language = language
		}
		if !validLanguage(in.Language) || in.Language == "random" {
			return nil, errf(ErrBadRequest, "stage %d language must be one of the bundled languages", position)
		}
		in.LanguageMode = entity.ContestLanguageMode(strings.ToLower(strings.TrimSpace(string(in.LanguageMode))))
		if in.LanguageMode == "" {
			in.LanguageMode = entity.ContestLanguageFixed
		}
		if in.LanguageMode != entity.ContestLanguageFixed && in.LanguageMode != entity.ContestLanguageChoice {
			return nil, errf(ErrBadRequest, "stage %d language mode must be fixed or choice", position)
		}
		if in.LanguageMode == entity.ContestLanguageChoice && in.Format != entity.ContestStageQualifier {
			return nil, errf(ErrBadRequest, "only a qualifier can let each player choose a language")
		}
		if in.Format == entity.ContestStageHotLaps || in.Format == entity.ContestStagePractice {
			in.LanguageMode = entity.ContestLanguageFixed
		}
		in.DelayMinutes = max(0, in.DelayMinutes)
		if i == 0 {
			in.DelayMinutes = 0
		}
		if in.DelayMinutes > 10080 {
			return nil, errf(ErrBadRequest, "stage delays must be seven days or less")
		}
		in.WindowMinutes = max(0, in.WindowMinutes)
		if in.Format.IsOpen() {
			if in.WindowMinutes == 0 {
				in.WindowMinutes = 60
			}
			if in.WindowMinutes > 10080 {
				return nil, errf(ErrBadRequest, "stage windows must be seven days or less")
			}
		} else {
			in.WindowMinutes = 0
		}
		in.MaxTries = max(0, in.MaxTries)
		if in.MaxTries > 1000 {
			return nil, errf(ErrBadRequest, "stage try limits must be 1000 or lower")
		}
		// HotLaps and practice stages may be unlimited (0) or host-capped.
		// Leaving the value untouched gives hosts that choice.
		in.MinWPM = max(0, in.MinWPM)
		if in.Format == entity.ContestStageQualifier {
			if in.MinWPM == 0 {
				in.MinWPM = 80
			}
			if in.MinWPM > 300 {
				return nil, errf(ErrBadRequest, "qualifier WPM must be 300 or lower")
			}
		} else {
			in.MinWPM = 0
		}
		in.AdvanceType = entity.ContestAdvanceType(strings.ToLower(strings.TrimSpace(string(in.AdvanceType))))
		if in.Format == entity.ContestStagePractice {
			// A warm-up never cuts players, regardless of what the client sent.
			in.AdvanceType = entity.ContestAdvanceAll
			in.AdvanceValue = 0
		}
		if in.AdvanceType == "" {
			in.AdvanceType = entity.ContestAdvanceAll
		}
		switch in.AdvanceType {
		case entity.ContestAdvanceCount:
			if in.AdvanceValue < 1 || in.AdvanceValue > 512 {
				return nil, errf(ErrBadRequest, "stage %d must advance between 1 and 512 players", position)
			}
		case entity.ContestAdvancePercent:
			if in.AdvanceValue < 1 || in.AdvanceValue > 100 {
				return nil, errf(ErrBadRequest, "stage %d advancement percentage must be between 1 and 100", position)
			}
		case entity.ContestAdvanceAll:
			in.AdvanceValue = 0
		default:
			return nil, errf(ErrBadRequest, "stage %d advancement must be count, percent or all", position)
		}
		in.MaxPlayers = max(2, in.MaxPlayers)
		if in.Format == entity.ContestStageMultiplayer || in.Format == entity.ContestStageRanked {
			if in.MaxPlayers > 10 {
				return nil, errf(ErrBadRequest, "live heats are limited to 10 players")
			}
		} else {
			in.MaxPlayers = 1
		}
		stage := entity.ContestStage{
			ID:            s.intGen.GenerateID(),
			Position:      position,
			Name:          in.Name,
			Format:        in.Format,
			Status:        entity.ContestStagePending,
			StartsAt:      time.Time{},
			DelayMinutes:  in.DelayMinutes,
			WindowMinutes: in.WindowMinutes,
			Duration:      in.Duration,
			LanguageMode:  in.LanguageMode,
			Language:      in.Language,
			MaxTries:      in.MaxTries,
			MinWPM:        in.MinWPM,
			AdvanceType:   in.AdvanceType,
			AdvanceValue:  in.AdvanceValue,
			MaxPlayers:    in.MaxPlayers,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if i == 0 {
			stage.StartsAt = startsAt
		}
		stages = append(stages, stage)
	}
	return stages, nil
}

func (s *ServiceLayer) newContestCode(ctx context.Context) string {
	for i := 0; i < 10; i++ {
		code := newContestCode()
		if _, err := s.db.GetContestByCode(ctx, code); errors.Is(err, gorm.ErrRecordNotFound) {
			return code
		}
	}
	return fmt.Sprintf("C%07d", time.Now().UnixNano()%10000000)
}

func (s *ServiceLayer) CreateContest(ctx context.Context, uid uint32, req CreateContestRequest) (ContestView, error) {
	if _, err := s.GetUser(ctx, uid); err != nil {
		return ContestView{}, err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.BannerURL = strings.TrimSpace(req.BannerURL)
	req.Accent = strings.TrimSpace(req.Accent)
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	req.StartsAt = req.StartsAt.UTC()
	if req.Accent == "" {
		req.Accent = "#00a19c"
	}
	if req.Duration == 0 {
		req.Duration = 90
	}
	if req.Language == "" {
		req.Language = "go"
	}
	if req.MaxPlayers == 0 {
		req.MaxPlayers = 8
	}
	visibility, err := normalizeVisibility(req.Visibility)
	if err != nil {
		return ContestView{}, err
	}
	stages, err := s.normalizeContestStages(req.Stages, req.StartsAt, req.Duration, req.Language, min(10, req.MaxPlayers))
	if err != nil {
		return ContestView{}, err
	}
	// The first stage is the event's public summary format. Capacity is the
	// total number of enrolled players, not the live heat size.
	req.Duration = stages[0].Duration
	req.Language = stages[0].Language
	if err := validateContestFields(req.Title, req.Description, req.BannerURL, req.Accent, req.CustomCSS, req.StartsAt, req.Duration, req.Language, req.MaxPlayers, true); err != nil {
		return ContestView{}, err
	}
	if len(req.Password) > 72 {
		return ContestView{}, errf(ErrBadRequest, "password is too long")
	}
	if visibility == entity.ContestPublic && req.Password != "" {
		return ContestView{}, errf(ErrBadRequest, "public contests cannot be password protected")
	}
	hash := ""
	if visibility == entity.ContestPrivate && req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return ContestView{}, err
		}
		hash = string(h)
	}

	now := time.Now().UTC()
	contest := entity.Contest{
		ID:           s.intGen.GenerateID(),
		Code:         s.newContestCode(ctx),
		HostID:       uid,
		Title:        req.Title,
		Description:  req.Description,
		BannerURL:    req.BannerURL,
		Accent:       req.Accent,
		CustomCSS:    req.CustomCSS,
		Visibility:   visibility,
		PasswordHash: hash,
		StartsAt:     req.StartsAt,
		Duration:     req.Duration,
		Language:     req.Language,
		MaxPlayers:   req.MaxPlayers,
		Status:       entity.ContestUpcoming,
		CreatedAt:    now,
	}
	if err := s.db.CreateContest(ctx, &contest); err != nil {
		return ContestView{}, err
	}
	for i := range stages {
		stages[i].ContestID = contest.ID
		stages[i].StartsAt = time.Time{}
		if i == 0 {
			stages[i].StartsAt = contest.StartsAt
		}
	}
	if err := s.db.CreateContestStages(ctx, stagePointers(stages)); err != nil {
		contest.Status = entity.ContestCancelled
		_ = s.db.SaveContest(ctx, &contest)
		return ContestView{}, err
	}
	contest.Stages = stages
	if err := s.db.AddContestEntry(ctx, &entity.ContestEntry{
		ContestID:    contest.ID,
		UserID:       uid,
		JoinedAt:     now,
		CurrentStage: 1,
	}); err != nil {
		s.logger.Warnw("contest host enrollment failed", "contest", contest.ID, "error", err)
	}
	if err := s.db.CreateTask(ctx, &entity.Task{
		Category: entity.CONTEST_STAGE,
		JobID:    stages[0].ID,
		Time:     stages[0].StartsAt,
	}); err != nil {
		contest.Status = entity.ContestCancelled
		_ = s.db.SaveContest(ctx, &contest)
		return ContestView{}, err
	}
	return s.contestView(ctx, contest, uid, true)
}

func stagePointers(stages []entity.ContestStage) []*entity.ContestStage {
	out := make([]*entity.ContestStage, 0, len(stages))
	for i := range stages {
		out = append(out, &stages[i])
	}
	return out
}

func (s *ServiceLayer) ListContests(ctx context.Context, viewerID uint32) (ContestListView, error) {
	publicContests, err := s.db.ListPublicContests(ctx, 60)
	if err != nil {
		return ContestListView{}, err
	}
	out := ContestListView{
		Contests:   make([]ContestView, 0, len(publicContests)),
		Trending:   []ContestView{},
		Mine:       []ContestView{},
		ServerTime: time.Now().UTC(),
	}
	for _, c := range publicContests {
		view, err := s.contestView(ctx, c, viewerID, false)
		if err != nil {
			return ContestListView{}, err
		}
		out.Contests = append(out.Contests, view)
	}
	out.Trending = append(out.Trending, out.Contests...)
	sort.SliceStable(out.Trending, func(i, j int) bool {
		if out.Trending[i].EntryCount != out.Trending[j].EntryCount {
			return out.Trending[i].EntryCount > out.Trending[j].EntryCount
		}
		return out.Trending[i].StartsAt.Before(out.Trending[j].StartsAt)
	})
	if len(out.Trending) > 3 {
		out.Trending = out.Trending[:3]
	}
	if viewerID != 0 {
		mine, err := s.db.ListContestsForUser(ctx, viewerID, 50)
		if err != nil {
			return ContestListView{}, err
		}
		for _, c := range mine {
			view, err := s.contestView(ctx, c, viewerID, false)
			if err != nil {
				return ContestListView{}, err
			}
			out.Mine = append(out.Mine, view)
		}
	}
	return out, nil
}

func (s *ServiceLayer) GetContestByCode(ctx context.Context, viewerID uint32, code string) (ContestView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	return s.contestView(ctx, c, viewerID, true)
}

func (s *ServiceLayer) contestView(ctx context.Context, c entity.Contest, viewerID uint32, detail bool) (ContestView, error) {
	hostUser := c.Host
	if hostUser == nil {
		if u, err := s.db.GetUser(c.HostID); err == nil {
			hostUser = &u
		}
	}
	host := entity.UserView{}
	if hostUser != nil {
		host = entity.NewUserView(*hostUser)
	}
	stages, err := s.db.ListContestStages(ctx, c.ID)
	if err != nil {
		return ContestView{}, err
	}
	if len(stages) == 0 {
		// Legacy contests created before funnels keep rendering as a single
		// standard race. Their stage view is synthesized, not written back.
		stages = []entity.ContestStage{legacyContestStage(c)}
	}
	out := ContestView{
		ID:               c.ID,
		Code:             c.Code,
		Title:            c.Title,
		Description:      c.Description,
		BannerURL:        c.BannerURL,
		Accent:           c.Accent,
		CustomCSS:        c.CustomCSS,
		Visibility:       c.Visibility,
		StartsAt:         c.StartsAt,
		Duration:         c.Duration,
		Language:         c.Language,
		Status:           c.Status,
		MaxPlayers:       c.MaxPlayers,
		Host:             host,
		IsHost:           viewerID != 0 && c.HostID == viewerID,
		RequiresPassword: c.Visibility == entity.ContestPrivate && c.PasswordHash != "",
		ShareLink:        strings.TrimRight(s.frontendURL, "/") + "/contests/" + c.Code,
		GameID:           c.GameID,
		StageCount:       len(stages),
		CurrentStage:     currentContestStagePosition(c.Status, stages),
		CreatedAt:        c.CreatedAt,
	}
	if viewerID != 0 {
		enrolled, err := s.db.IsContestEntry(ctx, c.ID, viewerID)
		if err != nil {
			return ContestView{}, err
		}
		out.IsEnrolled = enrolled
	}
	visibleParticipants := c.Visibility == entity.ContestPublic || out.IsHost || out.IsEnrolled
	var entries []entity.ContestEntry
	if detail && visibleParticipants {
		entries, err = s.db.ListContestEntries(ctx, c.ID)
		if err != nil {
			return ContestView{}, err
		}
		out.EntryCount = len(entries)
		out.Participants = make([]ContestParticipant, 0, len(entries))
		for _, entry := range entries {
			if entry.User == nil {
				continue
			}
			out.Participants = append(out.Participants, ContestParticipant{
				User:     entity.NewUserView(*entry.User),
				JoinedAt: entry.JoinedAt,
			})
		}
	} else {
		count, err := s.db.CountContestEntries(ctx, c.ID)
		if err != nil {
			return ContestView{}, err
		}
		out.EntryCount = int(count)
	}
	stageViews, err := s.contestStageViews(ctx, c, stages, entries, viewerID, detail && visibleParticipants)
	if err != nil {
		return ContestView{}, err
	}
	out.Stages = stageViews
	if detail && visibleParticipants && len(c.Leaderboard) > 0 {
		_ = json.Unmarshal(c.Leaderboard, &out.Leaderboard)
	}
	return out, nil
}

func legacyContestStage(c entity.Contest) entity.ContestStage {
	return entity.ContestStage{
		ID:            0,
		ContestID:     c.ID,
		Position:      1,
		Name:          "Live race",
		Format:        entity.ContestStageMultiplayer,
		Status:        entity.ContestStagePending,
		StartsAt:      c.StartsAt,
		WindowMinutes: 0,
		Duration:      c.Duration,
		LanguageMode:  entity.ContestLanguageFixed,
		Language:      c.Language,
		AdvanceType:   entity.ContestAdvanceAll,
		MaxPlayers:    c.MaxPlayers,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func currentContestStagePosition(status entity.ContestStatus, stages []entity.ContestStage) int {
	for _, stage := range stages {
		if stage.Status == entity.ContestStageActive {
			return stage.Position
		}
	}
	if status == entity.ContestEnded {
		if len(stages) > 0 {
			return stages[len(stages)-1].Position
		}
	}
	for _, stage := range stages {
		if stage.Status == entity.ContestStagePending {
			return stage.Position
		}
	}
	if len(stages) > 0 {
		return stages[len(stages)-1].Position
	}
	return 0
}

func (s *ServiceLayer) UpdateContest(ctx context.Context, uid uint32, code string, req UpdateContestRequest) (ContestView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	if c.HostID != uid {
		return ContestView{}, errf(ErrForbidden, "only the host can edit this contest")
	}
	if c.Status != entity.ContestUpcoming {
		return ContestView{}, errf(ErrConflict, "this contest can no longer be edited")
	}

	if req.Title != nil {
		c.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		c.Description = strings.TrimSpace(*req.Description)
	}
	if req.BannerURL != nil {
		c.BannerURL = strings.TrimSpace(*req.BannerURL)
	}
	if req.Accent != nil {
		c.Accent = strings.TrimSpace(*req.Accent)
	}
	if req.CustomCSS != nil {
		c.CustomCSS = *req.CustomCSS
	}
	if req.Duration != nil {
		c.Duration = *req.Duration
	}
	if req.Language != nil {
		c.Language = strings.ToLower(strings.TrimSpace(*req.Language))
	}
	if req.StartsAt != nil {
		c.StartsAt = req.StartsAt.UTC()
	}
	if req.Visibility != nil {
		visibility, err := normalizeVisibility(*req.Visibility)
		if err != nil {
			return ContestView{}, err
		}
		c.Visibility = visibility
	}
	if req.MaxPlayers != nil {
		count, err := s.db.CountContestEntries(ctx, c.ID)
		if err != nil {
			return ContestView{}, err
		}
		if *req.MaxPlayers < int(count) {
			return ContestView{}, errf(ErrConflict, "capacity cannot be lower than the number of enrolled players")
		}
		c.MaxPlayers = *req.MaxPlayers
	}
	if c.Accent == "" {
		c.Accent = "#00a19c"
	}
	if req.Password != nil {
		if len(*req.Password) > 72 {
			return ContestView{}, errf(ErrBadRequest, "password is too long")
		}
		if c.Visibility == entity.ContestPublic && *req.Password != "" {
			return ContestView{}, errf(ErrBadRequest, "public contests cannot be password protected")
		}
		if c.Visibility == entity.ContestPublic || *req.Password == "" {
			c.PasswordHash = ""
		} else {
			h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				return ContestView{}, err
			}
			c.PasswordHash = string(h)
		}
	} else if c.Visibility == entity.ContestPublic {
		c.PasswordHash = ""
	}
	if req.Stages != nil && len(req.Stages) == 0 {
		return ContestView{}, errf(ErrBadRequest, "a contest must keep at least one stage")
	}

	existingStages, err := s.db.ListContestStages(ctx, c.ID)
	if err != nil {
		return ContestView{}, err
	}
	var replacement []entity.ContestStage
	if req.Stages != nil {
		replacement, err = s.normalizeContestStages(req.Stages, c.StartsAt, c.Duration, c.Language, min(10, c.MaxPlayers))
		if err != nil {
			return ContestView{}, err
		}
		c.Duration = replacement[0].Duration
		c.Language = replacement[0].Language
	} else if len(existingStages) == 0 {
		replacement, err = s.normalizeContestStages(nil, c.StartsAt, c.Duration, c.Language, min(10, c.MaxPlayers))
		if err != nil {
			return ContestView{}, err
		}
		c.Duration = replacement[0].Duration
		c.Language = replacement[0].Language
	} else {
		// Summary-only edits keep the funnel. If the host changes the first
		// stage's clock or language, mirror it into that stage definition.
		if req.Duration != nil || req.Language != nil {
			existingStages[0].Duration = c.Duration
			existingStages[0].Language = c.Language
			if existingStages[0].LanguageMode == "" {
				existingStages[0].LanguageMode = entity.ContestLanguageFixed
			}
		}
		if req.StartsAt != nil {
			existingStages[0].StartsAt = c.StartsAt
		}
	}
	if err := validateContestFields(c.Title, c.Description, c.BannerURL, c.Accent, c.CustomCSS, c.StartsAt, c.Duration, c.Language, c.MaxPlayers, true); err != nil {
		return ContestView{}, err
	}
	c.UpdatedAt = time.Now().UTC()
	if err := s.db.SaveContest(ctx, &c); err != nil {
		return ContestView{}, err
	}
	if err := s.db.CompleteContestTasks(ctx, c.ID); err != nil {
		return ContestView{}, err
	}
	firstStage := entity.ContestStage{}
	if len(replacement) > 0 {
		for i := range replacement {
			replacement[i].ContestID = c.ID
			replacement[i].StartsAt = time.Time{}
			if i == 0 {
				replacement[i].StartsAt = c.StartsAt
			}
		}
		if err := s.db.DeleteContestStages(ctx, c.ID); err != nil {
			return ContestView{}, err
		}
		if err := s.db.CreateContestStages(ctx, stagePointers(replacement)); err != nil {
			return ContestView{}, err
		}
		firstStage = replacement[0]
	} else {
		if len(existingStages) == 0 {
			return ContestView{}, errf(ErrConflict, "contest has no stages")
		}
		firstStage = existingStages[0]
		firstStage.StartsAt = c.StartsAt
		firstStage.UpdatedAt = c.UpdatedAt
		if err := s.db.SaveContestStage(ctx, &firstStage); err != nil {
			return ContestView{}, err
		}
	}
	if err := s.db.CreateTask(ctx, &entity.Task{
		Category: entity.CONTEST_STAGE,
		JobID:    firstStage.ID,
		Time:     firstStage.StartsAt,
	}); err != nil {
		return ContestView{}, err
	}
	return s.contestView(ctx, c, uid, true)
}

func (s *ServiceLayer) CancelContest(ctx context.Context, uid uint32, code string) (ContestView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	if c.HostID != uid {
		return ContestView{}, errf(ErrForbidden, "only the host can cancel this contest")
	}
	if c.Status == entity.ContestCancelled {
		return s.contestView(ctx, c, uid, true)
	}
	if c.Status != entity.ContestUpcoming {
		return ContestView{}, errf(ErrConflict, "only upcoming contests can be cancelled")
	}
	c.Status = entity.ContestCancelled
	c.UpdatedAt = time.Now().UTC()
	if err := s.db.SaveContest(ctx, &c); err != nil {
		return ContestView{}, err
	}
	s.cancelContestStages(ctx, c.ID)
	_ = s.db.CompleteContestTasks(ctx, c.ID)
	s.notifyContest(ctx, c, "contest_cancelled", map[string]any{
		"contest_code": c.Code,
		"title":        c.Title,
	})
	return s.contestView(ctx, c, uid, true)
}

func (s *ServiceLayer) EnrollContest(ctx context.Context, uid uint32, code, password string) (ContestView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	already, err := s.db.IsContestEntry(ctx, c.ID, uid)
	if err != nil {
		return ContestView{}, err
	}
	if already {
		return s.contestView(ctx, c, uid, true)
	}
	if c.Status != entity.ContestUpcoming {
		return ContestView{}, errf(ErrConflict, "enrollment is closed")
	}
	if !time.Now().UTC().Before(c.StartsAt) {
		return ContestView{}, errf(ErrConflict, "enrollment is closed")
	}
	if c.Visibility == entity.ContestPrivate && c.PasswordHash != "" {
		if password == "" {
			return ContestView{}, errf(ErrPasswordRequired, "this contest requires a password")
		}
		if bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password)) != nil {
			return ContestView{}, errf(ErrWrongPassword, "incorrect contest password")
		}
	}
	count, err := s.db.CountContestEntries(ctx, c.ID)
	if err != nil {
		return ContestView{}, err
	}
	if int(count) >= c.MaxPlayers {
		return ContestView{}, errf(ErrFull, "contest is full")
	}
	if err := s.db.AddContestEntry(ctx, &entity.ContestEntry{
		ContestID:    c.ID,
		UserID:       uid,
		JoinedAt:     time.Now().UTC(),
		CurrentStage: 1,
	}); err != nil {
		return ContestView{}, err
	}
	s.hub().SendToUser(c.HostID, hub.Ev("contest_enrollment", map[string]any{
		"contest_code": c.Code,
		"entry_count":  count + 1,
	}))
	return s.contestView(ctx, c, uid, true)
}

func (s *ServiceLayer) LeaveContest(ctx context.Context, uid uint32, code string) (ContestView, error) {
	c, err := s.db.GetContestByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ContestView{}, errf(ErrNotFound, "contest not found")
		}
		return ContestView{}, err
	}
	if c.HostID == uid {
		return ContestView{}, errf(ErrForbidden, "the host cannot leave their own contest")
	}
	if c.Status != entity.ContestUpcoming {
		return ContestView{}, errf(ErrConflict, "enrollment is closed")
	}
	if err := s.db.RemoveContestEntry(ctx, c.ID, uid); err != nil {
		return ContestView{}, err
	}
	return s.contestView(ctx, c, uid, true)
}

func (s *ServiceLayer) notifyContest(ctx context.Context, c entity.Contest, event string, payload map[string]any) {
	entries, err := s.db.ListContestEntries(ctx, c.ID)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.User != nil && entry.User.IsBot {
			continue
		}
		s.hub().SendToUser(entry.UserID, hub.Ev(event, payload))
	}
}

// HandleContestTask is driven by the scheduler. A contest moves directly from
// upcoming to started; the hidden staging room exists only long enough to
// reuse the game launcher.
func (s *ServiceLayer) HandleContestTask(ctx context.Context, task entity.Task) error {
	if task.Category == entity.CONTEST_STAGE {
		return s.HandleContestStageTask(ctx, task)
	}
	if task.Category != entity.CONTEST {
		return nil
	}
	contest, err := s.db.GetContest(ctx, task.JobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if contest.Status != entity.ContestUpcoming {
		return nil
	}
	now := time.Now().UTC()
	if contest.StartsAt.After(now.Add(2 * time.Second)) {
		return s.db.CreateTask(ctx, &entity.Task{
			Category: entity.CONTEST,
			JobID:    contest.ID,
			Time:     contest.StartsAt,
		})
	}

	entries, err := s.db.ListContestEntries(ctx, contest.ID)
	if err != nil {
		return err
	}
	humans := make([]entity.ContestEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.User != nil && !entry.User.IsBot {
			humans = append(humans, entry)
		}
	}
	if len(humans) < 2 {
		return s.cancelContestForStart(ctx, contest, "at least two enrolled players are needed to start")
	}

	room := entity.Room{
		ID:          s.intGen.GenerateID(),
		Code:        newRoomCode(),
		Name:        truncateRunes(contest.Title, 64),
		Description: "contest staging room",
		IsPublic:    false,
		MaxPlayers:  contest.MaxPlayers,
		HostID:      contest.HostID,
		Status:      entity.RoomLobby,
		Duration:    contest.Duration,
		Language:    contest.Language,
		CreatedAt:   now,
	}
	if err := s.db.CreateRoom(ctx, &room); err != nil {
		return err
	}
	hostEnrolled := false
	for _, entry := range humans {
		role := entity.RoleMember
		if entry.UserID == contest.HostID {
			role = entity.RoleOwner
			hostEnrolled = true
		}
		if err := s.db.AddMember(ctx, &entity.RoomMember{
			RoomID:   room.ID,
			UserID:   entry.UserID,
			Role:     role,
			JoinedAt: now,
		}); err != nil {
			s.cleanupContestRoom(ctx, room.ID)
			return err
		}
	}
	if !hostEnrolled {
		_ = s.db.AddMember(ctx, &entity.RoomMember{
			RoomID: room.ID, UserID: contest.HostID, Role: entity.RoleOwner, JoinedAt: now,
		})
	}
	g, err := s.createRoomGame(ctx, room, entity.GameContest, contest.ID)
	if err != nil {
		s.cleanupContestRoom(ctx, room.ID)
		return s.cancelContestForStart(ctx, contest, "the contest could not start")
	}
	contest.Status = entity.ContestStarted
	contest.RoomID = room.ID
	contest.GameID = g.ID
	contest.UpdatedAt = time.Now().UTC()
	if err := s.db.SaveContest(ctx, &contest); err != nil {
		return err
	}
	s.notifyContest(ctx, contest, "contest_started", map[string]any{
		"contest_code": contest.Code,
		"title":        contest.Title,
		"game_id":      g.ID,
	})
	return nil
}

func (s *ServiceLayer) cancelContestForStart(ctx context.Context, contest entity.Contest, _ string) error {
	contest.Status = entity.ContestCancelled
	contest.UpdatedAt = time.Now().UTC()
	if err := s.db.SaveContest(ctx, &contest); err != nil {
		return err
	}
	s.cancelContestStages(ctx, contest.ID)
	s.notifyContest(ctx, contest, "contest_cancelled", map[string]any{
		"contest_code": contest.Code,
		"title":        contest.Title,
	})
	return nil
}

func (s *ServiceLayer) cleanupContestRoom(ctx context.Context, roomID uint32) {
	members, _ := s.db.ListMembers(ctx, roomID)
	_ = s.db.CloseRoom(ctx, roomID)
	for _, member := range members {
		_ = s.db.RemoveMember(ctx, roomID, member.UserID)
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
