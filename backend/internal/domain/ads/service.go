package ads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	frontdoorpayload "koditon/internal/providers/frontdoor"
	shortcutpayload "koditon/internal/providers/shortcut"
)

type SearchParams struct {
	Query           string
	Source          string
	Kind            string
	Grouping        string
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
	CanonicalID             string
	Source                  string
	Kind                    string
	NativeID                string
	ListingID               string
	OfferingID              string
	HousingCompanyID        string
	HousingCompanyName      string
	LinkStatus              string
	LinkMethod              string
	LinkScore               *int32
	PriceMatchTransactionID string
	PriceMatchScope         string
	PriceMatchStatus        string
	PriceMatchMethod        string
	PriceMatchScore         *int32
	PriceMatchPrice         *int64
	InsightCount            int32
	InsightTopSeverity      string
	Headline                string
	Address                 string
	City                    string
	Postal                  string
	Latitude                *float64
	Longitude               *float64
	Price                   *int64
	Area                    *float64
	RoomLayout              string
	URL                     string
	ExternalURLAvailable    bool
	LastSeenAt              time.Time
}

type AddressLookupParams struct {
	Address  string
	City     string
	Postal   string
	Source   string
	PageSize int32
}

type AddressLookupResult struct {
	Address         string                  `json:"address"`
	City            string                  `json:"city,omitempty"`
	Postal          string                  `json:"postal,omitempty"`
	Source          string                  `json:"source"`
	ListingCount    int                     `json:"listing_count"`
	HasMoreListings bool                    `json:"has_more_listings"`
	Offerings       []AddressOffering       `json:"offerings"`
	Listings        []AddressListing        `json:"listings"`
	RawTransactions []AddressRawTransaction `json:"raw_transactions"`
}

type AddressOffering struct {
	OfferingID           string                   `json:"offering_id"`
	HousingCompanyID     string                   `json:"housing_company_id,omitempty"`
	HousingCompanyName   string                   `json:"housing_company_name,omitempty"`
	Headline             string                   `json:"headline"`
	Address              string                   `json:"address,omitempty"`
	City                 string                   `json:"city,omitempty"`
	Postal               string                   `json:"postal,omitempty"`
	Latitude             *float64                 `json:"latitude,omitempty"`
	Longitude            *float64                 `json:"longitude,omitempty"`
	AskingPrice          *int64                   `json:"asking_price,omitempty"`
	DebtFreePrice        *int64                   `json:"debt_free_price,omitempty"`
	Area                 *float64                 `json:"area,omitempty"`
	RoomLayout           string                   `json:"room_layout,omitempty"`
	FirstSeenAt          *time.Time               `json:"first_seen_at,omitempty"`
	LastSeenAt           *time.Time               `json:"last_seen_at,omitempty"`
	SourceCount          int                      `json:"source_count"`
	Sources              []string                 `json:"sources"`
	SourceRecords        []AddressSourceRecord    `json:"source_records"`
	Transactions         []AddressTransactionLink `json:"transactions"`
	Insights             []AddressInsight         `json:"insights,omitempty"`
	Representative       AddressSourceRecord      `json:"representative"`
	SourceCandidateCount int                      `json:"source_candidate_count"`
}

type AddressListing struct {
	ListingID             string                   `json:"listing_id"`
	CanonicalID           string                   `json:"canonical_id"`
	Source                string                   `json:"source"`
	Kind                  string                   `json:"kind"`
	NativeID              string                   `json:"native_id"`
	HousingCompanyID      string                   `json:"housing_company_id,omitempty"`
	HousingCompanyName    string                   `json:"housing_company_name,omitempty"`
	Headline              string                   `json:"headline"`
	Address               string                   `json:"address"`
	City                  string                   `json:"city,omitempty"`
	Postal                string                   `json:"postal,omitempty"`
	Latitude              *float64                 `json:"latitude,omitempty"`
	Longitude             *float64                 `json:"longitude,omitempty"`
	AskingPrice           *int64                   `json:"asking_price,omitempty"`
	DebtFreePrice         *int64                   `json:"debt_free_price,omitempty"`
	Area                  *float64                 `json:"area,omitempty"`
	RoomLayout            string                   `json:"room_layout,omitempty"`
	URL                   string                   `json:"url,omitempty"`
	ExternalURLAvailable  bool                     `json:"external_url_available"`
	FirstSeenAt           *time.Time               `json:"first_seen_at,omitempty"`
	LastSeenAt            *time.Time               `json:"last_seen_at,omitempty"`
	PublishedAt           *time.Time               `json:"published_at,omitempty"`
	CreatedAt             *time.Time               `json:"created_at,omitempty"`
	UpdatedAt             *time.Time               `json:"updated_at,omitempty"`
	PreviousAskingPrice   *int64                   `json:"previous_asking_price,omitempty"`
	PreviousDebtFreePrice *int64                   `json:"previous_debt_free_price,omitempty"`
	PriceMatchStatus      string                   `json:"price_match_status,omitempty"`
	SourceMatchStatus     string                   `json:"source_match_status,omitempty"`
	OfferingID            string                   `json:"offering_id,omitempty"`
	Texts                 *AddressListingTexts     `json:"texts,omitempty"`
	SourceRecords         []AddressSourceRecord    `json:"source_records"`
	SourceCandidates      []AddressSourceCandidate `json:"source_candidates"`
	Transactions          []AddressTransactionLink `json:"transactions"`
	Insights              []AddressInsight         `json:"insights,omitempty"`
}

type AddressListingTexts struct {
	Availability       string `json:"availability,omitempty"`
	RenovationsDone    string `json:"renovations_done,omitempty"`
	RenovationsPlanned string `json:"renovations_planned,omitempty"`
	AdditionalInfo     string `json:"additional_info,omitempty"`
	Charges            string `json:"charges,omitempty"`
}

type AddressSourceRecord struct {
	ListingID             string               `json:"listing_id"`
	CanonicalID           string               `json:"canonical_id"`
	Source                string               `json:"source"`
	Kind                  string               `json:"kind"`
	NativeID              string               `json:"native_id"`
	Headline              string               `json:"headline"`
	Address               string               `json:"address,omitempty"`
	City                  string               `json:"city,omitempty"`
	Postal                string               `json:"postal,omitempty"`
	Latitude              *float64             `json:"latitude,omitempty"`
	Longitude             *float64             `json:"longitude,omitempty"`
	AskingPrice           *int64               `json:"asking_price,omitempty"`
	DebtFreePrice         *int64               `json:"debt_free_price,omitempty"`
	Area                  *float64             `json:"area,omitempty"`
	RoomLayout            string               `json:"room_layout,omitempty"`
	URL                   string               `json:"url,omitempty"`
	ExternalURLAvailable  bool                 `json:"external_url_available"`
	FirstSeenAt           *time.Time           `json:"first_seen_at,omitempty"`
	LastSeenAt            *time.Time           `json:"last_seen_at,omitempty"`
	UpdatedAt             *time.Time           `json:"updated_at,omitempty"`
	PreviousAskingPrice   *int64               `json:"previous_asking_price,omitempty"`
	PreviousDebtFreePrice *int64               `json:"previous_debt_free_price,omitempty"`
	LinkStatus            string               `json:"link_status,omitempty"`
	LinkMethod            string               `json:"link_method,omitempty"`
	LinkScore             *int32               `json:"link_score,omitempty"`
	Texts                 *AddressListingTexts `json:"texts,omitempty"`
	Insights              []AddressInsight     `json:"insights,omitempty"`
}

type AddressInsight struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Direction   string  `json:"direction,omitempty"`
	Severity    string  `json:"severity,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	SourceField string  `json:"source_field,omitempty"`
	Text        string  `json:"text,omitempty"`
}

type AddressSourceCandidate struct {
	ListingID            string          `json:"listing_id"`
	CanonicalID          string          `json:"canonical_id"`
	Source               string          `json:"source"`
	Kind                 string          `json:"kind"`
	NativeID             string          `json:"native_id"`
	Headline             string          `json:"headline"`
	Address              string          `json:"address,omitempty"`
	City                 string          `json:"city,omitempty"`
	Postal               string          `json:"postal,omitempty"`
	AskingPrice          *int64          `json:"asking_price,omitempty"`
	DebtFreePrice        *int64          `json:"debt_free_price,omitempty"`
	Area                 *float64        `json:"area,omitempty"`
	RoomLayout           string          `json:"room_layout,omitempty"`
	URL                  string          `json:"url,omitempty"`
	ExternalURLAvailable bool            `json:"external_url_available"`
	SelectedOfferingID   string          `json:"selected_offering_id,omitempty"`
	CandidateOfferingID  string          `json:"candidate_offering_id,omitempty"`
	Direction            string          `json:"direction"`
	Status               string          `json:"status"`
	Score                int32           `json:"score"`
	Confidence           string          `json:"confidence"`
	PriceDeltaPercent    *float64        `json:"price_delta_percent,omitempty"`
	Reasons              json.RawMessage `json:"reasons,omitempty"`
	ReasonsSummary       []string        `json:"reasons_summary,omitempty"`
	CreatedAt            *time.Time      `json:"created_at,omitempty"`
}

type AddressTransactionLink struct {
	TransactionID       string          `json:"transaction_id"`
	LinkType            string          `json:"link_type"`
	LinkStatus          string          `json:"link_status,omitempty"`
	LinkMethod          string          `json:"link_method,omitempty"`
	Score               *int32          `json:"score,omitempty"`
	Confidence          string          `json:"confidence,omitempty"`
	PriceDeltaPercent   *float64        `json:"price_delta_percent,omitempty"`
	Reasons             json.RawMessage `json:"reasons,omitempty"`
	ReasonsSummary      []string        `json:"reasons_summary,omitempty"`
	Description         string          `json:"description"`
	Type                string          `json:"type,omitempty"`
	Category            string          `json:"category,omitempty"`
	Area                *float64        `json:"area,omitempty"`
	Price               *int64          `json:"price,omitempty"`
	PricePerSquareMeter *int64          `json:"price_per_square_meter,omitempty"`
	BuildYear           *int32          `json:"build_year,omitempty"`
	Floor               string          `json:"floor,omitempty"`
	Elevator            *bool           `json:"elevator,omitempty"`
	Condition           string          `json:"condition,omitempty"`
	Plot                string          `json:"plot,omitempty"`
	EnergyClass         string          `json:"energy_class,omitempty"`
	PeriodIdentifier    string          `json:"period_identifier,omitempty"`
	City                string          `json:"city,omitempty"`
	Neighborhood        string          `json:"neighborhood,omitempty"`
	Postal              string          `json:"postal,omitempty"`
	CreatedAt           *time.Time      `json:"created_at,omitempty"`
	UpdatedAt           *time.Time      `json:"updated_at,omitempty"`
}

type AddressRawTransaction struct {
	TransactionID        string                       `json:"transaction_id"`
	Description          string                       `json:"description"`
	Type                 string                       `json:"type,omitempty"`
	Category             string                       `json:"category,omitempty"`
	Area                 *float64                     `json:"area,omitempty"`
	Price                *int64                       `json:"price,omitempty"`
	PricePerSquareMeter  *int64                       `json:"price_per_square_meter,omitempty"`
	BuildYear            *int32                       `json:"build_year,omitempty"`
	Floor                string                       `json:"floor,omitempty"`
	Elevator             *bool                        `json:"elevator,omitempty"`
	Condition            string                       `json:"condition,omitempty"`
	Plot                 string                       `json:"plot,omitempty"`
	EnergyClass          string                       `json:"energy_class,omitempty"`
	PeriodIdentifier     string                       `json:"period_identifier,omitempty"`
	City                 string                       `json:"city,omitempty"`
	Neighborhood         string                       `json:"neighborhood,omitempty"`
	Postal               string                       `json:"postal,omitempty"`
	CreatedAt            *time.Time                   `json:"created_at,omitempty"`
	UpdatedAt            *time.Time                   `json:"updated_at,omitempty"`
	IsMatched            bool                         `json:"is_matched"`
	LinkedToLookup       bool                         `json:"linked_to_lookup"`
	CandidateToLookup    bool                         `json:"candidate_to_lookup"`
	Scope                string                       `json:"scope"`
	MatchedListingCount  int32                        `json:"matched_listing_count"`
	MatchedOfferingCount int32                        `json:"matched_offering_count"`
	Matches              []AddressRawTransactionMatch `json:"matches"`
}

type AddressRawTransactionMatch struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	OfferingID  string `json:"offering_id,omitempty"`
	CanonicalID string `json:"canonical_id,omitempty"`
	Source      string `json:"source,omitempty"`
	NativeID    string `json:"native_id,omitempty"`
	Headline    string `json:"headline,omitempty"`
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	Postal      string `json:"postal,omitempty"`
	Status      string `json:"status,omitempty"`
	Method      string `json:"method,omitempty"`
	Score       *int32 `json:"score,omitempty"`
}

type ReportPage struct {
	Rows     []UnifiedEntityRow
	Total    int64
	Page     int32
	PageSize int32
}

type GroupedOfferingSearchPage struct {
	Rows     []GroupedOfferingRow
	Total    int64
	Page     int32
	PageSize int32
}

type GroupedOfferingRow struct {
	OfferingID              string
	HousingCompanyID        string
	HousingCompanyName      string
	Headline                string
	Address                 string
	City                    string
	Postal                  string
	Price                   *int64
	Area                    *float64
	RoomLayout              string
	LastSeenAt              *time.Time
	SourceCount             int32
	Sources                 []string
	PriceMatchTransactionID string
	PriceMatchScope         string
	PriceMatchStatus        string
	PriceMatchMethod        string
	PriceMatchScore         *int32
	PriceMatchPrice         *int64
	InsightCount            int32
	InsightTopSeverity      string
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
	CanonicalID          string
	Source               string
	Kind                 string
	NativeID             string
	Headline             string
	Address              string
	City                 string
	Postal               string
	Latitude             *float64
	Longitude            *float64
	Price                *int64
	Area                 *float64
	RoomLayout           string
	URL                  string
	ExternalURLAvailable bool
	LastSeenAt           time.Time
}

type UnifiedEntityDetail struct {
	Canonical      UnifiedCanonicalFields
	CanonicalExtra []DetailField
	SourceSpecific []DetailField
	Related        []DetailField
	Normalized     NormalizedDetailFields
	Raw            RawPayload
}

type NormalizedDetailFields struct {
	CanonicalID              string   `json:"canonical_id"`
	Source                   string   `json:"source"`
	Kind                     string   `json:"kind"`
	URL                      string   `json:"url"`
	StreetAddress            string   `json:"street_address,omitempty"`
	City                     string   `json:"city,omitempty"`
	Postal                   string   `json:"postal,omitempty"`
	Latitude                 *float64 `json:"latitude,omitempty"`
	Longitude                *float64 `json:"longitude,omitempty"`
	AskingPrice              *int64   `json:"asking_price,omitempty"`
	DebtFreePrice            *int64   `json:"debt_free_price,omitempty"`
	DebtShareAmount          *int64   `json:"debt_share_amount,omitempty"`
	PricePerSquareMeter      *float64 `json:"price_per_m2,omitempty"`
	AreaM2                   *float64 `json:"area_m2,omitempty"`
	RoomLayout               string   `json:"room_layout,omitempty"`
	RoomsCount               *int32   `json:"rooms_count,omitempty"`
	FloorLevel               *int32   `json:"floor_level,omitempty"`
	TotalFloors              *int32   `json:"total_floors,omitempty"`
	BuildYear                *int32   `json:"build_year,omitempty"`
	Condition                string   `json:"condition,omitempty"`
	EnergyClass              string   `json:"energy_class,omitempty"`
	PlotType                 string   `json:"plot_type,omitempty"`
	Elevator                 *bool    `json:"elevator,omitempty"`
	Sauna                    *bool    `json:"sauna,omitempty"`
	MaintenanceChargeMonthly *float64 `json:"maintenance_charge_monthly,omitempty"`
	TotalChargeMonthly       *float64 `json:"total_charge_monthly,omitempty"`
	WaterCharge              *float64 `json:"water_charge,omitempty"`
	DescriptionText          string   `json:"description_text,omitempty"`
	AvailabilityText         string   `json:"availability_text,omitempty"`
	RenovationsDoneText      string   `json:"renovations_done_text,omitempty"`
	RenovationsPlannedText   string   `json:"renovations_planned_text,omitempty"`
	AdditionalInfoText       string   `json:"additional_info_text,omitempty"`
	ChargesText              string   `json:"charges_text,omitempty"`
}

type Service struct {
	db      db.DBTX
	queries *db.Queries
}

var ErrNotFound = errors.New("ads detail not found")

func NewService(dbtx db.DBTX) *Service {
	return &Service{db: dbtx, queries: db.New(dbtx)}
}

func (s *Service) Search(ctx context.Context, params SearchParams) (ReportPage, error) {
	normalized := normalizeSearchParams(params)
	offset := (normalized.Page - 1) * normalized.PageSize
	minPrice, maxPrice := int64(0), int64(9223372036854775807)
	minArea, maxArea := float64(0), math.MaxFloat64
	if normalized.MinPrice != nil {
		minPrice = *normalized.MinPrice
	}
	if normalized.MaxPrice != nil {
		maxPrice = *normalized.MaxPrice
	}
	if normalized.MinArea != nil {
		minArea = *normalized.MinArea
	}
	if normalized.MaxArea != nil {
		maxArea = *normalized.MaxArea
	}
	listingType := strings.TrimSpace(normalized.ListingType)
	if listingType == "all" {
		listingType = ""
	}
	publishedAfter := time.Time{}
	publishedBefore := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if normalized.PublishedAfter != nil {
		publishedAfter = *normalized.PublishedAfter
	}
	if normalized.PublishedBefore != nil {
		publishedBefore = *normalized.PublishedBefore
	}
	count, err := s.queries.CountUnifiedEntities(ctx, db.CountUnifiedEntitiesParams{
		Column1:  &normalized.Source,
		Column2:  &normalized.Kind,
		Column3:  &normalized.Grouping,
		Column4:  stringPtr(strings.TrimSpace(normalized.Query)),
		Column5:  stringPtr(strings.TrimSpace(normalized.City)),
		Column6:  stringPtr(strings.TrimSpace(normalized.Postal)),
		Column7:  &minPrice,
		Column8:  &maxPrice,
		Column9:  &minArea,
		Column10: &maxArea,
		Column11: &listingType,
		Column12: &publishedAfter,
		Column13: &publishedBefore,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("count unified entities: %w", err)
	}
	rows, err := s.queries.SearchUnifiedEntities(ctx, db.SearchUnifiedEntitiesParams{
		Column1:  normalized.Source,
		Column2:  normalized.Kind,
		Column3:  normalized.Grouping,
		Column4:  strings.TrimSpace(normalized.Query),
		Column5:  strings.TrimSpace(normalized.City),
		Column6:  strings.TrimSpace(normalized.Postal),
		Column7:  minPrice,
		Column8:  maxPrice,
		Column9:  minArea,
		Column10: maxArea,
		Column11: listingType,
		Column12: publishedAfter,
		Column13: publishedBefore,
		Column14: normalized.Sort,
		Column15: normalized.PageSize,
		Column16: offset,
	})
	if err != nil {
		return ReportPage{}, fmt.Errorf("search unified entities: %w", err)
	}
	mapped := make([]UnifiedEntityRow, 0, len(rows))
	for _, row := range rows {
		mapped = append(mapped, UnifiedEntityRow{
			CanonicalID:             strings.TrimSpace(valueOrEmpty(row.CanonicalID)),
			Source:                  valueOrEmpty(row.Source),
			Kind:                    valueOrEmpty(row.Kind),
			NativeID:                valueOrEmpty(row.NativeID),
			ListingID:               strings.TrimSpace(valueOrEmpty(row.ListingID)),
			OfferingID:              strings.TrimSpace(valueOrEmpty(row.OfferingID)),
			HousingCompanyID:        strings.TrimSpace(valueOrEmpty(row.HousingCompanyID)),
			HousingCompanyName:      strings.TrimSpace(valueOrEmpty(row.HousingCompanyName)),
			LinkStatus:              strings.TrimSpace(valueOrEmpty(row.LinkStatus)),
			LinkMethod:              strings.TrimSpace(valueOrEmpty(row.LinkMethod)),
			LinkScore:               row.LinkScore,
			PriceMatchTransactionID: strings.TrimSpace(valueOrEmpty(row.PriceMatchTransactionID)),
			PriceMatchScope:         strings.TrimSpace(valueOrEmpty(row.PriceMatchScope)),
			PriceMatchStatus:        strings.TrimSpace(valueOrEmpty(row.PriceMatchStatus)),
			PriceMatchMethod:        strings.TrimSpace(valueOrEmpty(row.PriceMatchMethod)),
			PriceMatchScore:         row.PriceMatchScore,
			PriceMatchPrice:         int64PtrIf(valueOrEmpty(row.PriceMatchTransactionID) != "", ptrInt64Value(row.PriceMatchPriceEur)),
			InsightCount:            ptrInt32Value(row.InsightCount),
			InsightTopSeverity:      strings.TrimSpace(valueOrEmpty(row.InsightTopSeverity)),
			Headline:                strings.TrimSpace(valueOrEmpty(row.Headline)),
			Address:                 strings.TrimSpace(valueOrEmpty(row.Address)),
			City:                    strings.TrimSpace(valueOrEmpty(row.City)),
			Postal:                  strings.TrimSpace(valueOrEmpty(row.Postal)),
			Latitude:                row.Latitude,
			Longitude:               row.Longitude,
			Price:                   row.Price,
			Area:                    row.Area,
			RoomLayout:              strings.TrimSpace(valueOrEmpty(row.RoomLayout)),
			URL:                     strings.TrimSpace(valueOrEmpty(row.Url)),
			ExternalURLAvailable:    boolPtrValue(row.ExternalUrlAvailable),
			LastSeenAt:              timePtrValue(row.LastSeenAt),
		})
	}
	return ReportPage{Rows: mapped, Total: ptrInt64Value(count), Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) SearchGroupedOfferings(ctx context.Context, params SearchParams) (GroupedOfferingSearchPage, error) {
	normalized := normalizeSearchParams(params)
	offset := (normalized.Page - 1) * normalized.PageSize
	source := normalizeSource(normalized.Source)
	kind := normalizeKind(normalized.Kind)
	filter := db.CountGroupedOfferingsParams{Source: &source, Kind: &kind, QueryText: emptyToNil(normalized.Query), City: emptyToNil(normalized.City), Postal: emptyToNil(normalized.Postal), MinPrice: normalized.MinPrice, MaxPrice: normalized.MaxPrice, MinArea: normalized.MinArea, MaxArea: normalized.MaxArea, PublishedAfter: normalized.PublishedAfter, PublishedBefore: normalized.PublishedBefore}
	total, err := s.queries.CountGroupedOfferings(ctx, filter)
	if err != nil {
		return GroupedOfferingSearchPage{}, fmt.Errorf("count grouped offerings: %w", err)
	}
	rows, err := s.queries.SearchGroupedOfferings(ctx, db.SearchGroupedOfferingsParams{SortMode: normalized.Sort, OffsetCount: offset, LimitCount: normalized.PageSize, Source: source, Kind: kind, QueryText: filter.QueryText, City: filter.City, Postal: filter.Postal, MinPrice: filter.MinPrice, MaxPrice: filter.MaxPrice, MinArea: filter.MinArea, MaxArea: filter.MaxArea, PublishedAfter: filter.PublishedAfter, PublishedBefore: filter.PublishedBefore})
	if err != nil {
		return GroupedOfferingSearchPage{}, fmt.Errorf("search grouped offerings: %w", err)
	}
	out := make([]GroupedOfferingRow, 0, len(rows))
	for _, result := range rows {
		row := GroupedOfferingRow{OfferingID: valueOrEmpty(result.PropertyOfferingID), HousingCompanyID: valueOrEmpty(result.HousingCompanyID), HousingCompanyName: valueOrEmpty(result.HousingCompanyName), Headline: valueOrEmpty(result.Headline), Address: valueOrEmpty(result.Address), City: valueOrEmpty(result.City), Postal: valueOrEmpty(result.Postal), Price: result.Price, Area: result.Area, RoomLayout: valueOrEmpty(result.RoomLayout), LastSeenAt: result.LastSeenAt, SourceCount: ptrInt32Value(result.SourceCount), Sources: splitCSV(valueOrEmpty(result.Sources)), PriceMatchTransactionID: valueOrEmpty(result.PriceMatchTransactionID), PriceMatchScope: valueOrEmpty(result.PriceMatchScope), PriceMatchStatus: valueOrEmpty(result.PriceMatchStatus), PriceMatchMethod: valueOrEmpty(result.PriceMatchMethod), InsightCount: ptrInt32Value(result.InsightCount), InsightTopSeverity: valueOrEmpty(result.InsightTopSeverity)}
		if row.PriceMatchTransactionID != "" {
			row.PriceMatchScore = result.PriceMatchScore
			row.PriceMatchPrice = result.PriceMatchPriceEur
		}
		out = append(out, row)
	}
	return GroupedOfferingSearchPage{Rows: out, Total: ptrInt64Value(total), Page: normalized.Page, PageSize: normalized.PageSize}, nil
}

func (s *Service) LookupAddress(ctx context.Context, params AddressLookupParams) (AddressLookupResult, error) {
	address := strings.TrimSpace(params.Address)
	if address == "" {
		return AddressLookupResult{}, fmt.Errorf("address is required")
	}
	queryAddress, city, postal := normalizeAddressLookupInput(address, params.City, params.Postal)
	if city == "" && postal != "" {
		resolvedCity, cityAliases, err := s.lookupPostalCities(ctx, postal)
		if err != nil {
			return AddressLookupResult{}, err
		}
		if stripped := stripTrailingAddressCity(queryAddress, cityAliases); stripped != queryAddress {
			queryAddress = stripped
			city = resolvedCity
		}
	}
	source := normalizeSource(params.Source)
	limit := params.PageSize
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.queries.LookupAddressListings(ctx, db.LookupAddressListingsParams{Column1: queryAddress, Column2: city, Column3: postal, Column4: source, Column5: limit})
	if err != nil {
		return AddressLookupResult{}, fmt.Errorf("lookup address listings: %w", err)
	}
	lookupRows := make([]addressLookupRow, 0, len(rows))
	for _, row := range rows {
		lookupRows = append(lookupRows, addressLookupRowFromDB(row))
	}
	result := buildAddressLookupResult(queryAddress, city, postal, source, lookupRows)
	if err := s.attachAddressSourceCandidates(ctx, &result); err != nil {
		return AddressLookupResult{}, err
	}
	result.Offerings = addressOfferings(result.Listings)
	rawTransactions, err := s.lookupAddressRawTransactions(ctx, result, 40)
	if err != nil {
		return AddressLookupResult{}, err
	}
	result.RawTransactions = rawTransactions
	return result, nil
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
			row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, &adID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: shortcut ad", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), strconv.FormatInt(row.ShortcutAdID, 10)), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Latitude: row.AdLatitude, Longitude: row.AdLongitude, Price: row.AdPrice, Area: row.AdArea, RoomLayout: strings.TrimSpace(valueOrEmpty(row.AdRoomLayout)), URL: strings.TrimSpace(row.ShortcutAdUrl), ExternalURLAvailable: strings.TrimSpace(row.ShortcutAdUrl) != "" && row.ShortcutAdLastSeenAt.After(time.Now().AddDate(0, 0, -7)), LastSeenAt: row.ShortcutAdLastSeenAt}}
			detail.Normalized = normalizedFromShortcutAdDetail(canonicalID, source, kind, detail.Canonical, row)
			detail.SourceSpecific = []DetailField{{Label: "Ad Type", Value: row.ShortcutAdType}, {Label: "Building ID", Value: ptrUUIDToString(row.ShortcutBuildingID)}, {Label: "Building External ID", Value: formatInt64Value(row.ShortcutBuildingExternalID)}, {Label: "Building Address", Value: valueOrEmpty(row.ShortcutBuildingAddress)}, {Label: "Housing Company", Value: valueOrEmpty(row.ShortcutBuildingHousingCompany)}, {Label: "Building URL", Value: row.ShortcutBuildingUrl}}
			detail.Related = []DetailField{{Label: "Building Listings", Value: strconv.FormatInt(ptrInt64Value(row.BuildingListingCount), 10)}, {Label: "Building Rentals", Value: strconv.FormatInt(ptrInt64Value(row.BuildingRentalCount), 10)}}
			detail.Raw = buildRawPayload(row.ShortcutAdData)
			detail = promoteCanonicalFields(detail, "Ad Type", "Building ID", "Building External ID", "Housing Company")
			return cleanDetail(detail), nil
		case "building":
			buildingID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse shortcut building id: %w", err)
			}
			row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, &buildingID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: shortcut building", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut building detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.ShortcutBuildingAddress), valueOrEmpty(row.ShortcutBuildingHousingCompany), formatInt64Value(row.ShortcutBuildingExternalID)), Address: valueOrEmpty(row.ShortcutBuildingAddress), Latitude: row.ShortcutBuildingLatitude, Longitude: row.ShortcutBuildingLongitude, URL: strings.TrimSpace(row.ShortcutBuildingUrl), ExternalURLAvailable: row.ShortcutBuildingPageNotFound != nil && !*row.ShortcutBuildingPageNotFound, LastSeenAt: firstTimeValue(row.ShortcutBuildingUpdatedAt, row.ShortcutBuildingProcessedAt)}}
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: formatInt64Value(row.ShortcutBuildingExternalID)}, {Label: "Housing Company", Value: valueOrEmpty(row.ShortcutBuildingHousingCompany)}, {Label: "Building Type", Value: valueOrEmpty(row.ShortcutBuildingBuildingType)}, {Label: "Building Subtype", Value: valueOrEmpty(row.ShortcutBuildingBuildingSubtype)}, {Label: "Construction Year", Value: formatInt32(row.ShortcutBuildingConstructionYear)}, {Label: "Floor Count", Value: formatInt32(row.ShortcutBuildingFloorCount)}, {Label: "Apartment Count", Value: formatInt32(row.ShortcutBuildingApartmentCount)}, {Label: "Heating System", Value: valueOrEmpty(row.ShortcutBuildingHeatingSystem)}, {Label: "Building Material", Value: valueOrEmpty(row.ShortcutBuildingBuildingMaterial)}, {Label: "Plot Type", Value: valueOrEmpty(row.ShortcutBuildingPlotType)}, {Label: "Wall Structure", Value: valueOrEmpty(row.ShortcutBuildingWallStructure)}, {Label: "Heat Source", Value: valueOrEmpty(row.ShortcutBuildingHeatSource)}, {Label: "Has Elevator", Value: valueOrEmpty(row.ShortcutBuildingHasElevator)}, {Label: "Has Sauna", Value: valueOrEmpty(row.ShortcutBuildingHasSauna)}, {Label: "Latitude", Value: formatFloat64Ptr(row.ShortcutBuildingLatitude)}, {Label: "Longitude", Value: formatFloat64Ptr(row.ShortcutBuildingLongitude)}, {Label: "Page Not Found", Value: formatBoolPtr(row.ShortcutBuildingPageNotFound)}}
			detail.Related = []DetailField{{Label: "Linked Ads", Value: strconv.FormatInt(ptrInt64Value(row.AdCount), 10)}, {Label: "Listings", Value: strconv.FormatInt(ptrInt64Value(row.ListingCount), 10)}, {Label: "Rentals", Value: strconv.FormatInt(ptrInt64Value(row.RentalCount), 10)}}
			detail.Raw = buildRawPayload(row.RawJson)
			detail = promoteCanonicalFields(detail, "External ID", "Housing Company", "Building Type", "Building Subtype", "Construction Year", "Floor Count", "Apartment Count")
			return cleanDetail(detail), nil
		default:
			return UnifiedEntityDetail{}, fmt.Errorf("unsupported shortcut kind: %s", kind)
		}
	case "frontdoor":
		switch kind {
		case "ad":
			row, err := s.queries.GetFrontdoorAdUnifiedDetail(ctx, &nativeID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor ad", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), row.FrontdoorAdExternalID), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Latitude: row.AdLatitude, Longitude: row.AdLongitude, Price: row.AdPrice, Area: row.AdArea, RoomLayout: strings.TrimSpace(valueOrEmpty(row.AdRoomLayout)), URL: strings.TrimSpace(row.FrontdoorAdUrl), ExternalURLAvailable: !row.FrontdoorAdPageNotFound, LastSeenAt: row.FrontdoorAdLastSeenAt}}
			detail.Normalized = normalizedFromFrontdoorAdDetail(canonicalID, source, kind, detail.Canonical, row)
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: row.FrontdoorAdExternalID}, {Label: "Property Type", Value: strings.TrimSpace(valueOrEmpty(row.AdPropertyType))}, {Label: "Condition", Value: strings.TrimSpace(valueOrEmpty(row.AdCondition))}, {Label: "Page Not Found", Value: formatBool(row.FrontdoorAdPageNotFound)}}
			detail.Raw = buildRawPayload(row.FrontdoorAdData)
			detail = promoteCanonicalFields(detail, "External ID", "Property Type", "Condition")
			return cleanDetail(detail), nil
		case "announcement":
			announcementID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
			}
			row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, &announcementID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor announcement", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor announcement detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), formatInt32(row.FrontdoorBuildingAnnouncementExternalID)), Address: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Latitude: row.FrontdoorBuildingLatitude, Longitude: row.FrontdoorBuildingLongitude, Price: float64ToInt64Ptr(row.FrontdoorBuildingAnnouncementSearchPrice), Area: row.FrontdoorBuildingAnnouncementArea, RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), URL: valueOrEmpty(row.FrontdoorBuildingUrl), ExternalURLAvailable: boolPtrValue(row.FrontdoorBuildingAnnouncementPublished), LastSeenAt: row.FrontdoorBuildingAnnouncementLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: formatInt32(row.FrontdoorBuildingAnnouncementExternalID)}, {Label: "Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID)}, {Label: "Property Type", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType)}, {Label: "Property Subtype", Value: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype)}, {Label: "Published", Value: formatBoolPtr(row.FrontdoorBuildingAnnouncementPublished)}}
			detail.Related = []DetailField{{Label: "Building ID", Value: row.FrontdoorBuildingID.String()}, {Label: "Housing Company ID", Value: formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)}, {Label: "Housing Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID)}, {Label: "Company", Value: valueOrEmpty(row.FrontdoorBuildingCompanyName)}, {Label: "Building Street", Value: valueOrEmpty(row.FrontdoorBuildingStreetAddress)}, {Label: "Building House #", Value: valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, {Label: "Building Post Area", Value: valueOrEmpty(row.FrontdoorBuildingPostArea)}, {Label: "Building Municipality", Value: valueOrEmpty(row.FrontdoorBuildingMunicipality)}}
			detail.Raw = buildRawPayload(row.RawJson)
			detail = promoteCanonicalFields(detail, "External ID", "Friendly ID", "Property Type", "Property Subtype", "Published")
			return cleanDetail(detail), nil
		case "building":
			buildingID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse frontdoor building id: %w", err)
			}
			row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, &buildingID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor building", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor building detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingCompanyName), strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)), Address: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Latitude: row.FrontdoorBuildingLatitude, Longitude: row.FrontdoorBuildingLongitude, URL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: row.FrontdoorBuildingLastSeenAt}}
			detail.SourceSpecific = []DetailField{{Label: "Company", Value: valueOrEmpty(row.FrontdoorBuildingCompanyName)}, {Label: "Business ID", Value: valueOrEmpty(row.FrontdoorBuildingBusinessID)}, {Label: "Housing Company ID", Value: formatInt64Ptr(row.FrontdoorBuildingHousingCompanyID)}, {Label: "Housing Friendly ID", Value: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID)}, {Label: "Apartment Count", Value: formatInt32(row.FrontdoorBuildingApartmentCount)}, {Label: "Floor Count", Value: formatInt32(row.FrontdoorBuildingFloorCount)}, {Label: "Build Year", Value: formatInt32(row.FrontdoorBuildingBuildYear)}, {Label: "Has Elevator", Value: formatBoolPtr(row.FrontdoorBuildingHasElevator)}, {Label: "Has Sauna", Value: formatBoolPtr(row.FrontdoorBuildingHasSauna)}, {Label: "Energy Certificate", Value: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode)}, {Label: "Heating", Value: valueOrEmpty(row.FrontdoorBuildingHeating)}, {Label: "Post Area", Value: valueOrEmpty(row.FrontdoorBuildingPostArea)}, {Label: "Latitude", Value: formatFloat64Ptr(row.FrontdoorBuildingLatitude)}, {Label: "Longitude", Value: formatFloat64Ptr(row.FrontdoorBuildingLongitude)}}
			detail.Related = []DetailField{{Label: "Announcement Count", Value: strconv.FormatInt(ptrInt64Value(row.AnnouncementCount), 10)}}
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

type addressLookupRow struct {
	ListingID                        uuid.UUID
	CanonicalID                      string
	Source                           string
	Kind                             string
	NativeID                         string
	Headline                         string
	Address                          string
	City                             string
	Postal                           string
	Latitude                         *float64
	Longitude                        *float64
	AskingPrice                      *int64
	DebtFreePrice                    *int64
	Area                             *float64
	RoomLayout                       string
	URL                              string
	ExternalURLAvailable             bool
	FirstSeenAt                      *time.Time
	LastSeenAt                       *time.Time
	PublishedAt                      *time.Time
	CreatedAt                        *time.Time
	UpdatedAt                        *time.Time
	PreviousAskingPrice              *int64
	PreviousDebtFreePrice            *int64
	PriceMatchStatus                 string
	SourceMatchStatus                string
	OfferingID                       *uuid.UUID
	HousingCompanyID                 *uuid.UUID
	HousingCompanyName               string
	AvailabilityText                 string
	RenovationsDoneText              string
	RenovationsPlannedText           string
	AdditionalInfoText               string
	ChargesText                      string
	InsightsJSON                     json.RawMessage
	ListingCount                     int
	TransactionID                    *uuid.UUID
	LinkType                         string
	LinkStatus                       string
	LinkMethod                       string
	Score                            *int32
	Confidence                       string
	PriceDeltaPercent                *float64
	Reasons                          json.RawMessage
	TransactionDescription           string
	TransactionType                  string
	TransactionCategory              string
	TransactionArea                  *float64
	TransactionPrice                 *int64
	TransactionPricePerM2            *int64
	TransactionBuildYear             *int32
	TransactionFloor                 string
	TransactionElevator              *bool
	TransactionCondition             string
	TransactionPlot                  string
	TransactionEnergyClass           string
	TransactionPeriodIdentifier      string
	TransactionCity                  string
	TransactionNeighborhood          string
	TransactionPostal                string
	TransactionCreatedAt             *time.Time
	TransactionUpdatedAt             *time.Time
	SourceRecordListingID            *uuid.UUID
	SourceRecordCanonicalID          string
	SourceRecordSource               string
	SourceRecordKind                 string
	SourceRecordNativeID             string
	SourceRecordHeadline             string
	SourceRecordAddress              string
	SourceRecordCity                 string
	SourceRecordPostal               string
	SourceRecordLatitude             *float64
	SourceRecordLongitude            *float64
	SourceRecordAskingPrice          *int64
	SourceRecordDebtFreePrice        *int64
	SourceRecordArea                 *float64
	SourceRecordRoomLayout           string
	SourceRecordURL                  string
	SourceRecordExternalURLAvailable bool
	SourceRecordFirstSeenAt          *time.Time
	SourceRecordLastSeenAt           *time.Time
	SourceRecordUpdatedAt            *time.Time
	SourceRecordPreviousAsk          *int64
	SourceRecordPreviousDebt         *int64
	SourceRecordLinkStatus           string
	SourceRecordLinkMethod           string
	SourceRecordLinkScore            *int32
	SourceRecordAvailability         string
	SourceRecordRenovationsDone      string
	SourceRecordRenovationsPlan      string
	SourceRecordAdditionalInfo       string
	SourceRecordCharges              string
	SourceRecordInsightsJSON         json.RawMessage
}

type addressRawTransactionRow struct {
	TransactionID        uuid.UUID
	Description          string
	Type                 string
	Category             string
	Area                 *float64
	Price                *int64
	PricePerSquareMeter  *int64
	BuildYear            *int32
	Floor                string
	Elevator             *bool
	Condition            string
	Plot                 string
	EnergyClass          string
	PeriodIdentifier     string
	City                 string
	Neighborhood         string
	Postal               string
	CreatedAt            *time.Time
	UpdatedAt            *time.Time
	IsMatched            bool
	LinkedToLookup       bool
	CandidateToLookup    bool
	MatchedListingCount  int32
	MatchedOfferingCount int32
	Matches              json.RawMessage
}

type addressSourceCandidateRow struct {
	SelectedListingID    uuid.UUID
	CandidateListingID   uuid.UUID
	CanonicalID          string
	Source               string
	Kind                 string
	NativeID             string
	Headline             string
	Address              string
	City                 string
	Postal               string
	AskingPrice          *int64
	DebtFreePrice        *int64
	Area                 *float64
	RoomLayout           string
	URL                  string
	ExternalURLAvailable bool
	SelectedOfferingID   *uuid.UUID
	CandidateOfferingID  *uuid.UUID
	Direction            string
	Status               string
	Score                int32
	Confidence           string
	PriceDeltaPercent    *float64
	Reasons              json.RawMessage
	CreatedAt            *time.Time
}

func addressLookupRowFromDB(row db.LookupAddressListingsRow) addressLookupRow {
	return addressLookupRow{
		ListingID: row.SaleListingID, CanonicalID: row.SaleListingCanonicalID, Source: row.SaleListingSourceProvider, Kind: row.SaleListingSourceKind, NativeID: row.SaleListingNativeID,
		Headline: valueOrEmpty(row.Headline), Address: valueOrEmpty(row.Address), City: valueOrEmpty(row.City), Postal: valueOrEmpty(row.Postal),
		Latitude: row.SaleListingLatitude, Longitude: row.SaleListingLongitude, AskingPrice: row.SaleListingAskingPrice, DebtFreePrice: row.SaleListingDebtFreePrice, Area: row.SaleListingAreaValue,
		RoomLayout: valueOrEmpty(row.RoomLayout), URL: valueOrEmpty(row.Url), ExternalURLAvailable: boolPtrValue(row.ExternalUrlAvailable),
		FirstSeenAt: row.SaleListingFirstSeenAt, LastSeenAt: row.SaleListingLastSeenAt, PublishedAt: row.SaleListingPublishedAt, CreatedAt: new(row.SaleListingCreatedAt), UpdatedAt: new(row.SaleListingUpdatedAt),
		PreviousAskingPrice: row.SaleListingPreviousAskingPrice, PreviousDebtFreePrice: row.SaleListingPreviousDebtFreePrice, PriceMatchStatus: valueOrEmpty(row.PricesMatchStatus), SourceMatchStatus: valueOrEmpty(row.SourceMatchStatus),
		OfferingID: row.PropertyOfferingID, HousingCompanyID: &row.HousingCompanyID, HousingCompanyName: valueOrEmpty(row.HousingCompanyName),
		AvailabilityText: valueOrEmpty(row.AvailabilityText), RenovationsDoneText: valueOrEmpty(row.RenovationsDoneText), RenovationsPlannedText: valueOrEmpty(row.RenovationsPlannedText), AdditionalInfoText: valueOrEmpty(row.AdditionalInfoText), ChargesText: valueOrEmpty(row.ChargesText), InsightsJSON: row.InsightsJson, ListingCount: int(ptrInt32Value(row.ListingCount)),
		TransactionID: &row.PricesTransactionID, LinkType: valueOrEmpty(row.Coalesce), LinkStatus: valueOrEmpty(row.Coalesce_2), LinkMethod: valueOrEmpty(row.Coalesce_3), Score: row.Score, Confidence: valueOrEmpty(row.Coalesce_4), PriceDeltaPercent: row.PriceDeltaPercent, Reasons: row.Coalesce_5,
		TransactionDescription: valueOrEmpty(row.Coalesce_6), TransactionType: valueOrEmpty(row.Coalesce_7), TransactionCategory: valueOrEmpty(row.Coalesce_8), TransactionArea: &row.PricesTransactionArea, TransactionPrice: row.PricesTransactionPrice, TransactionPricePerM2: row.PricesTransactionPricePerSquareMeter, TransactionBuildYear: &row.PricesTransactionBuildYear, TransactionFloor: valueOrEmpty(row.Coalesce_9), TransactionElevator: &row.PricesTransactionElevator, TransactionCondition: valueOrEmpty(row.Coalesce_10), TransactionPlot: valueOrEmpty(row.Coalesce_11), TransactionEnergyClass: valueOrEmpty(row.Coalesce_12), TransactionPeriodIdentifier: valueOrEmpty(row.Coalesce_13), TransactionCity: valueOrEmpty(row.Coalesce_14), TransactionNeighborhood: valueOrEmpty(row.Coalesce_15), TransactionPostal: valueOrEmpty(row.Coalesce_16), TransactionCreatedAt: &row.PricesTransactionCreatedAt, TransactionUpdatedAt: &row.PricesTransactionUpdatedAt,
		SourceRecordListingID: &row.SaleListingID_2, SourceRecordCanonicalID: valueOrEmpty(row.Coalesce_17), SourceRecordSource: valueOrEmpty(row.Coalesce_18), SourceRecordKind: valueOrEmpty(row.Coalesce_19), SourceRecordNativeID: valueOrEmpty(row.Coalesce_20), SourceRecordHeadline: valueOrEmpty(row.Coalesce_21), SourceRecordAddress: valueOrEmpty(row.Coalesce_22), SourceRecordCity: valueOrEmpty(row.Coalesce_23), SourceRecordPostal: valueOrEmpty(row.Coalesce_24),
		SourceRecordLatitude: row.SaleListingLatitude_2, SourceRecordLongitude: row.SaleListingLongitude_2, SourceRecordAskingPrice: row.SaleListingAskingPrice_2, SourceRecordDebtFreePrice: row.SaleListingDebtFreePrice_2, SourceRecordArea: row.SaleListingAreaValue_2, SourceRecordRoomLayout: valueOrEmpty(row.Coalesce_25), SourceRecordURL: valueOrEmpty(row.Coalesce_26), SourceRecordExternalURLAvailable: boolPtrValue(row.Coalesce_27), SourceRecordFirstSeenAt: row.SaleListingFirstSeenAt_2, SourceRecordLastSeenAt: row.SaleListingLastSeenAt_2, SourceRecordUpdatedAt: new(row.SaleListingUpdatedAt_2), SourceRecordPreviousAsk: row.SaleListingPreviousAskingPrice_2, SourceRecordPreviousDebt: row.SaleListingPreviousDebtFreePrice_2,
		SourceRecordLinkStatus: valueOrEmpty(row.Coalesce_28), SourceRecordLinkMethod: valueOrEmpty(row.Coalesce_29), SourceRecordLinkScore: &row.PropertyOfferingSourceLinkScore, SourceRecordAvailability: valueOrEmpty(row.Coalesce_30), SourceRecordRenovationsDone: valueOrEmpty(row.Coalesce_31), SourceRecordRenovationsPlan: valueOrEmpty(row.Coalesce_32), SourceRecordAdditionalInfo: valueOrEmpty(row.Coalesce_33), SourceRecordCharges: valueOrEmpty(row.Coalesce_34), SourceRecordInsightsJSON: row.Coalesce_35,
	}
}

func buildAddressLookupResult(address, city, postal, source string, rows []addressLookupRow) AddressLookupResult {
	result := AddressLookupResult{Address: address, City: city, Postal: postal, Source: source, Offerings: []AddressOffering{}, Listings: []AddressListing{}, RawTransactions: []AddressRawTransaction{}}
	index := map[uuid.UUID]int{}
	sourceRecordsByOffering := map[uuid.UUID][]AddressSourceRecord{}
	seenSourceRecords := map[uuid.UUID]map[uuid.UUID]struct{}{}
	seenTransactions := map[string]struct{}{}
	for _, row := range rows {
		if row.ListingCount > result.ListingCount {
			result.ListingCount = row.ListingCount
		}
		if row.OfferingID != nil {
			seen := seenSourceRecords[*row.OfferingID]
			if seen == nil {
				seen = map[uuid.UUID]struct{}{}
				seenSourceRecords[*row.OfferingID] = seen
			}
			sourceRecordID := row.ListingID
			sourceRecord := AddressSourceRecord{ListingID: row.ListingID.String(), CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, Headline: row.Headline, Address: row.Address, City: row.City, Postal: row.Postal, Latitude: row.Latitude, Longitude: row.Longitude, AskingPrice: row.AskingPrice, DebtFreePrice: row.DebtFreePrice, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, UpdatedAt: row.UpdatedAt, PreviousAskingPrice: row.PreviousAskingPrice, PreviousDebtFreePrice: row.PreviousDebtFreePrice, Texts: addressListingTexts(row.AvailabilityText, row.RenovationsDoneText, row.RenovationsPlannedText, row.AdditionalInfoText, row.ChargesText), Insights: parseAddressInsights(row.InsightsJSON)}
			if row.SourceRecordListingID != nil {
				sourceRecordID = *row.SourceRecordListingID
				sourceRecord = AddressSourceRecord{ListingID: row.SourceRecordListingID.String(), CanonicalID: row.SourceRecordCanonicalID, Source: row.SourceRecordSource, Kind: row.SourceRecordKind, NativeID: row.SourceRecordNativeID, Headline: row.SourceRecordHeadline, Address: row.SourceRecordAddress, City: row.SourceRecordCity, Postal: row.SourceRecordPostal, Latitude: row.SourceRecordLatitude, Longitude: row.SourceRecordLongitude, AskingPrice: row.SourceRecordAskingPrice, DebtFreePrice: row.SourceRecordDebtFreePrice, Area: row.SourceRecordArea, RoomLayout: row.SourceRecordRoomLayout, URL: row.SourceRecordURL, ExternalURLAvailable: row.SourceRecordExternalURLAvailable, FirstSeenAt: row.SourceRecordFirstSeenAt, LastSeenAt: row.SourceRecordLastSeenAt, UpdatedAt: row.SourceRecordUpdatedAt, PreviousAskingPrice: row.SourceRecordPreviousAsk, PreviousDebtFreePrice: row.SourceRecordPreviousDebt, LinkStatus: row.SourceRecordLinkStatus, LinkMethod: row.SourceRecordLinkMethod, LinkScore: row.SourceRecordLinkScore, Texts: addressListingTexts(row.SourceRecordAvailability, row.SourceRecordRenovationsDone, row.SourceRecordRenovationsPlan, row.SourceRecordAdditionalInfo, row.SourceRecordCharges), Insights: parseAddressInsights(row.SourceRecordInsightsJSON)}
			}
			if _, ok := seen[sourceRecordID]; !ok {
				seen[sourceRecordID] = struct{}{}
				sourceRecordsByOffering[*row.OfferingID] = append(sourceRecordsByOffering[*row.OfferingID], sourceRecord)
			}
		}
		listingIndex, ok := index[row.ListingID]
		if !ok {
			listing := AddressListing{ListingID: row.ListingID.String(), CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, HousingCompanyID: uuidPtrString(row.HousingCompanyID), HousingCompanyName: row.HousingCompanyName, Headline: row.Headline, Address: row.Address, City: row.City, Postal: row.Postal, Latitude: row.Latitude, Longitude: row.Longitude, AskingPrice: row.AskingPrice, DebtFreePrice: row.DebtFreePrice, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, PublishedAt: row.PublishedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, PreviousAskingPrice: row.PreviousAskingPrice, PreviousDebtFreePrice: row.PreviousDebtFreePrice, PriceMatchStatus: row.PriceMatchStatus, SourceMatchStatus: row.SourceMatchStatus, OfferingID: uuidPtrString(row.OfferingID), Texts: addressListingTexts(row.AvailabilityText, row.RenovationsDoneText, row.RenovationsPlannedText, row.AdditionalInfoText, row.ChargesText), SourceRecords: []AddressSourceRecord{}, SourceCandidates: []AddressSourceCandidate{}, Transactions: []AddressTransactionLink{}, Insights: parseAddressInsights(row.InsightsJSON)}
			result.Listings = append(result.Listings, listing)
			listingIndex = len(result.Listings) - 1
			index[row.ListingID] = listingIndex
		}
		if row.TransactionID == nil {
			continue
		}
		key := row.ListingID.String() + ":" + row.TransactionID.String()
		if _, ok := seenTransactions[key]; ok {
			continue
		}
		seenTransactions[key] = struct{}{}
		result.Listings[listingIndex].Transactions = append(result.Listings[listingIndex].Transactions, AddressTransactionLink{TransactionID: row.TransactionID.String(), LinkType: row.LinkType, LinkStatus: row.LinkStatus, LinkMethod: row.LinkMethod, Score: row.Score, Confidence: row.Confidence, PriceDeltaPercent: row.PriceDeltaPercent, Reasons: row.Reasons, ReasonsSummary: addressMatchReasonSummary(row.Reasons), Description: row.TransactionDescription, Type: row.TransactionType, Category: row.TransactionCategory, Area: row.TransactionArea, Price: row.TransactionPrice, PricePerSquareMeter: row.TransactionPricePerM2, BuildYear: row.TransactionBuildYear, Floor: row.TransactionFloor, Elevator: row.TransactionElevator, Condition: row.TransactionCondition, Plot: row.TransactionPlot, EnergyClass: row.TransactionEnergyClass, PeriodIdentifier: row.TransactionPeriodIdentifier, City: row.TransactionCity, Neighborhood: row.TransactionNeighborhood, Postal: row.TransactionPostal, CreatedAt: row.TransactionCreatedAt, UpdatedAt: row.TransactionUpdatedAt})
	}
	for i := range result.Listings {
		offeringID, err := uuid.Parse(result.Listings[i].OfferingID)
		if err == nil {
			result.Listings[i].SourceRecords = slices.Clone(sourceRecordsByOffering[offeringID])
		}
	}
	result.Offerings = addressOfferings(result.Listings)
	if result.ListingCount == 0 {
		result.ListingCount = len(result.Listings)
	}
	result.HasMoreListings = result.ListingCount > len(result.Listings)
	return result
}

func addressOfferings(listings []AddressListing) []AddressOffering {
	index := map[string]int{}
	out := []AddressOffering{}
	for _, listing := range listings {
		if listing.OfferingID == "" {
			continue
		}
		sourceRecords := listing.SourceRecords
		if len(sourceRecords) == 0 {
			sourceRecords = []AddressSourceRecord{addressSourceRecordFromListing(listing)}
		}
		offeringIndex, ok := index[listing.OfferingID]
		if !ok {
			representative := representativeAddressSourceRecord(sourceRecords)
			out = append(out, AddressOffering{OfferingID: listing.OfferingID, HousingCompanyID: listing.HousingCompanyID, HousingCompanyName: listing.HousingCompanyName, Headline: firstNonEmpty(representative.Headline, listing.Headline, representative.Address, listing.OfferingID), Address: firstNonEmpty(representative.Address, listing.Address), City: firstNonEmpty(representative.City, listing.City), Postal: firstNonEmpty(representative.Postal, listing.Postal), Latitude: firstFloat64(representative.Latitude, listing.Latitude), Longitude: firstFloat64(representative.Longitude, listing.Longitude), AskingPrice: firstInt64(representative.AskingPrice, listing.AskingPrice), DebtFreePrice: firstInt64(representative.DebtFreePrice, listing.DebtFreePrice), Area: firstFloat64(representative.Area, listing.Area), RoomLayout: firstNonEmpty(representative.RoomLayout, listing.RoomLayout), FirstSeenAt: earliestTimePtr(representative.FirstSeenAt, listing.FirstSeenAt), LastSeenAt: latestTimePtr(representative.LastSeenAt, listing.LastSeenAt), SourceRecords: []AddressSourceRecord{}, Transactions: []AddressTransactionLink{}, Representative: representative})
			offeringIndex = len(out) - 1
			index[listing.OfferingID] = offeringIndex
		}
		out[offeringIndex].HousingCompanyID = firstNonEmpty(out[offeringIndex].HousingCompanyID, listing.HousingCompanyID)
		out[offeringIndex].HousingCompanyName = firstNonEmpty(out[offeringIndex].HousingCompanyName, listing.HousingCompanyName)
		out[offeringIndex].SourceRecords = appendUniqueAddressSourceRecords(out[offeringIndex].SourceRecords, sourceRecords)
		out[offeringIndex].Transactions = appendUniqueAddressTransactions(out[offeringIndex].Transactions, listing.Transactions)
		out[offeringIndex].FirstSeenAt = earliestTimePtr(out[offeringIndex].FirstSeenAt, listing.FirstSeenAt)
		out[offeringIndex].LastSeenAt = latestTimePtr(out[offeringIndex].LastSeenAt, listing.LastSeenAt)
		out[offeringIndex].SourceCandidateCount += len(listing.SourceCandidates)
	}
	for i := range out {
		out[i].SourceRecords = sortAddressSourceRecords(out[i].SourceRecords)
		out[i].SourceCount = len(out[i].SourceRecords)
		out[i].Sources = addressOfferingSourceLabels(out[i].SourceRecords)
		out[i].Insights = addressOfferingInsights(out[i].SourceRecords)
		out[i].Representative = representativeAddressSourceRecord(out[i].SourceRecords)
		out[i].Headline = firstNonEmpty(out[i].Representative.Headline, out[i].Headline)
		out[i].Address = firstNonEmpty(out[i].Representative.Address, out[i].Address)
		out[i].City = firstNonEmpty(out[i].Representative.City, out[i].City)
		out[i].Postal = firstNonEmpty(out[i].Representative.Postal, out[i].Postal)
		out[i].AskingPrice = firstInt64(out[i].Representative.AskingPrice, out[i].AskingPrice)
		out[i].DebtFreePrice = firstInt64(out[i].Representative.DebtFreePrice, out[i].DebtFreePrice)
		out[i].Area = firstFloat64(out[i].Representative.Area, out[i].Area)
		out[i].RoomLayout = firstNonEmpty(out[i].Representative.RoomLayout, out[i].RoomLayout)
	}
	slices.SortFunc(out, func(a, b AddressOffering) int {
		return compareTimeDesc(a.LastSeenAt, b.LastSeenAt)
	})
	return out
}

func (s *Service) attachAddressSourceCandidates(ctx context.Context, result *AddressLookupResult) error {
	if len(result.Listings) == 0 {
		return nil
	}
	listingIDs := make([]uuid.UUID, 0, len(result.Listings))
	index := map[uuid.UUID]int{}
	for i, listing := range result.Listings {
		id, err := uuid.Parse(listing.ListingID)
		if err != nil {
			continue
		}
		listingIDs = append(listingIDs, id)
		index[id] = i
	}
	if len(listingIDs) == 0 {
		return nil
	}
	rows, err := s.queries.LookupAddressSourceCandidates(ctx, listingIDs)
	if err != nil {
		return fmt.Errorf("lookup address source candidates: %w", err)
	}
	candidates := make([]addressSourceCandidateRow, 0, len(rows))
	for _, row := range rows {
		if row.SelectedSaleListingID == nil {
			continue
		}
		candidates = append(candidates, addressSourceCandidateRow{SelectedListingID: *row.SelectedSaleListingID, CandidateListingID: row.SaleListingID, CanonicalID: row.SaleListingCanonicalID, Source: row.SaleListingSourceProvider, Kind: row.SaleListingSourceKind, NativeID: row.SaleListingNativeID, Headline: valueOrEmpty(row.Headline), Address: valueOrEmpty(row.Address), City: valueOrEmpty(row.City), Postal: valueOrEmpty(row.Postal), AskingPrice: row.SaleListingAskingPrice, DebtFreePrice: row.SaleListingDebtFreePrice, Area: row.SaleListingAreaValue, RoomLayout: valueOrEmpty(row.RoomLayout), URL: valueOrEmpty(row.Url), ExternalURLAvailable: boolPtrValue(row.ExternalUrlAvailable), SelectedOfferingID: &row.SelectedPropertyOfferingID, CandidateOfferingID: &row.CandidatePropertyOfferingID, Direction: valueOrEmpty(row.Direction), Status: row.MatchStatus, Score: row.MatchScore, Confidence: valueOrEmpty(row.MatchConfidence), PriceDeltaPercent: row.PriceDeltaPercent, Reasons: row.MatchReasons, CreatedAt: &row.MatchCreatedAt})
	}
	appendAddressSourceCandidateRows(result, index, candidates)
	return nil
}

func appendAddressSourceCandidateRows(result *AddressLookupResult, index map[uuid.UUID]int, rows []addressSourceCandidateRow) {
	seen := map[string]struct{}{}
	for _, row := range rows {
		listingIndex, ok := index[row.SelectedListingID]
		if !ok {
			continue
		}
		key := row.SelectedListingID.String() + ":" + row.CandidateListingID.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result.Listings[listingIndex].SourceCandidates = append(result.Listings[listingIndex].SourceCandidates, AddressSourceCandidate{ListingID: row.CandidateListingID.String(), CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, Headline: row.Headline, Address: row.Address, City: row.City, Postal: row.Postal, AskingPrice: row.AskingPrice, DebtFreePrice: row.DebtFreePrice, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, SelectedOfferingID: uuidPtrString(row.SelectedOfferingID), CandidateOfferingID: uuidPtrString(row.CandidateOfferingID), Direction: row.Direction, Status: row.Status, Score: row.Score, Confidence: row.Confidence, PriceDeltaPercent: row.PriceDeltaPercent, Reasons: row.Reasons, ReasonsSummary: addressMatchReasonSummary(row.Reasons), CreatedAt: row.CreatedAt})
	}
}

func (s *Service) lookupAddressRawTransactions(ctx context.Context, result AddressLookupResult, limit int32) ([]AddressRawTransaction, error) {
	city, postal := rawTransactionLocation(result)
	if city == "" && postal == "" {
		return []AddressRawTransaction{}, nil
	}
	linkedTransactionIDs := linkedTransactionIDs(result)
	candidateTransactionIDs := candidateTransactionIDs(result)
	rows, err := s.queries.LookupAddressRawTransactions(ctx, db.LookupAddressRawTransactionsParams{Column1: city, Column2: postal, Column3: linkedTransactionIDs, Column4: candidateTransactionIDs, Column5: limit})
	if err != nil {
		return nil, fmt.Errorf("lookup address raw transactions: %w", err)
	}
	transactions := make([]AddressRawTransaction, 0, len(rows))
	for _, result := range rows {
		row := addressRawTransactionRow{TransactionID: result.TransactionID, Description: valueOrEmpty(result.Description), Type: valueOrEmpty(result.Type), Category: valueOrEmpty(result.Category), Area: &result.Area, Price: result.Price, PricePerSquareMeter: result.PricePerSquareMeter, BuildYear: &result.BuildYear, Floor: valueOrEmpty(result.Floor), Elevator: &result.Elevator, Condition: valueOrEmpty(result.Condition), Plot: valueOrEmpty(result.Plot), EnergyClass: valueOrEmpty(result.EnergyClass), PeriodIdentifier: valueOrEmpty(result.PeriodIdentifier), City: valueOrEmpty(result.City), Neighborhood: valueOrEmpty(result.Neighborhood), Postal: valueOrEmpty(result.Postal), CreatedAt: &result.CreatedAt, UpdatedAt: &result.UpdatedAt, IsMatched: valueOrFalse(result.IsMatched), LinkedToLookup: valueOrFalse(result.LinkedToLookup), CandidateToLookup: valueOrFalse(result.CandidateToLookup), MatchedListingCount: ptrInt32Value(result.MatchedListingCount), MatchedOfferingCount: ptrInt32Value(result.MatchedOfferingCount), Matches: result.Matches}
		matches, err := decodeRawTransactionMatches(row.Matches)
		if err != nil {
			return nil, fmt.Errorf("decode address raw transaction matches: %w", err)
		}
		transactions = append(transactions, AddressRawTransaction{TransactionID: row.TransactionID.String(), Description: row.Description, Type: row.Type, Category: row.Category, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, BuildYear: row.BuildYear, Floor: row.Floor, Elevator: row.Elevator, Condition: row.Condition, Plot: row.Plot, EnergyClass: row.EnergyClass, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Neighborhood: row.Neighborhood, Postal: row.Postal, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, IsMatched: row.IsMatched, LinkedToLookup: row.LinkedToLookup, CandidateToLookup: row.CandidateToLookup, Scope: rawTransactionScope(row.LinkedToLookup, row.CandidateToLookup, row.IsMatched), MatchedListingCount: row.MatchedListingCount, MatchedOfferingCount: row.MatchedOfferingCount, Matches: matches})
	}
	return transactions, nil
}

func rawTransactionScope(linkedToLookup bool, candidateToLookup bool, isMatched bool) string {
	if linkedToLookup {
		return "linked_here"
	}
	if candidateToLookup {
		return "candidate_here"
	}
	if isMatched {
		return "matched_elsewhere"
	}
	return "postal_history"
}

func decodeRawTransactionMatches(raw json.RawMessage) ([]AddressRawTransactionMatch, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return []AddressRawTransactionMatch{}, nil
	}
	var matches []AddressRawTransactionMatch
	if err := json.Unmarshal(raw, &matches); err != nil {
		return nil, err
	}
	if matches == nil {
		return []AddressRawTransactionMatch{}, nil
	}
	return matches, nil
}

func addressMatchReasonSummary(raw json.RawMessage) []string {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("{}")) || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var reasons map[string]any
	if err := json.Unmarshal(raw, &reasons); err != nil {
		return nil
	}
	summary := []string{}
	if value, ok := stringReasonValue(reasons["matched_by"]); ok {
		summary = append(summary, "Matched by "+value)
	}
	if value, ok := providerReasonSummary(reasons["source_provider"], reasons["target_provider"]); ok {
		summary = append(summary, value)
	}
	if value, ok := stringReasonValue(reasons["postal"]); ok {
		summary = append(summary, "Postal "+value)
	} else if value, ok := sourceTargetReasonSummary("Postal", reasons["postal"]); ok {
		summary = append(summary, value)
	}
	for _, item := range []struct {
		key   string
		label string
	}{{"address", "Address"}, {"street_name", "Street"}, {"street_match_key", "Street key"}, {"building_match_key", "Building key"}, {"unit_match_key", "Unit key"}, {"prices_transaction_id", "Prices transaction"}} {
		if value, ok := sourceTargetReasonSummary(item.label, reasons[item.key]); ok {
			summary = append(summary, value)
		}
	}
	if value, ok := reasonPairSummary("Area", reasons["area"]); ok {
		summary = append(summary, value)
	}
	if value, ok := layoutReasonSummary(reasons["layout"]); ok {
		summary = append(summary, value)
	}
	if value, ok := boolReasonValue(reasons["layout_prefix"]); ok && value {
		summary = append(summary, "Layout prefix matched")
	}
	if value, ok := stringReasonValue(reasons["property_type"]); ok {
		summary = append(summary, "Type "+value)
	}
	for _, item := range []struct {
		key   string
		label string
	}{{"floor_level", "Floor"}, {"total_floors", "Total floors"}, {"energy", "Energy"}, {"build_year", "Build year"}, {"room_category", "Rooms"}, {"elevator", "Elevator"}, {"condition", "Condition"}, {"plot", "Plot"}, {"plot_owned", "Plot owned"}} {
		if value, ok := reasonPairSummary(item.label, reasons[item.key]); ok {
			summary = append(summary, value)
		}
	}
	if value, ok := stringReasonValue(reasons["transaction_period_month"]); ok {
		summary = append(summary, "Transaction month "+value)
	}
	if value, ok := reasonScoreSummary(reasons["score"]); ok {
		summary = append(summary, value)
	}
	return summary
}

func reasonPairSummary(label string, value any) (string, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		text, textOK := reasonValueString(value)
		if !textOK {
			return "", false
		}
		return label + " " + text, true
	}
	listing, listingOK := reasonValueString(object["listing"])
	transaction, transactionOK := reasonValueString(object["transaction"])
	if !listingOK && !transactionOK {
		return sourceTargetReasonSummary(label, object)
	}
	if !listingOK {
		listing = "n/a"
	}
	if !transactionOK {
		transaction = "n/a"
	}
	return label + " " + listing + " / " + transaction, true
}

func layoutReasonSummary(value any) (string, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return reasonPairSummary("Layout", value)
	}
	listing, listingOK := reasonValueString(object["listing"])
	transaction, transactionOK := reasonValueString(object["transaction"])
	code, codeOK := stringReasonValue(object["code"])
	if !listingOK && !transactionOK && !codeOK {
		return sourceTargetReasonSummary("Layout", object)
	}
	prefix := "Layout"
	if codeOK {
		prefix += " " + code
	}
	if !listingOK {
		listing = "n/a"
	}
	if !transactionOK {
		transaction = "n/a"
	}
	return prefix + " " + listing + " / " + transaction, true
}

func reasonScoreSummary(value any) (string, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	total, ok := reasonValueString(object["total"])
	if ok {
		return "Score total " + total, true
	}
	parts := []string{}
	for _, item := range []string{"address", "unit", "building", "street", "street_area_layout", "street_area_floor_price", "area", "layout", "floor", "build_year", "elevator", "plot", "energy", "condition", "price", "temporal", "transaction", "postal", "city"} {
		score, scoreOK := positiveReasonScore(object[item])
		if scoreOK {
			parts = append(parts, strings.ReplaceAll(item, "_", " ")+" "+score)
		}
		if len(parts) == 4 {
			break
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return "Score " + strings.Join(parts, ", "), true
}

func providerReasonSummary(source any, target any) (string, bool) {
	sourceText, sourceOK := reasonValueString(source)
	targetText, targetOK := reasonValueString(target)
	if !sourceOK && !targetOK {
		return "", false
	}
	if !sourceOK {
		sourceText = "n/a"
	}
	if !targetOK {
		targetText = "n/a"
	}
	return "Sources " + sourceText + " / " + targetText, true
}

func sourceTargetReasonSummary(label string, value any) (string, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	source, sourceOK := reasonValueString(object["source"])
	target, targetOK := reasonValueString(object["target"])
	if !sourceOK && !targetOK {
		return "", false
	}
	if !sourceOK {
		source = "n/a"
	}
	if !targetOK {
		target = "n/a"
	}
	return label + " " + source + " / " + target, true
}

func positiveReasonScore(value any) (string, bool) {
	switch typed := value.(type) {
	case float64:
		if typed <= 0 {
			return "", false
		}
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case int:
		if typed <= 0 {
			return "", false
		}
		return strconv.Itoa(typed), true
	default:
		text, ok := reasonValueString(value)
		if !ok || text == "0" {
			return "", false
		}
		return text, true
	}
}

func stringReasonValue(value any) (string, bool) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func boolReasonValue(value any) (bool, bool) {
	result, ok := value.(bool)
	return result, ok
}

func reasonValueString(value any) (string, bool) {
	switch typed := value.(type) {
	case nil:
		return "", false
	case string:
		text := strings.TrimSpace(typed)
		return text, text != ""
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(typed), true
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", false
		}
		text := strings.TrimSpace(string(encoded))
		return text, text != "" && text != "null"
	}
}

func rawTransactionLocation(result AddressLookupResult) (string, string) {
	city := strings.TrimSpace(result.City)
	postal := strings.TrimSpace(result.Postal)
	for _, listing := range result.Listings {
		if city == "" {
			city = strings.TrimSpace(listing.City)
		}
		if postal == "" {
			postal = strings.TrimSpace(listing.Postal)
		}
		if city != "" && postal != "" {
			return city, postal
		}
	}
	return city, postal
}

func linkedTransactionIDs(result AddressLookupResult) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	ids := []uuid.UUID{}
	for _, listing := range result.Listings {
		for _, transaction := range listing.Transactions {
			if !isLinkedAddressTransaction(transaction) {
				continue
			}
			id, err := uuid.Parse(transaction.TransactionID)
			if err != nil {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func candidateTransactionIDs(result AddressLookupResult) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	ids := []uuid.UUID{}
	for _, listing := range result.Listings {
		for _, transaction := range listing.Transactions {
			if isLinkedAddressTransaction(transaction) {
				continue
			}
			id, err := uuid.Parse(transaction.TransactionID)
			if err != nil {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func isLinkedAddressTransaction(transaction AddressTransactionLink) bool {
	linkType := strings.ToLower(strings.TrimSpace(transaction.LinkType))
	linkStatus := strings.ToLower(strings.TrimSpace(transaction.LinkStatus))
	return linkType == "direct" || linkType == "offering" || linkType == "source_record" || linkStatus == "linked" || linkStatus == "auto_linked"
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
	if slices.Contains(shortcutAdPrefixes, category) {
		return CanonicalID("shortcut", "ad", id), nil
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
	normalized := SearchParams{Query: strings.TrimSpace(params.Query), Source: normalizeSource(params.Source), Kind: normalizeKind(params.Kind), Grouping: normalizeGrouping(params.Grouping), ListingType: normalizeListingType(params.ListingType), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, City: strings.TrimSpace(params.City), Postal: strings.TrimSpace(params.Postal), Page: params.Page, PageSize: normalizePageSize(params.PageSize), Sort: normalizeSort(params.Sort), PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore}
	if normalized.Page < 1 {
		normalized.Page = 1
	}
	return normalized
}

var (
	pastedAddressPostalCityRE = regexp.MustCompile(`^(.+?)\s+(\d{5})\s+(.+)$`)
	pastedAddressPostalRE     = regexp.MustCompile(`^(.+?)\s+(\d{5})$`)
	pastedPostalCityRE        = regexp.MustCompile(`^(\d{5})\s+(.+)$`)
	pastedPostalRE            = regexp.MustCompile(`^(\d{5})$`)
	pastedCityPostalRE        = regexp.MustCompile(`^(.+?)\s+(\d{5})$`)
)

func normalizeAddressLookupInput(address, city, postal string) (string, string, string) {
	queryAddress := strings.TrimSpace(address)
	queryCity := strings.TrimSpace(city)
	queryPostal := strings.TrimSpace(postal)
	if strings.Contains(queryAddress, ",") {
		parts := strings.SplitN(queryAddress, ",", 2)
		street := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(strings.ReplaceAll(parts[1], ",", " "))
		if street != "" {
			queryAddress = street
		}
		queryCity, queryPostal = applyPastedPostalCity(rest, queryCity, queryPostal)
		return finalizeAddressLookupInput(queryAddress, queryCity, queryPostal)
	}
	if matches := pastedAddressPostalCityRE.FindStringSubmatch(queryAddress); matches != nil {
		if strings.TrimSpace(matches[1]) != "" {
			queryAddress = strings.TrimSpace(matches[1])
		}
		if queryPostal == "" {
			queryPostal = strings.TrimSpace(matches[2])
		}
		if queryCity == "" {
			queryCity = strings.TrimSpace(matches[3])
		}
		return finalizeAddressLookupInput(queryAddress, queryCity, queryPostal)
	}
	if matches := pastedAddressPostalRE.FindStringSubmatch(queryAddress); matches != nil {
		if strings.TrimSpace(matches[1]) != "" {
			queryAddress = strings.TrimSpace(matches[1])
		}
		if queryPostal == "" {
			queryPostal = strings.TrimSpace(matches[2])
		}
	}
	return finalizeAddressLookupInput(queryAddress, queryCity, queryPostal)
}

func finalizeAddressLookupInput(address, city, postal string) (string, string, string) {
	if city != "" {
		address = stripTrailingAddressCity(address, []string{city})
	}
	return address, city, postal
}

func (s *Service) lookupPostalCities(ctx context.Context, postal string) (string, []string, error) {
	normalizedPostal := strings.TrimSpace(postal)
	if normalizedPostal == "" {
		return "", nil, nil
	}
	row, err := s.queries.LookupPostalCity(ctx, &normalizedPostal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("lookup postal city: %w", err)
	}
	cityFI := strings.TrimSpace(row.CityFi)
	citySV := strings.TrimSpace(valueOrEmpty(row.CitySv))
	return cityFI, uniqueNonEmptyStrings(cityFI, citySV), nil
}

func stripTrailingAddressCity(address string, cities []string) string {
	trimmedAddress := strings.TrimSpace(address)
	if trimmedAddress == "" || len(cities) == 0 {
		return trimmedAddress
	}
	addressFields := strings.Fields(trimmedAddress)
	for _, city := range cities {
		cityFields := strings.Fields(strings.TrimSpace(city))
		if len(addressFields) <= len(cityFields) {
			continue
		}
		addressTail := strings.Join(addressFields[len(addressFields)-len(cityFields):], " ")
		if foldAddressText(addressTail) == foldAddressText(city) {
			return strings.Join(addressFields[:len(addressFields)-len(cityFields)], " ")
		}
	}
	return trimmedAddress
}

func foldAddressText(value string) string {
	return strings.NewReplacer("å", "a", "ä", "a", "ö", "o").Replace(strings.ToLower(strings.TrimSpace(value)))
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := foldAddressText(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func applyPastedPostalCity(value, city, postal string) (string, string) {
	text := strings.TrimSpace(value)
	if text == "" {
		return city, postal
	}
	if matches := pastedPostalCityRE.FindStringSubmatch(text); matches != nil {
		if postal == "" {
			postal = strings.TrimSpace(matches[1])
		}
		if city == "" {
			city = strings.TrimSpace(matches[2])
		}
		return city, postal
	}
	if matches := pastedPostalRE.FindStringSubmatch(text); matches != nil {
		if postal == "" {
			postal = strings.TrimSpace(matches[1])
		}
		return city, postal
	}
	if matches := pastedCityPostalRE.FindStringSubmatch(text); matches != nil {
		if city == "" {
			city = strings.TrimSpace(matches[1])
		}
		if postal == "" {
			postal = strings.TrimSpace(matches[2])
		}
	}
	return city, postal
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

func normalizeGrouping(grouping string) string {
	switch strings.ToLower(strings.TrimSpace(grouping)) {
	case "grouped", "ungrouped":
		return strings.ToLower(strings.TrimSpace(grouping))
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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

func normalizedFromShortcutAdDetail(canonicalID, source, kind string, canonical UnifiedCanonicalFields, row db.GetShortcutAdUnifiedDetailRow) NormalizedDetailFields {
	payload := parseShortcutJSONMap(row.ShortcutAdData)
	return NormalizedDetailFields{
		CanonicalID:              canonicalID,
		Source:                   source,
		Kind:                     kind,
		URL:                      canonical.URL,
		StreetAddress:            canonical.Address,
		City:                     canonical.City,
		Postal:                   canonical.Postal,
		Latitude:                 canonical.Latitude,
		Longitude:                canonical.Longitude,
		AskingPrice:              row.AdPrice,
		DebtFreePrice:            int64Path(payload, "priceData", "priceDebtFree"),
		DebtShareAmount:          int64Path(payload, "priceData", "debtShare"),
		PricePerSquareMeter:      float64Path(payload, "priceData", "pricePerSqm"),
		AreaM2:                   canonical.Area,
		RoomLayout:               canonical.RoomLayout,
		RoomsCount:               int32Path(payload, "adData", "rooms"),
		FloorLevel:               int32Path(payload, "adData", "floor"),
		TotalFloors:              firstInt32(int32Path(payload, "adData", "totalFloors"), int32Path(payload, "buildingData", "floors")),
		BuildYear:                firstInt32(int32Path(payload, "adData", "constructionYear"), int32Path(payload, "buildingData", "year")),
		Condition:                firstNonEmpty(valueAtPath(payload, "adData", "condition"), valueAtPath(payload, "property", "condition")),
		EnergyClass:              firstNonEmpty(valueAtPath(payload, "adData", "energyClass"), valueAtPath(payload, "property", "energyClass")),
		PlotType:                 firstNonEmpty(valueAtPath(payload, "adData", "plotType"), valueAtPath(payload, "property", "plotType")),
		Elevator:                 boolPath(payload, "adData", "hasElevator"),
		Sauna:                    boolPath(payload, "adData", "hasSauna"),
		MaintenanceChargeMonthly: firstFloat64(float64Path(payload, "priceData", "maintenanceCharge"), float64Path(payload, "priceData", "monthlyFee")),
		TotalChargeMonthly:       firstFloat64(float64Path(payload, "priceData", "totalCharge"), float64Path(payload, "priceData", "monthlyFee")),
		WaterCharge:              float64Path(payload, "priceData", "waterFee"),
		DescriptionText:          firstNonEmpty(valueAtPath(payload, "adData", "description"), valueAtPath(payload, "description"), valueAtPath(payload, "text")),
		AvailabilityText:         firstNonEmpty(valueAtPath(payload, "adData", "availabilityDescription"), valueAtPath(payload, "availabilityDescription"), valueAtPath(payload, "adData", "availableFrom")),
		RenovationsDoneText:      firstNonEmpty(valueAtPath(payload, "adData", "renovationsDoneDescription"), valueAtPath(payload, "property", "renovationsDoneDescription")),
		RenovationsPlannedText:   firstNonEmpty(valueAtPath(payload, "adData", "renovationsPlannedDescription"), valueAtPath(payload, "property", "renovationsPlannedDescription")),
		AdditionalInfoText:       firstNonEmpty(valueAtPath(payload, "adData", "additionalInfo"), valueAtPath(payload, "moreInformationAvailableFrom"), valueAtPath(payload, "property", "otherInfo")),
		ChargesText:              firstNonEmpty(valueAtPath(payload, "priceData", "chargesText"), valueAtPath(payload, "priceData", "additionalInfo"), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), valueAtPath(payload, "property", "managementChargesAdditionalInfo")),
	}
}

func normalizedFromFrontdoorAdDetail(canonicalID, source, kind string, canonical UnifiedCanonicalFields, row db.GetFrontdoorAdUnifiedDetailRow) NormalizedDetailFields {
	payload := parseFrontdoorJSONMap(row.FrontdoorAdData)
	return NormalizedDetailFields{
		CanonicalID:              canonicalID,
		Source:                   source,
		Kind:                     kind,
		URL:                      canonical.URL,
		StreetAddress:            canonical.Address,
		City:                     canonical.City,
		Postal:                   canonical.Postal,
		Latitude:                 canonical.Latitude,
		Longitude:                canonical.Longitude,
		AskingPrice:              row.AdPrice,
		DebtFreePrice:            int64Path(payload, "debfFreePrice"),
		DebtShareAmount:          int64Path(payload, "debtShareAmount"),
		PricePerSquareMeter:      float64Path(payload, "pricePerSquareMeter"),
		AreaM2:                   canonical.Area,
		RoomLayout:               canonical.RoomLayout,
		RoomsCount:               int32Path(payload, "residenceDetailsDTO", "totalRoomCount"),
		FloorLevel:               int32Path(payload, "residenceDetailsDTO", "housingCompanyApartmentInformationDTO", "floorLevel"),
		TotalFloors:              firstInt32(int32Path(payload, "property", "housingCompany", "floorCount"), int32Path(payload, "residenceDetailsDTO", "floorCount")),
		BuildYear:                firstInt32(int32Path(payload, "residenceDetailsDTO", "constructionFinishedYear"), int32Path(payload, "property", "housingCompany", "usageStartYear")),
		Condition:                firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "inspection", "overallCondition"), strings.TrimSpace(valueOrEmpty(row.AdCondition)), valueAtPath(payload, "property", "condition")),
		EnergyClass:              firstNonEmpty(valueAtPath(payload, "property", "housingCompany", "energyCertificate", "energyCertificateType"), valueAtPath(payload, "property", "energyCertificate", "energyCertificateType")),
		PlotType:                 firstNonEmpty(valueAtPath(payload, "property", "plot", "plotType"), valueAtPath(payload, "property", "plot", "holdingType")),
		Elevator:                 boolPath(payload, "property", "housingCompany", "hasElevator"),
		Sauna:                    boolPath(payload, "property", "housingCompany", "hasSauna"),
		MaintenanceChargeMonthly: periodicCharge(payload, "HOUSING_COMPANY_MAINTENANCE_CHARGE"),
		TotalChargeMonthly:       periodicCharge(payload, "HOUSING_COMPANY_TOTAL_CHARGE"),
		WaterCharge:              periodicCharge(payload, "WATER"),
		DescriptionText:          firstNonEmpty(valueAtPath(payload, "text"), valueAtPath(payload, "property", "description")),
		AvailabilityText:         valueAtPath(payload, "availabilityDescription"),
		RenovationsDoneText:      firstNonEmpty(valueAtPath(payload, "property", "renovationsDoneDescription"), valueAtPath(payload, "property", "housingCompany", "renovationsDoneDescription")),
		RenovationsPlannedText:   firstNonEmpty(valueAtPath(payload, "property", "renovationsPlannedDescription"), valueAtPath(payload, "property", "housingCompany", "renovationsPlannedDescription")),
		AdditionalInfoText:       firstNonEmpty(valueAtPath(payload, "moreInformationAvailableFrom"), valueAtPath(payload, "property", "housingCompany", "otherInfo"), valueAtPath(payload, "additionalItemsIncludedInSale")),
		ChargesText:              firstNonEmpty(valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), valueAtPath(payload, "property", "managementChargesAdditionalInfo")),
	}
}

func parseJSONMap(payload []byte) map[string]any {
	if len(payload) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return out
}

func parseShortcutJSONMap(payload []byte) map[string]any {
	_, out, err := shortcutpayload.DecodeStoredAd(payload)
	if err != nil {
		return nil
	}
	return map[string]any(out)
}

func parseFrontdoorJSONMap(payload []byte) map[string]any {
	_, out, err := frontdoorpayload.DecodeStoredAd(payload)
	if err != nil {
		return nil
	}
	return map[string]any(out)
}

func valueAtPath(value any, path ...string) string {
	current := value
	for _, part := range path {
		m, ok := current.(map[string]any)
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
		return strings.TrimSpace(v)
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

func float64Path(value any, path ...string) *float64 {
	raw := valueAtPath(value, path...)
	if raw == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func int64Path(value any, path ...string) *int64 {
	raw := valueAtPath(value, path...)
	if raw == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	v := int64(parsed)
	return &v
}

func int32Path(value any, path ...string) *int32 {
	raw := valueAtPath(value, path...)
	if raw == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	v := int32(parsed)
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

func periodicCharge(value any, charge string) *float64 {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	property, ok := root["property"].(map[string]any)
	if !ok {
		return nil
	}
	items, ok := property["periodicCharges"].([]any)
	if !ok {
		return nil
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key, ok := m["periodicCharge"].(string)
		if !ok || key != charge {
			continue
		}
		price, ok := m["price"].(float64)
		if !ok {
			continue
		}
		return &price
	}
	return nil
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

func valueOrEmpty[T interface{ ~string | *string }](value T) string {
	switch v := any(value).(type) {
	case string:
		return strings.TrimSpace(v)
	case *string:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(*v)
	default:
		return ""
	}
}

//go:fix inline
func int64Ptr(value int64) *int64 {
	return new(value)
}

func int64PtrIf(ok bool, value int64) *int64 {
	if !ok {
		return nil
	}
	return &value
}

func ptrInt64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func ptrInt32Value(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

//go:fix inline
func float64Ptr(value float64) *float64 {
	return new(value)
}

//go:fix inline
func timePtr(value time.Time) *time.Time {
	return new(value)
}

func timePtrValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func uuidPtrString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func addressListingTexts(availability, renovationsDone, renovationsPlanned, additionalInfo, charges string) *AddressListingTexts {
	texts := AddressListingTexts{
		Availability:       strings.TrimSpace(availability),
		RenovationsDone:    strings.TrimSpace(renovationsDone),
		RenovationsPlanned: strings.TrimSpace(renovationsPlanned),
		AdditionalInfo:     strings.TrimSpace(additionalInfo),
		Charges:            strings.TrimSpace(charges),
	}
	if texts.Availability == "" && texts.RenovationsDone == "" && texts.RenovationsPlanned == "" && texts.AdditionalInfo == "" && texts.Charges == "" {
		return nil
	}
	return &texts
}

func parseAddressInsights(raw json.RawMessage) []AddressInsight {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var out []AddressInsight
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func addressSourceRecordFromListing(listing AddressListing) AddressSourceRecord {
	return AddressSourceRecord{ListingID: listing.ListingID, CanonicalID: listing.CanonicalID, Source: listing.Source, Kind: listing.Kind, NativeID: listing.NativeID, Headline: listing.Headline, Address: listing.Address, City: listing.City, Postal: listing.Postal, Latitude: listing.Latitude, Longitude: listing.Longitude, AskingPrice: listing.AskingPrice, DebtFreePrice: listing.DebtFreePrice, Area: listing.Area, RoomLayout: listing.RoomLayout, URL: listing.URL, ExternalURLAvailable: listing.ExternalURLAvailable, FirstSeenAt: listing.FirstSeenAt, LastSeenAt: listing.LastSeenAt, UpdatedAt: listing.UpdatedAt, PreviousAskingPrice: listing.PreviousAskingPrice, PreviousDebtFreePrice: listing.PreviousDebtFreePrice, Texts: listing.Texts, Insights: listing.Insights}
}

func addressOfferingInsights(records []AddressSourceRecord) []AddressInsight {
	seen := map[string]struct{}{}
	out := []AddressInsight{}
	for _, record := range records {
		for _, insight := range record.Insights {
			key := record.ListingID + ":" + insight.SourceField + ":" + insight.Key + ":" + insight.Value
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, insight)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func representativeAddressSourceRecord(records []AddressSourceRecord) AddressSourceRecord {
	if len(records) == 0 {
		return AddressSourceRecord{}
	}
	return sortAddressSourceRecords(records)[0]
}

func sortAddressSourceRecords(records []AddressSourceRecord) []AddressSourceRecord {
	out := slices.Clone(records)
	slices.SortFunc(out, func(a, b AddressSourceRecord) int {
		if cmp := compareTimeDesc(a.LastSeenAt, b.LastSeenAt); cmp != 0 {
			return cmp
		}
		if a.Source != b.Source {
			return strings.Compare(a.Source, b.Source)
		}
		return strings.Compare(a.NativeID, b.NativeID)
	})
	return out
}

func appendUniqueAddressSourceRecords(dst []AddressSourceRecord, src []AddressSourceRecord) []AddressSourceRecord {
	seen := map[string]struct{}{}
	out := make([]AddressSourceRecord, 0, len(dst)+len(src))
	for _, record := range append(dst, src...) {
		key := record.ListingID
		if key == "" {
			key = record.Source + ":" + record.Kind + ":" + record.NativeID
		}
		if key == "::" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, record)
	}
	return out
}

func appendUniqueAddressTransactions(dst []AddressTransactionLink, src []AddressTransactionLink) []AddressTransactionLink {
	seen := map[string]struct{}{}
	out := make([]AddressTransactionLink, 0, len(dst)+len(src))
	for _, transaction := range append(dst, src...) {
		key := transaction.TransactionID + ":" + transaction.LinkType
		if key == ":" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, transaction)
	}
	return out
}

func addressOfferingSourceLabels(records []AddressSourceRecord) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, record := range records {
		if record.Source == "" {
			continue
		}
		if _, ok := seen[record.Source]; ok {
			continue
		}
		seen[record.Source] = struct{}{}
		out = append(out, record.Source)
	}
	slices.Sort(out)
	return out
}

func earliestTimePtr(values ...*time.Time) *time.Time {
	var out *time.Time
	for _, value := range values {
		if value != nil && (out == nil || value.Before(*out)) {
			out = value
		}
	}
	return out
}

func latestTimePtr(values ...*time.Time) *time.Time {
	var out *time.Time
	for _, value := range values {
		if value != nil && (out == nil || value.After(*out)) {
			out = value
		}
	}
	return out
}

func compareTimeDesc(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	if a.After(*b) {
		return -1
	}
	if a.Before(*b) {
		return 1
	}
	return 0
}

func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func valueOrFalse(value *bool) bool {
	return value != nil && *value
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

func int32Ptr(value int32) *int32 {
	if value == 0 {
		return nil
	}
	return &value
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

func ptrUUIDToString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
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
