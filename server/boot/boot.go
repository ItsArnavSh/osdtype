package boot

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func newLogger() (*zap.Logger, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json") {
		cfg := zap.NewProductionConfig()
		return cfg.Build()
	}
	return zap.NewDevelopment()
}

// Initialize_App loads .env, config.toml and environment overrides, then
// returns the application logger.
func Initialize_App() (*zap.SugaredLogger, error) {
	// Load .env before creating the logger so LOG_FORMAT applies locally too.
	_ = godotenv.Load(".env", "../.env", "server/.env")

	logger, err := newLogger()
	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}
	sugar := logger.Sugar()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./boot")
	viper.AddConfigPath("../server/boot")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		sugar.Warn("config.toml not found, using defaults")
	}

	setDefaults()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("Core.port", "PORT")
	_ = viper.BindEnv("Core.jwt_key", "JWTKEY")
	_ = viper.BindEnv("Core.frontend_url", "FRONTEND_URL")
	_ = viper.BindEnv("Core.cookie_secure", "COOKIE_SECURE")
	_ = viper.BindEnv("Core.cookie_samesite", "COOKIE_SAMESITE")
	_ = viper.BindEnv("Core.dev_login", "DEV_LOGIN")
	_ = viper.BindEnv("Core.oauth_redirect_url", "GITHUB_REDIRECT_URL")
	_ = viper.BindEnv("DB.url", "DATABASE_URL")
	_ = viper.BindEnv("DB.sslmode", "DB_SSLMODE")
	_ = viper.BindEnv("DB.max_open_conns", "DB_MAX_OPEN_CONNS")
	_ = viper.BindEnv("DB.max_idle_conns", "DB_MAX_IDLE_CONNS")
	_ = viper.BindEnv("Redis.url", "REDIS_URL")
	_ = viper.BindEnv("Redis.addr", "REDIS_ADDR")
	_ = viper.BindEnv("Redis.tls", "REDIS_TLS")
	_ = viper.BindEnv("CodeGen.service_url", "CODEGEN_URL")
	_ = viper.BindEnv("BugReport.webhook_url", "BUG_WEBHOOK_URL")
	_ = viper.BindEnv("Metrics.token", "METRICS_TOKEN")

	redisDesc := viper.GetString("Redis.addr")
	if strings.TrimSpace(viper.GetString("Redis.url")) != "" {
		redisDesc = "configured via REDIS_URL"
	}
	sugar.Infof("config loaded (port=%s, redis=%s)", viper.GetString("Core.port"), redisDesc)
	return sugar, nil
}

func setDefaults() {
	viper.SetDefault("Core.port", ":8080")
	viper.SetDefault("Core.frontend_url", "http://localhost:5173")
	viper.SetDefault("Core.jwt_key", "dev-insecure-change-me")
	viper.SetDefault("Core.cookie_secure", false)
	viper.SetDefault("Core.cookie_samesite", "lax")
	viper.SetDefault("Core.dev_login", true)
	viper.SetDefault("Core.oauth_redirect_url", "http://localhost:8080/auth/github/callback")
	viper.SetDefault("DB.url", "")
	viper.SetDefault("DB.host", "localhost")
	viper.SetDefault("DB.user", "user")
	viper.SetDefault("DB.password", "password")
	viper.SetDefault("DB.dbname", "typedata")
	viper.SetDefault("DB.port", 5432)
	viper.SetDefault("DB.sslmode", "disable")
	viper.SetDefault("DB.max_open_conns", 10)
	viper.SetDefault("DB.max_idle_conns", 5)
	viper.SetDefault("Redis.url", "")
	viper.SetDefault("Redis.addr", "localhost:6379")
	viper.SetDefault("Redis.db", 0)
	viper.SetDefault("Redis.tls", false)
	viper.SetDefault("CodeGen.service_url", "http://localhost:8081/generate")
	viper.SetDefault("BugReport.webhook_url", "")
	viper.SetDefault("Metrics.token", "")
	viper.SetDefault("CodeGen.tokens", 120)
	viper.SetDefault("Game.countdown_ms", 3000)
	viper.SetDefault("Matchmaking.tick_ms", 500)
	viper.SetDefault("Matchmaking.base_range", 100)
	viper.SetDefault("Matchmaking.growth_range", 50)
	viper.SetDefault("Matchmaking.max_players", 4)
	viper.SetDefault("Matchmaking.bot_after_seconds", 20)
	viper.SetDefault("Contests.lobby_minutes", 2)
}
