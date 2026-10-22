package postgresql

import (
	"context"
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
		Where("game_players.user_id = ? AND games.status = ?", uid, entity.GameFinished).
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
		Where("game_players.user_id = ? AND games.status = ? AND game_players.acked = ? AND game_players.left = ?", uid, entity.GameFinished, false, false).
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

func (d *Database) UserGameHistory(ctx context.Context, uid uint32, limit int) ([]entity.MatchHistory, error) {
	const query = `SELECT g.id, g.mode, COALESCE(r.code, ''), g.language, g.duration,
		gp.wpm, gp.accuracy, gp.placement, gp.rating_after - gp.rating_before,
		gp.flagged, (SELECT COUNT(*) FROM game_players x WHERE x.game_id = g.id),
		g.finished_at
		FROM game_players AS gp
		JOIN games AS g ON g.id = gp.game_id
		LEFT JOIN rooms AS r ON r.id = g.room_id
		WHERE gp.user_id = ? AND g.status = ?
		ORDER BY g.finished_at DESC LIMIT ?`
	rows, err := d.db.WithContext(ctx).Raw(query, uid, entity.GameFinished, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]entity.MatchHistory, 0, limit)
	for rows.Next() {
		var item entity.MatchHistory
		var playedAt any
		if err := rows.Scan(
			&item.GameID, &item.Mode, &item.RoomCode, &item.Language, &item.Duration,
			&item.WPM, &item.Accuracy, &item.Rank, &item.RatingDelta,
			&item.Flagged, &item.Players, &playedAt,
		); err != nil {
			return nil, err
		}
		item.PlayedAt = anyTime(playedAt)
		out = append(out, item)
	}
	return out, rows.Err()
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
