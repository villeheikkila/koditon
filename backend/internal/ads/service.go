package ads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/db"
)

type SearchParams struct {
	Query           string
	Source          string
	Kind            string
	ListingType     string
	MinPrice        *int64
	MaxPrice        *int64
	MinArea         *float64
	MaxArea         *float64
	City            string
	Postal          string
	Page            int32
	PageSize        int32
	Sort            string
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
}

type UnifiedEntityRow struct {
	CanonicalID string
	Source      string
	Kind        string
	NativeID    string
	Headline    string
	Address     string
	City        string
	Postal      string
	Price       *int64
	Area        *float64
	RoomLayout  string
	URL         string
	LastSeenAt  time.Time
}

type ReportPage struct {
	Rows     []UnifiedEntityRow
	Total    int64
	Page     int32
	PageSize int32
}

type DetailField struct {
	Label string
	Value string
}

type RawPayload struct {
	Pretty        string
	OriginalBytes int
}

type UnifiedCanonicalFields struct {
	CanonicalID string
	Source      string
	Kind        string
	NativeID    string
	Headline    string
	Address     string
	City        string
	Postal      string
	Price       *int64
	Area        *float64
	RoomLayout  string
	URL         string
	LastSeenAt  time.Time
}

type UnifiedEntityDetail struct {
	Canonical      UnifiedCanonicalFields
	CanonicalExtra []DetailField
	SourceSpecific []DetailField
	Related        []DetailField
	Raw            RawPayload
}

type Service struct {
	queries *db.Queries
}

func NewService(dbtx db.DBTX) *Service {
	return &Service{queries: db.New(dbtx)}
}

func (s *Service) Search(ctx context.Context, params SearchParams) (ReportPage, error) {
	normalized := normalizeSearchParams(params)
	offset := (normalized.Page - 1) * normalized.PageSize
	source := stringPtr(normalized.Source)
	kind := stringPtr(normalized.Kind)
	sort := stringPtr(normalized.Sort)
	listingTypeFilter := emptyToNil(normalized.ListingType)
	if normalized.ListingType == "all" {
		listingTypeFilter = nil
	}
	publishedAfter := normalized.PublishedAfter
	publishedBefore := normalized.PublishedBefore
	count, err := s.queries.CountUnifiedEntities(ctx, &db.CountUnifiedEntitiesParams{
		SourceFilter:      source,
		KindFilter:        kind,
		QueryText:         emptyToNil(normalized.Query),
		CityFilter:        emptyToNil(normalized.City),
		PostalFilter:      emptyToNil(normalized.Postal),
		MinPrice:          normalized.MinPrice,
		MaxPrice:          normalized.MaxPrice,
		MinArea:           normalized.MinArea,
		MaxArea:           normalized.MaxArea,
		ListingTypeFilter: listingTypeFilter,
		PublishedAfter:    publishedAfter,
		PublishedBefore:   publishedBefore,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("count unified entities: %w", err)
	}
	rows, err := s.queries.SearchUnifiedEntities(ctx, &db.SearchUnifiedEntitiesParams{
		SortMode:          sort,
		OffsetCount:       offset,
		LimitCount:        normalized.PageSize,
		SourceFilter:      source,
		KindFilter:        kind,
		QueryText:         emptyToNil(normalized.Query),
		CityFilter:        emptyToNil(normalized.City),
		PostalFilter:      emptyToNil(normalized.Postal),
		MinPrice:          normalized.MinPrice,
		MaxPrice:          normalized.MaxPrice,
		MinArea:           normalized.MinArea,
		MaxArea:           normalized.MaxArea,
		ListingTypeFilter: listingTypeFilter,
		PublishedAfter:    publishedAfter,
		PublishedBefore:   publishedBefore,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("search unified entities: %w", err)
	}
	mapped := make([]UnifiedEntityRow, 0, len(rows))
	for _, row := range rows {
		mapped = append(mapped, UnifiedEntityRow{
			CanonicalID: valueOrEmpty(row.CanonicalID),
			Source:      row.Source,
			Kind:        row.Kind,
			NativeID:    row.NativeID,
			Headline:    valueOrEmpty(row.Headline),
			Address:     valueOrEmpty(row.Address),
			City:        valueOrEmpty(row.City),
			Postal:      valueOrEmpty(row.Postal),
			Price:       row.Price,
			Area:        row.Area,
			RoomLayout:  valueOrEmpty(row.RoomLayout),
			URL:         strings.TrimSpace(row.Url),
			LastSeenAt:  row.LastSeenAt,
		})
	}
	return ReportPage{Rows: mapped, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) DetailByCanonicalID(ctx context.Context, canonicalID string) (UnifiedEntityDetail, error) {
	source, kind, nativeID, err := ParseCanonicalID(canonicalID)
	if err != nil {
		return UnifiedEntityDetail{}, err
	}
	switch source {
	case "shortcut":
		switch kind {
		case "ad":
			adID, err := strconv.ParseInt(nativeID, 10, 64)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse shortcut ad id: %w", err)
			}
			row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("shortcut ad not found")
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), strconv.FormatInt(row.ShortcutAdID, 10)), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Price: row.AdPrice, Area: row.AdArea, RoomLayout: valueOrEmpty(row.AdRoomLayout), URL: strings.TrimSpace(row.ShortcutAdUrl), LastSeenAt: row.ShortcutAdLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "Ad Type", Value: row.ShortcutAdType}, {Label: "Building ID", Value: pgUUIDToString(row.ShortcutBuildingID)}, {Label: "Building External ID", Value: formatInt64Ptr(row.ShortcutBuildingExternalID)}, {Label: "Building Address", Value: valueOrEmpty(row.ShortcutBuildingAddress)}, {Label: "Housing Company", Value: valueOrEmpty(row.ShortcutBuildingHousingCompany)}, {Label: "Building URL", Value: valueOrEmpty(row.ShortcutBuildingUrl)}}
			detail.Related = []DetailField{{Label: "Building Listings", Value: strconv.FormatInt(row.BuildingListingCount, 10)}, {Label: "Building Rentals", Value: strconv.FormatInt(row.BuildingRentalCount, 10)}}
			detail.Raw = buildRawPayload(row.ShortcutAdData)
			detail = promoteCanonicalFields(detail, "Ad Type", "Building ID", "Building External ID", "Housing Company")
			return cleanDetail(detail), nil
		case "building":
			buildingID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse shortcut building id: %w", err)
			}
			row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, pgtype.UUID{Bytes: buildingID, Valid: true})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("shortcut building not found")
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut building detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.ShortcutBuildingAddress), valueOrEmpty(row.ShortcutBuildingHousingCompany), formatInt64Value(row.ShortcutBuildingExternalID)), Address: valueOrEmpty(row.ShortcutBuildingAddress), URL: strings.TrimSpace(row.ShortcutBuildingUrl), LastSeenAt: firstTimeValue(row.ShortcutBuildingUpdatedAt, row.ShortcutBuildingProcessedAt)}}
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: formatInt64Value(row.ShortcutBuildingExternalID)}, {Label: "Housing Company", Value: valueOrEmpty(row.ShortcutBuildingHousingCompany)}, {Label: "Building Type", Value: valueOrEmpty(row.ShortcutBuildingBuildingType)}, {Label: "Building Subtype", Value: valueOrEmpty(row.ShortcutBuildingBuildingSubtype)}, {Label: "Construction Year", Value: formatInt32(row.ShortcutBuildingConstructionYear)}, {Label: "Floor Count", Value: formatInt32(row.ShortcutBuildingFloorCount)}, {Label: "Apartment Count", Value: formatInt32(row.ShortcutBuildingApartmentCount)}, {Label: "Heating System", Value: valueOrEmpty(row.ShortcutBuildingHeatingSystem)}, {Label: "Building Material", Value: valueOrEmpty(row.ShortcutBuildingBuildingMaterial)}, {Label: "Plot Type", Value: valueOrEmpty(row.ShortcutBuildingPlotType)}, {Label: "Wall Structure", Value: valueOrEmpty(row.ShortcutBuildingWallStructure)}, {Label: "Heat Source", Value: valueOrEmpty(row.ShortcutBuildingHeatSource)}, {Label: "Has Elevator", Value: valueOrEmpty(row.ShortcutBuildingHasElevator)}, {Label: "Has Sauna", Value: valueOrEmpty(row.ShortcutBuildingHasSauna)}, {Label: "Latitude", Value: formatFloat64Ptr(row.ShortcutBuildingLatitude)}, {Label: "Longitude", Value: formatFloat64Ptr(row.ShortcutBuildingLongitude)}, {Label: "Page Not Found", Value: formatBoolPtr(row.ShortcutBuildingPageNotFound)}}
			detail.Related = []DetailField{{Label: "Linked Ads", Value: strconv.FormatInt(row.AdCount, 10)}, {Label: "Listings", Value: strconv.FormatInt(row.ListingCount, 10)}, {Label: "Rentals", Value: strconv.FormatInt(row.RentalCount, 10)}}
			detail.Raw = buildRawPayload(row.RawJson)
			detail = promoteCanonicalFields(detail, "External ID", "Housing Company", "Building Type", "Building Subtype", "Construction Year", "Floor Count", "Apartment Count")
			return cleanDetail(detail), nil
		default:
			return UnifiedEntityDetail{}, fmt.Errorf("unsupported shortcut kind: %s", kind)
		}
	case "frontdoor":
		switch kind {
		case "ad":
			row, err := s.queries.GetFrontdoorAdUnifiedDetail(ctx, nativeID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("frontdoor ad not found")
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), row.FrontdoorAdExternalID), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Price: row.AdPrice, Area: row.AdArea, RoomLayout: valueOrEmpty(row.AdRoomLayout), URL: strings.TrimSpace(row.FrontdoorAdUrl), LastSeenAt: row.FrontdoorAdLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: row.FrontdoorAdExternalID}, {Label: "Property Type", Value: valueOrEmpty(row.AdPropertyType)}, {Label: "Condition", Value: valueOrEmpty(row.AdCondition)}, {Label: "Page Not Found", Value: formatBool(row.FrontdoorAdPageNotFound)}}
			detail.Raw = buildRawPayload(row.FrontdoorAdData)
			detail = promoteCanonicalFields(detail, "External ID", "Property Type", "Condition")
			return cleanDetail(detail), nil
		case "announcement":
			announcementID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
			}
			row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, pgtype.UUID{Bytes: announcementID, Valid: true})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("frontdoor announcement not found")
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor announcement detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), formatInt32(row.FrontdoorBuildingAnnouncementExternalID)), Address: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Price: float64ToInt64Ptr(row.FrontdoorBuildingAnnouncementSearchPrice), Area: row.FrontdoorBuildingAnnouncementArea, RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), URL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: row.FrontdoorBuildingAnnouncementLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: formatInt32(row.FrontdoorBuildingAnnouncementExternalID)}, {Label: "Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID)}, {Label: "Property Type", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType)}, {Label: "Property Subtype", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype)}, {Label: "Published", Value: formatBoolPtr(row.FrontdoorBuildingAnnouncementPublished)}}
			detail.Related = []DetailField{{Label: "Building ID", Value: pgUUIDToString(row.FrontdoorBuildingID)}, {Label: "Housing Company ID", Value: formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)}, {Label: "Housing Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID)}, {Label: "Company", Value: valueOrEmpty(row.FrontdoorBuildingCompanyName)}, {Label: "Building Street", Value: valueOrEmpty(row.FrontdoorBuildingStreetAddress)}, {Label: "Building House #", Value: valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, {Label: "Building Post Area", Value: valueOrEmpty(row.FrontdoorBuildingPostArea)}, {Label: "Building Municipality", Value: valueOrEmpty(row.FrontdoorBuildingMunicipality)}}
			detail.Raw = buildRawPayload(row.RawJson)
			detail = promoteCanonicalFields(detail, "External ID", "Friendly ID", "Property Type", "Property Subtype", "Published")
			return cleanDetail(detail), nil
		case "building":
			buildingID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse frontdoor building id: %w", err)
			}
			row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, pgtype.UUID{Bytes: buildingID, Valid: true})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("frontdoor building not found")
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor building detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingCompanyName), strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)), Address: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), URL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: row.FrontdoorBuildingLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "Company", Value: valueOrEmpty(row.FrontdoorBuildingCompanyName)}, {Label: "Business ID", Value: valueOrEmpty(row.FrontdoorBuildingBusinessID)}, {Label: "Housing Company ID", Value: formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)}, {Label: "Housing Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID)}, {Label: "Apartment Count", Value: formatInt32(row.FrontdoorBuildingApartmentCount)}, {Label: "Floor Count", Value: formatInt32(row.FrontdoorBuildingFloorCount)}, {Label: "Build Year", Value: formatInt32(row.FrontdoorBuildingBuildYear)}, {Label: "Has Elevator", Value: formatBoolPtr(row.FrontdoorBuildingHasElevator)}, {Label: "Has Sauna", Value: formatBoolPtr(row.FrontdoorBuildingHasSauna)}, {Label: "Energy Certificate", Value: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode)}, {Label: "Heating", Value: valueOrEmpty(row.FrontdoorBuildingHeating)}, {Label: "Post Area", Value: valueOrEmpty(row.FrontdoorBuildingPostArea)}, {Label: "Latitude", Value: formatFloat64Ptr(row.FrontdoorBuildingLatitude)}, {Label: "Longitude", Value: formatFloat64Ptr(row.FrontdoorBuildingLongitude)}}
			detail.Related = []DetailField{{Label: "Announcement Count", Value: strconv.FormatInt(row.AnnouncementCount, 10)}}
			detail.Raw = buildRawPayload(row.FrontdoorBuildingData)
			detail = promoteCanonicalFields(detail, "Company", "Business ID", "Housing Company ID", "Housing Friendly ID", "Apartment Count", "Floor Count", "Build Year")
			return cleanDetail(detail), nil
		default:
			return UnifiedEntityDetail{}, fmt.Errorf("unsupported frontdoor kind: %s", kind)
		}
	default:
		return UnifiedEntityDetail{}, fmt.Errorf("unsupported source: %s", source)
	}
}

func CanonicalID(source, kind, nativeID string) string {
	return strings.TrimSpace(source) + ":" + strings.TrimSpace(kind) + ":" + strings.TrimSpace(nativeID)
}

func ParseCanonicalID(value string) (string, string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid canonical id: %s", value)
	}
	source := normalizeSource(parts[0])
	kind := normalizeKind(parts[1])
	nativeID := strings.TrimSpace(parts[2])
	if source == "all" || kind == "all" || nativeID == "" {
		return "", "", "", fmt.Errorf("invalid canonical id: %s", value)
	}
	return source, kind, nativeID, nil
}

// shortcutAdPrefixes are URL path prefixes that identify shortcut ad pages.
var shortcutAdPrefixes = []string{
	"/myytavat-asunnot/",
	"/vuokra-asunnot/",
	"/myytavat-toimitilat/",
	"/vuokrattavat-toimitilat/",
	"/myytavat-tontit/",
	"/myytavat-metsatilat-ja-maatilat/",
	"/myytavat-autotallit/",
	"/myytavat-loma-asunnot/",
	"/vuokrattavat-loma-asunnot/",
	"/vuokrattavat-autotallit/",
}

// ResolveInput takes user input that may be a URL or a canonical ID and returns
// a canonical ID. shortcutBase and frontdoorBase are the base URLs used to
// distinguish which source a /talo/ URL belongs to.
func ResolveInput(input, shortcutBase, frontdoorBase string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return resolveURL(input, shortcutBase, frontdoorBase)
	}
	_, _, _, err := ParseCanonicalID(input)
	if err != nil {
		return "", fmt.Errorf("not a valid URL or canonical ID: %s", input)
	}
	return input, nil
}

func resolveURL(raw, shortcutBase, frontdoorBase string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	path := strings.TrimRight(u.Path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 3 {
		return "", fmt.Errorf("URL path too short: %s", path)
	}

	// Last segment is the ID, second-to-last is the category.
	id := segments[len(segments)-1]
	category := "/" + segments[len(segments)-2] + "/"

	// Frontdoor-only path: /kohde/{id}
	if category == "/kohde/" {
		return CanonicalID("frontdoor", "ad", id), nil
	}

	// Shortcut ad paths
	for _, prefix := range shortcutAdPrefixes {
		if category == prefix {
			return CanonicalID("shortcut", "ad", id), nil
		}
	}

	// /talo/{uuid} exists on both sources — use host to disambiguate.
	if category == "/talo/" {
		host := strings.ToLower(u.Host)
		scHost := hostFromBase(shortcutBase)
		fdHost := hostFromBase(frontdoorBase)

		switch {
		case scHost != "" && strings.Contains(host, scHost):
			return CanonicalID("shortcut", "building", id), nil
		case fdHost != "" && strings.Contains(host, fdHost):
			return CanonicalID("frontdoor", "building", id), nil
		default:
			return "", fmt.Errorf("cannot determine source for /talo/ URL (host %q does not match shortcut or frontdoor)", host)
		}
	}

	return "", fmt.Errorf("unrecognized URL path: %s", path)
}

func hostFromBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func normalizeSearchParams(params SearchParams) SearchParams {
	normalized := SearchParams{Query: strings.TrimSpace(params.Query), Source: normalizeSource(params.Source), Kind: normalizeKind(params.Kind), ListingType: normalizeListingType(params.ListingType), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, City: strings.TrimSpace(params.City), Postal: strings.TrimSpace(params.Postal), Page: params.Page, PageSize: normalizePageSize(params.PageSize), Sort: normalizeSort(params.Sort), PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore}
	if normalized.Page < 1 {
		normalized.Page = 1
	}
	return normalized
}

func normalizeSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "shortcut", "frontdoor":
		return strings.ToLower(strings.TrimSpace(source))
	default:
		return "all"
	}
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ad", "announcement", "building":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return "all"
	}
}

func normalizeListingType(listingType string) string {
	switch strings.ToLower(strings.TrimSpace(listingType)) {
	case "listing", "rental":
		return strings.ToLower(strings.TrimSpace(listingType))
	default:
		return "all"
	}
}

func normalizeSort(sortMode string) string {
	switch strings.ToLower(strings.TrimSpace(sortMode)) {
	case "price_asc", "price_desc", "area_asc", "area_desc", "seen_desc":
		return strings.ToLower(strings.TrimSpace(sortMode))
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

func stringPtr(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func emptyToNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cleanDetail(detail UnifiedEntityDetail) UnifiedEntityDetail {
	detail.CanonicalExtra = compactFields(detail.CanonicalExtra)
	detail.SourceSpecific = compactFields(detail.SourceSpecific)
	detail.Related = compactFields(detail.Related)
	return detail
}

func promoteCanonicalFields(detail UnifiedEntityDetail, labels ...string) UnifiedEntityDetail {
	if len(labels) == 0 || len(detail.SourceSpecific) == 0 {
		return detail
	}
	wanted := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		normalized := strings.ToLower(strings.TrimSpace(label))
		if normalized == "" {
			continue
		}
		wanted[normalized] = struct{}{}
	}
	if len(wanted) == 0 {
		return detail
	}
	remaining := make([]DetailField, 0, len(detail.SourceSpecific))
	for _, field := range detail.SourceSpecific {
		key := strings.ToLower(strings.TrimSpace(field.Label))
		if _, ok := wanted[key]; ok {
			detail.CanonicalExtra = append(detail.CanonicalExtra, field)
			continue
		}
		remaining = append(remaining, field)
	}
	detail.SourceSpecific = remaining
	return detail
}

func compactFields(fields []DetailField) []DetailField {
	out := make([]DetailField, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.Label) == "" || strings.TrimSpace(field.Value) == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func buildRawPayload(payload []byte) RawPayload {
	if len(payload) == 0 {
		return RawPayload{}
	}
	pretty := payload
	var buf bytes.Buffer
	if err := json.Indent(&buf, payload, "", "  "); err == nil {
		pretty = buf.Bytes()
	}
	return RawPayload{Pretty: string(pretty), OriginalBytes: len(pretty)}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func formatBoolPtr(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}

func formatInt32(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
}

func formatInt64Ptr(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatInt64Value(value int64) string {
	return strconv.FormatInt(value, 10)
}

func formatFloat64Ptr(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.6f", *value)
}

func float64ToInt64Ptr(value *float64) *int64 {
	if value == nil {
		return nil
	}
	v := int64(*value)
	return &v
}

func pgUUIDToString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func firstTimeValue(first time.Time, rest ...*time.Time) time.Time {
	if !first.IsZero() {
		return first
	}
	for _, v := range rest {
		if v != nil && !v.IsZero() {
			return *v
		}
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
