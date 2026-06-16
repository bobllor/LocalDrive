package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

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

// HashString joins the given strings and returns a hash
// of the string.
func HashString(s ...string) string {
	joined := strings.Join(s, "")

	hash := sha256.Sum256([]byte(joined))

	return fmt.Sprintf("%x", hash)
}
