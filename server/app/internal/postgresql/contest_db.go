package postgresql

import (
	"context"
	"time"

	"osdtyp/app/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *Database) CreateContest(ctx context.Context, contest *entity.Contest) error {
	return d.db.WithContext(ctx).Create(contest).Error
}

func (d *Database) SaveContest(ctx context.Context, contest *entity.Contest) error {
	return d.db.WithContext(ctx).Save(contest).Error
}

func (d *Database) GetContest(ctx context.Context, id uint32) (entity.Contest, error) {
	var contest entity.Contest
	if err := d.db.WithContext(ctx).Preload("Host").First(&contest, "id = ?", id).Error; err != nil {
		return entity.Contest{}, err
	}
	return contest, nil
}

func (d *Database) GetContestByCode(ctx context.Context, code string) (entity.Contest, error) {
	var contest entity.Contest
	if err := d.db.WithContext(ctx).Preload("Host").Where("code = ?", code).First(&contest).Error; err != nil {
		return entity.Contest{}, err
	}
	return contest, nil
}

func (d *Database) ListPublicContests(ctx context.Context, limit int) ([]entity.Contest, error) {
	var contests []entity.Contest
	err := d.db.WithContext(ctx).Preload("Host").
		Where("visibility = ? AND status IN ?", entity.ContestPublic, []entity.ContestStatus{
			entity.ContestUpcoming,
			entity.ContestLobby,
			entity.ContestStarted,
		}).
		Order("CASE status WHEN 'started' THEN 0 WHEN 'lobby' THEN 1 ELSE 2 END, starts_at ASC, created_at DESC").
		Limit(limit).Find(&contests).Error
	return contests, err
}

func (d *Database) ListContestsForUser(ctx context.Context, uid uint32, limit int) ([]entity.Contest, error) {
	var contests []entity.Contest
	err := d.db.WithContext(ctx).Preload("Host").
		Where("host_id = ? OR id IN (SELECT contest_id FROM contest_entries WHERE user_id = ?)", uid, uid).
		Order("CASE status WHEN 'started' THEN 0 WHEN 'upcoming' THEN 1 WHEN 'lobby' THEN 1 ELSE 2 END, starts_at ASC").
		Limit(limit).Find(&contests).Error
	return contests, err
}

func (d *Database) CountContestEntries(ctx context.Context, contestID uint32) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&entity.ContestEntry{}).
		Where("contest_id = ?", contestID).Count(&count).Error
	return count, err
}

func (d *Database) ListContestEntries(ctx context.Context, contestID uint32) ([]entity.ContestEntry, error) {
	var entries []entity.ContestEntry
	err := d.db.WithContext(ctx).Preload("User").
		Where("contest_id = ?", contestID).Order("joined_at ASC").Find(&entries).Error
	return entries, err
}

func (d *Database) IsContestEntry(ctx context.Context, contestID, uid uint32) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&entity.ContestEntry{}).
		Where("contest_id = ? AND user_id = ?", contestID, uid).Count(&count).Error
	return count > 0, err
}

func (d *Database) AddContestEntry(ctx context.Context, entry *entity.ContestEntry) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(entry).Error
}

func (d *Database) SaveContestEntry(ctx context.Context, entry *entity.ContestEntry) error {
	return d.db.WithContext(ctx).Save(entry).Error
}

func (d *Database) RemoveContestEntry(ctx context.Context, contestID, uid uint32) error {
	return d.db.WithContext(ctx).Where("contest_id = ? AND user_id = ?", contestID, uid).
		Delete(&entity.ContestEntry{}).Error
}

func (d *Database) CreateContestStages(ctx context.Context, stages []*entity.ContestStage) error {
	if len(stages) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(stages).Error
}

func (d *Database) SaveContestStage(ctx context.Context, stage *entity.ContestStage) error {
	return d.db.WithContext(ctx).Save(stage).Error
}

func (d *Database) GetContestStage(ctx context.Context, id uint32) (entity.ContestStage, error) {
	var stage entity.ContestStage
	if err := d.db.WithContext(ctx).First(&stage, "id = ?", id).Error; err != nil {
		return entity.ContestStage{}, err
	}
	return stage, nil
}

func (d *Database) ListContestStages(ctx context.Context, contestID uint32) ([]entity.ContestStage, error) {
	var stages []entity.ContestStage
	err := d.db.WithContext(ctx).Where("contest_id = ?", contestID).
		Order("position ASC").Find(&stages).Error
	return stages, err
}

func (d *Database) DeleteContestStages(ctx context.Context, contestID uint32) error {
	return d.db.WithContext(ctx).Where("contest_id = ?", contestID).
		Delete(&entity.ContestStage{}).Error
}

// ListContestStageGames returns every finished or live game belonging to one
// funnel stage. Attempt games and live heats share the same table, so the
// stage aggregate can rank them uniformly.
func (d *Database) ListContestStageGames(ctx context.Context, contestID, stageID uint32) ([]entity.Game, error) {
	var games []entity.Game
	err := d.db.WithContext(ctx).
		Where("contest_id = ? AND contest_stage_id = ?", contestID, stageID).
		Order("created_at ASC").Find(&games).Error
	return games, err
}

func (d *Database) CountContestStageGames(ctx context.Context, stageID, uid uint32) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Table("games").
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("games.contest_stage_id = ? AND game_players.user_id = ?", stageID, uid).
		Count(&count).Error
	return count, err
}

func (d *Database) ActiveContestStageGame(ctx context.Context, stageID, uid uint32) (entity.Game, error) {
	var game entity.Game
	err := d.db.WithContext(ctx).
		Table("games").
		Joins("JOIN game_players ON game_players.game_id = games.id").
		Where("games.contest_stage_id = ? AND game_players.user_id = ? AND games.status IN ?",
			stageID, uid, []entity.GameStatus{entity.GameCountdown, entity.GameActive}).
		Order("games.created_at DESC").First(&game).Error
	return game, err
}

// CompleteContestTasks closes every pending scheduler row for a contest,
// including legacy contest and per-stage tasks.
func (d *Database) CompleteContestTasks(ctx context.Context, contestID uint32) error {
	err := d.db.WithContext(ctx).Model(&entity.Task{}).
		Where("category = ? AND job_id = ? AND completed = ?", entity.CONTEST, contestID, false).
		Update("completed", true).Error
	if err != nil {
		return err
	}
	return d.db.WithContext(ctx).Model(&entity.Task{}).
		Where("category = ? AND completed = ? AND job_id IN (SELECT id FROM contest_stages WHERE contest_id = ?)",
			entity.CONTEST_STAGE, false, contestID).
		Update("completed", true).Error
}

func (d *Database) CompleteContestStageTasks(ctx context.Context, stageID uint32) error {
	return d.db.WithContext(ctx).Model(&entity.Task{}).
		Where("category = ? AND job_id = ? AND completed = ?", entity.CONTEST_STAGE, stageID, false).
		Update("completed", true).Error
}

func (d *Database) CreateTask(ctx context.Context, task *entity.Task) error {
	return d.db.WithContext(ctx).Create(task).Error
}

// PopDueTask claims the next due task. The row is marked completed inside a
// transaction so two scheduler instances cannot run the same job.
func (d *Database) PopDueTask(ctx context.Context, now time.Time) (entity.Task, error) {
	var task entity.Task
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("completed = ? AND time <= ?", false, now).
			Order("time ASC").Limit(1).Find(&task)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&entity.Task{}).Where("id = ?", task.ID).Update("completed", true).Error
	})
	return task, err
}

func (d *Database) NextTaskTime(ctx context.Context) (time.Time, error) {
	var task entity.Task
	res := d.db.WithContext(ctx).Where("completed = ?", false).Order("time ASC").Limit(1).Find(&task)
	if res.Error != nil {
		return time.Time{}, res.Error
	}
	if res.RowsAffected == 0 {
		return time.Time{}, nil
	}
	return task.Time, nil
}
