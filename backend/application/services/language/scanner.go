package langauge

import (
	"osdtype/application/entity"
	"unicode"
)

// Tokenizer takes the snippet and breaks it down into non-spaced strings
func Tokenize(snippet string) []string {
	//Iterating over the character in this string
	current := ""
	var tokens []string //Well not exactly tokens...but "what can be seperated by spaces pretty much
	scanning := entity.FREE
	for _, char := range snippet {
		switch {
		case char == '\n':
			if scanning != entity.FREE {
				if current != "" {
					tokens = append(tokens, current)
					current = ""
				}
				scanning = entity.FREE
			}
			tokens = append(tokens, "\n")
		case unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '.':
			if scanning == entity.OPERATOR {
				if current != "" {
					tokens = append(tokens, current)
					current = ""
				}

			}
			scanning = entity.ALPHANUM
			current += string(char)
		case char == ' ' || char == '\t':
			if scanning == entity.FREE || current == "" {
				//Just in case they are not matched...we can still do with missed tokens but not longer glitches
				scanning = entity.FREE
				current = ""
			} else {
				tokens = append(tokens, current)
				scanning = entity.FREE
				current = ""
			}
			tokens = append(tokens, " ")
		default:
			//For Special Characters
			if scanning == entity.ALPHANUM {
				if current != "" {
					tokens = append(tokens, current)
					current = ""
				}
				scanning = entity.OPERATOR
			}
			current += string(char)
		}
	}
	//The last string
	if current != "" {
		tokens = append(tokens, current)
	}
	return tokens
}

func PurgeExcess(tokens []string) []string {
	// Helper function: is token a space or newline?
	isWhitespace := func(token string) bool {
		return token == " " || token == "\n"
	}

	// 1. Remove leading whitespace tokens
	start := 0
	for start < len(tokens) && isWhitespace(tokens[start]) {
		start++
	}

	// 2. Remove trailing whitespace tokens
	end := len(tokens) - 1
	for end >= start && isWhitespace(tokens[end]) {
		end--
	}

	if start > end {
		// All tokens are whitespace => return empty slice
		return []string{}
	}

	// 3. Iterate through tokens from start to end and collapse multiples
	var result []string
	prevWhitespace := false
	whitespaceType := "" // keep track of whether last whitespace was " " or "\n"

	for i := start; i <= end; i++ {
		token := tokens[i]
		if !isWhitespace(token) {
			// Normal token, just append and reset whitespace state
			result = append(result, token)
			prevWhitespace = false
			whitespaceType = ""
		} else {
			// Whitespace token
			if !prevWhitespace {
				// First whitespace after token, just add it temporarily
				whitespaceType = token
				prevWhitespace = true
			} else {
				// Already previous whitespace exists, determine if mixed
				if whitespaceType != token {
					// Mixed whitespace encountered (space then newline or vice versa)
					// Upgrade whitespaceType to newline
					whitespaceType = "\n"
				}
				// Else same whitespace type – keep it as is
			}
		}

		// If next token is not whitespace or end reached, append the whitespace token once
		nextIsWhitespace := false
		if i+1 <= end {
			nextIsWhitespace = isWhitespace(tokens[i+1])
		}
		if prevWhitespace && (!nextIsWhitespace || i == end) {
			// Append the collapsed whitespace
			result = append(result, whitespaceType)
			prevWhitespace = false
			whitespaceType = ""
		}
	}

	return result
}
