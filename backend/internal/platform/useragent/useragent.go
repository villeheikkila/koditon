package useragent

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxLength = 512

// Normalize converts a user-agent header into bounded, storage-safe metadata.
func Normalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	lastWasSpace := false

	for _, r := range value {
		switch {
		case unicode.IsSpace(r):
			if builder.Len() == 0 || lastWasSpace {
				continue
			}
			builder.WriteByte(' ')
			lastWasSpace = true
		case unicode.IsControl(r):
			continue
		default:
			builder.WriteRune(r)
			lastWasSpace = false
		}
	}

	normalized := strings.TrimSpace(builder.String())
	if normalized == "" {
		return ""
	}
	if utf8.RuneCountInString(normalized) <= MaxLength {
		return normalized
	}
	return string([]rune(normalized)[:MaxLength])
}
