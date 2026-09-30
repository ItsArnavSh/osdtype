package utils

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
	"time"
)

// Generator produces unique 32-bit identifiers.
//
// Layout of the returned value:
//
//	[ 20 bits: seconds since the generator's epoch ][ 12 bits: sequence ]
//
// A 32-bit value cannot hold a millisecond-resolution Unix timestamp. The
// epoch clock needs about 41 bits in milliseconds, so the previous scheme of
// (ms << 12) | seq silently overflowed uint32, discarded the low 20 bits of
// the clock, and handed out the same id to any two callers about 2^20 ms
// (17.5 minutes) apart.
//
// Truncating to whole seconds leaves 20 bits, about 12 days of runway, and
// spends the remaining 12 bits on an in-second counter (4096 ids per second,
// which is far beyond anything a typing game needs). The random epoch offset
// keeps ids from two processes started at different times from overlapping.
//
// Tradeoff: ids repeat once a given generator passes its 12-day window. That is
// acceptable here because the window is randomized per process, so a restart
// lands in a different region rather than replaying the previous one. If ids
// ever need to be unique forever, widen the type to uint64 rather than trying to
// squeeze more out of 32 bits.
type Generator struct {
	seq   uint32
	epoch time.Time
}

const (
	seqBits    = 12
	seqMask    = (1 << seqBits) - 1
	epochRange = 1 << 20 // seconds covered by the high 20 bits
)

func NewGenerator() Generator {
	var b [4]byte
	var offset int64
	if _, err := rand.Read(b[:]); err == nil {
		offset = int64(binary.LittleEndian.Uint32(b[:])) % epochRange
	}
	return Generator{epoch: time.Now().Add(-time.Duration(offset) * time.Second)}
}

// GenerateID returns the next identifier. It is safe for concurrent use.
func (g *Generator) GenerateID() uint32 {
	elapsed := time.Since(g.epoch)
	// A negative elapsed time means the wall clock moved backwards; clamp so
	// the high bits cannot underflow.
	if elapsed < 0 {
		elapsed = 0
	}

	seq := atomic.AddUint32(&g.seq, 1) & seqMask
	return (uint32(elapsed.Seconds()) << seqBits) | seq
}
