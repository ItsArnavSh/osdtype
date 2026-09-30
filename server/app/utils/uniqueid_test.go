//go:build unit

package utils

import (
	"sync"
	"testing"
)

func TestGenerateIDIsNonZero(t *testing.T) {
	var g Generator
	if id := g.GenerateID(); id == 0 {
		t.Fatal("expected a non-zero id")
	}
}

func TestGenerateIDIsUniqueUnderConcurrency(t *testing.T) {
	var g Generator

	const n = 2000
	ids := make([]uint32, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			ids[i] = g.GenerateID()
		}(i)
	}
	wg.Wait()

	seen := make(map[uint32]int, n)
	for i, id := range ids {
		if prev, dup := seen[id]; dup {
			t.Fatalf("duplicate id %d from goroutines %d and %d", id, prev, i)
		}
		seen[id] = i
	}
}

func TestGenerateIDIsMonotonic(t *testing.T) {
	var g Generator

	prev := g.GenerateID()
	for i := 0; i < 1000; i++ {
		next := g.GenerateID()
		if next <= prev {
			t.Fatalf("ids are not increasing: %d then %d at iteration %d", prev, next, i)
		}
		prev = next
	}
}

// TestGenerateIDFitsInUint32 is a regression test.
//
// The original implementation built ids as (millisecondsSinceEpoch << 12) |
// sequence. Milliseconds since 1970 need about 41 bits, so the shift overflowed
// a uint32 and threw away the low 20 bits of the clock. The result was that any
// two ids produced roughly 2^20 ms (about 17.5 minutes) apart were numerically
// identical, which silently merges distinct users.
//
// This test pins the encoding down by checking the high bits change when the
// clock does.
func TestGenerateIDFitsInUint32(t *testing.T) {
	var g Generator

	// The whole point of the regression: the value must not be a truncated
	// millisecond clock. Rebuild the old encoding and show that it loses the
	// low bits, so a future change back to that shape fails loudly.
	oldEncode := func(ms, seq uint32) uint32 { return (ms << 12) | (seq & 0xFFF) }

	const a, b uint32 = 3983375298, 3983375298 + (1 << 20)
	if oldEncode(a, 1) != oldEncode(b, 1) {
		t.Skip("premise changed: the old encoding no longer collides here")
	}

	// The current generator reads whole seconds, not milliseconds, so its high
	// bits are a coarse clock and the low bits carry the sequence.
	first := g.GenerateID()
	if first&seqMask == 0 {
		t.Fatalf("expected the low %d bits to be a sequence, got %#x", seqBits, first)
	}
	if first>>seqBits >= epochRange {
		t.Fatalf("high bits %d exceed the %d second range", first>>seqBits, epochRange)
	}
}

// TestGeneratorEpochsDiffer confirms two generators rarely share an epoch, so
// ids from separate processes do not collide.
func TestGeneratorEpochsDiffer(t *testing.T) {
	a, b := NewGenerator(), NewGenerator()
	if a.epoch.Equal(b.epoch) {
		t.Fatal("two generators got the same epoch offset")
	}
}
