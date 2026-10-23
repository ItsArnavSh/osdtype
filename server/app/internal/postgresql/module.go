package postgresql

import (
	"fmt"
	"time"

	"osdtyp/app/entity"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func ConnectDatabase(log *zap.SugaredLogger) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		viper.GetString("DB.host"),
		viper.GetString("DB.user"),
		viper.GetString("DB.password"),
		viper.GetString("DB.dbname"),
		viper.GetInt("DB.port"),
	)
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	d := &Database{db: gdb, logger: log}
	if err := d.Migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

// NewFromGorm is the test/embedding constructor.
func NewFromGorm(gdb *gorm.DB, log *zap.SugaredLogger) *Database {
	return &Database{db: gdb, logger: log}
}

func (d *Database) Migrate() error {
	// The first room-scoped contest implementation created `contests.id` as
	// text and never wrote a contest through the new service. GORM cannot
	// AutoMigrate that primary key into the standalone contest schema, so
	// rebuild the empty table (or preserve a populated legacy table under a
	// timestamped name for manual recovery).
	if d.db.Dialector.Name() == "postgres" && d.db.Migrator().HasTable("contests") {
		var idType string
		if err := d.db.Raw(`SELECT data_type FROM information_schema.columns
			WHERE table_name = 'contests' AND column_name = 'id'`).Scan(&idType).Error; err == nil && idType != "" && idType != "bigint" {
			var count int64
			if err := d.db.Table("contests").Count(&count).Error; err != nil {
				return fmt.Errorf("inspect legacy contests: %w", err)
			}
			if count == 0 {
				if err := d.db.Exec(`DROP TABLE contests`).Error; err != nil {
					return fmt.Errorf("drop empty legacy contests: %w", err)
				}
			} else {
				legacy := fmt.Sprintf("contests_legacy_%d", time.Now().UTC().Unix())
				if err := d.db.Exec(fmt.Sprintf(`ALTER TABLE contests RENAME TO %s`, legacy)).Error; err != nil {
					return fmt.Errorf("rename legacy contests: %w", err)
				}
				if d.logger != nil {
					d.logger.Warnw("legacy contest table preserved", "table", legacy, "rows", count)
				}
			}
			_ = d.db.Exec(`DELETE FROM tasks WHERE category = ?`, entity.CONTEST).Error
		}
	}

	// Older Postgres deployments stored contest result bytes in a JSONB
	// column. AutoMigrate cannot cast JSONB to BYTEA directly, so normalise
	// the column before GORM inspects the schema.
	if d.db.Dialector.Name() == "postgres" && d.db.Migrator().HasTable(&entity.Contest{}) {
		var dataType string
		if err := d.db.Raw(`SELECT data_type FROM information_schema.columns
			WHERE table_name = 'contests' AND column_name = 'leaderboard'`).Scan(&dataType).Error; err == nil && dataType == "jsonb" {
			if err := d.db.Exec(`ALTER TABLE contests ALTER COLUMN leaderboard TYPE bytea USING convert_to(leaderboard::text, 'UTF8')`).Error; err != nil {
				return fmt.Errorf("migrate contest leaderboard: %w", err)
			}
		}
	}

	// Earlier builds named the friends join columns a/b. The Go model now uses
	// a_id/b_id; convert legacy tables in place so ON CONFLICT has a primary
	// key that matches the model.
	if d.db.Dialector.Name() == "postgres" && d.db.Migrator().HasTable("friends") &&
		d.postgresColumnExists("friends", "a") && d.postgresColumnExists("friends", "b") {
		if err := d.db.Transaction(func(tx *gorm.DB) error {
			steps := []string{
				`ALTER TABLE friends ADD COLUMN IF NOT EXISTS a_id BIGINT`,
				`ALTER TABLE friends ADD COLUMN IF NOT EXISTS b_id BIGINT`,
				`UPDATE friends SET a_id = COALESCE(a_id, a), b_id = COALESCE(b_id, b)`,
				`ALTER TABLE friends DROP CONSTRAINT IF EXISTS friends_pkey`,
				`ALTER TABLE friends DROP COLUMN IF EXISTS a`,
				`ALTER TABLE friends DROP COLUMN IF EXISTS b`,
				`ALTER TABLE friends ALTER COLUMN a_id SET NOT NULL`,
				`ALTER TABLE friends ALTER COLUMN b_id SET NOT NULL`,
				`ALTER TABLE friends ADD PRIMARY KEY (a_id, b_id)`,
			}
			for _, step := range steps {
				if err := tx.Exec(step).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("migrate friends columns: %w", err)
		}
	}

	// The original scheduler used `job_id` as the task primary key. Standalone
	// contests need to reschedule the same job ID (for example when a host edits
	// the start time), which a job_id primary key rejects. GORM expects `id` to
	// be the primary key, so move legacy databases onto that schema before
	// AutoMigrate runs.
	if d.db.Dialector.Name() == "postgres" && d.db.Migrator().HasTable("tasks") {
		var primaryColumn string
		if err := d.db.Raw(`SELECT a.attname
			FROM pg_index i
			JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = 'tasks'::regclass AND i.indisprimary
			LIMIT 1`).Scan(&primaryColumn).Error; err == nil && primaryColumn == "job_id" {
			if err := d.db.Transaction(func(tx *gorm.DB) error {
				steps := []string{
					`ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_pkey`,
					`ALTER TABLE tasks ADD CONSTRAINT tasks_pkey PRIMARY KEY (id)`,
					`CREATE INDEX IF NOT EXISTS idx_tasks_job_id ON tasks (job_id)`,
				}
				for _, step := range steps {
					if err := tx.Exec(step).Error; err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return fmt.Errorf("migrate scheduler primary key: %w", err)
			}
		}
	}

	// Detect the first boot after per-clock ratings were introduced so the
	// legacy aggregate can be copied into each of the three ladders once.
	hadUserTable := d.db.Migrator().HasTable(&entity.User{})
	hadClockRatings := true
	if hadUserTable && d.db.Dialector.Name() == "postgres" {
		hadClockRatings = d.postgresColumnExists("users", "rating_30")
	}

	// Results acknowledgements are durable so a Redis flush cannot resurrect
	// already-dismissed results. Existing finished games predate the column and
	// must be treated as acknowledged; only new games start unacknowledged.
	hadAckColumn := d.db.Migrator().HasColumn(&entity.GamePlayer{}, "Acked")
	if err := d.db.AutoMigrate(
		&entity.User{},
		&entity.Room{},
		&entity.RoomMember{},
		&entity.RoomBan{},
		&entity.RoomInvite{},
		&entity.Friends{},
		&entity.Game{},
		&entity.GamePlayer{},
		&entity.RatingHistory{},
		&entity.CheatFlag{},
		&entity.Recording{},
		&entity.Contest{},
		&entity.ContestStage{},
		&entity.ContestEntry{},
		&entity.Task{},
		&entity.IdempotencyKey{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if !hadAckColumn {
		if err := d.db.Exec(`UPDATE game_players SET acked = ? WHERE game_id IN (SELECT id FROM games WHERE status = ?)`, true, entity.GameFinished).Error; err != nil {
			return fmt.Errorf("backfill result acknowledgements: %w", err)
		}
	}
	if hadUserTable && !hadClockRatings {
		// Copy the old single Elo into all three ladders on the first boot
		// after the split. Future boots leave the independent values alone.
		if err := d.db.Exec(`UPDATE users SET
			rating_30 = rating, peak_rating_30 = peak_rating,
			rating_90 = rating, peak_rating_90 = peak_rating,
			rating_300 = rating, peak_rating_300 = peak_rating`).Error; err != nil {
			return fmt.Errorf("backfill clock ratings: %w", err)
		}
		// Per-clock records can be reconstructed from finished ranked games.
		// This keeps existing players on the ladder after the split instead of
		// making them look unranked until they race again.
		if err := d.db.Exec(`UPDATE users u SET
			games_played_30 = s.g30, wins_30 = s.w30,
			games_played_90 = s.g90, wins_90 = s.w90,
			games_played_300 = s.g300, wins_300 = s.w300
			FROM (
				SELECT gp.user_id,
					count(*) FILTER (WHERE g.duration = 30) AS g30,
					count(*) FILTER (WHERE g.duration = 30 AND gp.placement = 1 AND NOT gp.flagged AND NOT gp.left) AS w30,
					count(*) FILTER (WHERE g.duration = 90) AS g90,
					count(*) FILTER (WHERE g.duration = 90 AND gp.placement = 1 AND NOT gp.flagged AND NOT gp.left) AS w90,
					count(*) FILTER (WHERE g.duration = 300) AS g300,
					count(*) FILTER (WHERE g.duration = 300 AND gp.placement = 1 AND NOT gp.flagged AND NOT gp.left) AS w300
				FROM game_players gp
				JOIN games g ON g.id = gp.game_id
				WHERE g.mode = ? AND g.status = ?
				GROUP BY gp.user_id
			) s
			WHERE u.id = s.user_id AND u.is_bot = ?`,
			entity.GameRanked, entity.GameFinished, false).Error; err != nil {
			return fmt.Errorf("backfill clock records: %w", err)
		}
	}
	// Accounts created under the old tier system started at 1000. Rebase
	// anyone who has never finished a ranked game onto the new 400 starting
	// Elo on every clock. Room and practice games do not put a player on a
	// ladder.
	if err := d.db.Exec(`UPDATE users SET
		rating = ?, peak_rating = ?,
		rating_30 = ?, peak_rating_30 = ?,
		rating_90 = ?, peak_rating_90 = ?,
		rating_300 = ?, peak_rating_300 = ?
		WHERE is_bot = ? AND games_played_30 = 0 AND games_played_90 = 0 AND games_played_300 = 0`,
		entity.StartingElo, entity.StartingElo,
		entity.StartingElo, entity.StartingElo,
		entity.StartingElo, entity.StartingElo,
		entity.StartingElo, entity.StartingElo,
		false,
	).Error; err != nil {
		return fmt.Errorf("rebase starting elo: %w", err)
	}
	return nil
}

func (d *Database) postgresColumnExists(table, column string) bool {
	var count int64
	err := d.db.Raw(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = ? AND column_name = ?`, table, column).Scan(&count).Error
	return err == nil && count > 0
}

func (d *Database) DB() *gorm.DB { return d.db }

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) Ping() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
