package mcpserver

import (
	"strings"
	"testing"
	"time"

	"koditon/internal/domain/ads"
)

func TestBuildPropertyDetailResultUsesTypedJSONFields(t *testing.T) {
	t.Parallel()
	price := int64(250000)
	area := 52.5
	lastSeenAt := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	impl := &toolImpl{config: toolImplConfig{webBaseURL: "https://app.example"}}
	result := impl.buildPropertyDetailResult(ads.UnifiedEntityDetail{
		Canonical:      ads.UnifiedCanonicalFields{CanonicalID: "frontdoor:ad:123", Source: "frontdoor", Kind: "ad", NativeID: "123", Headline: "Askvägen 4", Address: "Askvägen 4", City: "Maarianhamina", Postal: "22100", Price: &price, Area: &area, RoomLayout: "2h+k", URL: "https://source.example/ad/123", ExternalURLAvailable: true, LastSeenAt: lastSeenAt},
		SourceSpecific: []ads.DetailField{{Label: "Condition", Value: "Good"}},
		Related:        []ads.DetailField{{Label: "Housing company", Value: "Asunto Oy Example"}},
		Normalized:     ads.NormalizedDetailFields{CanonicalID: "frontdoor:ad:123", Source: "frontdoor", Kind: "ad", StreetAddress: "Askvägen 4", City: "Maarianhamina", Postal: "22100", AskingPrice: &price, AreaM2: &area, RoomLayout: "2h+k"},
		Raw:            ads.RawPayload{Pretty: `{"source":"frontdoor"}`},
	}, true, true)
	if result.Canonical.CanonicalID != "frontdoor:ad:123" {
		t.Fatalf("canonical id = %q", result.Canonical.CanonicalID)
	}
	if result.CanonicalExtra == nil {
		t.Fatalf("canonical_extra should be an empty typed slice, not nil")
	}
	if len(result.SourceSpecific) != 1 || result.SourceSpecific[0].Label != "Condition" {
		t.Fatalf("source_specific not mapped: %#v", result.SourceSpecific)
	}
	if result.Raw == nil || result.RawJSON == nil {
		t.Fatalf("expected raw payloads when requested")
	}
	if !strings.Contains(result.Markdown, "Open in Koditon: https://app.example/listing/frontdoor:ad:123") {
		t.Fatalf("markdown did not include web link: %s", result.Markdown)
	}
}

func TestPropertyFacetsCountsTypedRows(t *testing.T) {
	t.Parallel()
	rows := []PropertySummary{
		{Source: "frontdoor", Kind: "ad", City: "Helsinki", Transactions: []ComparableSale{{ID: "tx1"}}},
		{Source: "frontdoor", Kind: "ad", City: "Helsinki"},
		{Source: "shortcut", Kind: "building", City: "Espoo", Insights: propertyInsightSummary{Count: 2}},
	}
	facets := propertyFacets(rows)
	if facets.Sources["frontdoor"] != 2 || facets.Sources["shortcut"] != 1 {
		t.Fatalf("unexpected source facets: %#v", facets.Sources)
	}
	if facets.WithSales != 1 || facets.WithInsights != 1 {
		t.Fatalf("unexpected counters: sales=%d insights=%d", facets.WithSales, facets.WithInsights)
	}
}
