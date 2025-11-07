package bots

import (
	"osdtyp/app/entity"

	"github.com/spf13/viper"
)

// This can take the snapshot and generate the
type Bot struct {
	stats map[entity.Persona]entity.BotConfig
}

func NewBot() Bot {
	botmap := make(map[entity.Persona]entity.BotConfig)

	botmap[entity.MAX] = entity.BotConfig{
		Accuracy:      viper.GetInt("Bots.MAX.accuracy"),
		RelativeSpeed: float32(viper.GetFloat64("Bots.MAX.relative_speed")),
	}

	botmap[entity.LEWIS] = entity.BotConfig{
		Accuracy:      viper.GetInt("Bots.LEWIS.accuracy"),
		RelativeSpeed: float32(viper.GetFloat64("Bots.LEWIS.relative_speed")),
	}

	botmap[entity.LANCE] = entity.BotConfig{
		Accuracy:      viper.GetInt("Bots.LANCE.accuracy"),
		RelativeSpeed: float32(viper.GetFloat64("Bots.LANCE.relative_speed")),
	}

	return Bot{
		stats: botmap,
	}
}
func (b *Bot) GenerateRun() {

}
