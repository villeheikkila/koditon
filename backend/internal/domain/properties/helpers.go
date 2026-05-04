package properties

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func publicID(prefix, canonicalID string) string {
	sum := md5Hex(canonicalID)
	if len(sum) > 16 {
		sum = sum[:16]
	}
	return prefix + "_" + sum
}

func md5Hex(value string) string {
	// #nosec G401 -- public IDs are non-security identifiers that mirror Postgres md5().
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeParams(params SearchParams) SearchParams {
	out := SearchParams{Query: strings.TrimSpace(params.Query), Source: normalizeSource(params.Source), Kind: normalizeListingKind(params.Kind), City: strings.TrimSpace(params.City), Postal: strings.TrimSpace(params.Postal), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, MinPricePerM2: params.MinPricePerM2, MaxPricePerM2: params.MaxPricePerM2, Rooms: params.Rooms, Floor: params.Floor, MinBuildYear: params.MinBuildYear, MaxBuildYear: params.MaxBuildYear, Condition: strings.TrimSpace(params.Condition), EnergyClass: strings.TrimSpace(params.EnergyClass), Page: params.Page, PageSize: normalizePageSize(params.PageSize), Sort: normalizeSort(params.Sort), PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore}
	if out.Page < 1 {
		out.Page = 1
	}
	return out
}

func normalizeSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "shortcut", "frontdoor":
		return strings.ToLower(strings.TrimSpace(source))
	default:
		return "all"
	}
}

func normalizeListingKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ad", "announcement":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return "all"
	}
}

func normalizeSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "price_asc", "price_desc", "area_asc", "area_desc", "price_m2_asc", "price_m2_desc", "build_year_desc", "seen_desc":
		return strings.ToLower(strings.TrimSpace(sort))
	default:
		return "seen_desc"
	}
}

func normalizePageSize(pageSize int32) int32 {
	switch pageSize {
	case 25, 50, 100:
		return pageSize
	default:
		return 50
	}
}

func emptyToNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return cleanDisplayString(*value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := cleanDisplayString(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cleanDisplayString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func firstFloat64(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstInt64(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstInt32(values ...*int32) *int32 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func int64ToInt32(value *int64) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}

func float64ToInt64(value *float64) *int64 {
	if value == nil {
		return nil
	}
	v := int64(math.Round(*value))
	return &v
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func millisToTime(value *int64) *time.Time {
	if value == nil || *value <= 0 {
		return nil
	}
	t := time.UnixMilli(*value).UTC()
	return &t
}

func valueAtPath(value any, path ...string) string {
	current := value
	for _, part := range path {
		m, ok := mapAtPathValue(current)
		if !ok {
			return ""
		}
		next, ok := m[part]
		if !ok {
			return ""
		}
		current = next
	}
	switch v := current.(type) {
	case string:
		return cleanDisplayString(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func mapAtPathValue(value any) (map[string]any, bool) {
	switch v := value.(type) {
	case map[string]any:
		return v, true
	case rawMap:
		return map[string]any(v), true
	default:
		return nil, false
	}
}

func float64Path(value any, path ...string) *float64 {
	raw := valueAtPath(value, path...)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, ",", ".")
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func int64Path(value any, path ...string) *int64 {
	f := float64Path(value, path...)
	if f == nil {
		return nil
	}
	v := int64(math.Round(*f))
	return &v
}

func int32Path(value any, path ...string) *int32 {
	i := int64Path(value, path...)
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func boolPath(value any, path ...string) *bool {
	raw := strings.ToLower(strings.TrimSpace(valueAtPath(value, path...)))
	if raw == "" {
		return nil
	}
	switch raw {
	case "true", "1", "yes", "on", "kylla", "kyllä":
		v := true
		return &v
	case "false", "0", "no", "off", "ei":
		v := false
		return &v
	default:
		return nil
	}
}

func stringSlicePath(value any, path ...string) []string {
	current := value
	for _, part := range path {
		m, ok := mapAtPathValue(current)
		if !ok {
			return nil
		}
		current = m[part]
	}
	items, ok := current.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func sourceMetadata(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				out[key] = strings.TrimSpace(v)
			}
		case nil:
		default:
			out[key] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func computedBuildingIdentity(provider, kind, nativeID string, location Location, company string, businessID string, sourceExternalID string) BuildingIdentity {
	inputs := map[string]string{}
	if location.StreetAddress != "" {
		inputs["address"] = normalizeIdentityPart(location.StreetAddress)
	}
	if location.Postal != "" {
		inputs["postal"] = normalizeIdentityPart(location.Postal)
	}
	if location.City != "" {
		inputs["city"] = normalizeIdentityPart(location.City)
	}
	if company != "" {
		inputs["housing_company"] = normalizeIdentityPart(company)
	}
	if businessID != "" {
		inputs["business_id"] = normalizeIdentityPart(businessID)
	}
	if location.Latitude != nil && location.Longitude != nil {
		inputs["coordinates"] = fmt.Sprintf("%.4f,%.4f", *location.Latitude, *location.Longitude)
	}
	basis := inputs["business_id"]
	confidence := 0.95
	if basis == "" {
		basis = strings.Join([]string{inputs["postal"], inputs["city"], inputs["address"], inputs["housing_company"], inputs["coordinates"]}, "|")
		confidence = 0.8
	}
	if strings.TrimSpace(basis) == "" {
		basis = strings.Join([]string{provider, kind, nativeID}, "|")
		confidence = 0.4
	}
	sum := sha1.Sum([]byte(basis))
	sources := []BuildingSourceID{{Provider: provider, Kind: kind, NativeID: nativeID, ExternalID: sourceExternalID}}
	return BuildingIdentity{Key: "building:" + hex.EncodeToString(sum[:10]), Strategy: "deterministic_v1", Confidence: confidence, Inputs: inputs, Sources: sources}
}

func normalizeIdentityPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastSpace := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}
