package properties

import (
	"encoding/json"
	"testing"
)

func TestSaleSummaryUsesNormalizedSections(t *testing.T) {
	area := 42.5
	price := int64(210000)
	row := listingSearchRow{Source: "frontdoor", Kind: "ad", NativeID: "123", CanonicalID: "frontdoor:ad:123", PublicID: "08daecb8-dabb-44ef-8566-92ca0ca889a4", URL: "https://example.test/a", Headline: "Test listing", Address: "Testikatu 1", City: "Helsinki", Postal: "00100", Price: &price, Area: &area, RoomLayout: "2h+k"}
	body, err := json.Marshal(row.toSaleSummary())
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	for _, legacy := range []string{"location", "property", "sale_terms", "building_identity", "main_image"} {
		if _, ok := decoded[legacy]; ok {
			t.Fatalf("legacy key %q present in summary: %s", legacy, body)
		}
	}
	for _, section := range []string{"unit", "building", "commercial"} {
		if _, ok := decoded[section]; !ok {
			t.Fatalf("section %q missing from summary: %s", section, body)
		}
	}
	if decoded["id"] != row.PublicID {
		t.Fatalf("expected canonical offering UUID as id: %s", body)
	}
}
