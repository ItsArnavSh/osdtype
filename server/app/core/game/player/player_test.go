//go:build unit

package player

import (
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"osdtyp/app/entity"
)

func newTestPlayer(snippet string, duration time.Duration) *Player {
	return &Player{
		State:    strings.Builder{},
		Snippet:  snippet,
		Duration: duration,
		Name:     "tester",
		ID:       1,
		Logger:   zap.NewNop().Sugar(),
	}
}

func TestHandlePressAppendsCharacters(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "h"})
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "i"})

	if got := p.State.String(); got != "hi" {
		t.Fatalf("expected \"hi\", got %q", got)
	}
}

// TestHandlePressBackspaceRemovesSuffix covers the normal backspace path.
func TestHandlePressBackspaceRemovesSuffix(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "abc"})
	p.HandlePress(entity.Keypress{Action: entity.BACKSPACE, Value: "c"})

	if got := p.State.String(); got != "ab" {
		t.Fatalf("expected \"ab\", got %q", got)
	}
}

// TestHandlePressBackspaceWrongSuffixIsIgnored makes sure a mismatched delete
// cannot corrupt the state.
func TestHandlePressBackspaceWrongSuffixIsIgnored(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "abc"})
	p.HandlePress(entity.Keypress{Action: entity.BACKSPACE, Value: "x"})

	if got := p.State.String(); got != "abc" {
		t.Fatalf("expected the state to be unchanged, got %q", got)
	}
}

// TestHandlePressBackspaceOnEmptyState guards the empty-string case.
func TestHandlePressBackspaceOnEmptyState(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.BACKSPACE, Value: "x"})

	if got := p.State.String(); got != "" {
		t.Fatalf("expected an empty state, got %q", got)
	}
}

func TestHandlePressIgnoresUnknownAction(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.Action(99), Value: "zz"})

	if got := p.State.String(); got != "" {
		t.Fatalf("expected an unknown action to be ignored, got %q", got)
	}
}

// TestCalculateScoreWithinSnippet is the ordinary case.
func TestCalculateScoreWithinSnippet(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "hello"})

	got := p.CalculateScore()
	if got.Correct != 5 {
		t.Errorf("expected 5 correct, got %d", got.Correct)
	}
	if got.Wrong != 0 {
		t.Errorf("expected 0 wrong, got %d", got.Wrong)
	}
	if got.Name != "tester" {
		t.Errorf("expected the player name on the result, got %q", got.Name)
	}
}

// TestCalculateScoreOvershootDoesNotPanic is a regression test.
//
// CalculateScore used to slice the snippet with p.Snippet[:p.State.Len()].
// Nothing capped State against the snippet length, so a client that kept
// sending keypresses past the end of the snippet made the game handler panic
// with a slice bounds error and took the match down. Overshooting has to score
// as errors instead.
func TestCalculateScoreOvershootDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CalculateScore panicked on overshoot: %v", r)
		}
	}()

	p := newTestPlayer("abc", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "abcdefghij"})

	got := p.CalculateScore()
	if got.Correct != 3 {
		t.Errorf("expected the 3 overlapping characters to be correct, got %d", got.Correct)
	}
	if got.Wrong != 7 {
		t.Errorf("expected the 7 excess characters to be wrong, got %d", got.Wrong)
	}
}

func TestReachableSnippetCapsAtSnippetLength(t *testing.T) {
	p := newTestPlayer("abc", time.Minute)
	p.HandlePress(entity.Keypress{Action: entity.KEYPRESS, Value: "abcdefghij"})

	if got := p.reachableSnippet(); got != "abc" {
		t.Fatalf("expected the snippet capped to \"abc\", got %q", got)
	}
}

func TestReachableSnippetWithNoInput(t *testing.T) {
	p := newTestPlayer("abc", time.Minute)
	if got := p.reachableSnippet(); got != "" {
		t.Fatalf("expected an empty string, got %q", got)
	}
}

func TestCalculateScoreEmptyStateIsZero(t *testing.T) {
	p := newTestPlayer("hello", time.Minute)
	got := p.CalculateScore()
	if got.WPM != 0 {
		t.Errorf("expected a zero score, got %v", got.WPM)
	}
}
