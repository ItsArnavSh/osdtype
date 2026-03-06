package utils

import (
	"fmt"
	"osdtyp/app/entity"
)

func cleanSnippet(s string) []rune {
	var res []rune
	for _, r := range s {
		if r == ' ' || r == '\n' {
			continue
		}
		res = append(res, r)
	}
	return res
}

func Calculate_WPM(wpm entity.WPM) entity.WPMRes {
	fmt.Println("Original: ", wpm.OriginalSnippet)
	fmt.Println("User: ", wpm.UserSnippet)
	originalRunes := cleanSnippet(wpm.OriginalSnippet)
	userRunes := cleanSnippet(wpm.UserSnippet)

	var correct, wrong int32 = 0, 0

	minLen := min(len(userRunes), len(originalRunes))

	// Count correct and wrong characters
	for i := 0; i < minLen; i++ {
		if userRunes[i] == originalRunes[i] {
			correct++
		} else {
			wrong++
		}
	}

	// Extra characters typed
	extra := len(userRunes) - len(originalRunes)
	if extra > 0 {
		wrong += int32(extra)
	}

	totalTyped := int32(len(userRunes))
	if totalTyped == 0 || wpm.DurationMS == 0 {
		return entity.WPMRes{}
	}

	durationMinutes := float32(wpm.DurationMS) / 60000.0

	rawWPM := (float32(totalTyped) / 5.0) / durationMinutes
	accuracy := float32(correct) / float32(totalTyped)
	netWPM := rawWPM * accuracy

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
