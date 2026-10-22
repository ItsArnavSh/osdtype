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

// BotWPM is the base speed scaled by the persona multiplier.
func (c BotConfig) BotWPM() float64 {
	return 70 * c.RelativeSpeed
}
