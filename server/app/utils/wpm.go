package utils

import "osdtyp/app/entity"

func Calculate_WPM(wpm entity.WPM) entity.WPMRes {
	originalRunes := []rune(wpm.OriginalSnippet)
	userRunes := []rune(wpm.UserSnippet)

	var correct, wrong int32 = 0, 0

	minLen := min(len(userRunes), len(originalRunes))

	// Count correct and wrong characters by comparing positions
	for i := range minLen {
		if userRunes[i] == originalRunes[i] {
			correct++
		} else {
			wrong++
		}
	}

	// If user typed extra characters beyond original snippet, consider all extra as wrong
	extra := len(userRunes) - len(originalRunes)
	if extra > 0 {
		wrong += int32(extra)
	}

	totalTyped := int32(len(userRunes))
	if totalTyped == 0 || wpm.DurationMS == 0 {
		// Avoid division by zero
		return entity.WPMRes{}
	}

	durationMinutes := float64(wpm.DurationMS) / 60000.0

	rawWPM := (float64(totalTyped) / 5.0) / durationMinutes

	accuracy := float64(correct) / float64(totalTyped)

	netWPM := rawWPM * accuracy

	// Clamp accuracy and WPM minimum to 0
	if accuracy < 0 {
		accuracy = 0
	}
	if netWPM < 0 {
		netWPM = 0
	}

	return entity.WPMRes{
		RAW:      rawWPM,
		WPM:      netWPM,
		Accuracy: accuracy,
		Correct:  correct,
		Wrong:    wrong,
	}
}
