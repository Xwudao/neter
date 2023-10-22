package conv

import (
	"regexp"
	"strings"

	"github.com/samber/lo"
)

type CamelCaseConverter interface {
	ToCamelCase(s string) string
}

type DefaultCamelCaseConverter struct {
	allCaps   map[string]bool
	converted map[string]string
}

func NewDefaultCamelCaseConverter(allCaps []string) DefaultCamelCaseConverter {
	return DefaultCamelCaseConverter{
		allCaps: lo.SliceToMap(allCaps, func(s string) (string, bool) {
			return s, true
		}),
		converted: make(map[string]string),
	}
}

var (
	splitRE       = regexp.MustCompile(`_+|-+|\s+`)
	allCapsRE     = regexp.MustCompile(`^[A-Z]{2,}\d*$`)
	capitalizedRE = regexp.MustCompile(`[A-Z][a-z]+\d*`)
	ToCamelCase   = NewDefaultCamelCaseConverter(nil).ToCamelCase
)

func (c DefaultCamelCaseConverter) ToCamelCase(s string) string {
	if r, ok := c.converted[s]; ok {
		return r
	}

	words := splitRE.Split(s, -1)

	for i, word := range words {
		if len(word) != 0 {
			if allCapsRE.MatchString(word) {
				words[i] = word[:1] + strings.ToLower(word[1:])
			} else {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
	}

	r := capitalizedRE.ReplaceAllStringFunc(strings.Join(words, ""), func(s string) string {
		if word := strings.ToUpper(s); c.allCaps[word] {
			return word
		}
		return s
	})

	c.converted[s] = r
	return r
}

var (
	followNonCapRE = regexp.MustCompile(`([a-z\d])([A-Z])`)
	followCapRE    = regexp.MustCompile(`([A-Z])([A-Z][a-z]+)`)
)

func ToSnakeCase(s string) string {
	s = followNonCapRE.ReplaceAllString(s, "${1}_$2")
	s = followCapRE.ReplaceAllString(s, "${1}_$2")
	return strings.ToLower(s)
}
