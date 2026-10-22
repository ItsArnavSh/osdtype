package utils

import "math/rand"

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
	p.number ^= p.number << 13
	p.number ^= p.number >> 7
	p.number ^= p.number << 17
	return p.number
}

// Random returns a float32 in [0,1).
func (p *prng) Random() float32 {
	return float32(p.nextPRN()) / float32(1<<32)
}

func (p *prng) RandomInt(min, max int) int {
	if max <= min {
		return min
	}
	return min + int(p.Random()*float32(max-min))
}

func NewPRNG(seed uint32) prng {
	p := prng{}
	if seed == 0 {
		p.generateSeed()
	} else {
		p.setSeed(seed)
	}
	return p
}
