package identity

import (
	"regexp"
	"strings"
)

var (
	prefixPattern = regexp.MustCompile(`^(pt\.?|cv\.?|ud\.?|firma|koperasi|yayasan)\s+`)
	suffixPattern = regexp.MustCompile(`\s+(tbk\.?|persero)$`)
	punctPattern  = regexp.MustCompile(`[.,\-'()"]`)
	spacePattern  = regexp.MustCompile(`\s+`)
)

func NormalizeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	s = prefixPattern.ReplaceAllString(s, "")
	s = suffixPattern.ReplaceAllString(s, "")
	s = punctPattern.ReplaceAllString(s, " ")
	s = spacePattern.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}
