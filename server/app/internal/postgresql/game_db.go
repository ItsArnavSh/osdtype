package postgresql

import (
	"context"
	"errors"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
)

func (d *Database) CreateGame(ctx context.Context, game *entity.Game, players []entity.GamePlayer) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(game).Error; err != nil {
			return err
		}
		if len(players) == 0 {
			return nil
		}
		// Callers construct players before the game ID exists. Always bind
		// them to the persisted game, otherwise the composite key collapses
		// onto game_id=0 and Postgres rejects the second player.
		for i := range players {
			players[i].GameID = game.ID
		}
		return tx.Create(&players).Error
	})
}

func (d *Database) GetGame(ctx context.Context, id uint32) (entity.Game, error) {
	var game entity.Game
	if err := d.db.WithContext(ctx).First(&game, "id = ?", id).Error; err != nil {
		return entity.Game{}, err
	}
	return game, nil
}

func (d *Database) GetGamePlayers(ctx context.Context, gameID uint32) ([]entity.GamePlayer, error) {
	var players []entity.GamePlayer
	err := d.db.WithContext(ctx).Preload("User").Where("game_id = ?", gameID).Order("user_id ASC").Find(&players).Error
	return players, err
}

func (d *Database) ActiveGameForUser(ctx context.Context, uid uint32) (entity.Game, error) {
	var game entity.Game
	err := d.db.WithContext(ctx).
		Table("games").
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("game_players.user_id = ? AND games.status IN ?", uid, []entity.GameStatus{entity.GameCountdown, entity.GameActive}).
		Order("games.created_at DESC").First(&game).Error
	return game, err
}

func (d *Database) LatestFinishedGameForUser(ctx context.Context, uid uint32) (entity.Game, error) {
	var game entity.Game
	err := d.db.WithContext(ctx).
		Table("games").
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("game_players.user_id = ? AND games.status = ? AND games.mode <> ?", uid, entity.GameFinished, entity.GamePractice).
		Order("games.finished_at DESC").First(&game).Error
	return game, err
}

// LatestUnacknowledgedGame finds the newest finished game whose results the
// player has not explicitly dismissed. It is used to rebuild the results state
// from Postgres after Redis is flushed.
func (d *Database) LatestUnacknowledgedGame(ctx context.Context, uid uint32) (entity.Game, error) {
	var game entity.Game
	err := d.db.WithContext(ctx).
		Table("games").
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("game_players.user_id = ? AND games.status = ? AND game_players.acked = ? AND game_players.left = ? AND games.mode <> ?", uid, entity.GameFinished, false, false, entity.GamePractice).
		Order("games.finished_at DESC").First(&game).Error
	return game, err
}

func (d *Database) AckResults(ctx context.Context, gameID, uid uint32) error {
	return d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Where("game_id = ? AND user_id = ?", gameID, uid).
		Update("acked", true).Error
}

func (d *Database) AckAllResults(ctx context.Context, uid uint32) error {
	return d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Where("user_id = ? AND acked = ?", uid, false).
		Update("acked", true).Error
}

func (d *Database) UpdateGameFields(ctx context.Context, id uint32, fields map[string]any) error {
	return d.db.WithContext(ctx).Model(&entity.Game{}).Where("id = ?", id).Updates(fields).Error
}

func (d *Database) StartGame(ctx context.Context, id uint32, startsAt, endsAt time.Time) error {
	return d.UpdateGameFields(ctx, id, map[string]any{
		"status":    entity.GameActive,
		"starts_at": startsAt,
		"ends_at":   endsAt,
	})
}

func (d *Database) UpdatePlayerProgress(ctx context.Context, gameID, uid uint32, points int) error {
	return d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Where("game_id = ? AND user_id = ?", gameID, uid).
		Update("current_points", points).Error
}

func (d *Database) MarkPlayerLeft(ctx context.Context, gameID, uid uint32) error {
	return d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Where("game_id = ? AND user_id = ?", gameID, uid).
		Updates(map[string]any{"left": true, "finished": true}).Error
}

func (d *Database) FinishGame(ctx context.Context, game *entity.Game, players []entity.GamePlayer, flags []entity.CheatFlag) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		game.Status = entity.GameFinished
		game.FinishedAt = &now
		if err := tx.Save(game).Error; err != nil {
			return err
		}
		for i := range players {
			if err := tx.Save(&players[i]).Error; err != nil {
				return err
			}
		}
		if len(flags) > 0 {
			if err := tx.Create(&flags).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AbandonGame closes a persisted game that will never produce results, such as
// a room game interrupted by the owner closing the room. Without this, the
// durable state rebuild would keep pointing at a game no runtime owns.
func (d *Database) AbandonGame(ctx context.Context, id uint32) error {
	if id == 0 {
		return nil
	}
	now := time.Now().UTC()
	return d.UpdateGameFields(ctx, id, map[string]any{
		"status":      entity.GameAbandoned,
		"finished_at": now,
	})
}

func (d *Database) SaveRatings(ctx context.Context, histories []entity.RatingHistory) error {
	if len(histories) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(&histories).Error
}

func (d *Database) SaveRecordings(ctx context.Context, recordings []entity.Recording) error {
	if len(recordings) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(&recordings).Error
}

func (d *Database) GetRecordings(ctx context.Context, gameID uint32) ([]entity.Recording, error) {
	var recordings []entity.Recording
	err := d.db.WithContext(ctx).Where("game_id = ?", gameID).Order("user_id ASC").Find(&recordings).Error
	return recordings, err
}

func (d *Database) GetRecording(ctx context.Context, gameID, userID uint32) (entity.Recording, error) {
	var recording entity.Recording
	err := d.db.WithContext(ctx).Where("game_id = ? AND user_id = ?", gameID, userID).First(&recording).Error
	return recording, err
}

// CreatePracticeRun atomically replaces the player's previous temporary
// practice attempt with a new one. Saved practice runs stay untouched.
func (d *Database) CreatePracticeRun(ctx context.Context, game *entity.Game, player *entity.GamePlayer, recording *entity.Recording) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var previous []uint32
		if err := tx.Table("games").
			Joins("JOIN game_players ON game_players.game_id = games.id").
			Where("games.mode = ? AND games.temporary = ? AND game_players.user_id = ?", entity.GamePractice, true, player.UserID).
			Pluck("games.id", &previous).Error; err != nil {
			return err
		}
		if len(previous) > 0 {
			if err := tx.Where("game_id IN ?", previous).Delete(&entity.Recording{}).Error; err != nil {
				return err
			}
			if err := tx.Where("game_id IN ?", previous).Delete(&entity.GamePlayer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", previous).Delete(&entity.Game{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(game).Error; err != nil {
			return err
		}
		player.GameID = game.ID
		recording.GameID = game.ID
		if err := tx.Create(player).Error; err != nil {
			return err
		}
		return tx.Create(recording).Error
	})
}

// SavePracticeRun makes a temporary practice run permanent. The bool reports
// whether the run was found and belonged to the player.
func (d *Database) SavePracticeRun(ctx context.Context, gameID, userID uint32) (bool, error) {
	saved := false
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game entity.Game
		if err := tx.Where("id = ? AND mode = ?", gameID, entity.GamePractice).First(&game).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		var playerCount int64
		if err := tx.Model(&entity.GamePlayer{}).
			Where("game_id = ? AND user_id = ?", gameID, userID).
			Count(&playerCount).Error; err != nil {
			return err
		}
		if playerCount == 0 {
			return nil
		}
		if err := tx.Model(&entity.Game{}).
			Where("id = ?", gameID).
			Update("temporary", false).Error; err != nil {
			return err
		}
		saved = true
		return nil
	})
	return saved, err
}

const matchHistorySelect = `SELECT g.id, gp.user_id, g.mode, COALESCE(r.code, ''), g.language, g.duration,
	gp.wpm, gp.accuracy, gp.placement, gp.rating_after - gp.rating_before,
	gp.flagged, (SELECT COUNT(*) FROM game_players x WHERE x.game_id = g.id),
	g.finished_at, gp.run_name, gp.run_description, gp.pinned, gp.pinned_at
	FROM game_players AS gp
	JOIN games AS g ON g.id = gp.game_id
	LEFT JOIN rooms AS r ON r.id = g.room_id`

const matchHistoryVisible = ` AND g.status = ? AND (g.mode <> ? OR g.temporary = ?)`

func (d *Database) UserGameHistory(ctx context.Context, uid uint32, limit int) ([]entity.MatchHistory, error) {
	query := matchHistorySelect + ` WHERE gp.user_id = ?` + matchHistoryVisible +
		` ORDER BY g.finished_at DESC LIMIT ?`
	return d.queryMatchHistory(ctx, query, uid, entity.GameFinished, entity.GamePractice, false, limit)
}

func (d *Database) PinnedGameHistory(ctx context.Context, uid uint32, limit int) ([]entity.MatchHistory, error) {
	query := matchHistorySelect + ` WHERE gp.user_id = ?` + matchHistoryVisible +
		` AND gp.pinned = ? ORDER BY gp.pinned_at DESC, g.finished_at DESC LIMIT ?`
	return d.queryMatchHistory(ctx, query, uid, entity.GameFinished, entity.GamePractice, false, true, limit)
}

func (d *Database) BestGameHistory(ctx context.Context, uid uint32) (*entity.MatchHistory, error) {
	query := matchHistorySelect + ` WHERE gp.user_id = ?` + matchHistoryVisible +
		` AND gp.left = ? AND gp.wpm > 0 ORDER BY gp.wpm DESC, g.finished_at DESC LIMIT 1`
	rows, err := d.queryMatchHistory(ctx, query, uid, entity.GameFinished, entity.GamePractice, false, false)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (d *Database) queryMatchHistory(ctx context.Context, query string, args ...any) ([]entity.MatchHistory, error) {
	rows, err := d.db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]entity.MatchHistory, 0)
	for rows.Next() {
		var item entity.MatchHistory
		var playedAt any
		var pinnedAt any
		if err := rows.Scan(
			&item.GameID, &item.UserID, &item.Mode, &item.RoomCode, &item.Language, &item.Duration,
			&item.WPM, &item.Accuracy, &item.Rank, &item.RatingDelta,
			&item.Flagged, &item.Players, &playedAt,
			&item.RunName, &item.RunDescription, &item.Pinned, &pinnedAt,
		); err != nil {
			return nil, err
		}
		item.PlayedAt = anyTime(playedAt)
		item.PinnedAt = anyTimePtr(pinnedAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

// UpdateRunMeta changes the player's presentation for one finished run.
func (d *Database) UpdateRunMeta(ctx context.Context, gameID, userID uint32, updates map[string]any) (bool, error) {
	var count int64
	query := `game_players.game_id = ? AND game_players.user_id = ? AND games.status = ? AND (games.mode <> ? OR games.temporary = ?)`
	err := d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Joins("JOIN games ON games.id = game_players.game_id").
		Where(query, gameID, userID, entity.GameFinished, entity.GamePractice, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	err = d.db.WithContext(ctx).Model(&entity.GamePlayer{}).
		Where("game_id = ? AND user_id = ?", gameID, userID).
		Updates(updates).Error
	return err == nil, err
}

func anyTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	t := anyTime(v)
	if t.IsZero() {
		return nil
	}
	return &t
}

// anyTime normalises the raw value returned by different SQL drivers for a
// timestamp. Postgres returns time.Time; sqlite (in tests) returns a string.
func anyTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, t); err == nil {
				return parsed
			}
		}
	case []byte:
		return anyTime(string(t))
	}
	return time.Time{}
}

func (d *Database) CountCheatFlags(ctx context.Context) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&entity.CheatFlag{}).Count(&n).Error
	return n, err
}
