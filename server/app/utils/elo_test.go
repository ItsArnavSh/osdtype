//go:build unit

package utils

import (
	"math"
	"testing"

	"osdtyp/app/entity"
)

// board builds a leaderboard of n equal-scoring players, which is the shape the
// rate engine actually receives from a game handler: sorted best first.
func board(n int) []entity.WPMRes {
	out := make([]entity.WPMRes, n)
	for i := range out {
		out[i] = entity.WPMRes{ID: uint32(i + 1), WPM: float32(n - i)}
	}
	return out
}

func TestRateEmptyInput(t *testing.T) {
	got := NewEloEngine().Rate(nil, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected an empty result, got %v", got)
	}
}

func TestRateMismatchedLengthsUsesFallback(t *testing.T) {
	// A short ratings slice must not panic or index out of bounds. The missing
	// entries fall back to the starting rating, which is the only defensible
	// answer: the caller has told us nothing about them.
	got := NewEloEngine().Rate(board(2), []uint16{1400}, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	for i, v := range got {
		if v < floorRating || v > ceilingRating {
			t.Fatalf("entry %d escaped the clamp: %d", i, v)
		}
	}
}

func TestRateSinglePlayerKeepsRating(t *testing.T) {
	got := NewEloEngine().Rate(board(1), []uint16{1750}, []int{40})
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0] != 1750 {
		t.Fatalf("a single player's rating changed: got %d, want 1750", got[0])
	}
}

func TestRateWinnerGainsLoserLoses(t *testing.T) {
	got := NewEloEngine().Rate(board(2), []uint16{1000, 1000}, []int{50, 50})

	if got[0] <= 1000 {
		t.Errorf("first place should have gained, got %d", got[0])
	}
	if got[1] >= 1000 {
		t.Errorf("last place should have lost, got %d", got[1])
	}
	if got[0] == got[1] {
		t.Errorf("the two ratings should have diverged, both are %d", got[0])
	}
}

func TestRateSymmetricFieldDoesNotDrift(t *testing.T) {
	// With every opponent the same rating as you, the expected score is
	// exactly 0.5 for everyone, so an even spread of places has to leave the
	// ladder where it was. A field that drifts here means the multiplayer
	// average is wrong, which would inflate or deflate every rating over time.
	ratings := []uint16{1000, 1000, 1000, 1000}
	played := []int{50, 50, 50, 50}
	spread := []entity.WPMRes{
		{ID: 1, WPM: 90}, {ID: 2, WPM: 80}, {ID: 3, WPM: 70}, {ID: 4, WPM: 60},
	}

	got := NewEloEngine().Rate(spread, ratings, played)
	total := 0
	for _, v := range got {
		total += int(v)
	}
	before := 0
	for _, v := range ratings {
		before += int(v)
	}
	if total != before {
		t.Errorf("a symmetric field drifted the ladder: %d became %d", before, total)
	}
}

func TestRateUnderdogGainsOnAnUpset(t *testing.T) {
	// The leaderboard and the ratings are both positional and both best-first:
	// a 1000 rated player wins against a 2000 rated one.
	got := NewEloEngine().Rate(
		[]entity.WPMRes{{ID: 2, WPM: 90}, {ID: 1, WPM: 40}},
		[]uint16{1000, 2000},
		[]int{80, 80})

	if got[0] <= 1000 {
		t.Errorf("the underdog should have gained rating, got %d", got[0])
	}
	if got[1] >= 2000 {
		t.Errorf("the favorite should have lost rating, got %d", got[1])
	}
}

// TestRateBeatingAWeakPlayerIsWorthLess is the property that distinguishes Elo
// from the old normaliser.
//
// Beating somebody rated 1000 while you are also rated 1000 is a coin flip
// that went your way. Beating somebody rated 500 is not. The old normaliser
// paid the same delta for both, which made a lobby of beginners the fastest
// route to the top of the board.
func TestRateBeatingAWeakPlayerIsWorthLess(t *testing.T) {
	againstEqual := NewEloEngine().Rate(board(2), []uint16{1000, 1000}, []int{50, 50})
	againstWeak := NewEloEngine().Rate(board(2), []uint16{1000, 500}, []int{50, 50})

	equalGain := int(againstEqual[0]) - 1000
	weakGain := int(againstWeak[0]) - 1000

	if equalGain <= weakGain {
		t.Errorf("beating an equal gained %d, beating a 500 rated player gained %d; the weaker opponent was worth more",
			equalGain, weakGain)
	}
}

// TestRateBeatingAStrongerPlayerIsWorthMore is the other half of the same
// property: an upset is the most valuable result a player can produce.
func TestRateBeatingAStrongerPlayerIsWorthMore(t *testing.T) {
	againstEqual := NewEloEngine().Rate(board(2), []uint16{1000, 1000}, []int{50, 50})
	againstStrong := NewEloEngine().Rate(board(2), []uint16{1000, 1600}, []int{50, 50})

	equalGain := int(againstEqual[0]) - 1000
	strongGain := int(againstStrong[0]) - 1000

	if strongGain <= equalGain {
		t.Errorf("an upset gained %d, an even match gained %d", strongGain, equalGain)
	}
}

func TestRateRespectsTheClamp(t *testing.T) {
	// A long run of heavy losses must not drive a rating below the floor, which
	// is what keeps the uint16 column safe and keeps a new account usable.
	got := NewEloEngine().Rate(
		[]entity.WPMRes{{ID: 1, WPM: 1}, {ID: 2, WPM: 999}},
		[]uint16{floorRating, ceilingRating},
		[]int{0, 0})

	if got[0] < floorRating {
		t.Errorf("rating %d is below the floor %d", got[0], floorRating)
	}
	if got[1] > ceilingRating {
		t.Errorf("rating %d is above the ceiling %d", got[1], ceilingRating)
	}
}

func TestRateProvisionalRatingsMoveMore(t *testing.T) {
	// A new account should be able to move quickly, and a settled one should
	// not be thrown around by a single round. If these were equal the K-factor
	// ladder would not be doing anything.
	newcomer := NewEloEngine().Rate(board(2), []uint16{1000, 1000}, []int{0, 0})
	settled := NewEloEngine().Rate(board(2), []uint16{1000, 1000}, []int{500, 500})

	newGain := int(newcomer[0]) - 1000
	oldGain := int(settled[0]) - 1000

	if newGain <= oldGain {
		t.Errorf("a provisional rating gained %d, a settled one gained %d", newGain, oldGain)
	}
}

func TestScoreForPosition(t *testing.T) {
	cases := []struct {
		pos, field int
		want       float64
	}{
		{0, 4, 1.0},
		{3, 4, 0.0},
		{0, 2, 1.0},
		{1, 2, 0.0},
		{0, 1, 0.5},
		{-5, 4, 1.0}, // out of range below
		{99, 4, 0.0}, // out of range above
	}
	for _, c := range cases {
		if got := ScoreForPosition(c.pos, c.field); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("ScoreForPosition(%d, %d) = %v, want %v", c.pos, c.field, got, c.want)
		}
	}
}

func TestRateSoloMovesLessThanARatedGame(t *testing.T) {
	e := NewEloEngine()

	// A very strong run from a rating that implies far less. The step is capped
	// so one session cannot teleport somebody across the ladder.
	solo := int(e.RateSolo(1000, 1000, 500)) - 1000
	if solo <= 0 {
		t.Fatalf("an excellent solo run should gain rating, gained %d", solo)
	}
	if solo > soloMaxStep {
		t.Errorf("a solo run gained %d, above the %d cap", solo, soloMaxStep)
	}
}

func TestRateSoloTargetIsWhereRatingHolds(t *testing.T) {
	e := NewEloEngine()
	if got := e.RateSolo(float32(SoloTargetWPM(1000)), 1000, 100); got != 1000 {
		t.Errorf("a run at the target WPM moved the rating to %d", got)
	}
}

func TestSoloTargetRisesWithRating(t *testing.T) {
	// A higher rating has to imply a higher expectation, or there is no fixed
	// point and practice is an infinite grind.
	low := SoloTargetWPM(1000)
	high := SoloTargetWPM(2000)
	if high <= low {
		t.Errorf("a 2000 rating expects %v wpm and a 1000 rating expects %v; the target did not rise", high, low)
	}
	if math.Abs(low-soloBaseWPM) > 1e-9 {
		t.Errorf("the starting rating should expect exactly %v wpm, got %v", soloBaseWPM, low)
	}
}

func TestSoloTargetAndRatingAreInverses(t *testing.T) {
	// The two functions are used to explain the ladder to a player, so they
	// have to agree with each other.
	for _, rating := range []uint16{1000, 1250, 1500, 1750, 2100} {
		wpm := SoloTargetWPM(rating)
		if back := SoloRatingForWPM(wpm); back > rating+1 || back+1 < rating {
			t.Errorf("%.1f wpm implies %d, not %d", wpm, back, rating)
		}
	}
}

func TestRateSoloLosesRatingBelowTarget(t *testing.T) {
	if got := NewEloEngine().RateSolo(20, 1000, 100); got >= 1000 {
		t.Errorf("a 20 wpm run left the rating at %d", got)
	}
}

func TestRateSoloStaysInsideTheClamp(t *testing.T) {
	// An implausible WPM must not produce a rating outside the ladder, even
	// though the anticheat has not run yet by the time this is called.
	e := NewEloEngine()
	if got := e.RateSolo(10_000, floorRating, 0); got > ceilingRating {
		t.Errorf("an absurd run produced a rating of %d, above the ceiling", got)
	}
	if got := e.RateSolo(-50, ceilingRating, 0); got < floorRating {
		t.Errorf("a negative run produced a rating of %d, below the floor", got)
	}
}

// TestRateSoloCannotBeGrinded is the regression that matters for the feature.
//
// The first version of this paid a constant delta for any run above the
// target, so a thousand repetitions of one good run walked the rating to the
// ceiling and the number stopped meaning anything. A pull toward the implied
// rating has a fixed point instead: it converges to where the WPM implied and
// the WPM achieved agree, and stays there however many more runs are done.
func TestRateSoloCannotBeGrinded(t *testing.T) {
	const wpm = 180.0

	e := NewEloEngine()
	rating := uint16(1000)
	for i := 0; i < 500; i++ {
		rating = e.RateSolo(wpm, rating, 500)
	}
	settled := rating

	// A thousand more identical runs must move the rating by nothing at all.
	for i := 0; i < 1000; i++ {
		rating = e.RateSolo(wpm, rating, 500)
	}
	if rating != settled {
		t.Errorf("a thousand more identical runs moved the rating from %d to %d; practice alone is still a route up", settled, rating)
	}

	// And the fixed point is where the arithmetic says it should be: 180 wpm
	// at 0.2 wpm per point is 550 points above the starting rating.
	if want := SoloRatingForWPM(wpm); math.Abs(float64(settled)-float64(want)) > 3 {
		t.Errorf("the fixed point settled at %d for a %.0f wpm run, want about %d", settled, wpm, want)
	}
}

// TestRateSoloIsWorthLessThanARatedGame is the relationship that makes the
// ladder honest: a practice run is worth less than a game against a person, so
// there is no reason to only ever practice.
func TestRateSoloIsWorthLessThanARatedGame(t *testing.T) {
	e := NewEloEngine()

	// The largest step a solo run can take, measured from a rating far below
	// what the WPM implies. The first run is the one that matters here: by the
	// hundredth the rating has converged and the step is legitimately zero.
	step := int(e.RateSolo(250, floorRating, 500)) - int(floorRating)

	if step == 0 {
		t.Fatal("a run far above the implied target did not move the rating at all")
	}
	if step > soloMaxStep {
		t.Errorf("a solo run moved the rating %d, above the %d cap", step, soloMaxStep)
	}
	if step >= int(kEstablished) {
		t.Errorf("a solo run moved the rating %d, as much as or more than a rated game (K=%v)", step, kEstablished)
	}

	// And the step shrinks as the rating approaches what the run implies, which
	// is what the cap and the pull together buy. Convergence is geometric, so
	// it takes a couple of hundred runs to get inside the cap's reach.
	later := 0
	rating := uint16(1000)
	for i := 0; i < 200; i++ {
		next := e.RateSolo(250, rating, 500)
		later = int(next) - int(rating)
		rating = next
	}
	if later >= step {
		t.Errorf("a solo run moved the rating %d two hundred runs in, against %d on the first", later, step)
	}
}

// TestRateSoloRewardsRealImprovement is the other half: a genuinely faster
// player must still be able to climb, or the fixed point is a wall.
func TestRateSoloRewardsRealImprovement(t *testing.T) {
	e := NewEloEngine()

	improver := uint16(1000)
	for i := 0; i < 50; i++ {
		improver = e.RateSolo(160, improver, 100)
	}
	if improver <= 1100 {
		t.Errorf("fifty improving runs only reached %d", improver)
	}

	// A player who never improves stays exactly where they were.
	stagnant := uint16(1000)
	for i := 0; i < 50; i++ {
		stagnant = e.RateSolo(70, stagnant, 100)
	}
	if stagnant != 1000 {
		t.Errorf("a player typing at the target drifted to %d", stagnant)
	}
}

func TestTierBands(t *testing.T) {
	cases := []struct {
		rating uint16
		name   string
	}{
		{100, "Bronze"},
		{1099, "Bronze"},
		{1100, "Silver"},
		{1250, "Gold"},
		{1400, "Platinum"},
		{1550, "Diamond"},
		{1750, "Master"},
		{1900, "Grandmaster"},
		{2100, "Legend"},
		{3000, "Legend"},
	}
	for _, c := range cases {
		if got := TierOf(c.rating).Name; got != c.name {
			t.Errorf("TierOf(%d) = %s, want %s", c.rating, got, c.name)
		}
	}
}

func TestAllTiersIsACopy(t *testing.T) {
	first := AllTiers()
	if len(first) == 0 {
		t.Fatal("the tier table is empty")
	}
	first[0].Name = "tampered"
	if AllTiers()[0].Name == "tampered" {
		t.Error("AllTiers handed out the live table; a caller can corrupt every other reader")
	}
}

func TestTiersAreAscending(t *testing.T) {
	// A table out of order would make TierOf pick the last match rather than
	// the highest band, which is a silent mislabel rather than a crash.
	all := AllTiers()
	for i := 1; i < len(all); i++ {
		if all[i].Min <= all[i-1].Min {
			t.Fatalf("tier %s (%d) is not above %s (%d)",
				all[i].Name, all[i].Min, all[i-1].Name, all[i-1].Min)
		}
	}
}
