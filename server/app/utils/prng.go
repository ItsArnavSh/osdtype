package utils

import (
	"math/rand"
)

type prng struct {
	seed   uint32
	number uint32
}

func (p *prng) nextPRN() uint32 {
	// Using the XOR shift method for PRN generation
	p.number ^= p.number << 13
	p.number ^= p.number >> 7
	p.number ^= p.number << 17
	return p.number
}

// Random returns a float32 in [0,1).
//
// The generator state is 32 bits, so there is no way to obtain the 53 random
// bits this function used to claim: the previous version shifted the value
// right by 11 and divided by 1<<53, which yielded a maximum of 1/2048 instead
// of 1. Everything built on top of it, RandomInt in particular, was therefore
// stuck near zero. Normalise by the full 32-bit range instead.
func (p *prng) Random() float32 {
	return float32(p.nextPRN()) / (1 << 32)
}

// randomInt returns an int in [min, max)
func (p *prng) RandomInt(min, max int) int {
	if max <= min {
		return min // avoid division by zero or negative range
	}
	r := p.Random()
	return min + int(r*float32(max-min))
}

// NewPRNG returns a seeded generator. A zero seed means "pick one at random",
// which is also the only way a caller can get a non-reproducible sequence.
func NewPRNG(seed uint32) prng {
	if seed == 0 {
		seed = rand.Uint32()
	}
	return prng{seed: seed, number: seed}
}
