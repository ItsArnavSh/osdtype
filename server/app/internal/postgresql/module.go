package postgresql

import (
	"fmt"
	"strings"

	"osdtyp/app/entity"
	"osdtyp/app/utils"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database is the persistence layer.
type Database struct {
	db *gorm.DB
	// codes mints room share codes. It is a field rather than a package global
	// so two databases in one process, as the integration tests create, cannot
	// interfere with each other's codes.
	codes *utils.ShareCoder
}

// ConnectDatabase opens the pool and runs the migrations.
//
// An optional "DB.search_path" viper key isolates the connection to a single
// Postgres schema. The integration suite sets it so that the packages go test
// runs in parallel each get their own tables instead of trampling on one shared
// set. It is empty in production, where the connection uses the server default.
func ConnectDatabase(logger *zap.SugaredLogger) (Database, error) {
	searchPath := strings.TrimSpace(viper.GetString("DB.search_path"))

	if searchPath != "" {
		if err := ensureSchema(logger, searchPath); err != nil {
			return Database{}, err
		}
	}

	dsn := buildDSN(viper.GetString("DB.host"),
		viper.GetString("DB.user"),
		viper.GetString("DB.password"),
		viper.GetString("DB.dbname"),
		viper.GetInt("DB.port"),
		searchPath)

	logger.Infof("Connecting to database with DSN: %s", redactDSN(dsn))

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
		&entity.Task{},
		&entity.Contest{},
		&entity.Notification{},
		&entity.Run{},
		&entity.Settings{},
	)
	if err != nil {
		logger.Errorf("Failed to run migrations: %s", err)
		return Database{}, err
	}

	logger.Info("Database migrations completed successfully")

	return Database{db: db, codes: utils.NewShareCoder()}, nil
}

func buildDSN(host, user, password, dbname string, port int, searchPath string) string {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		host, user, password, dbname, port)

	if searchPath != "" {
		// Conninfo values are whitespace delimited, so a multi-entry search path
		// has to be quoted.
		dsn += " search_path=" + quoteConninfo(searchPath)
	}
	return dsn
}

// quoteConninfo wraps a conninfo value in single quotes, escaping any it
// already contains.
func quoteConninfo(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `\'`) + "'"
}

// ensureSchema creates the target schema if it is missing.
//
// A connection whose search_path names a schema that does not exist cannot
// create tables ("no schema has been selected"), so the schema is created over
// a separate, short-lived connection that uses the server default.
func ensureSchema(logger *zap.SugaredLogger, searchPath string) error {
	bootstrap, err := gorm.Open(postgres.Open(buildDSN(viper.GetString("DB.host"),
		viper.GetString("DB.user"),
		viper.GetString("DB.password"),
		viper.GetString("DB.dbname"),
		viper.GetInt("DB.port"),
		"")), &gorm.Config{})
	if err != nil {
		logger.Errorf("Could not open the database to create a schema: %s", err)
		return err
	}
	defer func() {
		if sqlDB, err := bootstrap.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	for _, name := range strings.Split(searchPath, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Doubling the quote is how a Postgres identifier is escaped.
		stmt := `CREATE SCHEMA IF NOT EXISTS "` + strings.ReplaceAll(name, `"`, `""`) + `"`
		if err := bootstrap.Exec(stmt).Error; err != nil {
			logger.Errorf("Could not create schema %q: %s", name, err)
			return err
		}
	}
	return nil
}

// redactDSN hides the password so the DSN is safe to log.
func redactDSN(dsn string) string {
	for _, field := range strings.Fields(dsn) {
		if strings.HasPrefix(field, "password=") {
			return strings.Replace(dsn, field, "password=*****", 1)
		}
	}
	return dsn
}
