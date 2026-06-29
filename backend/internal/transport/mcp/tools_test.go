package mcpserver

import (
	"testing"
	"time"

	"koditon/internal/domain/ads"
)

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

func TestNormalizeTransactionSort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"price_asc", "price_asc"},
		{"price_desc", "price_desc"},
		{"area_asc", "area_asc"},
		{"area_desc", "area_desc"},
		{"date_asc", "date_asc"},
		{"date_desc", "date_desc"},
		{"invalid", "date_desc"},
		{"", "date_desc"},
		{"  PRICE_ASC  ", "price_asc"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := normalizeTransactionSort(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeTransactionSort(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseUUIDs(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		out, err := parseUUIDs(nil, "ids")
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 0 {
			t.Fatalf("expected empty, got %v", out)
		}
	})
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		out, err := parseUUIDs([]string{"00000000-0000-0000-0000-000000000001"}, "ids")
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1, got %d", len(out))
		}
	})
	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseUUIDs([]string{"not-a-uuid"}, "ids")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSourceEntityRouteMatchesWebRoutes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind string
		want string
	}{
		{"ad", "listing"},
		{"rental", "rental"},
		{"building", "housing-company"},
		{"announcement", "housing-company"},
		{"", "listing"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			t.Parallel()
			got := sourceEntityRoute(tt.kind)
			if got != tt.want {
				t.Fatalf("sourceEntityRoute(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestListingsAppWebPathUsesConfiguredBaseURL(t *testing.T) {
	t.Parallel()
	impl := &toolImpl{config: toolImplConfig{webBaseURL: "http://localhost:5173/"}}
	got := impl.listingWebURL("frontdoor:ad:123", "ad", "")
	want := "http://localhost:5173/listing/frontdoor:ad:123"
	if got != want {
		t.Fatalf("listingWebURL = %q, want %q", got, want)
	}
}
