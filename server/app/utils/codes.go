package utils

import (
	"crypto/rand"
	"math/big"
)

// Share codes for private lobbies.
//
// The codes are short and typed by hand, so the alphabet drops the characters
// people confuse: no O/0, no I/1, no l/1. Six characters from a 31 symbol
// alphabet is about 887 million combinations, which is far more than a small
// deployment needs, and the database enforces uniqueness anyway.

// CodeLength is how many characters a share code has.
const CodeLength = 6

// CodeChars is the alphabet a share code is drawn from, in a fixed order so
// the index of a character is stable.
var CodeChars = []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

// CodeAlphabet is the set of valid code characters, for membership tests.
var CodeAlphabet = func() map[rune]bool {
	m := make(map[rune]bool, len(CodeChars))
	for _, r := range CodeChars {
		m[r] = true
	}
	return m
}()

// NewCode returns a share code.
//
// Every code is drawn uniformly from the alphabet, so no character is
// favored. crypto/rand is used rather than math/rand because a predictable
// share code would let anyone walk the space of lobby URLs; a weak fallback
// here would silently produce guessable private rooms, so a failure to reach
// the entropy source panics instead.
func NewCode() string {
	out := make([]byte, CodeLength)
	max := big.NewInt(int64(len(CodeChars)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic("crypto/rand unavailable: " + err.Error())
		}
		out[i] = byte(CodeChars[n.Int64()])
	}
	return string(out)
}

// ShareCoder mints share codes.
//
// It exists as a type rather than being called directly so that a dependency on
// code generation can be held and replaced. The database holds one, which is
// why the integration tests can create two databases in one process without
// their codes interfering.
type ShareCoder struct{}

// NewShareCoder builds a coder.
func NewShareCoder() *ShareCoder { return &ShareCoder{} }

// Generate returns a share code. See NewCode.
func (ShareCoder) Generate() string { return NewCode() }
