package anticheat

import (
	"math"
	"osdtyp/app/entity"
	"osdtyp/app/utils"
	"sort"
)

// Report is the outcome of a single player's cheat analysis.
type Report struct {
	Risk    int      `json:"risk"`
	Reasons []string `json:"reasons"`
	Flagged bool     `json:"flagged"`
}

// Analyze inspects a run. Timestamps are milliseconds since the match start;
// events are the raw keypresses (used to spot pasted chunks).
func Analyze(timestamps []int32, events []entity.Keypress, typedLen int, durationMS int64, isBot bool) Report {
	if isBot || typedLen < 20 || len(timestamps) < 8 {
		return Report{}
	}

	var reasons []string
	risk := 0

	intervals := utils.CumulativeToDiffs(timestamps)
	sd := utils.StandardDeviation(intervals)
	if sd < 12 {
		risk += 2
		reasons = append(reasons, "keystroke variance is machine-like")
	}
	minInterval := utils.FindMinIgnoringFirst(intervals)
	if minInterval > 0 && minInterval < 25 {
		risk += 2
		reasons = append(reasons, "impossibly short key interval")
	}
	if durationMS > 0 {
		wpm := (float64(typedLen) / 5.0) / (float64(durationMS) / 60000.0)
		if wpm > 230 {
			risk += 2
			reasons = append(reasons, "sustained wpm above human range")
		}
	}

	pastes := 0
	for _, ev := range events {
		if ev.Action == entity.KEYPRESS && len([]rune(ev.Value)) > 3 {
			pastes++
		}
	}
	if pastes > 0 {
		risk += 3
		reasons = append(reasons, "multi-character keypress (paste)")
	}

	// Identical millisecond stamps repeated often means events were injected.
	sorted := append([]int32(nil), timestamps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	dupes := 0
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			dupes++
		}
	}
	if float64(dupes)/float64(len(sorted)) > 0.25 {
		risk += 2
		reasons = append(reasons, "batched keystroke timestamps")
	}

	// Sanity clamp so a report always has a stable shape.
	if math.IsNaN(sd) {
		risk = 0
		reasons = nil
	}

	return Report{Risk: risk, Reasons: reasons, Flagged: risk >= 3}
}
