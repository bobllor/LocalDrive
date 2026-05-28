package utils

import "unicode"

// ToUpperFirstChar uppercases the first letter of the string.
func ToUpperFirstChar(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	out := string(runes)

	return out
}
