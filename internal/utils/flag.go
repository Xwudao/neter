package utils

import (
	"strings"
	"unicode"
)

// ParseArgsString parse args string to exec.Command to use
func ParseArgsString(args string) []string {
	quote := '\000'
	start := 0
	lastSpace := true
	backslash := false
	var cc []string
	for i, c := range args {
		if quote == '\000' && unicode.IsSpace(c) {
			if !lastSpace {
				cc = append(cc, args[start:i])
				lastSpace = true
			}
		} else {
			if lastSpace {
				start = i
				lastSpace = false
			}
			if quote == '\000' && !backslash && (c == '"' || c == '\'') {
				quote = c
				backslash = false
			} else if !backslash && quote == c {
				quote = '\000'
			} else if (quote == '\000' || quote == '"') && !backslash && c == '\\' {
				backslash = true
			} else {
				backslash = false
			}
		}
	}
	for i := 0; i < len(cc); i++ {
		if strings.HasPrefix(cc[i], `"`) {
			cc[i] = strings.Trim(cc[i], `"`)
		}
		if strings.HasPrefix(cc[i], `'`) {
			cc[i] = strings.Trim(cc[i], `'`)
		}
	}
	return cc
}
