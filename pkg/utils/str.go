package utils

import (
	"strings"
	"unicode"
)

func ExtractInitials(s string) string {
	var initials string
	for _, word := range strings.Split(s, "") {
		if unicode.IsUpper([]rune(word)[0]) {
			initials += string([]rune(word)[0])
		}
	}
	return strings.ToLower(initials)
}
