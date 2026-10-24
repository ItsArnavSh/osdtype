package services

import (
	"context"
	"errors"
	"osdtyp/app/core/game"
	"osdtyp/app/core/matchmaker"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"time"

	"gorm.io/gorm"
)

func (s *ServiceLayer) JoinQueue(ctx context.Context, uid uint32, duration int) (matchmaker.QueueStatus, error) {
	if _, ok := entity.LobbyTypeFromSeconds(duration); !ok {
		return matchmaker.QueueStatus{}, errf(ErrBadRequest, "duration must be 30, 90 or 300 seconds")
	}
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return matchmaker.QueueStatus{}, err
	}
	if _, inGame := s.g().IsInGame(uid); inGame {
		return matchmaker.QueueStatus{}, errf(ErrConflict, "finish or leave your current game first")
	}
	if _, inRoom, _ := s.db.ActiveRoomForUser(ctx, uid); inRoom {
		return matchmaker.QueueStatus{}, errf(ErrConflict, "leave your current room before playing ranked")
	}
	if st, ok, _ := s.rdb().GetState(ctx, uid); ok && st.Status == "queue" && st.QueueDuration != duration {
		_ = s.mm().Remove(ctx, uid, st.QueueDuration)
	}
	rating := user.RatingFor(duration)
	if err := s.mm().Add(ctx, uid, rating, duration); err != nil {
		return matchmaker.QueueStatus{}, err
	}
	status, ok, err := s.mm().Status(ctx, uid, rating, duration)
	if err != nil {
		return matchmaker.QueueStatus{}, err
	}
	if !ok {
		return matchmaker.QueueStatus{}, errf(ErrConflict, "could not join the queue")
	}
	return status, nil
}

func (s *ServiceLayer) LeaveQueue(ctx context.Context, uid uint32, duration int) error {
	if _, ok := entity.LobbyTypeFromSeconds(duration); !ok {
		return errf(ErrBadRequest, "invalid duration")
	}
	return s.mm().Remove(ctx, uid, duration)
}

func (s *ServiceLayer) QueueStatus(ctx context.Context, uid uint32, duration int) (matchmaker.QueueStatus, bool, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return matchmaker.QueueStatus{}, false, err
	}
	return s.mm().Status(ctx, uid, user.RatingFor(duration), duration)
}

// StartRankedMatch is called by the matchmaker after it has claimed a group
// from the Redis queue.
func (s *ServiceLayer) StartRankedMatch(ctx context.Context, users []entity.User, durationSec int) error {
	if len(users) < 2 {
		return errf(ErrConflict, "not enough players")
	}
	seed := randomSeed()
	language := entity.LanguageForSeed(seed)
	genCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	snippet, _ := s.codegen.GenerateSnippet(genCtx, language, seed)
	cancel()
	startOffset := utils.SnippetStartOffset(seed, snippet)

	now := time.Now().UTC()
	starts := now.Add(s.countdown)
	ends := starts.Add(time.Duration(durationSec) * time.Second)
	g := entity.Game{
		ID:          s.intGen.GenerateID(),
		Mode:        entity.GameRanked,
		Duration:    durationSec,
		Language:    language,
		Seed:        seed,
		Snippet:     snippet,
		StartOffset: startOffset,
		Status:      entity.GameCountdown,
		StartsAt:    &starts,
		EndsAt:      &ends,
		CreatedAt:   now,
	}
	players := make([]entity.GamePlayer, 0, len(users))
	userMap := make(map[uint32]entity.User, len(users))
	botConfigs := map[uint32]entity.BotConfig{}

	// A bot plays at the pace of the humans it was matched with. The bot's
	// persisted rating still drives matchmaking and Elo, but using the match
	// rating for speed avoids a 600-rated bot feeling sluggish to a much
	// stronger player, or a low-Elo bot feeling superhuman.
	matchRating := 0
	humans := 0
	for _, u := range users {
		if u.IsBot {
			continue
		}
		matchRating += u.RatingFor(durationSec)
		humans++
	}
	if humans > 0 {
		matchRating /= humans
	}

	for _, u := range users {
		userMap[u.ID] = u
		players = append(players, entity.GamePlayer{UserID: u.ID, RatingBefore: u.RatingFor(durationSec), IsBot: u.IsBot})
		if u.IsBot {
			if _, cfg, ok := s.bots().PersonaForUsername(u.Username); ok {
				if matchRating > 0 {
					cfg.Rating = matchRating
				}
				botConfigs[u.ID] = cfg
			}
		}
	}
	if err := s.db.CreateGame(ctx, &g, players); err != nil {
		return err
	}
	for _, u := range users {
		if !u.IsBot {
			s.state().SetGame(ctx, u.ID, g.ID, "")
		}
	}
	opts := game.StartOptions{Game: g, Players: players, Users: userMap, Bots: botConfigs}
	return s.g().Start(ctx, opts)
}

func (s *ServiceLayer) PickRankedBot(ctx context.Context, rating, duration int) (entity.User, bool) {
	if err := s.Bootstrap(ctx); err != nil {
		s.logger.Warnw("bot bootstrap failed", "error", err)
	}
	bot, err := s.db.GetBotForRating(ctx, rating, duration)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, false
		}
		s.logger.Warnw("pick ranked bot", "error", err)
		return entity.User{}, false
	}
	return bot, true
}
