package util

import (
	"fmt"
	"math"
	"strings"
)

func NormalizeString(v string) string {
	return strings.TrimSpace(v)
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

func ValidateEqualLengths(name string, lengths ...int) error {
	if len(lengths) == 0 {
		return fmt.Errorf("%s length check missing values", name)
	}
	expected := lengths[0]
	for _, l := range lengths[1:] {
		if l != expected {
			return fmt.Errorf("%s length mismatch (expected %d, got %d)", name, expected, l)
		}
		if l == 0 {
			return fmt.Errorf("%s are empty", name)
		}
	}
	if expected == 0 {
		return fmt.Errorf("%s are empty", name)
	}
	if expected > math.MaxInt32 {
		return fmt.Errorf("%s too large: %d", name, expected)
	}
	return nil
}
