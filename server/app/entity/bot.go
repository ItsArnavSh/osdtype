package entity

type Persona uint8

const (
	LEWIS Persona = iota
	// Average, more consistent pace with less errors
	MAX   // More aggressive pace with errors and more varied speed and accuracy swings
	LANCE // Highest variance, most aggressive, and extremely high variance
)

type BotConfig struct {
	Accuracy      int
	RelativeSpeed float32
}
