package entity

// WPM is the raw input to a score calculation.
type WPM struct {
	OriginalSnippet string
	UserSnippet     string
	DurationMS      int64
}

// WPMRes is one player's result.
//
// The rating fields are filled in by whoever applied the result. Carrying them
// on the result rather than looking them up separately is what lets a run's
// stored row say what the rating did without replaying every earlier run.
type WPMRes struct {
	Name     string  `json:"name"`
	ID       uint32  `json:"id"`
	RAW      float32 `json:"raw"`
	WPM      float32 `json:"wpm"`
	Accuracy float32 `json:"accuracy"`
	Correct  int32   `json:"correct"`
	Wrong    int32   `json:"wrong"`

	// Place is the finishing position, zero based, so first place is 0.
	Place int `json:"place"`

	// RatingBefore and RatingAfter bracket the rating change for this game.
	RatingBefore uint16 `json:"rating_before"`
	RatingAfter  uint16 `json:"rating_after"`
	// Delta is RatingAfter minus RatingBefore.
	Delta int `json:"delta"`
}

// AvgWPM is words per minute, corrected for accuracy.
//
// Raw speed is a bad thing to rank on: a player who types fast and wrongly
// scores better raw speed than a player who types a little slower and
// correctly, which is the exact opposite of what the game is rewarding. Raw is
// still reported, because it is interesting, but Accuracy-weighted is what the
// leaderboard and the rating use.
func (r WPMRes) AvgWPM() float32 {
	if r.Accuracy <= 0 {
		return 0
	}
	return r.WPM * (r.Accuracy / 100)
}
