package usage

import (
	"math"
	"testing"
)

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestExtractWords(t *testing.T) {
	got := extractWords("Hello, it's mother-in-law!")
	want := []string{"Hello", "it's", "mother-in-law"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("word %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestIsPalindrome(t *testing.T) {
	cases := map[string]bool{
		"A man, a plan, a canal: Panama": true,
		"racecar":                        true,
		"hello":                          false,
		"":                               false,
		"No 'x' in Nixon":                true,
	}
	for in, want := range cases {
		if got := isPalindrome(in); got != want {
			t.Errorf("isPalindrome(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCalculateTextCredits_ShortNonPalindrome(t *testing.T) {
	// "Hi": base 1 + 0.05*2 + 0.1 (1-3 char word) = 1.2; unique → -2 → clamp 1.
	if got := CalculateTextCredits("Hi"); !approxEq(got, 1.0) {
		t.Errorf("got %v, want 1.0", got)
	}
}

func TestCalculateTextCredits_Palindrome(t *testing.T) {
	// "racecar" → clamps to 1, then doubled by palindrome rule → 2.
	if got := CalculateTextCredits("racecar"); !approxEq(got, 2.0) {
		t.Errorf("got %v, want 2.0", got)
	}
}
