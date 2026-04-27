package shortcut

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const ShortcutAdPayloadSchemaVersion int16 = 1

var ErrInvalidShortcutAdPayload = errors.New("invalid shortcut ad payload")

type ShortcutAdPayloadV1 struct {
	AdID               int64
	AdType             AdType
	BuildingExternalID *int64
	Raw                json.RawMessage
	SchemaVersion      int16
}

func ValidateShortcutAdPayloadV1(raw json.RawMessage, expectedAdID int64) (*ShortcutAdPayloadV1, error) {
	root, err := decodeJSONObject(raw)
	if err != nil {
		return nil, err
	}
	adID, err := requireFirstInt(root, "cardId", "id", "adId")
	if err != nil {
		return nil, err
	}
	if expectedAdID > 0 && adID != expectedAdID {
		return nil, payloadError("ad id mismatch")
	}
	cardType, err := requireInt(root, "cardType")
	if err != nil {
		return nil, err
	}
	adType, err := adTypeFromCardType(cardType)
	if err != nil {
		return nil, err
	}
	if err := validateShortcutAdAddress(root); err != nil {
		return nil, err
	}
	if err := validateShortcutAdPriceData(root); err != nil {
		return nil, err
	}
	if err := validateShortcutAdData(root); err != nil {
		return nil, err
	}
	if err := validateShortcutAdBuilding(root); err != nil {
		return nil, err
	}
	buildingID := optionalNestedInt(root, []string{"buildingData", "buildingId"})
	if buildingID == nil {
		buildingID = optionalNestedInt(root, []string{"building", "buildingId"})
	}
	return &ShortcutAdPayloadV1{AdID: adID, AdType: adType, BuildingExternalID: buildingID, Raw: raw, SchemaVersion: ShortcutAdPayloadSchemaVersion}, nil
}

func adTypeFromCardType(cardType int64) (AdType, error) {
	switch cardType {
	case 100:
		return AdTypeListing, nil
	case 101:
		return AdTypeRental, nil
	default:
		return "", payloadError(fmt.Sprintf("unsupported card type %d", cardType))
	}
}

func validateShortcutAdAddress(root map[string]any) error {
	address, err := requireObject(root, "address")
	if err != nil {
		return err
	}
	if !hasNestedString(address, []string{"street", "name"}) && !hasString(address, "street") && !hasString(address, "formattedAddress") {
		return payloadError("address missing street")
	}
	if !hasNestedString(address, []string{"city", "name"}) && !hasString(address, "city") {
		return payloadError("address missing city")
	}
	if !hasNestedString(address, []string{"zipCode", "value"}) && !hasNestedString(address, []string{"zipCode", "name"}) && !hasString(address, "zipCode") {
		return payloadError("address missing postal code")
	}
	return nil
}

func validateShortcutAdPriceData(root map[string]any) error {
	priceData, err := requireObject(root, "priceData")
	if err != nil {
		return err
	}
	if !hasNumberLike(priceData, "priceSell") && !hasNumberLike(priceData, "price") && !hasNumberLike(priceData, "priceDebtFree") {
		return payloadError("price data missing price")
	}
	return nil
}

func validateShortcutAdData(root map[string]any) error {
	adData, err := requireObject(root, "adData")
	if err != nil {
		return err
	}
	if !hasNumberLike(adData, "size") {
		return payloadError("ad data missing size")
	}
	return nil
}

func validateShortcutAdBuilding(root map[string]any) error {
	if _, err := requireObject(root, "buildingData"); err == nil {
		return nil
	}
	if _, err := requireObject(root, "building"); err == nil {
		return nil
	}
	return payloadError("missing building data")
}

func decodeJSONObject(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, payloadError("empty payload")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, payloadError(fmt.Sprintf("decode payload: %v", err))
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, payloadError("payload root must be object")
	}
	return root, nil
}

func requireObject(root map[string]any, key string) (map[string]any, error) {
	value, ok := root[key]
	if !ok {
		return nil, payloadError("missing " + key)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, payloadError(key + " must be object")
	}
	return obj, nil
}

func requireFirstInt(root map[string]any, keys ...string) (int64, error) {
	for _, key := range keys {
		value, ok := root[key]
		if !ok {
			continue
		}
		if parsed, ok := parseIntValue(value); ok {
			return parsed, nil
		}
	}
	return 0, payloadError("missing numeric id")
}

func requireInt(root map[string]any, key string) (int64, error) {
	value, ok := root[key]
	if !ok {
		return 0, payloadError("missing " + key)
	}
	parsed, ok := parseIntValue(value)
	if !ok {
		return 0, payloadError(key + " must be numeric")
	}
	return parsed, nil
}

func optionalNestedInt(root map[string]any, path []string) *int64 {
	value, ok := nestedValue(root, path)
	if !ok {
		return nil
	}
	parsed, ok := parseIntValue(value)
	if !ok {
		return nil
	}
	return &parsed
}

func nestedValue(root map[string]any, path []string) (any, bool) {
	current := any(root)
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func hasNestedString(root map[string]any, path []string) bool {
	value, ok := nestedValue(root, path)
	if !ok {
		return false
	}
	return stringValue(value) != ""
}

func hasString(root map[string]any, key string) bool {
	value, ok := root[key]
	if !ok {
		return false
	}
	return stringValue(value) != ""
}

func hasNumberLike(root map[string]any, key string) bool {
	value, ok := root[key]
	if !ok {
		return false
	}
	_, ok = parseFloatValue(value)
	return ok
}

func parseIntValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed, true
		}
		parsed, err := strconv.ParseFloat(typed.String(), 64)
		if err != nil {
			return 0, false
		}
		return int64(parsed), true
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(cleanNumberString(typed), 10, 64)
		if err == nil {
			return parsed, true
		}
		asFloat, err := strconv.ParseFloat(cleanNumberString(typed), 64)
		if err != nil {
			return 0, false
		}
		return int64(asFloat), true
	default:
		return 0, false
	}
}

func parseFloatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(cleanNumberString(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func cleanNumberString(value string) string {
	cleaned := strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return cleaned
}

func payloadError(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidShortcutAdPayload, message)
}
