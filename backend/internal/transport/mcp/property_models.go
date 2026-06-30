package mcpserver

import "time"

const propertySchemaVersion = "1.0"

type PropertySchema struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PropertyQueryInput struct {
	Entity         string                 `json:"entity,omitempty" jsonschema:"Entity to query: property, listing, address, transaction, or all. Defaults to property/listing search."`
	Query          string                 `json:"query,omitempty" jsonschema:"Natural language or portal-style free text search across listing headlines, addresses, source native IDs, and transaction descriptions."`
	Location       string                 `json:"location,omitempty" jsonschema:"Location text such as city, neighborhood, postal code, or address. Address is preferred for exact lookup."`
	Address        string                 `json:"address,omitempty" jsonschema:"Exact or pasted address for address lookup mode."`
	BBox           []float64              `json:"bbox,omitempty" jsonschema:"Optional map bounding box as west,south,east,north. Accepted for contract compatibility; current backend search may ignore it."`
	Radius         *float64               `json:"radius,omitempty" jsonschema:"Optional search radius in kilometers around location or coordinates."`
	PropertyTypes  []string               `json:"property_types,omitempty" jsonschema:"Portal property type filters such as apartment, house, row_house, plot, or commercial."`
	ListingTypes   []string               `json:"listing_types,omitempty" jsonschema:"Listing type filters such as listing, rental, sale, or all."`
	OwnershipTypes []string               `json:"ownership_types,omitempty" jsonschema:"Ownership filters such as own, rental, right_of_occupancy, or shared_plot."`
	Price          PropertyRangeInt64     `json:"price,omitempty" jsonschema:"Asking or sale price range in EUR."`
	DebtFreePrice  PropertyRangeInt64     `json:"debt_free_price,omitempty" jsonschema:"Debt-free price range in EUR."`
	AreaM2         PropertyRangeFloat64   `json:"area_m2,omitempty" jsonschema:"Area range in square meters."`
	Rooms          PropertyRangeFloat64   `json:"rooms,omitempty" jsonschema:"Room count range."`
	BuildYear      PropertyRangeInt32     `json:"build_year,omitempty" jsonschema:"Build year range."`
	Floor          PropertyRangeInt32     `json:"floor,omitempty" jsonschema:"Floor range."`
	Condition      []string               `json:"condition,omitempty" jsonschema:"Condition filters."`
	EnergyClass    []string               `json:"energy_class,omitempty" jsonschema:"Energy class filters."`
	Features       []string               `json:"features,omitempty" jsonschema:"Feature filters such as sauna, balcony, elevator, parking, or floorplan."`
	Costs          []string               `json:"costs,omitempty" jsonschema:"Cost-related filters or natural language hints."`
	Lifecycle      []string               `json:"lifecycle,omitempty" jsonschema:"Lifecycle filters such as new, price_changed, stale, or first_seen_recently."`
	Market         []string               `json:"market,omitempty" jsonschema:"Market filters such as has_comps, under_market, or over_market."`
	Sort           string                 `json:"sort,omitempty" jsonschema:"Sort mode such as seen_desc, price_asc, price_desc, area_asc, area_desc, newest, cheapest, or expensive."`
	Page           *int32                 `json:"page,omitempty" jsonschema:"One-based result page for listings."`
	PageSize       *int32                 `json:"page_size,omitempty" jsonschema:"Listing result count. Must be 25, 50, or 100 for search mode; address lookup is capped at 100."`
	Include        propertyIncludeOptions `json:"include,omitempty" jsonschema:"Controls optional linked records and raw payloads."`
	Source         string                 `json:"source,omitempty" jsonschema:"Source filter: shortcut, frontdoor, or all."`
	Kind           string                 `json:"kind,omitempty" jsonschema:"Entity kind filter: ad, building, announcement, rental, or all."`
	ListingType    string                 `json:"listing_type,omitempty" jsonschema:"Compatibility listing type filter: listing, rental, or all."`
	City           string                 `json:"city,omitempty" jsonschema:"Compatibility municipality or city filter."`
	Postal         string                 `json:"postal,omitempty" jsonschema:"Compatibility Finnish postal code filter."`
	MinPrice       *int64                 `json:"min_price,omitempty" jsonschema:"Compatibility minimum asking or sale price in EUR."`
	MaxPrice       *int64                 `json:"max_price,omitempty" jsonschema:"Compatibility maximum asking or sale price in EUR."`
	MinArea        *float64               `json:"min_area,omitempty" jsonschema:"Compatibility minimum area in square meters."`
	MaxArea        *float64               `json:"max_area,omitempty" jsonschema:"Compatibility maximum area in square meters."`
	Limit          *int32                 `json:"limit,omitempty" jsonschema:"Transaction result count, between 1 and 5000."`
}

type PropertyRangeInt64 struct {
	Min *int64 `json:"min,omitempty"`
	Max *int64 `json:"max,omitempty"`
}

type PropertyRangeInt32 struct {
	Min *int32 `json:"min,omitempty"`
	Max *int32 `json:"max,omitempty"`
}

type PropertyRangeFloat64 struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

type PropertyQueryResult struct {
	Schema       PropertySchema           `json:"schema"`
	View         string                   `json:"view"`
	Summary      string                   `json:"summary"`
	Mode         string                   `json:"mode"`
	Entity       string                   `json:"entity"`
	Query        PropertyQueryEcho        `json:"query"`
	Rows         []PropertySummary        `json:"rows"`
	Transactions []ComparableSale         `json:"transactions,omitempty"`
	Facets       propertyQueryFacets      `json:"facets,omitempty"`
	DataQuality  PropertyDataQuality      `json:"data_quality"`
	Diagnostics  propertyQueryDiagnostics `json:"diagnostics,omitempty"`
	WebURL       string                   `json:"web_url,omitempty"`
	Total        int64                    `json:"total"`
	Page         int32                    `json:"page"`
	PageSize     int32                    `json:"page_size"`
}

type PropertyQueryEcho struct {
	Text           string                 `json:"text,omitempty"`
	Address        string                 `json:"address,omitempty"`
	Location       string                 `json:"location,omitempty"`
	Source         string                 `json:"source,omitempty"`
	Kind           string                 `json:"kind,omitempty"`
	ListingType    string                 `json:"listing_type,omitempty"`
	ListingTypes   []string               `json:"listing_types,omitempty"`
	City           string                 `json:"city,omitempty"`
	Postal         string                 `json:"postal,omitempty"`
	Price          PropertyRangeInt64     `json:"price,omitempty"`
	DebtFreePrice  PropertyRangeInt64     `json:"debt_free_price,omitempty"`
	AreaM2         PropertyRangeFloat64   `json:"area_m2,omitempty"`
	PropertyTypes  []string               `json:"property_types,omitempty"`
	OwnershipTypes []string               `json:"ownership_types,omitempty"`
	Features       []string               `json:"features,omitempty"`
	Sort           string                 `json:"sort,omitempty"`
	Include        propertyIncludeOptions `json:"include,omitempty"`
}

type PropertySummary struct {
	Schema               PropertySchema          `json:"schema"`
	ID                   string                  `json:"id"`
	EntityID             string                  `json:"entity_id"`
	EntityType           string                  `json:"entity_type"`
	CanonicalID          string                  `json:"canonical_id,omitempty"`
	SourceIDs            []string                `json:"source_ids,omitempty"`
	OfferingID           string                  `json:"offering_id,omitempty"`
	GroupingID           string                  `json:"grouping_id,omitempty"`
	ListingID            string                  `json:"listing_id,omitempty"`
	NativeID             string                  `json:"native_id,omitempty"`
	Source               string                  `json:"source,omitempty"`
	Kind                 string                  `json:"kind,omitempty"`
	Title                string                  `json:"title"`
	Subtitle             string                  `json:"subtitle,omitempty"`
	Badges               []string                `json:"badges,omitempty"`
	Location             PropertyLocation        `json:"location"`
	Facts                PropertyFacts           `json:"facts"`
	Costs                PropertyCosts           `json:"costs"`
	Features             PropertyFeatures        `json:"features"`
	Lifecycle            PropertyLifecycle       `json:"lifecycle"`
	Media                PropertyMedia           `json:"media"`
	Market               MarketContext           `json:"market"`
	Links                propertyEntityLinks     `json:"links"`
	Actions              []PropertyAction        `json:"actions"`
	DataQuality          PropertyDataQuality     `json:"data_quality"`
	Evidence             []PropertyEvidence      `json:"evidence,omitempty"`
	Address              string                  `json:"address,omitempty"`
	City                 string                  `json:"city,omitempty"`
	Postal               string                  `json:"postal,omitempty"`
	Price                *int64                  `json:"price,omitempty"`
	Area                 *float64                `json:"area,omitempty"`
	RoomLayout           string                  `json:"room_layout,omitempty"`
	URL                  string                  `json:"url,omitempty"`
	ExternalURLAvailable bool                    `json:"external_url_available,omitempty"`
	WebURL               string                  `json:"web_url,omitempty"`
	LastSeenAt           *time.Time              `json:"last_seen_at,omitempty"`
	Match                propertyMatchSummary    `json:"match,omitempty"`
	Insights             propertyInsightSummary  `json:"insights,omitempty"`
	Transactions         []ComparableSale        `json:"transactions,omitempty"`
	SourceRecords        []propertySourceSummary `json:"source_records,omitempty"`
}

type PropertyDetail struct {
	Schema         PropertySchema           `json:"schema"`
	View           string                   `json:"view"`
	ID             string                   `json:"id"`
	EntityType     string                   `json:"entity_type"`
	Title          string                   `json:"title"`
	Overview       []propertyDetailField    `json:"overview"`
	Location       PropertyLocation         `json:"location"`
	Facts          PropertyFacts            `json:"facts"`
	Costs          PropertyCosts            `json:"costs"`
	Features       PropertyFeatures         `json:"features"`
	Lifecycle      PropertyLifecycle        `json:"lifecycle"`
	Media          PropertyMedia            `json:"media"`
	HousingCompany []propertyDetailField    `json:"housing_company,omitempty"`
	Building       []propertyDetailField    `json:"building,omitempty"`
	Renovations    []propertyDetailField    `json:"renovations,omitempty"`
	MarketContext  MarketContext            `json:"market_context"`
	NearbySales    []ComparableSale         `json:"nearby_sales,omitempty"`
	SourceRecords  []propertyDetailField    `json:"source_records,omitempty"`
	RawEvidence    []PropertyEvidence       `json:"raw_evidence,omitempty"`
	Reports        []PropertyReport         `json:"reports"`
	Actions        []PropertyAction         `json:"actions"`
	DataQuality    PropertyDataQuality      `json:"data_quality"`
	Summary        string                   `json:"summary"`
	Canonical      propertyCanonicalFields  `json:"canonical"`
	CanonicalExtra []propertyDetailField    `json:"canonical_extra,omitempty"`
	SourceSpecific []propertyDetailField    `json:"source_specific,omitempty"`
	Related        []propertyDetailField    `json:"related,omitempty"`
	Normalized     normalizedPropertyFields `json:"normalized"`
	Links          propertyEntityLinks      `json:"links"`
	Report         []propertyReportSection  `json:"report"`
	Markdown       string                   `json:"markdown"`
	Raw            *propertyRawPayload      `json:"raw,omitempty"`
	RawJSON        any                      `json:"raw_json,omitempty"`
}

type PropertyFacts struct {
	Price        *int64   `json:"price,omitempty"`
	AreaM2       *float64 `json:"area_m2,omitempty"`
	Rooms        string   `json:"rooms,omitempty"`
	RoomsCount   *int32   `json:"rooms_count,omitempty"`
	BuildYear    *int32   `json:"build_year,omitempty"`
	Floor        *int32   `json:"floor,omitempty"`
	TotalFloors  *int32   `json:"total_floors,omitempty"`
	Condition    string   `json:"condition,omitempty"`
	EnergyClass  string   `json:"energy_class,omitempty"`
	PropertyType string   `json:"property_type,omitempty"`
}

type PropertyLifecycle struct {
	FirstSeenAt  *time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	DaysOnMarket *int       `json:"days_on_market,omitempty"`
	PriceChanged *bool      `json:"price_changed,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

type PropertyMedia struct {
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
	ImageCount       *int   `json:"image_count,omitempty"`
	HasFloorplan     *bool  `json:"has_floorplan,omitempty"`
	HasVirtualTour   *bool  `json:"has_virtual_tour,omitempty"`
	HasSourceGallery *bool  `json:"has_source_gallery,omitempty"`
}

type PropertyLocation struct {
	Address     string    `json:"address,omitempty"`
	Street      string    `json:"street,omitempty"`
	City        string    `json:"city,omitempty"`
	Postal      string    `json:"postal,omitempty"`
	Coordinates []float64 `json:"coordinates,omitempty"`
}

type PropertyCosts struct {
	AskingPrice              *int64   `json:"asking_price,omitempty"`
	DebtFreePrice            *int64   `json:"debt_free_price,omitempty"`
	DebtShareAmount          *int64   `json:"debt_share_amount,omitempty"`
	PricePerSquareMeter      *float64 `json:"price_per_m2,omitempty"`
	MaintenanceChargeMonthly *float64 `json:"maintenance_charge_monthly,omitempty"`
	TotalChargeMonthly       *float64 `json:"total_charge_monthly,omitempty"`
	WaterCharge              *float64 `json:"water_charge,omitempty"`
}

type PropertyFeatures struct {
	Sauna    *bool    `json:"sauna,omitempty"`
	Balcony  *bool    `json:"balcony,omitempty"`
	Elevator *bool    `json:"elevator,omitempty"`
	Parking  string   `json:"parking,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	PlotType string   `json:"plot_type,omitempty"`
}

type MarketContext struct {
	LinkedSalesCount     int              `json:"linked_sales_count,omitempty"`
	ComparableSales      []ComparableSale `json:"comparable_sales,omitempty"`
	NearestComparable    *ComparableSale  `json:"nearest_comparable,omitempty"`
	MedianPricePerM2     *float64         `json:"median_price_per_m2,omitempty"`
	SubjectPricePerM2    *float64         `json:"subject_price_per_m2,omitempty"`
	SubjectVsMarketPct   *float64         `json:"subject_vs_market_pct,omitempty"`
	OverUnderMarketHint  string           `json:"over_under_market_hint,omitempty"`
	Confidence           string           `json:"confidence,omitempty"`
	Explanation          string           `json:"explanation,omitempty"`
	RecommendedFollowUps []string         `json:"recommended_follow_ups,omitempty"`
}

type ComparableSale struct {
	Schema              PropertySchema `json:"schema,omitempty"`
	ID                  string         `json:"id,omitempty"`
	TransactionID       string         `json:"transaction_id,omitempty"`
	Description         string         `json:"description,omitempty"`
	Category            string         `json:"category,omitempty"`
	Type                string         `json:"type,omitempty"`
	Area                *float64       `json:"area,omitempty"`
	Price               *int64         `json:"price,omitempty"`
	PricePerSquareMeter *int64         `json:"price_per_square_meter,omitempty"`
	BuildYear           *int32         `json:"build_year,omitempty"`
	Floor               string         `json:"floor,omitempty"`
	Elevator            *bool          `json:"elevator,omitempty"`
	Condition           string         `json:"condition,omitempty"`
	Plot                string         `json:"plot,omitempty"`
	EnergyClass         string         `json:"energy_class,omitempty"`
	PeriodIdentifier    string         `json:"period_identifier,omitempty"`
	City                string         `json:"city,omitempty"`
	Neighborhood        string         `json:"neighborhood,omitempty"`
	Postal              string         `json:"postal,omitempty"`
	Confidence          string         `json:"confidence,omitempty"`
	LinkStatus          string         `json:"link_status,omitempty"`
	LinkMethod          string         `json:"link_method,omitempty"`
	Score               *int32         `json:"score,omitempty"`
	CreatedAt           *time.Time     `json:"created_at,omitempty"`
	UpdatedAt           *time.Time     `json:"updated_at,omitempty"`
}

type PropertyReport struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

type PropertyAction struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Type   string         `json:"type"`
	Target string         `json:"target,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

type PropertyDataQuality struct {
	Completeness    float64            `json:"completeness"`
	MissingFields   []string           `json:"missing_fields,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
	SourceConflicts []PropertyConflict `json:"source_conflicts,omitempty"`
}

type PropertyConflict struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

type PropertyEvidence struct {
	Field      string `json:"field"`
	Value      string `json:"value,omitempty"`
	Source     string `json:"source,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

type PropertyComparisonInput struct {
	IDs          []string `json:"ids" jsonschema:"Canonical property IDs, source URLs, or listing/address text values to compare."`
	Criteria     []string `json:"criteria,omitempty" jsonschema:"Optional buyer criteria such as price, commute, condition, market_value, monthly_cost, or renovations."`
	BuyerProfile string   `json:"buyer_profile,omitempty" jsonschema:"Optional natural language buyer profile for tradeoff framing."`
}

type PropertyComparisonResult struct {
	Schema                   PropertySchema      `json:"schema"`
	View                     string              `json:"view"`
	Summary                  string              `json:"summary"`
	Rows                     []PropertySummary   `json:"rows"`
	Ranking                  []PropertyRank      `json:"ranking"`
	Tradeoffs                []string            `json:"tradeoffs"`
	MissingDataWarnings      []string            `json:"missing_data_warnings"`
	MarketComparison         MarketContext       `json:"market_comparison"`
	RecommendedFollowUpCalls []PropertyAction    `json:"recommended_follow_up_detail_calls"`
	DataQuality              PropertyDataQuality `json:"data_quality"`
	Markdown                 string              `json:"markdown"`
}

type PropertyRank struct {
	ID      string   `json:"id"`
	Rank    int      `json:"rank"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`
}

type PropertyMarketContextInput struct {
	ID                string   `json:"id,omitempty" jsonschema:"Canonical property ID, source URL, or listing/address text for the subject property."`
	Location          string   `json:"location,omitempty" jsonschema:"Location text when no listing ID is provided."`
	City              string   `json:"city,omitempty" jsonschema:"City filter for comparable sales."`
	Postal            string   `json:"postal,omitempty" jsonschema:"Postal code filter for comparable sales."`
	Radius            *float64 `json:"radius,omitempty" jsonschema:"Radius in kilometers around the subject or location."`
	ComparableFilters []string `json:"comparable_filters,omitempty" jsonschema:"Optional comparable filters such as same_postal, similar_area, same_property_type, or recent."`
	Limit             *int32   `json:"limit,omitempty" jsonschema:"Maximum comparable sales to return."`
}

type PropertyMarketContextResult struct {
	Schema      PropertySchema      `json:"schema"`
	View        string              `json:"view"`
	Summary     string              `json:"summary"`
	Subject     *PropertySummary    `json:"subject,omitempty"`
	Market      MarketContext       `json:"market"`
	DataQuality PropertyDataQuality `json:"data_quality"`
	Markdown    string              `json:"markdown"`
}

type normalizedPropertyFields struct {
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
