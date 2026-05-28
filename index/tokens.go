package index

import (
	"strings"
	"unicode"
)

// Tokenize lowercases text and splits identifiers into subwords:
// camelCase, snake_case, and digit boundaries are all broken apart.
// Both the whole identifier and its parts are emitted so that lexical
// matching works on full names and on individual subwords.
func Tokenize(text string) []string {
	var out []string
	var word []rune
	flush := func() {
		if len(word) == 0 {
			return
		}
		whole := string(word)
		out = append(out, strings.ToLower(whole))
		for _, p := range splitIdentifier(word) {
			if p != whole {
				out = append(out, strings.ToLower(p))
			}
		}
		word = word[:0]
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func splitIdentifier(word []rune) []string {
	var parts []string
	var cur []rune
	emit := func() {
		if len(cur) > 0 {
			parts = append(parts, string(cur))
			cur = cur[:0]
		}
	}
	for i, r := range word {
		prevLower := i > 0 && unicode.IsLower(word[i-1])
		prevDigit := i > 0 && unicode.IsDigit(word[i-1])
		isUpper := unicode.IsUpper(r)
		isDigit := unicode.IsDigit(r)
		// boundary on lower->Upper (camelCase) or letter<->digit transitions
		if (isUpper && prevLower) || (isDigit && !prevDigit && i > 0) || (!isDigit && prevDigit) {
			emit()
		}
		cur = append(cur, r)
	}
	emit()
	return parts
}
