package usage

import (
	"strings"
	"unicode"
)

// CalculateTextCredits computes credits for a message based on its text content,
// per the rules in the task spec.
func CalculateTextCredits(text string) float64 {
	credits := 1.0

	runes := []rune(text)
	credits += 0.05 * float64(len(runes))

	words := extractWords(text)
	for _, w := range words {
		n := len([]rune(w))
		switch {
		case n >= 1 && n <= 3:
			credits += 0.1
		case n >= 4 && n <= 7:
			credits += 0.2
		case n >= 8:
			credits += 0.3
		}
	}

	// Third vowels: positions 3, 6, 9... (1-indexed) → indices 2, 5, 8...
	for i := 2; i < len(runes); i += 3 {
		if isVowel(runes[i]) {
			credits += 0.3
		}
	}

	if len(runes) > 100 {
		credits += 5
	}

	if len(words) > 0 && allUnique(words) {
		credits -= 2
	}

	if credits < 1 {
		credits = 1
	}

	if isPalindrome(text) {
		credits *= 2
	}

	return credits
}

// extractWords parses words per the spec: any continual sequence of letters,
// apostrophes and hyphens.
func extractWords(text string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || r == '\'' || r == '-' {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

func allUnique(words []string) bool {
	seen := make(map[string]struct{}, len(words))
	for _, w := range words {
		if _, ok := seen[w]; ok {
			return false
		}
		seen[w] = struct{}{}
	}
	return true
}

func isPalindrome(text string) bool {
	var cleaned []rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned = append(cleaned, unicode.ToLower(r))
		}
	}
	if len(cleaned) == 0 {
		return false
	}
	for i, j := 0, len(cleaned)-1; i < j; i, j = i+1, j-1 {
		if cleaned[i] != cleaned[j] {
			return false
		}
	}
	return true
}
