package properties

import "testing"

func TestComputedBuildingIdentityDeterministic(t *testing.T) {
	t.Parallel()
	lat := 60.1699
	lon := 24.9384
	location := Location{StreetAddress: "Testikatu 1 A", City: "Helsinki", Postal: "00100", Latitude: &lat, Longitude: &lon}
	first := computedBuildingIdentity("shortcut", "ad", "1", location, "As Oy Testi", "1234567-8", "987")
	second := computedBuildingIdentity("frontdoor", "building", "2", location, "As Oy Testi", "1234567-8", "654")
	if first.Key != second.Key {
		t.Fatalf("expected same business id to produce same key, got %q and %q", first.Key, second.Key)
	}
	if first.Confidence < 0.9 {
		t.Fatalf("expected high confidence for business id match, got %v", first.Confidence)
	}
}

func TestSourceMetadataDropsEmptyValues(t *testing.T) {
	t.Parallel()
	metadata := sourceMetadata(map[string]any{"empty": " ", "kind": "listing", "count": 2, "nil": nil})
	if _, ok := metadata["empty"]; ok {
		t.Fatal("expected empty string metadata to be omitted")
	}
	if _, ok := metadata["nil"]; ok {
		t.Fatal("expected nil metadata to be omitted")
	}
	if metadata["kind"] != "listing" || metadata["count"] != 2 {
		t.Fatalf("expected non-empty metadata to remain, got %#v", metadata)
	}
}
