package useragent

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "preserves normal app user agent",
			input: "Koditon/1.0 (iPhone; iOS 26.0)",
			want:  "Koditon/1.0 (iPhone; iOS 26.0)",
		},
		{
			name:  "trims whitespace",
			input: "  Koditon/1.0 (iPhone; iOS 26.0)  ",
			want:  "Koditon/1.0 (iPhone; iOS 26.0)",
		},
		{
			name:  "collapses whitespace and strips controls",
			input: "Koditon/1.0\t(\niPhone;\r iOS 26.0)\u0000",
			want:  "Koditon/1.0 ( iPhone; iOS 26.0)",
		},
		{
			name:  "blank becomes empty",
			input: " \n\t ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(tt.input); got != tt.want {
				t.Fatalf("Normalize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalize_TruncatesLongValues(t *testing.T) {
	t.Parallel()

	input := strings.Repeat("a", MaxLength+32)
	got := Normalize(input)
	if len([]rune(got)) != MaxLength {
		t.Fatalf("expected length %d, got %d", MaxLength, len([]rune(got)))
	}
}
