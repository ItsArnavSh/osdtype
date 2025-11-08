package postgresql

import (
	"fmt"
	"osdtyp/app/entity"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func ConnectDatabase(logger *zap.SugaredLogger) (Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		viper.GetString("DB.host"),
		viper.GetString("DB.user"),
		viper.GetString("DB.password"),
		viper.GetString("DB.dbname"),
		viper.GetInt("DB.port"),
	)

	logger.Infof("Connecting to database with DSN: %s", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Errorf("Could not connect to db %s", err)
		return Database{}, err
	}

	logger.Info("Connection to database established")

	// Run migrations with error checking
	logger.Info("Running database migrations...")
	err = db.AutoMigrate(
		&entity.User{},
		&entity.Room{},
		&entity.Room_User{},
		&entity.Friends{},
	)
	if err != nil {
		logger.Errorf("Failed to run migrations: %s", err)
		return Database{}, err
	}

	logger.Info("Database migrations completed successfully")

	return Database{db}, nil
}
