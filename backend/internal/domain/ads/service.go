package ads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Listings        []AddressListing        `json:"listings"`
	RawTransactions []AddressRawTransaction `json:"raw_transactions"`
}

type AddressListing struct {
	ListingID             string                   `json:"listing_id"`
	CanonicalID           string                   `json:"canonical_id"`
	Source                string                   `json:"source"`
	Kind                  string                   `json:"kind"`
	NativeID              string                   `json:"native_id"`
	Headline              string                   `json:"headline"`
	Address               string                   `json:"address"`
	City                  string                   `json:"city,omitempty"`
	Postal                string                   `json:"postal,omitempty"`
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
	source := stringPtr(normalized.Source)
	kind := stringPtr(normalized.Kind)
	sort := stringPtr(normalized.Sort)
	listingTypeFilter := emptyToNil(normalized.ListingType)
	if normalized.ListingType == "all" {
		listingTypeFilter = nil
	}
	publishedAfter := normalized.PublishedAfter
	publishedBefore := normalized.PublishedBefore
	count, err := s.queries.CountUnifiedEntities(ctx, db.CountUnifiedEntitiesParams{
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
	rows, err := s.queries.SearchUnifiedEntities(ctx, db.SearchUnifiedEntitiesParams{
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
			CanonicalID: strings.TrimSpace(row.CanonicalID),
			Source:      row.Source,
			Kind:        row.Kind,
			NativeID:    row.NativeID,
			Headline:    strings.TrimSpace(row.Headline),
			Address:     strings.TrimSpace(row.Address),
			City:        strings.TrimSpace(row.City),
			Postal:      strings.TrimSpace(row.Postal),
			Price:       int64Ptr(row.Price),
			Area:        float64Ptr(row.Area),
			RoomLayout:  strings.TrimSpace(row.RoomLayout),
			URL:         strings.TrimSpace(row.Url),
			LastSeenAt:  row.LastSeenAt,
		})
	}
	return ReportPage{Rows: mapped, Total: count, Page: normalized.Page, PageSize: normalized.PageSize}, nil
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
	rows, err := s.db.Query(ctx, addressLookupSQL, queryAddress, city, postal, source, limit)
	if err != nil {
		return AddressLookupResult{}, fmt.Errorf("lookup address listings: %w", err)
	}
	defer rows.Close()
	lookupRows := []addressLookupRow{}
	for rows.Next() {
		var row addressLookupRow
		if err := rows.Scan(&row.ListingID, &row.CanonicalID, &row.Source, &row.Kind, &row.NativeID, &row.Headline, &row.Address, &row.City, &row.Postal, &row.AskingPrice, &row.DebtFreePrice, &row.Area, &row.RoomLayout, &row.URL, &row.ExternalURLAvailable, &row.FirstSeenAt, &row.LastSeenAt, &row.PublishedAt, &row.CreatedAt, &row.UpdatedAt, &row.PreviousAskingPrice, &row.PreviousDebtFreePrice, &row.PriceMatchStatus, &row.SourceMatchStatus, &row.OfferingID, &row.AvailabilityText, &row.RenovationsDoneText, &row.RenovationsPlannedText, &row.AdditionalInfoText, &row.ChargesText, &row.TransactionID, &row.LinkType, &row.LinkStatus, &row.LinkMethod, &row.Score, &row.Confidence, &row.PriceDeltaPercent, &row.Reasons, &row.TransactionDescription, &row.TransactionType, &row.TransactionCategory, &row.TransactionArea, &row.TransactionPrice, &row.TransactionPricePerM2, &row.TransactionBuildYear, &row.TransactionFloor, &row.TransactionElevator, &row.TransactionCondition, &row.TransactionPlot, &row.TransactionEnergyClass, &row.TransactionPeriodIdentifier, &row.TransactionCity, &row.TransactionNeighborhood, &row.TransactionPostal, &row.TransactionCreatedAt, &row.TransactionUpdatedAt, &row.SourceRecordListingID, &row.SourceRecordCanonicalID, &row.SourceRecordSource, &row.SourceRecordKind, &row.SourceRecordNativeID, &row.SourceRecordHeadline, &row.SourceRecordAddress, &row.SourceRecordCity, &row.SourceRecordPostal, &row.SourceRecordAskingPrice, &row.SourceRecordDebtFreePrice, &row.SourceRecordArea, &row.SourceRecordRoomLayout, &row.SourceRecordURL, &row.SourceRecordExternalURLAvailable, &row.SourceRecordFirstSeenAt, &row.SourceRecordLastSeenAt, &row.SourceRecordUpdatedAt, &row.SourceRecordPreviousAsk, &row.SourceRecordPreviousDebt, &row.SourceRecordLinkStatus, &row.SourceRecordLinkMethod, &row.SourceRecordLinkScore, &row.SourceRecordAvailability, &row.SourceRecordRenovationsDone, &row.SourceRecordRenovationsPlan, &row.SourceRecordAdditionalInfo, &row.SourceRecordCharges); err != nil {
			return AddressLookupResult{}, fmt.Errorf("scan address listing: %w", err)
		}
		lookupRows = append(lookupRows, row)
	}
	if err := rows.Err(); err != nil {
		return AddressLookupResult{}, fmt.Errorf("iterate address listings: %w", err)
	}
	result := buildAddressLookupResult(queryAddress, city, postal, source, lookupRows)
	if err := s.attachAddressSourceCandidates(ctx, &result); err != nil {
		return AddressLookupResult{}, err
	}
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
			row, err := s.queries.GetShortcutAdUnifiedDetail(ctx, adID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: shortcut ad", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), strconv.FormatInt(row.ShortcutAdID, 10)), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Price: row.AdPrice, Area: row.AdArea, RoomLayout: strings.TrimSpace(row.AdRoomLayout), URL: strings.TrimSpace(row.ShortcutAdUrl), ExternalURLAvailable: strings.TrimSpace(row.ShortcutAdUrl) != "" && row.ShortcutAdLastSeenAt.After(time.Now().AddDate(0, 0, -7)), LastSeenAt: row.ShortcutAdLastSeenAt}}
			detail.Normalized = normalizedFromShortcutAdDetail(canonicalID, source, kind, detail.Canonical, row)
			detail.SourceSpecific = []DetailField{{Label: "Ad Type", Value: row.ShortcutAdType}, {Label: "Building ID", Value: ptrUUIDToString(row.ShortcutBuildingID)}, {Label: "Building External ID", Value: formatInt64Ptr(row.ShortcutBuildingExternalID)}, {Label: "Building Address", Value: valueOrEmpty(row.ShortcutBuildingAddress)}, {Label: "Housing Company", Value: valueOrEmpty(row.ShortcutBuildingHousingCompany)}, {Label: "Building URL", Value: valueOrEmpty(row.ShortcutBuildingUrl)}}
			detail.Related = []DetailField{{Label: "Building Listings", Value: strconv.FormatInt(row.BuildingListingCount, 10)}, {Label: "Building Rentals", Value: strconv.FormatInt(row.BuildingRentalCount, 10)}}
			detail.Raw = buildRawPayload(row.ShortcutAdData)
			detail = promoteCanonicalFields(detail, "Ad Type", "Building ID", "Building External ID", "Housing Company")
			return cleanDetail(detail), nil
		case "building":
			buildingID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse shortcut building id: %w", err)
			}
			row, err := s.queries.GetShortcutBuildingUnifiedDetail(ctx, buildingID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: shortcut building", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get shortcut building detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.ShortcutBuildingAddress), valueOrEmpty(row.ShortcutBuildingHousingCompany), formatInt64Value(row.ShortcutBuildingExternalID)), Address: valueOrEmpty(row.ShortcutBuildingAddress), URL: strings.TrimSpace(row.ShortcutBuildingUrl), ExternalURLAvailable: row.ShortcutBuildingPageNotFound != nil && !*row.ShortcutBuildingPageNotFound, LastSeenAt: firstTimeValue(row.ShortcutBuildingUpdatedAt, row.ShortcutBuildingProcessedAt)}}
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
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor ad", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor ad detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.AdAddress), row.FrontdoorAdExternalID), Address: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Price: row.AdPrice, Area: row.AdArea, RoomLayout: strings.TrimSpace(row.AdRoomLayout), URL: strings.TrimSpace(row.FrontdoorAdUrl), ExternalURLAvailable: !row.FrontdoorAdPageNotFound, LastSeenAt: row.FrontdoorAdLastSeenAt}}
			detail.Normalized = normalizedFromFrontdoorAdDetail(canonicalID, source, kind, detail.Canonical, row)
			detail.SourceSpecific = []DetailField{{Label: "External ID", Value: row.FrontdoorAdExternalID}, {Label: "Property Type", Value: strings.TrimSpace(row.AdPropertyType)}, {Label: "Condition", Value: strings.TrimSpace(row.AdCondition)}, {Label: "Page Not Found", Value: formatBool(row.FrontdoorAdPageNotFound)}}
			detail.Raw = buildRawPayload(row.FrontdoorAdData)
			detail = promoteCanonicalFields(detail, "External ID", "Property Type", "Condition")
			return cleanDetail(detail), nil
		case "announcement":
			announcementID, err := uuid.Parse(nativeID)
			if err != nil {
				return UnifiedEntityDetail{}, fmt.Errorf("parse frontdoor announcement id: %w", err)
			}
			row, err := s.queries.GetFrontdoorAnnouncementUnifiedDetail(ctx, announcementID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor announcement", ErrNotFound)
				}
				return UnifiedEntityDetail{}, fmt.Errorf("get frontdoor announcement detail: %w", err)
			}
			detail := UnifiedEntityDetail{Canonical: UnifiedCanonicalFields{CanonicalID: canonicalID, Source: source, Kind: kind, NativeID: nativeID, Headline: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), formatInt32(row.FrontdoorBuildingAnnouncementExternalID)), Address: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Price: float64ToInt64Ptr(row.FrontdoorBuildingAnnouncementSearchPrice), Area: row.FrontdoorBuildingAnnouncementArea, RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), URL: valueOrEmpty(row.FrontdoorBuildingUrl), ExternalURLAvailable: boolPtrValue(row.FrontdoorBuildingAnnouncementPublished), LastSeenAt: row.FrontdoorBuildingAnnouncementLastSeenAt}}
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
			row, err := s.queries.GetFrontdoorBuildingUnifiedDetail(ctx, buildingID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return UnifiedEntityDetail{}, fmt.Errorf("%w: frontdoor building", ErrNotFound)
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
	AvailabilityText                 string
	RenovationsDoneText              string
	RenovationsPlannedText           string
	AdditionalInfoText               string
	ChargesText                      string
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

func buildAddressLookupResult(address, city, postal, source string, rows []addressLookupRow) AddressLookupResult {
	result := AddressLookupResult{Address: address, City: city, Postal: postal, Source: source, Listings: []AddressListing{}, RawTransactions: []AddressRawTransaction{}}
	index := map[uuid.UUID]int{}
	sourceRecordsByOffering := map[uuid.UUID][]AddressSourceRecord{}
	seenSourceRecords := map[uuid.UUID]map[uuid.UUID]struct{}{}
	seenTransactions := map[string]struct{}{}
	for _, row := range rows {
		if row.OfferingID != nil {
			seen := seenSourceRecords[*row.OfferingID]
			if seen == nil {
				seen = map[uuid.UUID]struct{}{}
				seenSourceRecords[*row.OfferingID] = seen
			}
			sourceRecordID := row.ListingID
			sourceRecord := AddressSourceRecord{ListingID: row.ListingID.String(), CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, Headline: row.Headline, Address: row.Address, City: row.City, Postal: row.Postal, AskingPrice: row.AskingPrice, DebtFreePrice: row.DebtFreePrice, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, UpdatedAt: row.UpdatedAt, PreviousAskingPrice: row.PreviousAskingPrice, PreviousDebtFreePrice: row.PreviousDebtFreePrice, Texts: addressListingTexts(row.AvailabilityText, row.RenovationsDoneText, row.RenovationsPlannedText, row.AdditionalInfoText, row.ChargesText)}
			if row.SourceRecordListingID != nil {
				sourceRecordID = *row.SourceRecordListingID
				sourceRecord = AddressSourceRecord{ListingID: row.SourceRecordListingID.String(), CanonicalID: row.SourceRecordCanonicalID, Source: row.SourceRecordSource, Kind: row.SourceRecordKind, NativeID: row.SourceRecordNativeID, Headline: row.SourceRecordHeadline, Address: row.SourceRecordAddress, City: row.SourceRecordCity, Postal: row.SourceRecordPostal, AskingPrice: row.SourceRecordAskingPrice, DebtFreePrice: row.SourceRecordDebtFreePrice, Area: row.SourceRecordArea, RoomLayout: row.SourceRecordRoomLayout, URL: row.SourceRecordURL, ExternalURLAvailable: row.SourceRecordExternalURLAvailable, FirstSeenAt: row.SourceRecordFirstSeenAt, LastSeenAt: row.SourceRecordLastSeenAt, UpdatedAt: row.SourceRecordUpdatedAt, PreviousAskingPrice: row.SourceRecordPreviousAsk, PreviousDebtFreePrice: row.SourceRecordPreviousDebt, LinkStatus: row.SourceRecordLinkStatus, LinkMethod: row.SourceRecordLinkMethod, LinkScore: row.SourceRecordLinkScore, Texts: addressListingTexts(row.SourceRecordAvailability, row.SourceRecordRenovationsDone, row.SourceRecordRenovationsPlan, row.SourceRecordAdditionalInfo, row.SourceRecordCharges)}
			}
			if _, ok := seen[sourceRecordID]; !ok {
				seen[sourceRecordID] = struct{}{}
				sourceRecordsByOffering[*row.OfferingID] = append(sourceRecordsByOffering[*row.OfferingID], sourceRecord)
			}
		}
		listingIndex, ok := index[row.ListingID]
		if !ok {
			listing := AddressListing{ListingID: row.ListingID.String(), CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, Headline: row.Headline, Address: row.Address, City: row.City, Postal: row.Postal, AskingPrice: row.AskingPrice, DebtFreePrice: row.DebtFreePrice, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, PublishedAt: row.PublishedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, PreviousAskingPrice: row.PreviousAskingPrice, PreviousDebtFreePrice: row.PreviousDebtFreePrice, PriceMatchStatus: row.PriceMatchStatus, SourceMatchStatus: row.SourceMatchStatus, OfferingID: uuidPtrString(row.OfferingID), Texts: addressListingTexts(row.AvailabilityText, row.RenovationsDoneText, row.RenovationsPlannedText, row.AdditionalInfoText, row.ChargesText), SourceRecords: []AddressSourceRecord{}, SourceCandidates: []AddressSourceCandidate{}, Transactions: []AddressTransactionLink{}}
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
	return result
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
	rows, err := s.db.Query(ctx, addressSourceCandidatesSQL, listingIDs)
	if err != nil {
		return fmt.Errorf("lookup address source candidates: %w", err)
	}
	defer rows.Close()
	candidates := []addressSourceCandidateRow{}
	for rows.Next() {
		var row addressSourceCandidateRow
		if err := rows.Scan(&row.SelectedListingID, &row.CandidateListingID, &row.CanonicalID, &row.Source, &row.Kind, &row.NativeID, &row.Headline, &row.Address, &row.City, &row.Postal, &row.AskingPrice, &row.DebtFreePrice, &row.Area, &row.RoomLayout, &row.URL, &row.ExternalURLAvailable, &row.SelectedOfferingID, &row.CandidateOfferingID, &row.Direction, &row.Status, &row.Score, &row.Confidence, &row.PriceDeltaPercent, &row.Reasons, &row.CreatedAt); err != nil {
			return fmt.Errorf("scan address source candidate: %w", err)
		}
		candidates = append(candidates, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate address source candidates: %w", err)
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
	rows, err := s.db.Query(ctx, addressRawTransactionsSQL, city, postal, linkedTransactionIDs, candidateTransactionIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("lookup address raw transactions: %w", err)
	}
	defer rows.Close()
	transactions := []AddressRawTransaction{}
	for rows.Next() {
		var row addressRawTransactionRow
		if err := rows.Scan(&row.TransactionID, &row.Description, &row.Type, &row.Category, &row.Area, &row.Price, &row.PricePerSquareMeter, &row.BuildYear, &row.Floor, &row.Elevator, &row.Condition, &row.Plot, &row.EnergyClass, &row.PeriodIdentifier, &row.City, &row.Neighborhood, &row.Postal, &row.CreatedAt, &row.UpdatedAt, &row.IsMatched, &row.LinkedToLookup, &row.CandidateToLookup, &row.MatchedListingCount, &row.MatchedOfferingCount, &row.Matches); err != nil {
			return nil, fmt.Errorf("scan address raw transaction: %w", err)
		}
		matches, err := decodeRawTransactionMatches(row.Matches)
		if err != nil {
			return nil, fmt.Errorf("decode address raw transaction matches: %w", err)
		}
		transactions = append(transactions, AddressRawTransaction{TransactionID: row.TransactionID.String(), Description: row.Description, Type: row.Type, Category: row.Category, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, BuildYear: row.BuildYear, Floor: row.Floor, Elevator: row.Elevator, Condition: row.Condition, Plot: row.Plot, EnergyClass: row.EnergyClass, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Neighborhood: row.Neighborhood, Postal: row.Postal, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, IsMatched: row.IsMatched, LinkedToLookup: row.LinkedToLookup, CandidateToLookup: row.CandidateToLookup, Scope: rawTransactionScope(row.LinkedToLookup, row.CandidateToLookup, row.IsMatched), MatchedListingCount: row.MatchedListingCount, MatchedOfferingCount: row.MatchedOfferingCount, Matches: matches})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate address raw transactions: %w", err)
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

const addressLookupSQL = `
WITH lookup_input AS (
    SELECT
        public.fnc__normalize_address_token($1::text) AS address_norm,
        translate(public.fnc__normalize_address_token($1::text), 'åäö', 'aao') AS address_ascii_norm,
        substring(public.fnc__normalize_address_token($1::text) from '^(.*)\s+[0-9]+(\s*[[:alpha:]])?\s*$') AS street_name_norm,
        substring(translate(public.fnc__normalize_address_token($1::text), 'åäö', 'aao') from '^(.*)\s+[0-9]+(\s*[[:alpha:]])?\s*$') AS street_name_ascii_norm,
        substring(public.fnc__normalize_address_token($1::text) from '\s([0-9]+)(\s*[[:alpha:]])?\s*$') AS street_number_norm,
        substring(translate(public.fnc__normalize_address_token($1::text), 'åäö', 'aao') from '\s[0-9]+\s*([[:alpha:]])\s*$') AS building_letter_ascii_norm,
        public.fnc__normalize_postal($3::text) AS postal_norm
),
selected_listing_matches AS (
    SELECT
        sl.sale_listing_id,
        pos.property_offering_id
    FROM public.property_source_offerings sl
    CROSS JOIN lookup_input li
    LEFT JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    WHERE ($4::text = 'all' OR sl.sale_listing_source_provider = $4::text)
        AND sl.sale_listing_source_kind = ANY(ARRAY['ad'::text, 'announcement'::text])
        AND trim($3::text) <> ''
        AND COALESCE(sl.sale_listing_postal_norm, public.fnc__normalize_postal(sl.sale_listing_postal)) = li.postal_norm
        AND (
            sl.sale_listing_address_norm = li.address_norm
            OR translate(sl.sale_listing_address_norm, 'åäö', 'aao') = li.address_ascii_norm
            OR lower(COALESCE(sl.sale_listing_street_address, '')) LIKE ('%' || lower(trim($1::text)) || '%')
            OR translate(lower(COALESCE(sl.sale_listing_street_address, '')), 'åäö', 'aao') LIKE ('%' || li.address_ascii_norm || '%')
            OR (
                sl.sale_listing_street_name_norm IS NOT NULL
                AND sl.sale_listing_street_number_norm IS NOT NULL
                AND (
                    li.building_letter_ascii_norm IS NULL
                    OR translate(COALESCE(sl.sale_listing_building_letter_norm, ''), 'åäö', 'aao') = li.building_letter_ascii_norm
                )
                AND (
                    (' ' || li.address_norm || ' ')
                        LIKE ('% ' || sl.sale_listing_street_name_norm || ' ' || sl.sale_listing_street_number_norm || ' %')
                    OR (' ' || li.address_ascii_norm || ' ')
                        LIKE ('% ' || translate(sl.sale_listing_street_name_norm, 'åäö', 'aao') || ' ' || sl.sale_listing_street_number_norm || ' %')
                )
            )
        )
        AND (trim($2::text) = '' OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim($2::text)) || '%'))
    UNION ALL
    SELECT
        sl.sale_listing_id,
        pos.property_offering_id
    FROM public.property_source_offerings sl
    CROSS JOIN lookup_input li
    LEFT JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    WHERE ($4::text = 'all' OR sl.sale_listing_source_provider = $4::text)
        AND sl.sale_listing_source_kind = ANY(ARRAY['ad'::text, 'announcement'::text])
        AND trim($3::text) = ''
        AND li.street_name_ascii_norm IS NOT NULL
        AND li.street_number_norm IS NOT NULL
        AND translate(sl.sale_listing_street_name_norm, 'åäö', 'aao') = li.street_name_ascii_norm
        AND sl.sale_listing_street_number_norm = li.street_number_norm
        AND (
            li.building_letter_ascii_norm IS NULL
            OR translate(COALESCE(sl.sale_listing_building_letter_norm, ''), 'åäö', 'aao') = li.building_letter_ascii_norm
        )
        AND (trim($2::text) = '' OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim($2::text)) || '%'))
),
selected_listing_ids AS (
    SELECT DISTINCT ON (sale_listing_id)
        sale_listing_id,
        property_offering_id
    FROM selected_listing_matches
    ORDER BY sale_listing_id, property_offering_id NULLS LAST
),
selected_listings AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_canonical_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        sl.sale_listing_native_id,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
        COALESCE(sl.sale_listing_street_address, '') AS address,
        COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
        COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
        sl.sale_listing_asking_price,
        sl.sale_listing_debt_free_price,
        sl.sale_listing_area_value,
        COALESCE(sl.sale_listing_room_layout, '') AS room_layout,
        COALESCE(sl.sale_listing_url, '') AS url,
        CASE
            WHEN sl.sale_listing_source_provider = 'shortcut' AND sl.sale_listing_source_kind = 'ad' THEN sl.shortcut_ad_id IS NOT NULL AND COALESCE(sl.sale_listing_url, '') <> '' AND sl.sale_listing_last_seen_at >= now() - interval '7 days'
            WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
            WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
            ELSE false
        END AS external_url_available,
        sl.sale_listing_first_seen_at,
        sl.sale_listing_last_seen_at,
        sl.sale_listing_published_at,
        sl.sale_listing_created_at,
        sl.sale_listing_updated_at,
        sl.sale_listing_previous_asking_price,
        sl.sale_listing_previous_debt_free_price,
        COALESCE(sl.sale_listing_prices_match_status, '') AS prices_match_status,
        COALESCE(sl.sale_listing_source_match_status, '') AS source_match_status,
        sli.property_offering_id,
        COALESCE(sl.sale_listing_availability_text, '') AS availability_text,
        COALESCE(sl.sale_listing_renovations_done_text, '') AS renovations_done_text,
        COALESCE(sl.sale_listing_renovations_planned_text, '') AS renovations_planned_text,
        COALESCE(sl.sale_listing_additional_info_text, '') AS additional_info_text,
        COALESCE(sl.sale_listing_charges_text, '') AS charges_text,
        ROW_NUMBER() OVER (
            ORDER BY
                CASE
                    WHEN sl.sale_listing_address_norm = li.address_norm THEN 0
                    WHEN translate(sl.sale_listing_address_norm, 'åäö', 'aao') = li.address_ascii_norm THEN 1
                    WHEN lower(COALESCE(sl.sale_listing_street_address, '')) = lower(trim($1::text)) THEN 1
                    ELSE 3
                END,
                sl.sale_listing_last_seen_at DESC NULLS LAST,
                sl.sale_listing_source_provider,
                sl.sale_listing_source_kind,
                sl.sale_listing_native_id
        ) AS listing_rank
    FROM selected_listing_ids sli
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = sli.sale_listing_id
    LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    CROSS JOIN lookup_input li
),
limited_listings AS (
    SELECT *
    FROM selected_listings
    WHERE listing_rank <= $5::int
),
matched_offerings AS (
    SELECT DISTINCT property_offering_id
    FROM limited_listings
    WHERE property_offering_id IS NOT NULL
),
offering_source_records AS (
    SELECT
        pos.property_offering_id,
        sr.sale_listing_id,
        sr.sale_listing_canonical_id,
        sr.sale_listing_source_provider,
        sr.sale_listing_source_kind,
        sr.sale_listing_native_id,
        COALESCE(sr.sale_listing_headline, sr.sale_listing_street_address, sr.sale_listing_native_id) AS headline,
        COALESCE(sr.sale_listing_street_address, '') AS address,
        COALESCE(sr.sale_listing_city, sr.sale_listing_city_norm, '') AS city,
        COALESCE(sr.sale_listing_postal, sr.sale_listing_postal_norm, '') AS postal,
        sr.sale_listing_asking_price,
        sr.sale_listing_debt_free_price,
        sr.sale_listing_area_value,
        COALESCE(sr.sale_listing_room_layout, '') AS room_layout,
        COALESCE(sr.sale_listing_url, '') AS url,
        CASE
            WHEN sr.sale_listing_source_provider = 'shortcut' AND sr.sale_listing_source_kind = 'ad' THEN sr.shortcut_ad_id IS NOT NULL AND COALESCE(sr.sale_listing_url, '') <> '' AND sr.sale_listing_last_seen_at >= now() - interval '7 days'
            WHEN sr.sale_listing_source_provider = 'frontdoor' AND sr.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
            WHEN sr.sale_listing_source_provider = 'frontdoor' AND sr.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
            ELSE false
        END AS external_url_available,
        sr.sale_listing_first_seen_at,
        sr.sale_listing_last_seen_at,
        sr.sale_listing_updated_at,
        sr.sale_listing_previous_asking_price,
        sr.sale_listing_previous_debt_free_price,
        sr.prices_transaction_id,
        COALESCE(sr.sale_listing_prices_match_status, '') AS prices_match_status,
        COALESCE(pos.property_offering_source_link_status, '') AS source_link_status,
        COALESCE(pos.property_offering_source_link_method, '') AS source_link_method,
        pos.property_offering_source_link_score,
        COALESCE(sr.sale_listing_availability_text, '') AS availability_text,
        COALESCE(sr.sale_listing_renovations_done_text, '') AS renovations_done_text,
        COALESCE(sr.sale_listing_renovations_planned_text, '') AS renovations_planned_text,
        COALESCE(sr.sale_listing_additional_info_text, '') AS additional_info_text,
        COALESCE(sr.sale_listing_charges_text, '') AS charges_text
    FROM matched_offerings mo
    JOIN public.property_offering_sources pos ON pos.property_offering_id = mo.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
    JOIN public.property_source_offerings sr ON sr.sale_listing_id = pos.sale_listing_id
    LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sr.frontdoor_ad_id
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sr.frontdoor_building_announcement_id
),
latest_candidates AS (
    SELECT DISTINCT ON (c.sale_listing_id, c.prices_transaction_id)
        c.sale_listing_id,
        c.prices_transaction_id,
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence,
        c.sale_listing_prices_transaction_match_status,
        c.sale_listing_prices_transaction_match_reasons,
        c.sale_listing_prices_transaction_match_price_delta_percent
    FROM public.sale_listing_prices_transaction_match_candidates c
    JOIN limited_listings sl ON sl.sale_listing_id = c.sale_listing_id
    ORDER BY c.sale_listing_id, c.prices_transaction_id, c.sale_listing_prices_transaction_match_created_at DESC
),
links AS (
    SELECT
        sl.sale_listing_id,
        sl.prices_transaction_id,
        'direct'::text AS link_type,
        COALESCE(sl.sale_listing_prices_match_status, 'linked') AS link_status,
        'source_listing'::text AS link_method,
        lc.sale_listing_prices_transaction_match_score AS score,
        lc.sale_listing_prices_transaction_match_confidence AS confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent AS price_delta_percent,
        lc.sale_listing_prices_transaction_match_reasons AS reasons,
        1 AS link_rank
    FROM public.property_source_offerings sl
    JOIN limited_listings selected ON selected.sale_listing_id = sl.sale_listing_id
    LEFT JOIN latest_candidates lc ON lc.sale_listing_id = sl.sale_listing_id
        AND lc.prices_transaction_id = sl.prices_transaction_id
    WHERE sl.prices_transaction_id IS NOT NULL
    UNION ALL
    SELECT
        selected.sale_listing_id,
        pot.prices_transaction_id,
        'offering'::text,
        pot.property_offering_transaction_link_status,
        pot.property_offering_transaction_link_method,
        pot.property_offering_transaction_link_score,
        lc.sale_listing_prices_transaction_match_confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent,
        COALESCE(lc.sale_listing_prices_transaction_match_reasons, pot.property_offering_transaction_link_reasons),
        2
    FROM limited_listings selected
    JOIN public.property_offering_transactions pot ON pot.property_offering_id = selected.property_offering_id
        AND pot.property_offering_transaction_link_status <> 'rejected'
    LEFT JOIN latest_candidates lc ON lc.sale_listing_id = selected.sale_listing_id
        AND lc.prices_transaction_id = pot.prices_transaction_id
    UNION ALL
    SELECT
        selected.sale_listing_id,
        osr.prices_transaction_id,
        'source_record'::text,
        COALESCE(NULLIF(osr.prices_match_status, ''), 'linked'),
        'offering_source_listing'::text,
        NULL::integer,
        ''::text,
        NULL::double precision,
        jsonb_build_object(
            'source_listing_id', osr.sale_listing_id,
            'source_listing_provider', osr.sale_listing_source_provider,
            'source_listing_native_id', osr.sale_listing_native_id
        ),
        3
    FROM limited_listings selected
    JOIN offering_source_records osr ON osr.property_offering_id = selected.property_offering_id
    WHERE osr.prices_transaction_id IS NOT NULL
    UNION ALL
    SELECT
        lc.sale_listing_id,
        lc.prices_transaction_id,
        'candidate'::text,
        lc.sale_listing_prices_transaction_match_status,
        'match_candidate'::text,
        lc.sale_listing_prices_transaction_match_score,
        lc.sale_listing_prices_transaction_match_confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent,
        lc.sale_listing_prices_transaction_match_reasons,
        4
    FROM latest_candidates lc
    WHERE lc.sale_listing_prices_transaction_match_status = ANY(ARRAY['candidate'::text, 'ambiguous'::text])
),
dedup_links AS (
    SELECT DISTINCT ON (sale_listing_id, prices_transaction_id)
        *
    FROM links
    ORDER BY sale_listing_id, prices_transaction_id, link_rank, score DESC NULLS LAST
)
SELECT
    sl.sale_listing_id,
    sl.sale_listing_canonical_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.headline,
    sl.address,
    sl.city,
    sl.postal,
    sl.sale_listing_asking_price,
    sl.sale_listing_debt_free_price,
    sl.sale_listing_area_value,
    sl.room_layout,
    sl.url,
    sl.external_url_available,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_published_at,
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at,
    sl.sale_listing_previous_asking_price,
    sl.sale_listing_previous_debt_free_price,
    sl.prices_match_status,
    sl.source_match_status,
    sl.property_offering_id,
    sl.availability_text,
    sl.renovations_done_text,
    sl.renovations_planned_text,
    sl.additional_info_text,
    sl.charges_text,
    pt.prices_transaction_id,
    COALESCE(dl.link_type, ''),
    COALESCE(dl.link_status, ''),
    COALESCE(dl.link_method, ''),
    dl.score,
    COALESCE(dl.confidence, ''),
    dl.price_delta_percent,
    COALESCE(dl.reasons, '{}'::jsonb),
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    pt.prices_transaction_area,
    pt.prices_transaction_price::bigint,
    pt.prices_transaction_price_per_square_meter::bigint,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, ''),
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, ''),
    COALESCE(pt.prices_transaction_plot, ''),
    COALESCE(pt.prices_transaction_energy_class, ''),
    COALESCE(pt.prices_transaction_period_identifier, ''),
    COALESCE(pc.prices_city_name, ''),
    COALESCE(pn.prices_neighborhood_name, ''),
    COALESCE(ppc.prices_postal_code_code, postal.postal_postal_code_code, ''),
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    osr.sale_listing_id,
    COALESCE(osr.sale_listing_canonical_id, ''),
    COALESCE(osr.sale_listing_source_provider, ''),
    COALESCE(osr.sale_listing_source_kind, ''),
    COALESCE(osr.sale_listing_native_id, ''),
    COALESCE(osr.headline, ''),
    COALESCE(osr.address, ''),
    COALESCE(osr.city, ''),
    COALESCE(osr.postal, ''),
    osr.sale_listing_asking_price,
    osr.sale_listing_debt_free_price,
    osr.sale_listing_area_value,
    COALESCE(osr.room_layout, ''),
    COALESCE(osr.url, ''),
    COALESCE(osr.external_url_available, false),
    osr.sale_listing_first_seen_at,
    osr.sale_listing_last_seen_at,
    osr.sale_listing_updated_at,
    osr.sale_listing_previous_asking_price,
    osr.sale_listing_previous_debt_free_price,
    COALESCE(osr.source_link_status, ''),
    COALESCE(osr.source_link_method, ''),
    osr.property_offering_source_link_score,
    COALESCE(osr.availability_text, ''),
    COALESCE(osr.renovations_done_text, ''),
    COALESCE(osr.renovations_planned_text, ''),
    COALESCE(osr.additional_info_text, ''),
    COALESCE(osr.charges_text, '')
FROM limited_listings sl
LEFT JOIN dedup_links dl ON dl.sale_listing_id = sl.sale_listing_id
LEFT JOIN public.prices_transactions pt ON pt.prices_transaction_id = dl.prices_transaction_id
LEFT JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN public.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN public.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN public.postal_postal_codes postal ON postal.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN offering_source_records osr ON osr.property_offering_id = sl.property_offering_id
ORDER BY sl.listing_rank, dl.link_rank NULLS LAST, dl.score DESC NULLS LAST, pt.prices_transaction_created_at DESC NULLS LAST, osr.sale_listing_source_provider, osr.sale_listing_native_id`

const addressSourceCandidatesSQL = `
WITH selected AS (
    SELECT unnest($1::uuid[]) AS sale_listing_id
),
candidates AS (
    SELECT
        selected.sale_listing_id AS selected_sale_listing_id,
        c.target_sale_listing_id AS candidate_sale_listing_id,
        c.source_property_offering_id AS selected_property_offering_id,
        c.target_property_offering_id AS candidate_property_offering_id,
        'source_to_target'::text AS direction,
        c.property_offering_source_match_score,
        c.property_offering_source_match_confidence,
        c.property_offering_source_match_status,
        c.property_offering_source_match_reasons,
        c.property_offering_source_match_price_delta_percent,
        c.property_offering_source_match_created_at
    FROM selected
    JOIN public.property_offering_source_match_candidates c ON c.source_sale_listing_id = selected.sale_listing_id
    WHERE c.property_offering_source_match_status <> 'rejected'
    UNION ALL
    SELECT
        selected.sale_listing_id AS selected_sale_listing_id,
        c.source_sale_listing_id AS candidate_sale_listing_id,
        c.target_property_offering_id AS selected_property_offering_id,
        c.source_property_offering_id AS candidate_property_offering_id,
        'target_to_source'::text AS direction,
        c.property_offering_source_match_score,
        c.property_offering_source_match_confidence,
        c.property_offering_source_match_status,
        c.property_offering_source_match_reasons,
        c.property_offering_source_match_price_delta_percent,
        c.property_offering_source_match_created_at
    FROM selected
    JOIN public.property_offering_source_match_candidates c ON c.target_sale_listing_id = selected.sale_listing_id
    WHERE c.property_offering_source_match_status <> 'rejected'
),
latest AS (
    SELECT DISTINCT ON (selected_sale_listing_id, candidate_sale_listing_id)
        *
    FROM candidates
    ORDER BY selected_sale_listing_id, candidate_sale_listing_id, property_offering_source_match_created_at DESC, property_offering_source_match_score DESC
)
SELECT
    latest.selected_sale_listing_id,
    candidate.sale_listing_id,
    candidate.sale_listing_canonical_id,
    candidate.sale_listing_source_provider,
    candidate.sale_listing_source_kind,
    candidate.sale_listing_native_id,
    COALESCE(candidate.sale_listing_headline, candidate.sale_listing_street_address, candidate.sale_listing_native_id) AS headline,
    COALESCE(candidate.sale_listing_street_address, '') AS address,
    COALESCE(candidate.sale_listing_city, candidate.sale_listing_city_norm, '') AS city,
    COALESCE(candidate.sale_listing_postal, candidate.sale_listing_postal_norm, '') AS postal,
    candidate.sale_listing_asking_price,
    candidate.sale_listing_debt_free_price,
    candidate.sale_listing_area_value,
    COALESCE(candidate.sale_listing_room_layout, '') AS room_layout,
    COALESCE(candidate.sale_listing_url, '') AS url,
    CASE
        WHEN candidate.sale_listing_source_provider = 'shortcut' AND candidate.sale_listing_source_kind = 'ad' THEN candidate.shortcut_ad_id IS NOT NULL AND COALESCE(candidate.sale_listing_url, '') <> '' AND candidate.sale_listing_last_seen_at >= now() - interval '7 days'
        WHEN candidate.sale_listing_source_provider = 'frontdoor' AND candidate.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
        WHEN candidate.sale_listing_source_provider = 'frontdoor' AND candidate.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
        ELSE false
    END AS external_url_available,
    latest.selected_property_offering_id,
    latest.candidate_property_offering_id,
    latest.direction,
    latest.property_offering_source_match_status,
    latest.property_offering_source_match_score,
    latest.property_offering_source_match_confidence,
    latest.property_offering_source_match_price_delta_percent,
    latest.property_offering_source_match_reasons,
    latest.property_offering_source_match_created_at
FROM latest
JOIN public.property_source_offerings candidate ON candidate.sale_listing_id = latest.candidate_sale_listing_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = candidate.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = candidate.frontdoor_building_announcement_id
ORDER BY latest.selected_sale_listing_id, latest.property_offering_source_match_score DESC, latest.property_offering_source_match_created_at DESC
LIMIT 100`

const addressRawTransactionsSQL = `
SELECT
    pt.prices_transaction_id,
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    pt.prices_transaction_area,
    pt.prices_transaction_price::bigint,
    pt.prices_transaction_price_per_square_meter::bigint,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, ''),
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, ''),
    COALESCE(pt.prices_transaction_plot, ''),
    COALESCE(pt.prices_transaction_energy_class, ''),
    COALESCE(pt.prices_transaction_period_identifier, ''),
    COALESCE(pc.prices_city_name, ''),
    COALESCE(pn.prices_neighborhood_name, ''),
    COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code, ''),
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    (
        EXISTS (
            SELECT 1
            FROM public.property_source_offerings sl
            WHERE sl.prices_transaction_id = pt.prices_transaction_id
        )
        OR EXISTS (
            SELECT 1
            FROM public.property_offering_transactions pot
            WHERE pot.prices_transaction_id = pt.prices_transaction_id
                AND pot.property_offering_transaction_link_status <> 'rejected'
        )
    ) AS is_matched,
    pt.prices_transaction_id = ANY($3::uuid[]) AS linked_to_lookup,
    pt.prices_transaction_id = ANY($4::uuid[]) AS candidate_to_lookup,
    (
        SELECT count(*)::integer
        FROM public.property_source_offerings sl
        WHERE sl.prices_transaction_id = pt.prices_transaction_id
    ) AS matched_listing_count,
    (
        SELECT count(*)::integer
        FROM public.property_offering_transactions pot
        WHERE pot.prices_transaction_id = pt.prices_transaction_id
            AND pot.property_offering_transaction_link_status <> 'rejected'
    ) AS matched_offering_count,
    COALESCE(
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'type', match_type,
                    'id', id,
                    'offering_id', offering_id,
                    'canonical_id', canonical_id,
                    'source', source,
                    'native_id', native_id,
                    'headline', headline,
                    'address', address,
                    'city', city,
                    'postal', postal,
                    'status', status,
                    'method', method,
                    'score', score
                )
                ORDER BY match_type, headline, id
            )
            FROM (
                SELECT
                    'listing'::text AS match_type,
                    sl.sale_listing_id::text AS id,
                    ''::text AS offering_id,
                    sl.sale_listing_canonical_id AS canonical_id,
                    sl.sale_listing_source_provider AS source,
                    sl.sale_listing_native_id AS native_id,
                    COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
                    COALESCE(sl.sale_listing_street_address, '') AS address,
                    COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
                    COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
                    COALESCE(sl.sale_listing_prices_match_status, '') AS status,
                    ''::text AS method,
                    NULL::integer AS score
                FROM public.property_source_offerings sl
                WHERE sl.prices_transaction_id = pt.prices_transaction_id
                UNION ALL
                SELECT
                    'offering_source'::text AS match_type,
                    pot.property_offering_transaction_id::text || ':' || sl.sale_listing_id::text AS id,
                    pot.property_offering_id::text AS offering_id,
                    sl.sale_listing_canonical_id AS canonical_id,
                    sl.sale_listing_source_provider AS source,
                    sl.sale_listing_native_id AS native_id,
                    COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
                    COALESCE(sl.sale_listing_street_address, '') AS address,
                    COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
                    COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
                    pot.property_offering_transaction_link_status AS status,
                    pot.property_offering_transaction_link_method AS method,
                    pot.property_offering_transaction_link_score AS score
                FROM public.property_offering_transactions pot
                JOIN public.property_offering_sources pos ON pos.property_offering_id = pot.property_offering_id
                    AND pos.property_offering_source_link_status <> 'rejected'
                JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
                WHERE pot.prices_transaction_id = pt.prices_transaction_id
                    AND pot.property_offering_transaction_link_status <> 'rejected'
                LIMIT 8
            ) match
        ),
        '[]'::jsonb
    ) AS matches
FROM public.prices_transactions pt
JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN public.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN public.prices_postal_codes ppc_prices ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN public.postal_postal_codes ppc_scraped ON ppc_scraped.postal_postal_code_code = ppc_prices.prices_postal_code_code
LEFT JOIN public.postal_municipalities pm_scraped ON pm_scraped.postal_municipality_id = ppc_scraped.postal_municipality_id
LEFT JOIN public.postal_postal_codes ppc ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN public.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE (trim($1::text) = '' OR lower(COALESCE(pc.prices_city_name, pm_scraped.postal_municipality_name_fi, pm.postal_municipality_name_fi, '')) LIKE ('%' || lower(trim($1::text)) || '%'))
    AND (trim($2::text) = '' OR COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code, '') = public.fnc__normalize_postal($2::text))
ORDER BY linked_to_lookup DESC, candidate_to_lookup DESC, pt.prices_transaction_created_at DESC, pt.prices_transaction_price ASC
LIMIT $5::int`

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
	normalized := SearchParams{Query: strings.TrimSpace(params.Query), Source: normalizeSource(params.Source), Kind: normalizeKind(params.Kind), ListingType: normalizeListingType(params.ListingType), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, City: strings.TrimSpace(params.City), Postal: strings.TrimSpace(params.Postal), Page: params.Page, PageSize: normalizePageSize(params.PageSize), Sort: normalizeSort(params.Sort), PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore}
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
		return queryAddress, queryCity, queryPostal
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
		return queryAddress, queryCity, queryPostal
	}
	if matches := pastedAddressPostalRE.FindStringSubmatch(queryAddress); matches != nil {
		if strings.TrimSpace(matches[1]) != "" {
			queryAddress = strings.TrimSpace(matches[1])
		}
		if queryPostal == "" {
			queryPostal = strings.TrimSpace(matches[2])
		}
	}
	return queryAddress, queryCity, queryPostal
}

func (s *Service) lookupPostalCities(ctx context.Context, postal string) (string, []string, error) {
	normalizedPostal := strings.TrimSpace(postal)
	if normalizedPostal == "" {
		return "", nil, nil
	}
	var cityFI, citySV string
	err := s.db.QueryRow(ctx, `
SELECT COALESCE(pm.postal_municipality_name_fi, ''), COALESCE(pm.postal_municipality_name_sv, '')
FROM public.postal_postal_codes ppc
JOIN public.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE ppc.postal_postal_code_code = public.fnc__normalize_postal($1::text)
ORDER BY pm.postal_municipality_name_fi
LIMIT 1`, normalizedPostal).Scan(&cityFI, &citySV)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("lookup postal city: %w", err)
	}
	cityFI = strings.TrimSpace(cityFI)
	citySV = strings.TrimSpace(citySV)
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
		Condition:                firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "inspection", "overallCondition"), strings.TrimSpace(row.AdCondition), valueAtPath(payload, "property", "condition")),
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

func firstInt32(values ...*int32) *int32 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
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

func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func boolPtrValue(value *bool) bool {
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
