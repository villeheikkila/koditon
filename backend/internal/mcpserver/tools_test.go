package mcpserver

import (
	"testing"
	"time"

	"koditon-go/internal/ads"
)

func TestInt32Arg(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		min     int64
		max     int64
		want    int32
		wantSet bool
		wantErr bool
	}{
		{name: "missing", args: map[string]any{}, key: "page", min: 1, max: 10, wantSet: false},
		{name: "valid", args: map[string]any{"page": float64(2)}, key: "page", min: 1, max: 10, want: 2, wantSet: true},
		{name: "fractional", args: map[string]any{"page": float64(2.5)}, key: "page", min: 1, max: 10, wantErr: true},
		{name: "out of range", args: map[string]any{"page": float64(0)}, key: "page", min: 1, max: 10, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, set, err := int32Arg(tt.args, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error mismatch: got err=%v wantErr=%v", err, tt.wantErr)
			}
			if set != tt.wantSet {
				t.Fatalf("set mismatch: got %v want %v", set, tt.wantSet)
			}
			if !tt.wantSet || tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("value mismatch: got %d want %d", got, tt.want)
			}
		})
	}
}

func TestInt64Arg(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		min     int64
		max     int64
		want    int64
		wantSet bool
		wantErr bool
	}{
		{name: "missing", args: map[string]any{}, key: "min_price", min: 0, max: 10, wantSet: false},
		{name: "valid", args: map[string]any{"min_price": float64(7)}, key: "min_price", min: 0, max: 10, want: 7, wantSet: true},
		{name: "fractional", args: map[string]any{"min_price": float64(7.1)}, key: "min_price", min: 0, max: 10, wantErr: true},
		{name: "out of range", args: map[string]any{"min_price": float64(11)}, key: "min_price", min: 0, max: 10, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, set, err := int64Arg(tt.args, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error mismatch: got err=%v wantErr=%v", err, tt.wantErr)
			}
			if set != tt.wantSet {
				t.Fatalf("set mismatch: got %v want %v", set, tt.wantSet)
			}
			if !tt.wantSet || tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("value mismatch: got %d want %d", got, tt.want)
			}
		})
	}
}

func TestBuildDetailResultOmitsRawJSONByDefault(t *testing.T) {
	t.Parallel()
	detail := ads.UnifiedEntityDetail{
		Canonical: ads.UnifiedCanonicalFields{
			CanonicalID: "frontdoor:ad:1",
			Source:      "frontdoor",
			Kind:        "ad",
			NativeID:    "1",
			LastSeenAt:  time.Now(),
		},
		Raw: ads.RawPayload{Pretty: `{"foo":"bar"}`},
	}
	result, ok := buildDetailResult(detail, false).(map[string]any)
	if !ok {
		t.Fatalf("expected map result")
	}
	if _, exists := result["raw_json"]; exists {
		t.Fatalf("expected raw_json to be omitted")
	}
	if _, exists := result["normalized"]; !exists {
		t.Fatalf("expected normalized field")
	}
}

func TestBuildDetailResultIncludesRawJSONWhenRequested(t *testing.T) {
	t.Parallel()
	detail := ads.UnifiedEntityDetail{
		Canonical: ads.UnifiedCanonicalFields{
			CanonicalID: "frontdoor:ad:1",
			Source:      "frontdoor",
			Kind:        "ad",
			NativeID:    "1",
			LastSeenAt:  time.Now(),
		},
		Raw: ads.RawPayload{Pretty: `{"foo":"bar"}`},
	}
	result, ok := buildDetailResult(detail, true).(map[string]any)
	if !ok {
		t.Fatalf("expected map result")
	}
	if _, exists := result["raw_json"]; !exists {
		t.Fatalf("expected raw_json to be present")
	}
}
