package utils

import (
	"osdtyp/app/entity"
	"strings"
)

// cleanSnippet drops whitespace because the game transports only printable
// keystrokes; it keeps the comparison independent of indentation.
func cleanSnippet(s string) []rune {
	s = strings.ReplaceAll(s, "\r", "")
	var res []rune
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		res = append(res, r)
	}
	return res
}

func Calculate_WPM(wpm entity.WPM) entity.WPMRes {
	originalRunes := cleanSnippet(wpm.OriginalSnippet)
	userRunes := cleanSnippet(wpm.UserSnippet)

	var correct, wrong int
	minLen := min(len(userRunes), len(originalRunes))
	for i := 0; i < minLen; i++ {
		if userRunes[i] == originalRunes[i] {
			correct++
		} else {
			wrong++
		}
	}
	if extra := len(userRunes) - len(originalRunes); extra > 0 {
		wrong += extra
	}

	totalTyped := len(userRunes)
	if totalTyped == 0 || wpm.DurationMS <= 0 {
		return entity.WPMRes{}
	}

	durationMinutes := float64(wpm.DurationMS) / 60000.0
	rawWPM := (float64(totalTyped) / 5.0) / durationMinutes
	accuracy := float64(correct) / float64(totalTyped)
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
