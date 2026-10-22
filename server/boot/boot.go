package boot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Initialize_App loads .env, config.toml and environment overrides, then
// returns the application logger.
func Initialize_App() (*zap.SugaredLogger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}
	sugar := logger.Sugar()

	// All of these are optional; the checked-in config and defaults are enough
	// for local development.
	_ = godotenv.Load(".env", "../.env", "server/.env")

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
	_ = viper.BindEnv("Core.dev_login", "DEV_LOGIN")
	_ = viper.BindEnv("Core.oauth_redirect_url", "GITHUB_REDIRECT_URL")
	_ = viper.BindEnv("Redis.addr", "REDIS_ADDR")
	_ = viper.BindEnv("CodeGen.service_url", "CODEGEN_URL")

	sugar.Infof("config loaded (port=%s, redis=%s)", viper.GetString("Core.port"), viper.GetString("Redis.addr"))
	return sugar, nil
}

func setDefaults() {
	viper.SetDefault("Core.port", ":8080")
	viper.SetDefault("Core.frontend_url", "http://localhost:5173")
	viper.SetDefault("Core.jwt_key", "dev-insecure-change-me")
	viper.SetDefault("Core.cookie_secure", false)
	viper.SetDefault("Core.dev_login", true)
	viper.SetDefault("Core.oauth_redirect_url", "http://localhost:8080/auth/github/callback")
	viper.SetDefault("DB.host", "localhost")
	viper.SetDefault("DB.user", "user")
	viper.SetDefault("DB.password", "password")
	viper.SetDefault("DB.dbname", "typedata")
	viper.SetDefault("DB.port", 5432)
	viper.SetDefault("Redis.addr", "localhost:6379")
	viper.SetDefault("Redis.db", 0)
	viper.SetDefault("CodeGen.service_url", "http://localhost:8081/generate")
	viper.SetDefault("CodeGen.tokens", 120)
	viper.SetDefault("Game.countdown_ms", 3000)
	viper.SetDefault("Matchmaking.tick_ms", 500)
	viper.SetDefault("Matchmaking.base_range", 100)
	viper.SetDefault("Matchmaking.growth_range", 50)
	viper.SetDefault("Matchmaking.max_players", 4)
	viper.SetDefault("Matchmaking.bot_after_seconds", 20)
	viper.SetDefault("Contests.lobby_minutes", 2)
}
