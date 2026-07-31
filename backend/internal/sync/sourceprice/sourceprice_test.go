package sourceprice

import (
	"math"
	"testing"
)

func TestRoundedAmount(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		value *float64
		want  *int64
	}{
		"nil":      {value: nil, want: nil},
		"negative": {value: floatPtr(-1), want: nil},
		"nan":      {value: floatPtr(math.NaN()), want: nil},
		"rounded":  {value: floatPtr(123.6), want: intPtr(124)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := RoundedAmount(tc.value)
			if tc.want == nil && got != nil {
				t.Fatalf("got %d, want nil", *got)
			}
			if tc.want != nil && (got == nil || *got != *tc.want) {
				t.Fatalf("got %v, want %d", got, *tc.want)
			}
		})
	}
}

func TestNonNegative(t *testing.T) {
	t.Parallel()
	if NonNegative(floatPtr(math.Inf(1))) != nil {
		t.Fatal("infinite value should be rejected")
	}
	value := floatPtr(1.5)
	if got := NonNegative(value); got == nil || *got != 1.5 {
		t.Fatalf("got %v, want 1.5", got)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int64) *int64 {
	return &value
}
