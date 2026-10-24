package entity

type Persona string

const (
	LEWIS Persona = "LEWIS"
	MAX   Persona = "MAX"
	LANCE Persona = "LANCE"
)

var Personas = []Persona{LEWIS, MAX, LANCE}

type BotConfig struct {
	Accuracy      int     `mapstructure:"accuracy"`
	RelativeSpeed float64 `mapstructure:"relative_speed"`
	Rating        int     `mapstructure:"rating"`
}

// BotWPM keeps a server bot close to the pace of a human at the same Elo.
// The base curve is deliberately gentle: 30 WPM at 400 Elo and about one
// extra WPM for every 20 rating points. Persona multipliers only add a small
// amount of variety around that target, instead of making a low-Elo bot
// unnaturally fast.
func (c BotConfig) BotWPM() float64 {
	base := 30 + (float64(c.Rating)-400)*0.05
	if base < 20 {
		base = 20
	}
	spread := 0.94 + (c.RelativeSpeed-1.2)*0.2
	if spread < 0.9 {
		spread = 0.9
	}
	if spread > 1.1 {
		spread = 1.1
	}
	return base * spread
}
