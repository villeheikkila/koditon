package ads

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/ads/db"
)

type SearchParams struct {
	Query    string
	Source   string
	Kind     string
	MinPrice *int64
	MaxPrice *int64
	MinArea  *float64
	MaxArea  *float64
	City     string
	Postal   string
	Page     int32
	PageSize int32
	Sort     string
}

type ReportRow struct {
	Source     string
	Kind       string
	EntityID   string
	Headline   string
	Address    string
	City       string
	Postal     string
	Price      *int64
	Area       *float64
	RoomLayout string
	URL        string
	LastSeenAt time.Time
}

type ReportPage struct {
	Rows     []ReportRow
	Total    int64
	Page     int32
	PageSize int32
}

type DetailField struct {
	Label string
	Value string
}

type Detail struct {
	Summary []DetailField
	Related []DetailField
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
	count, err := s.queries.CountAdsReports(ctx, &db.CountAdsReportsParams{
		SourceFilter: source,
		KindFilter:   kind,
		QueryText:    emptyToNil(normalized.Query),
		CityFilter:   emptyToNil(normalized.City),
		PostalFilter: emptyToNil(normalized.Postal),
		MinPrice:     normalized.MinPrice,
		MaxPrice:     normalized.MaxPrice,
		MinArea:      normalized.MinArea,
		MaxArea:      normalized.MaxArea,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("count ads reports: %w", err)
	}
	rows, err := s.queries.SearchAdsReports(ctx, &db.SearchAdsReportsParams{
		SortMode:     sort,
		OffsetCount:  offset,
		LimitCount:   normalized.PageSize,
		SourceFilter: source,
		KindFilter:   kind,
		QueryText:    emptyToNil(normalized.Query),
		CityFilter:   emptyToNil(normalized.City),
		PostalFilter: emptyToNil(normalized.Postal),
		MinPrice:     normalized.MinPrice,
		MaxPrice:     normalized.MaxPrice,
		MinArea:      normalized.MinArea,
		MaxArea:      normalized.MaxArea,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("search ads reports: %w", err)
	}
	mapped := make([]ReportRow, 0, len(rows))
	for _, row := range rows {
		mapped = append(mapped, ReportRow{
			Source:     row.Source,
			Kind:       row.Kind,
			EntityID:   row.EntityID,
			Headline:   valueOrEmpty(row.Headline),
			Address:    valueOrEmpty(row.Address),
			City:       valueOrEmpty(row.City),
			Postal:     valueOrEmpty(row.Postal),
			Price:      pgInt8ToPointer(row.Price),
			Area:       pgFloat8ToPointer(row.Area),
			RoomLayout: valueOrEmpty(row.RoomLayout),
			URL:        strings.TrimSpace(row.Url),
			LastSeenAt: row.LastSeenAt.Time,
		})
	}
	return ReportPage{Rows: mapped, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) Detail(ctx context.Context, source, kind, entityID string) (Detail, error) {
	switch normalizeSource(source) {
	case "shortcut":
		if normalizeKind(kind) != "ad" {
			return Detail{}, fmt.Errorf("unsupported shortcut kind: %s", kind)
		}
		adID, err := strconv.ParseInt(strings.TrimSpace(entityID), 10, 64)
		if err != nil {
			return Detail{}, fmt.Errorf("parse shortcut ad id: %w", err)
		}
		row, err := s.queries.GetShortcutAdReportDetail(ctx, pgtype.Int8{Int64: adID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Detail{}, fmt.Errorf("shortcut ad not found")
			}
			return Detail{}, fmt.Errorf("get shortcut ad detail: %w", err)
		}
		summary := []DetailField{}
		appendField(&summary, "Source", "shortcut/ad")
		appendField(&summary, "Ad ID", strconv.FormatInt(row.ShortcutAdID, 10))
		appendField(&summary, "Type", row.ShortcutAdType)
		appendField(&summary, "Address", valueOrEmpty(row.AdAddress))
		appendField(&summary, "Price", formatPgInt8(row.AdPrice))
		appendField(&summary, "Area", formatPgFloat8(row.AdArea))
		appendField(&summary, "Room Layout", valueOrEmpty(row.AdRoomLayout))
		appendField(&summary, "Last Seen", row.ShortcutAdLastSeenAt.Time.Format("2006-01-02 15:04:05Z07:00"))
		appendField(&summary, "URL", row.ShortcutAdUrl)
		related := []DetailField{}
		if row.ShortcutBuildingID.Valid {
			appendField(&related, "Building ID", uuid.UUID(row.ShortcutBuildingID.Bytes).String())
		}
		appendField(&related, "Building External ID", formatPgInt8(row.ShortcutBuildingExternalID))
		appendField(&related, "Building Address", valueOrEmpty(row.ShortcutBuildingAddress))
		appendField(&related, "Housing Company", valueOrEmpty(row.ShortcutBuildingHousingCompany))
		appendField(&related, "Building URL", valueOrEmpty(row.ShortcutBuildingUrl))
		appendField(&related, "Listing Rows", strconv.FormatInt(row.BuildingListingCount, 10))
		appendField(&related, "Rental Rows", strconv.FormatInt(row.BuildingRentalCount, 10))
		return Detail{Summary: summary, Related: related}, nil
	case "frontdoor":
		switch normalizeKind(kind) {
		case "ad":
			row, err := s.queries.GetFrontdoorAdReportDetail(ctx, strings.TrimSpace(entityID))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return Detail{}, fmt.Errorf("frontdoor ad not found")
				}
				return Detail{}, fmt.Errorf("get frontdoor ad detail: %w", err)
			}
			summary := []DetailField{}
			appendField(&summary, "Source", "frontdoor/ad")
			appendField(&summary, "External ID", row.FrontdoorAdExternalID)
			appendField(&summary, "Address", valueOrEmpty(row.AdAddress))
			appendField(&summary, "City", valueOrEmpty(row.AdCity))
			appendField(&summary, "Postal", valueOrEmpty(row.AdPostal))
			appendField(&summary, "Price", formatPgInt8(row.AdPrice))
			appendField(&summary, "Area", formatPgFloat8(row.AdArea))
			appendField(&summary, "Room Layout", valueOrEmpty(row.AdRoomLayout))
			appendField(&summary, "Property Type", valueOrEmpty(row.AdPropertyType))
			appendField(&summary, "Condition", valueOrEmpty(row.AdCondition))
			appendField(&summary, "Page Not Found", formatBool(row.FrontdoorAdPageNotFound))
			appendField(&summary, "Last Seen", row.FrontdoorAdLastSeenAt.Time.Format("2006-01-02 15:04:05Z07:00"))
			appendField(&summary, "URL", row.FrontdoorAdUrl)
			return Detail{Summary: summary, Related: nil}, nil
		case "announcement":
			parsedID, err := uuid.Parse(strings.TrimSpace(entityID))
			if err != nil {
				return Detail{}, fmt.Errorf("parse announcement id: %w", err)
			}
			row, err := s.queries.GetFrontdoorAnnouncementReportDetail(ctx, pgtype.UUID{Bytes: parsedID, Valid: true})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return Detail{}, fmt.Errorf("frontdoor announcement not found")
				}
				return Detail{}, fmt.Errorf("get frontdoor announcement detail: %w", err)
			}
			summary := []DetailField{}
			appendField(&summary, "Source", "frontdoor/announcement")
			appendField(&summary, "Announcement ID", uuid.UUID(row.FrontdoorBuildingAnnouncementID.Bytes).String())
			appendField(&summary, "External ID", formatInt32(row.FrontdoorBuildingAnnouncementExternalID))
			appendField(&summary, "Friendly ID", valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID))
			appendField(&summary, "Address 1", valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1))
			appendField(&summary, "Address 2", valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2))
			appendField(&summary, "Location", valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation))
			appendField(&summary, "Price", formatFloatToInt(row.FrontdoorBuildingAnnouncementSearchPrice))
			appendField(&summary, "Area", formatFloat(row.FrontdoorBuildingAnnouncementArea))
			appendField(&summary, "Room Layout", valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure))
			appendField(&summary, "Property Type", valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType))
			appendField(&summary, "Property Subtype", valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype))
			appendField(&summary, "Published", formatBoolPtr(row.FrontdoorBuildingAnnouncementPublished))
			appendField(&summary, "Last Seen", row.FrontdoorBuildingAnnouncementLastSeenAt.Time.Format("2006-01-02 15:04:05Z07:00"))
			related := []DetailField{}
			appendField(&related, "Building ID", uuid.UUID(row.FrontdoorBuildingID.Bytes).String())
			appendField(&related, "Building URL", valueOrEmpty(row.FrontdoorBuildingUrl))
			appendField(&related, "Housing Company ID", formatPgInt8(row.FrontdoorBuildingHousingCompanyID))
			appendField(&related, "Housing Friendly ID", valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID))
			appendField(&related, "Company", valueOrEmpty(row.FrontdoorBuildingCompanyName))
			appendField(&related, "Street", valueOrEmpty(row.FrontdoorBuildingStreetAddress))
			appendField(&related, "House #", valueOrEmpty(row.FrontdoorBuildingHouseNumber))
			appendField(&related, "Postal", valueOrEmpty(row.FrontdoorBuildingPostcode))
			appendField(&related, "Post Area", valueOrEmpty(row.FrontdoorBuildingPostArea))
			appendField(&related, "Municipality", valueOrEmpty(row.FrontdoorBuildingMunicipality))
			return Detail{Summary: summary, Related: related}, nil
		default:
			return Detail{}, fmt.Errorf("unsupported frontdoor kind: %s", kind)
		}
	default:
		return Detail{}, fmt.Errorf("unsupported source: %s", source)
	}
}

func normalizeSearchParams(params SearchParams) SearchParams {
	normalized := SearchParams{
		Query:    strings.TrimSpace(params.Query),
		Source:   normalizeSource(params.Source),
		Kind:     normalizeKind(params.Kind),
		MinPrice: params.MinPrice,
		MaxPrice: params.MaxPrice,
		MinArea:  params.MinArea,
		MaxArea:  params.MaxArea,
		City:     strings.TrimSpace(params.City),
		Postal:   strings.TrimSpace(params.Postal),
		Page:     params.Page,
		PageSize: normalizePageSize(params.PageSize),
		Sort:     normalizeSort(params.Sort),
	}
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
	case "ad", "announcement":
		return strings.ToLower(strings.TrimSpace(kind))
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

func appendField(fields *[]DetailField, label string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	*fields = append(*fields, DetailField{Label: label, Value: trimmed})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func pgInt8ToPointer(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func pgFloat8ToPointer(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func formatPgInt8(value pgtype.Int8) string {
	if !value.Valid {
		return ""
	}
	return strconv.FormatInt(value.Int64, 10)
}

func formatInt32(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
}

func formatFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *value)
}

func formatPgFloat8(value pgtype.Float8) string {
	if !value.Valid {
		return ""
	}
	return fmt.Sprintf("%.1f", value.Float64)
}

func formatFloatToInt(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
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
