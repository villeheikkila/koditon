package client

import (
	"encoding/json"
	"testing"
)

func TestSearchResultDecodesNumericAndStringScalars(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cards": [
			{
				"id": 1,
				"url": "/a",
				"rooms": "2",
				"price": 1000,
				"size": 42.5,
				"coordinates": {"latitude": 60.1, "longitude": "24.9"},
				"buildingData": {"year": "1970", "buildingType": 1}
			}
		],
		"found": 1,
		"start": 0
	}`)
	var result SearchResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if len(result.Cards) != 1 {
		t.Fatalf("expected one card, got %d", len(result.Cards))
	}
	rooms, ok := result.Cards[0].Rooms.Int64()
	if !ok || rooms != 2 {
		t.Fatalf("expected rooms 2, got %d valid=%t", rooms, ok)
	}
	lat, ok := result.Cards[0].Coordinates.Latitude.Float64()
	if !ok || lat != 60.1 {
		t.Fatalf("expected latitude 60.1, got %v valid=%t", lat, ok)
	}
	year, ok := result.Cards[0].BuildingData.Year.Int64()
	if !ok || year != 1970 {
		t.Fatalf("expected year 1970, got %d valid=%t", year, ok)
	}
}

func TestBuildingResponseDecodesNumericVrkID(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"buildingId":1,"vrkId":12345}`)
	var result BuildingResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode building response: %v", err)
	}
	vrkID, ok := result.VrkID.Int64()
	if !ok || vrkID != 12345 {
		t.Fatalf("expected vrk id 12345, got %d valid=%t", vrkID, ok)
	}
}
