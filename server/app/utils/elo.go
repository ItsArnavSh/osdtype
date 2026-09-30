package utils

import (
	"math"

	"osdtyp/app/entity"
)

// Elo rating for multiplayer typing.
//
// The previous implementation was not Elo at all: it took the finishing
// positions of one round and renormalised them, so a player's rating after a
// round depended entirely on who else was in that round. Winning alone moved
// you up, losing alone moved you down, and there was no notion of how likely a
// result was, which is the entire point of the system.
//
// This is standard Elo with an adaptive K-factor:
//
//   - expected score against an opponent is 1/(1 + 10^((opp - self)/400))
//   - against several opponents it is the mean of those expectations
//   - actual score is linear in finishing position, so first place in a four
//     way round is 1.0 and last place is 0.0
//   - K shrinks as a player accumulates games, so a new player can climb fast
//     and an established one is not thrown around by one bad round
//
// Ratings are stored as uint16, which caps them well above anything reachable,
// and the arithmetic is done in float so a rating cannot wrap mid-calculation.
const (
	// StartingRating is what a new account is given. A zero start made every
	// new player's first expected score 0.5 against another new player and
	// 1.0 against anyone established, which combined with the old normaliser
	// meant a fresh account could reach the top of the board immediately.
	StartingRating uint16 = 1000

	// eloScale is the conventional rating scale. A 400 point gap means the
	// stronger player is expected to win about ten times in a row.
	eloScale = 400.0

	// floorRating and ceilingRating bound a rating so the uint16 column can
	// never overflow and no rating is negative.
	floorRating   = 100
	ceilingRating = 3000

	// provisionalGames is how many games a rating stays volatile for.
	provisionalGames = 30
	// establishedGames is where the K-factor reaches its floor.
	establishedGames = 200

	// kProvisional, kIntermediate and kEstablished are the K-factors.
	kProvisional  = 40
	kIntermediate = 24
	kEstablished  = 16
	// kVolatility is added to K for a brand new account, so the first few
	// results move a new player decisively in either direction.
	kVolatility = 20
)

// EloEngine computes rating deltas for a finished round.
type EloEngine struct{}

// NewEloEngine builds an engine. It holds no state, so this is cheap and the
// value can be shared.
func NewEloEngine() EloEngine { return EloEngine{} }

// kFactor is how much a single round can move a rating, given how many games
// the player has already played.
func (EloEngine) kFactor(gamesPlayed int) float64 {
	switch {
	case gamesPlayed < provisionalGames:
		return kProvisional + kVolatility
	case gamesPlayed < establishedGames:
		return kIntermediate
	default:
		return kEstablished
	}
}

// expected is the probability that a player rated self beats an opponent
// rated opp.
func expected(self, opp float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (opp-self)/eloScale))
}

// ScoreForPosition converts a finishing position into an actual score.
//
// With one player there is no result, so the score is the expected value for a
// draw, which leaves the rating unchanged rather than pretending a solo
// round was a win.
func ScoreForPosition(position, fieldSize int) float64 {
	if fieldSize <= 1 {
		return 0.5
	}
	if position < 0 {
		position = 0
	}
	if position >= fieldSize {
		position = fieldSize - 1
	}
	return 1.0 - float64(position)/float64(fieldSize-1)
}

// Rate returns the new rating for every entry in a leaderboard, in the same
// order.
//
// leaderboard is expected to be sorted best first, which is how a game
// handler produces it. ratings and played are parallel slices holding each
// player's current rating and how many rated games they have completed.
func (e EloEngine) Rate(board []entity.WPMRes, ratings []uint16, played []int) []uint16 {
	n := len(board)
	out := make([]uint16, n)
	if n == 0 {
		return out
	}

	// A field of one is a practice run, not a match: there is no expected score
	// to compute, because there is no opponent. Dividing the accumulated
	// expectation by n-1 below is 0/0, and every delta then became NaN — which
	// int() converts to an implementation-defined value, so a lone player's
	// rating was clamped to the floor. The same NaN reached any field where a
	// player's own rating was the only one, so it is worth returning the
	// ratings unchanged rather than trying to make the mean work.
	if n == 1 {
		out[0] = clampRating(int(ratingsAt(ratings, 0, StartingRating)))
		return out
	}

	// One pass over the field per player. Expected score is the mean over
	// every opponent, which is the standard multiplayer simplification. Against
	// a symmetric field this reduces to 0.5, so a player who neither gained
	// nor lost rank from a mixed field does not drift.
	for i := range board {
		self := float64(ratingsAt(ratings, i, StartingRating))

		exp := 0.0
		for j := range board {
			if j == i {
				continue
			}
			exp += expected(self, float64(ratingsAt(ratings, j, StartingRating)))
		}
		exp /= float64(n - 1)

		actual := ScoreForPosition(i, n)
		games := playedAt(played, i)

		next := self + e.kFactor(games)*(actual-exp)
		out[i] = clampRating(int(math.Round(next)))
	}

	return out
}

// RatingAfter returns the rating a player would have after a round, given a
// delta. The solo run path uses this to report a projected change without
// writing anything.
func RatingAfter(current uint16, delta int) uint16 { return clampRating(int(current) + delta) }

// Solo practice constants.
//
// A solo run has no opponent, so there is nothing to compute an expected score
// against. The design has to answer a different question instead: how far is
// this run from what this player's rating says they should manage?
//
// The update is therefore a pull toward the rating a run's WPM implies, rather
// than a fixed bonus for beating a threshold. That distinction is the whole
// design. A fixed bonus pays out for every run above the line, so repeating one
// good run walks a rating to the ceiling and the number stops meaning anything.
// A pull has a fixed point: the rating converges to where the WPM implied and
// the WPM achieved agree, and stays there however many more runs are done. The
// only thing that moves it after that is getting faster, which is what a rating
// is supposed to measure.
const (
	// soloBaseWPM is the WPM that implies the starting rating. Seventy is
	// roughly an average typist, so a new account neither gains nor loses for
	// an ordinary session.
	soloBaseWPM = 70.0

	// soloWPMPerPoint is how much faster the implied WPM gets per rating point.
	// At 0.2 wpm a point, 1250 (Gold) implies 120 wpm and 2100 (Legend) implies
	// 250. Those are the numbers a rating ladder should imply, and they are in
	// the range a real typist can actually reach.
	soloWPMPerPoint = 0.2

	// soloConvergence is the fraction of the remaining distance one run
	// closes. An eighth converges in roughly fifty runs, which is about how
	// long it takes a person to actually get faster, so the rating keeps
	// moving for as long as there is something to learn.
	soloConvergence = 8.0

	// soloMaxStep caps how far one run can move a rating, in either
	// direction. Without it a single wild run would teleport somebody across
	// the whole ladder, and the cap is what makes the relationship to ranked
	// play honest: a practice run is worth less than a game against a person.
	// It sits below every rated K-factor, so playing people always moves a
	// rating more than practicing does.
	soloMaxStep = 12
)

// SoloTargetWPM is the WPM a rating implies.
//
// It is exported because it is a rule the interface would want to explain: a
// player told they did not gain rating deserves to know what was expected.
func SoloTargetWPM(rating uint16) float64 {
	return soloBaseWPM + soloWPMPerPoint*(float64(rating)-float64(StartingRating))
}

// SoloRatingForWPM is the rating a WPM implies, the inverse of SoloTargetWPM.
func SoloRatingForWPM(wpm float64) uint16 {
	return clampRating(int(math.Round(float64(StartingRating) + (wpm-soloBaseWPM)/soloWPMPerPoint)))
}

// RateSolo returns the rating a player would hold after a solo run.
//
// The rating moves an eighth of the way from where it is to the rating this
// run implies, capped at soloMaxStep, so a run always has a visible effect
// while there is somewhere to go and settles when there is not.
//
// gamesPlayed is accepted for symmetry with Rate and is deliberately unused.
// A rated game's K shrinks with experience because more games means a more
// reliable estimate of a player's ability. A practice run carries no new
// information about how a player does against other people, so there is
// nothing for experience to change.
func (EloEngine) RateSolo(wpm float32, current uint16, _ int) uint16 {
	target := SoloRatingForWPM(float64(wpm))

	step := (float64(target) - float64(current)) / soloConvergence
	if step > soloMaxStep {
		step = soloMaxStep
	}
	if step < -soloMaxStep {
		step = -soloMaxStep
	}

	return clampRating(int(math.Round(float64(current) + step)))
}

func clampRating(n int) uint16 {
	if n < floorRating {
		return floorRating
	}
	if n > ceilingRating {
		return ceilingRating
	}
	return uint16(n)
}

func ratingsAt(ratings []uint16, i int, fallback uint16) uint16 {
	if i < len(ratings) {
		return ratings[i]
	}
	return fallback
}

func playedAt(played []int, i int) int {
	if i < len(played) {
		return played[i]
	}
	return 0
}

// Tier is a named band of the rating range, for display.
type Tier struct {
	Name  string
	Min   uint16
	Color string
}

// tiers is ordered from lowest to highest.
var tiers = []Tier{
	{"Bronze", 0, "#cd7f32"},
	{"Silver", 1100, "#9fb4c7"},
	{"Gold", 1250, "#d4af37"},
	{"Platinum", 1400, "#7fd4c1"},
	{"Diamond", 1550, "#8ecae6"},
	{"Master", 1750, "#b388ff"},
	{"Grandmaster", 1900, "#ff8a65"},
	{"Legend", 2100, "#ff5252"},
}

// TierOf returns the named band a rating falls in.
func TierOf(rating uint16) Tier {
	out := tiers[0]
	for _, t := range tiers {
		if rating >= t.Min {
			out = t
		}
	}
	return out
}

// AllTiers returns the tier table, for a client that wants to render a ladder.
func AllTiers() []Tier {
	out := make([]Tier, len(tiers))
	copy(out, tiers)
	return out
}
