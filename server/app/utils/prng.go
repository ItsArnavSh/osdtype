package utils

import (
	"math/rand"
)

type prng struct {
	seed   uint32
	number uint32
}

func (p *prng) setSeed(seed uint32) {
	p.number = seed
	p.seed = seed
}
func (p *prng) generateSeed() {
	seed := rand.Uint32()
	p.number = seed
	p.seed = seed
}
func (p *prng) nextPRN() uint32 {
	//Using the XOR shift method for PRN generation
	p.number ^= p.number << 13
	p.number ^= p.number >> 7
	p.number ^= p.number << 17
	return p.number
}

// random returns a float32 in [0,1)
func (p *prng) Random() float32 {
	// Take the next 53 random bits (same precision as math/rand.Float32)
	v := p.nextPRN() >> 11        // keep top 53 bits
	return float32(v) / (1 << 53) // normalize to [0,1)
}

// randomInt returns an int in [min, max)
func (p *prng) RandomInt(min, max int) int {
	if max <= min {
		return min // avoid division by zero or negative range
	}
	r := p.Random()
	return min + int(r*float32(max-min))
}

func NewPRNG(seed uint32) prng {
	prng := prng{}
	if seed == 0 {
		prng.generateSeed()
	} else {
		prng.setSeed(seed)
	}
	return prng
}
