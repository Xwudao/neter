package utils

import (
	"strings"
	"unicode"
)

func ExtractInitials(s string) string {
	var initials strings.Builder
	for word := range strings.SplitSeq(s, "") {
		if unicode.IsUpper([]rune(word)[0]) {
			initials.WriteString(string([]rune(word)[0]))
		}
	}
	return strings.ToLower(initials.String())
}
