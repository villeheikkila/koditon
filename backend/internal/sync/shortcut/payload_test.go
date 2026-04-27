package shortcut

import (
	"errors"
	"testing"
)

func TestValidateShortcutAdPayloadV1AcceptsNumericAndStringValues(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": "123",
		"cardType": "100",
		"address": {
			"street": {"name": "Testikatu 1"},
			"city": {"name": "Helsinki"},
			"zipCode": {"value": "00100"}
		},
		"priceData": {
			"priceSell": "250000",
			"maintenanceCharge": "300.50"
		},
		"adData": {
			"size": "42,5",
			"rooms": "2"
		},
		"buildingData": {
			"buildingId": "987",
			"year": "1970"
		}
	}`)
	payload, err := ValidateShortcutAdPayloadV1(raw, 123)
	if err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}
	if payload.SchemaVersion != ShortcutAdPayloadSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", ShortcutAdPayloadSchemaVersion, payload.SchemaVersion)
	}
	if payload.AdType != AdTypeListing {
		t.Fatalf("expected listing type, got %s", payload.AdType)
	}
	if payload.BuildingExternalID == nil || *payload.BuildingExternalID != 987 {
		t.Fatalf("expected building external id 987, got %v", payload.BuildingExternalID)
	}
}

func TestValidateShortcutAdPayloadV1UsesBuildingFallback(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": 124,
		"cardType": 101,
		"address": {
			"street": "Testikatu 2",
			"city": "Espoo",
			"zipCode": "02100"
		},
		"priceData": {"price": 1200},
		"adData": {"size": 35},
		"buildingData": {},
		"building": {"buildingId": 654}
	}`)
	payload, err := ValidateShortcutAdPayloadV1(raw, 124)
	if err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}
	if payload.AdType != AdTypeRental {
		t.Fatalf("expected rental type, got %s", payload.AdType)
	}
	if payload.BuildingExternalID == nil || *payload.BuildingExternalID != 654 {
		t.Fatalf("expected fallback building external id 654, got %v", payload.BuildingExternalID)
	}
}

func TestValidateShortcutAdPayloadV1AcceptsBuildingObjectWithoutBuildingData(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": 125,
		"cardType": 100,
		"address": {
			"street": "Testikatu 3",
			"city": "Tampere",
			"zipCode": "33100"
		},
		"priceData": {"priceSell": 150000},
		"adData": {"size": 50},
		"building": {"buildingId": 321}
	}`)
	payload, err := ValidateShortcutAdPayloadV1(raw, 125)
	if err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}
	if payload.BuildingExternalID == nil || *payload.BuildingExternalID != 321 {
		t.Fatalf("expected building external id 321, got %v", payload.BuildingExternalID)
	}
}

func TestValidateShortcutAdPayloadV1AcceptsTopLevelBuildingIDAndRentPrice(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": 126,
		"cardType": 101,
		"buildingId": "765",
		"address": {
			"street": "Testikatu 4",
			"city": "Oulu",
			"zipCode": "90100"
		},
		"priceData": {"rentPerMonth": "990"},
		"adData": {
			"size": "38",
			"floor": "2",
			"totalFloors": "6",
			"rooms": "2",
			"elevator": "true",
			"sauna": "0"
		}
	}`)
	payload, err := ValidateShortcutAdPayloadV1(raw, 126)
	if err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}
	if payload.BuildingExternalID == nil || *payload.BuildingExternalID != 765 {
		t.Fatalf("expected building external id 765, got %v", payload.BuildingExternalID)
	}
}

func TestValidateShortcutAdPayloadV1AcceptsSizeTotalFallback(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": 128,
		"cardType": 100,
		"buildingId": 765,
		"address": {
			"street": "Testikatu 6",
			"city": "Oulu",
			"zipCode": "90100"
		},
		"priceData": {"priceSell": 190000},
		"adData": {
			"size": null,
			"sizeTotal": "187"
		}
	}`)
	if _, err := ValidateShortcutAdPayloadV1(raw, 128); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}
}

func TestValidateShortcutAdPayloadV1RejectsMalformedOptionalDetails(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"cardId": 127,
		"cardType": 100,
		"buildingId": 765,
		"address": {
			"street": "Testikatu 5",
			"city": "Oulu",
			"zipCode": "90100"
		},
		"priceData": {"priceSell": 190000},
		"adData": {
			"size": "38",
			"floor": "not-a-floor"
		}
	}`)
	_, err := ValidateShortcutAdPayloadV1(raw, 127)
	if !errors.Is(err, ErrInvalidShortcutAdPayload) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestValidateShortcutAdPayloadV1RejectsInvalidPayloads(t *testing.T) {
	t.Parallel()
	cases := map[string][]byte{
		"array root":        []byte(`[]`),
		"unsupported type":  []byte(`{"cardId":123,"cardType":102,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"adData":{"size":1},"buildingData":{}}`),
		"missing id":        []byte(`{"cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"adData":{"size":1},"buildingData":{}}`),
		"id mismatch":       []byte(`{"cardId":999,"cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"adData":{"size":1},"buildingData":{}}`),
		"missing address":   []byte(`{"cardId":123,"cardType":100,"priceData":{"price":1},"adData":{"size":1},"buildingData":{}}`),
		"missing price":     []byte(`{"cardId":123,"cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{},"adData":{"size":1},"buildingData":{}}`),
		"missing ad data":   []byte(`{"cardId":123,"cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"buildingData":{}}`),
		"missing building":  []byte(`{"cardId":123,"cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"adData":{"size":1}}`),
		"malformed numeric": []byte(`{"cardId":"abc","cardType":100,"address":{"street":"A","city":"B","zipCode":"00100"},"priceData":{"price":1},"adData":{"size":1},"buildingData":{}}`),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateShortcutAdPayloadV1(raw, 123)
			if !errors.Is(err, ErrInvalidShortcutAdPayload) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}
