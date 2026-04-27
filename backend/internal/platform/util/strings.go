package util

import (
	"strings"
	"unicode"
)

func TrimUnicodeSpace(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

func NormalizeString(v string) string {
	return TrimUnicodeSpace(v)
}

func UniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		v = NormalizeString(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
