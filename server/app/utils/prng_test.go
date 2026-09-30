//go:build unit

package utils

import "testing"

func TestPRNGSameSeedSameSequence(t *testing.T) {
	a := NewPRNG(12345)
	b := NewPRNG(12345)
	for i := 0; i < 50; i++ {
		if x, y := a.nextPRN(), b.nextPRN(); x != y {
			t.Fatalf("divergence at step %d: %d != %d", i, x, y)
		}
	}
}

func TestPRNGDifferentSeedsDiverge(t *testing.T) {
	a := NewPRNG(1)
	b := NewPRNG(2)
	same := 0
	for i := 0; i < 50; i++ {
		if a.nextPRN() == b.nextPRN() {
			same++
		}
	}
	if same > 1 {
		t.Errorf("expected the sequences to differ, %d/50 values matched", same)
	}
}

func TestPRNGZeroSeedStillWorks(t *testing.T) {
	// A zero seed is reseeded internally; the generator must still produce
	// values rather than staying stuck at zero.
	p := NewPRNG(0)
	if v := p.nextPRN(); v == 0 {
		t.Fatal("expected a non-zero value")
	}
}

func TestPRNGRandomStaysInUnitInterval(t *testing.T) {
	p := NewPRNG(999)
	for i := 0; i < 10000; i++ {
		v := p.Random()
		if v < 0 || v >= 1 {
			t.Fatalf("value %v out of [0,1) at iteration %d", v, i)
		}
	}
}

func TestPRNGRandomIsRoughlyUniform(t *testing.T) {
	p := NewPRNG(4242)
	const buckets = 10
	const draws = 100000
	counts := make([]int, buckets)
	for i := 0; i < draws; i++ {
		counts[int(p.Random()*buckets)]++
	}
	// Expect 10% per bucket; allow a generous 2 percentage point margin.
	lo, hi := draws/buckets*8/10, draws/buckets*12/10
	for i, c := range counts {
		if c < lo || c > hi {
			t.Errorf("bucket %d had %d hits, expected between %d and %d", i, c, lo, hi)
		}
	}
}

func TestPRNGRandomIntWithinBounds(t *testing.T) {
	p := NewPRNG(7)
	for i := 0; i < 10000; i++ {
		v := p.RandomInt(3, 9)
		if v < 3 || v >= 9 {
			t.Fatalf("value %d out of [3,9)", v)
		}
	}
}

func TestPRNGRandomIntEmptyRange(t *testing.T) {
	p := NewPRNG(7)
	if got := p.RandomInt(5, 5); got != 5 {
		t.Errorf("expected 5 for an empty range, got %d", got)
	}
	if got := p.RandomInt(9, 3); got != 9 {
		t.Errorf("expected the min for an inverted range, got %d", got)
	}
}

func TestPRNGRandomIntCoversRange(t *testing.T) {
	p := NewPRNG(31337)
	seen := make(map[int]bool)
	for i := 0; i < 5000; i++ {
		seen[p.RandomInt(0, 5)] = true
	}
	if len(seen) != 5 {
		t.Errorf("expected all 5 values to appear, saw %d", len(seen))
	}
}
