package utils

import (
	"sync/atomic"
	"time"
)

// Generates integer based unique ids
type Generator struct {
	seq uint32
}

func NewGenerator() Generator {
	return Generator{seq: 0}
}

func (g *Generator) GenerateID() uint32 {
	now := uint32(time.Now().UnixNano() / 1e6)   // milliseconds
	count := atomic.AddUint32(&g.seq, 1) & 0xFFF // 12 bits of sequence
	return (now << 12) | count
}
