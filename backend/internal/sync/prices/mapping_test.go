package prices

import "testing"

func TestParsePlotOwned(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value string
		want  *bool
		error bool
	}{
		{name: "empty", value: "", want: nil},
		{name: "owned finnish", value: "Oma", want: boolPtr(true)},
		{name: "owned english", value: "owned", want: boolPtr(true)},
		{name: "rented finnish", value: "Vuokra", want: boolPtr(false)},
		{name: "rented compound", value: "valinnainen vuokratontti", want: boolPtr(false)},
		{name: "unknown", value: "asumisoikeus", error: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePlotOwned(tt.value)
			if tt.error {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", *got)
				}
				return
			}
			if got == nil || *got != *tt.want {
				t.Fatalf("got %v, want %v", got, *tt.want)
			}
		})
	}
}

func TestParseConditionMatchCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value string
		want  string
		error bool
	}{
		{name: "empty", value: "", want: ""},
		{name: "listing good", value: "GOOD", want: "good"},
		{name: "transaction good", value: "hyvä", want: "good"},
		{name: "listing satisfactory", value: "SATISFACTORY", want: "satisfactory"},
		{name: "transaction satisfactory abbreviation", value: "tyyd.", want: "satisfactory"},
		{name: "poor", value: "huono", want: "poor"},
		{name: "unknown enum", value: "NOT_KNOWN", want: "unknown"},
		{name: "new unexpected value", value: "excellent", error: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConditionMatchCode(tt.value)
			if tt.error {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
