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
	return nil
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
