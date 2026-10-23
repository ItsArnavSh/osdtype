package bots

import (
	"osdtyp/app/entity"
	"strings"

	"github.com/spf13/viper"
)

// Manager owns the three built-in personas. Their usernames are reserved so
// they can be persisted as real users and appear on leaderboards/history.
type Manager struct {
	configs map[entity.Persona]entity.BotConfig
}

func New() *Manager {
	defaults := map[entity.Persona]entity.BotConfig{
		entity.LEWIS: {Accuracy: 95, RelativeSpeed: 1.2, Rating: 600},
		entity.MAX:   {Accuracy: 85, RelativeSpeed: 1.5, Rating: 500},
		entity.LANCE: {Accuracy: 75, RelativeSpeed: 1.8, Rating: 400},
	}
	out := make(map[entity.Persona]entity.BotConfig, len(defaults))
	for persona, cfg := range defaults {
		base := "Bots." + string(persona)
		if viper.IsSet(base + ".accuracy") {
			cfg.Accuracy = viper.GetInt(base + ".accuracy")
		}
		if viper.IsSet(base + ".relative_speed") {
			cfg.RelativeSpeed = viper.GetFloat64(base + ".relative_speed")
		}
		if viper.IsSet(base + ".rating") {
			cfg.Rating = viper.GetInt(base + ".rating")
		}
		out[persona] = cfg
	}
	return &Manager{configs: out}
}

func (m *Manager) Config(persona entity.Persona) (entity.BotConfig, bool) {
	cfg, ok := m.configs[persona]
	return cfg, ok
}

func (m *Manager) All() []entity.Persona { return entity.Personas }

func Username(persona entity.Persona) string {
	return "osd-bot-" + strings.ToLower(string(persona))
}

func (m *Manager) PersonaForUsername(username string) (entity.Persona, entity.BotConfig, bool) {
	username = strings.ToLower(username)
	for _, persona := range entity.Personas {
		if username == Username(persona) {
			cfg, ok := m.configs[persona]
			return persona, cfg, ok
		}
	}
	return "", entity.BotConfig{}, false
}

func (m *Manager) DefaultRating() int {
	for _, cfg := range m.configs {
		return cfg.Rating
	}
	return 1000
}
