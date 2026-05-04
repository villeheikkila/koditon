package properties

import (
	"encoding/json"
	"time"

	frontdoorpayload "koditon/internal/providers/frontdoor"
	shortcutpayload "koditon/internal/providers/shortcut"
)

type SearchParams struct {
	Query           string
	Source          string
	Kind            string
	City            string
	Postal          string
	MinPrice        *int64
	MaxPrice        *int64
	MinArea         *float64
	MaxArea         *float64
	MinPricePerM2   *float64
	MaxPricePerM2   *float64
	Rooms           *int32
	Floor           *int32
	MinBuildYear    *int32
	MaxBuildYear    *int32
	Condition       string
	EnergyClass     string
	Page            int32
	PageSize        int32
	Sort            string
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
}

type Page[T any] struct {
	Rows     []T   `json:"rows"`
	Total    int64 `json:"total"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

type MapBounds struct {
	MinLat         *float64
	MinLng         *float64
	MaxLat         *float64
	MaxLng         *float64
	Query          string
	City           string
	Postal         string
	Source         string
	Kind           string
	MinPrice       *int64
	MaxPrice       *int64
	MinArea        *float64
	MaxArea        *float64
	MinPricePerM2  *float64
	MaxPricePerM2  *float64
	Rooms          *int32
	MinBuildYear   *int32
	MaxBuildYear   *int32
	PropertyType   string
	Condition      string
	EnergyClass    string
	Elevator       *bool
	Sauna          *bool
	Balcony        *bool
	PlotOwned      *bool
	NewDevelopment *bool
	HasTransaction *bool
	Limit          int32
}

type SaleListingMapMarker struct {
	Lat                 float64                 `json:"lat"`
	Lng                 float64                 `json:"lng"`
	Count               int64                   `json:"count"`
	Address             string                  `json:"address,omitempty"`
	City                string                  `json:"city,omitempty"`
	Postal              string                  `json:"postal,omitempty"`
	MinPrice            *int64                  `json:"min_price,omitempty"`
	MaxPrice            *int64                  `json:"max_price,omitempty"`
	MinAreaM2           *float64                `json:"min_area_m2,omitempty"`
	MaxAreaM2           *float64                `json:"max_area_m2,omitempty"`
	LastSeenAt          *time.Time              `json:"last_seen_at,omitempty"`
	Providers           []string                `json:"providers,omitempty"`
	Kinds               []string                `json:"kinds,omitempty"`
	ListingIDs          []string                `json:"listing_ids,omitempty"`
	Listings            []SaleListingMapListing `json:"listings,omitempty"`
	HousingCompanyID    string                  `json:"housing_company_id,omitempty"`
	HousingCompanyCount int64                   `json:"housing_company_count"`
}

type SaleListingMapListing struct {
	ID         string     `json:"id"`
	Headline   string     `json:"headline,omitempty"`
	Address    string     `json:"address,omitempty"`
	City       string     `json:"city,omitempty"`
	Postal     string     `json:"postal,omitempty"`
	Layout     string     `json:"layout,omitempty"`
	AreaM2     *float64   `json:"area_m2,omitempty"`
	Price      *int64     `json:"price,omitempty"`
	PricePerM2 *float64   `json:"price_per_m2,omitempty"`
	BuildYear  *int32     `json:"build_year,omitempty"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	Providers  []string   `json:"providers,omitempty"`
	Kinds      []string   `json:"kinds,omitempty"`
}

type SaleListingMap struct {
	Markers []SaleListingMapMarker `json:"markers"`
}

type SaleListingMapFilterOption struct {
	Value string   `json:"value"`
	Label string   `json:"label"`
	Meta  string   `json:"meta,omitempty"`
	Lat   *float64 `json:"lat,omitempty"`
	Lng   *float64 `json:"lng,omitempty"`
}

type SaleListingMapFilterOptions struct {
	Cities  []SaleListingMapFilterOption `json:"cities"`
	Postals []SaleListingMapFilterOption `json:"postals"`
}

type ListingSource struct {
	Provider    string          `json:"provider"`
	Kind        string          `json:"kind"`
	CanonicalID string          `json:"-"`
	NativeID    string          `json:"native_id,omitempty"`
	ExternalID  string          `json:"external_id,omitempty"`
	FriendlyID  string          `json:"friendly_id,omitempty"`
	URL         string          `json:"url,omitempty"`
	OriginalURL string          `json:"original_url,omitempty"`
	FirstSeenAt *time.Time      `json:"first_seen_at,omitempty"`
	LastSeenAt  *time.Time      `json:"last_seen_at,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	Status      string          `json:"status,omitempty"`
	Flags       map[string]bool `json:"flags,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

type CanonicalOffering struct {
	OfferingID           string   `json:"offering_id"`
	HousingCompanyID     string   `json:"housing_company_id,omitempty"`
	UnitID               string   `json:"unit_id,omitempty"`
	PrimarySourceListing string   `json:"primary_source_listing_id,omitempty"`
	SourceCount          int32    `json:"source_count,omitempty"`
	MergeDecisionCount   int32    `json:"merge_decision_count,omitempty"`
	MergedFrom           []string `json:"merged_from,omitempty"`
}

type OfferingSourceRecord struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	Kind        string     `json:"kind"`
	NativeID    string     `json:"native_id,omitempty"`
	URL         string     `json:"url,omitempty"`
	Headline    string     `json:"headline,omitempty"`
	FirstSeenAt *time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
	LinkStatus  string     `json:"link_status"`
	LinkMethod  string     `json:"link_method"`
	LinkScore   int32      `json:"link_score"`
}

type OfferingSourceRawPayload struct {
	ID       string          `json:"id"`
	Provider string          `json:"provider"`
	Kind     string          `json:"kind"`
	NativeID string          `json:"native_id"`
	Payload  json.RawMessage `json:"payload"`
}

type BuildingIdentity struct {
	Key        string             `json:"key"`
	Strategy   string             `json:"strategy"`
	Confidence float64            `json:"confidence"`
	Inputs     map[string]string  `json:"inputs,omitempty"`
	Sources    []BuildingSourceID `json:"sources,omitempty"`
}

type BuildingSourceID struct {
	Provider   string `json:"provider"`
	Kind       string `json:"kind"`
	NativeID   string `json:"native_id"`
	ExternalID string `json:"external_id,omitempty"`
}

type Location struct {
	StreetAddress string   `json:"street_address,omitempty"`
	City          string   `json:"city,omitempty"`
	Postal        string   `json:"postal,omitempty"`
	District      string   `json:"district,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
}

type UnitDetails struct {
	Location                  Location `json:"location"`
	PropertyType              string   `json:"property_type,omitempty"`
	PropertySubtype           string   `json:"property_subtype,omitempty"`
	RoomLayout                string   `json:"room_layout,omitempty"`
	RoomsCount                *int32   `json:"rooms_count,omitempty"`
	BedroomsCount             *int32   `json:"bedrooms_count,omitempty"`
	AreaM2                    *float64 `json:"area_m2,omitempty"`
	LivingAreaM2              *float64 `json:"living_area_m2,omitempty"`
	TotalAreaM2               *float64 `json:"total_area_m2,omitempty"`
	OtherAreaM2               *float64 `json:"other_area_m2,omitempty"`
	FloorLevel                *int32   `json:"floor_level,omitempty"`
	Condition                 string   `json:"condition,omitempty"`
	Sauna                     *bool    `json:"sauna,omitempty"`
	Balcony                   *bool    `json:"balcony,omitempty"`
	Parking                   string   `json:"parking,omitempty"`
	Availability              string   `json:"availability,omitempty"`
	KitchenDescription        string   `json:"kitchen_description,omitempty"`
	BathroomDescription       string   `json:"bathroom_description,omitempty"`
	StorageDescription        string   `json:"storage_description,omitempty"`
	FloorMaterialsDescription string   `json:"floor_materials_description,omitempty"`
	WallMaterialsDescription  string   `json:"wall_materials_description,omitempty"`
	BalconyDescription        string   `json:"balcony_description,omitempty"`
	SaunaDescription          string   `json:"sauna_description,omitempty"`
	ViewsDescription          string   `json:"views_description,omitempty"`
	Appliances                []string `json:"appliances,omitempty"`
	Features                  []string `json:"features,omitempty"`
}

type BuildingDetails struct {
	Identity                  BuildingIdentity     `json:"identity"`
	Location                  Location             `json:"location"`
	HousingCompany            string               `json:"housing_company,omitempty"`
	BusinessID                string               `json:"business_id,omitempty"`
	BuildingType              string               `json:"building_type,omitempty"`
	BuildingSubtype           string               `json:"building_subtype,omitempty"`
	BuildYear                 *int32               `json:"build_year,omitempty"`
	ConstructionYear          *int32               `json:"construction_year,omitempty"`
	FloorCount                *int32               `json:"floor_count,omitempty"`
	ApartmentCount            *int32               `json:"apartment_count,omitempty"`
	BusinessPremiseCount      *int32               `json:"business_premise_count,omitempty"`
	EnergyClass               string               `json:"energy_class,omitempty"`
	Heating                   string               `json:"heating,omitempty"`
	HeatingDescription        string               `json:"heating_description,omitempty"`
	HeatingFuel               string               `json:"heating_fuel,omitempty"`
	BuildingMaterial          string               `json:"building_material,omitempty"`
	WallStructure             string               `json:"wall_structure,omitempty"`
	FrameConstructionMethod   string               `json:"frame_construction_method,omitempty"`
	RoofType                  string               `json:"roof_type,omitempty"`
	RoofMaterial              string               `json:"roof_material,omitempty"`
	CommonAreas               string               `json:"common_areas,omitempty"`
	CarStorage                string               `json:"car_storage,omitempty"`
	Connectivity              string               `json:"connectivity,omitempty"`
	OtherInfo                 string               `json:"other_info,omitempty"`
	Elevator                  *bool                `json:"elevator,omitempty"`
	Sauna                     *bool                `json:"sauna,omitempty"`
	Renovations               []BuildingRenovation `json:"renovations,omitempty"`
	ManagementMethod          string               `json:"management_method,omitempty"`
	PropertyManager           string               `json:"property_manager,omitempty"`
	MaintenanceResponsibility string               `json:"maintenance_responsibility,omitempty"`
}

type SiteDetails struct {
	PlotType           string   `json:"plot_type,omitempty"`
	PlotOwnershipType  string   `json:"plot_ownership_type,omitempty"`
	PlotAreaM2         *float64 `json:"plot_area_m2,omitempty"`
	LotRedemptionInfo  string   `json:"lot_redemption_info,omitempty"`
	LotRentalAgreement string   `json:"lot_rental_agreement,omitempty"`
	Yard               string   `json:"yard,omitempty"`
	Shore              string   `json:"shore,omitempty"`
	WaterSupply        string   `json:"water_supply,omitempty"`
	Sewer              string   `json:"sewer,omitempty"`
	RoadAccess         string   `json:"road_access,omitempty"`
	Zoning             string   `json:"zoning,omitempty"`
	DrivingDirections  string   `json:"driving_directions,omitempty"`
	Services           string   `json:"services,omitempty"`
	Transport          string   `json:"transport,omitempty"`
	WaterSupplyTypes   []string `json:"water_supply_types,omitempty"`
}

type CommercialDetails struct {
	Status                            string                 `json:"status,omitempty"`
	BookingStatus                     string                 `json:"booking_status,omitempty"`
	PublishedAt                       *time.Time             `json:"published_at,omitempty"`
	UnpublishedAt                     *time.Time             `json:"unpublished_at,omitempty"`
	FirstSeenAt                       *time.Time             `json:"first_seen_at,omitempty"`
	LastSeenAt                        *time.Time             `json:"last_seen_at,omitempty"`
	DaysOnMarket                      *int32                 `json:"days_on_market,omitempty"`
	MapVisible                        *bool                  `json:"map_visible,omitempty"`
	CanReceiveLeads                   *bool                  `json:"can_receive_leads,omitempty"`
	LeadOptions                       map[string]bool        `json:"lead_options,omitempty"`
	AskingPrice                       *int64                 `json:"asking_price,omitempty"`
	DebtFreePrice                     *int64                 `json:"debt_free_price,omitempty"`
	DebtShareAmount                   *int64                 `json:"debt_share_amount,omitempty"`
	PreviousAskingPrice               *int64                 `json:"previous_asking_price,omitempty"`
	PreviousDebtFreePrice             *int64                 `json:"previous_debt_free_price,omitempty"`
	PricePerSquareMeter               *float64               `json:"price_per_m2,omitempty"`
	Rent                              *int64                 `json:"rent,omitempty"`
	RentPeriod                        string                 `json:"rent_period,omitempty"`
	SecurityDeposit                   string                 `json:"security_deposit,omitempty"`
	AvailableFrom                     string                 `json:"available_from,omitempty"`
	MinimumTermMonths                 *int32                 `json:"minimum_term_months,omitempty"`
	FixedTerm                         *bool                  `json:"fixed_term,omitempty"`
	Furnished                         *bool                  `json:"furnished,omitempty"`
	PetsAllowed                       *bool                  `json:"pets_allowed,omitempty"`
	OwnershipType                     string                 `json:"ownership_type,omitempty"`
	DebtShareAdditionalInfo           string                 `json:"debt_share_additional_info,omitempty"`
	FeesInfo                          string                 `json:"fees_info,omitempty"`
	OtherTerms                        string                 `json:"other_terms,omitempty"`
	FinancingFeeInterestOnlyPeriod    string                 `json:"financing_fee_interest_only_period,omitempty"`
	FinancingFeeInterestOnlyStartDate string                 `json:"financing_fee_interest_only_start_date,omitempty"`
	FinancingFeeInterestOnlyEndDate   string                 `json:"financing_fee_interest_only_end_date,omitempty"`
	OpenBiddingInUse                  *bool                  `json:"open_bidding_in_use,omitempty"`
	OpenBiddingStartingSellingPrice   *int64                 `json:"open_bidding_starting_selling_price,omitempty"`
	OpenBiddingStartingDebtFreePrice  *int64                 `json:"open_bidding_starting_debt_free_price,omitempty"`
	OpenBiddingLatestOffer            *int64                 `json:"open_bidding_latest_offer,omitempty"`
	OpenBiddingTargetURL              string                 `json:"open_bidding_target_url,omitempty"`
	DevelopmentPhase                  string                 `json:"development_phase,omitempty"`
	NewDevelopment                    *bool                  `json:"new_development,omitempty"`
	NotifyPriceChanged                *bool                  `json:"notify_price_changed,omitempty"`
	MainImageHidden                   *bool                  `json:"main_image_hidden,omitempty"`
	IsCompanyAnnouncement             *bool                  `json:"is_company_announcement,omitempty"`
	ShowBiddingIndicators             *bool                  `json:"show_bidding_indicators,omitempty"`
	Charges                           Charges                `json:"charges,omitempty"`
	MatchedTransaction                *PriceTransactionMatch `json:"matched_transaction,omitempty"`
}

type PriceTransactionMatch struct {
	ID                  string     `json:"id"`
	FirstSeenAt         *time.Time `json:"first_seen_at,omitempty"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
	Description         string     `json:"description,omitempty"`
	Type                string     `json:"type,omitempty"`
	Category            string     `json:"category,omitempty"`
	AreaM2              *float64   `json:"area_m2,omitempty"`
	Price               *int64     `json:"price,omitempty"`
	PricePerSquareMeter *int64     `json:"price_per_m2,omitempty"`
	BuildYear           *int32     `json:"build_year,omitempty"`
	Floor               string     `json:"floor,omitempty"`
	Elevator            *bool      `json:"elevator,omitempty"`
	Condition           string     `json:"condition,omitempty"`
	Plot                string     `json:"plot,omitempty"`
	PlotOwned           *bool      `json:"plot_owned,omitempty"`
	EnergyClass         string     `json:"energy_class,omitempty"`
	PeriodIdentifier    string     `json:"period_identifier,omitempty"`
	City                string     `json:"city,omitempty"`
	Neighborhood        string     `json:"neighborhood,omitempty"`
	PostalCode          string     `json:"postal_code,omitempty"`
	MatchStatus         string     `json:"match_status,omitempty"`
	MatchScore          *int32     `json:"match_score,omitempty"`
	MatchConfidence     string     `json:"match_confidence,omitempty"`
}

type TransactionMatchPostalSummary struct {
	Postal           string `json:"postal"`
	NameFi           string `json:"name_fi,omitempty"`
	MunicipalityName string `json:"municipality_name,omitempty"`
	CandidateCount   int64  `json:"candidate_count"`
	ListingCount     int64  `json:"listing_count"`
	TransactionCount int64  `json:"transaction_count"`
	HighCount        int64  `json:"high_count"`
	MediumCount      int64  `json:"medium_count"`
	LowCount         int64  `json:"low_count"`
	AmbiguousCount   int64  `json:"ambiguous_count"`
	LatestAt         string `json:"latest_at,omitempty"`
}

type TransactionMatchCandidate struct {
	ID                string                           `json:"id"`
	Status            string                           `json:"status"`
	Score             int32                            `json:"score"`
	Confidence        string                           `json:"confidence"`
	PriceDeltaPercent *float64                         `json:"price_delta_percent,omitempty"`
	Reasons           json.RawMessage                  `json:"reasons,omitempty"`
	CreatedAt         string                           `json:"created_at,omitempty"`
	Listing           TransactionMatchListingCandidate `json:"listing"`
	Transaction       TransactionMatchTransaction      `json:"transaction"`
}

type TransactionMatchListingCandidate struct {
	ID                 string   `json:"id"`
	CanonicalID        string   `json:"canonical_id"`
	SourceProvider     string   `json:"source_provider"`
	URL                string   `json:"url,omitempty"`
	Headline           string   `json:"headline,omitempty"`
	StreetAddress      string   `json:"street_address,omitempty"`
	City               string   `json:"city,omitempty"`
	Postal             string   `json:"postal,omitempty"`
	RoomLayout         string   `json:"room_layout,omitempty"`
	Condition          string   `json:"condition,omitempty"`
	ConditionMatchCode string   `json:"condition_match_code,omitempty"`
	AreaM2             *float64 `json:"area_m2,omitempty"`
	AskingPrice        *int64   `json:"asking_price,omitempty"`
	PricePerM2         *float64 `json:"price_per_m2,omitempty"`
	BuildYear          *int32   `json:"build_year,omitempty"`
	FloorLevel         *int32   `json:"floor_level,omitempty"`
	TotalFloors        *int32   `json:"total_floors,omitempty"`
	Elevator           *bool    `json:"elevator,omitempty"`
	EnergyMatchCode    string   `json:"energy_match_code,omitempty"`
	EnergyLabel        string   `json:"energy_label,omitempty"`
	PlotOwnershipRaw   string   `json:"plot_ownership_raw,omitempty"`
	PlotOwned          *bool    `json:"plot_owned,omitempty"`
	FirstSeenAt        string   `json:"first_seen_at,omitempty"`
	LastSeenAt         string   `json:"last_seen_at,omitempty"`
}

type TransactionMatchTransaction struct {
	ID                  string  `json:"id"`
	Description         string  `json:"description,omitempty"`
	Type                string  `json:"type,omitempty"`
	Category            string  `json:"category,omitempty"`
	AreaM2              float64 `json:"area_m2"`
	Price               int64   `json:"price"`
	PricePerSquareMeter int64   `json:"price_per_m2"`
	BuildYear           int32   `json:"build_year,omitempty"`
	Floor               string  `json:"floor,omitempty"`
	Elevator            bool    `json:"elevator"`
	Condition           string  `json:"condition,omitempty"`
	ConditionMatchCode  string  `json:"condition_match_code,omitempty"`
	Plot                string  `json:"plot,omitempty"`
	PlotOwned           *bool   `json:"plot_owned,omitempty"`
	EnergyClass         string  `json:"energy_class,omitempty"`
	EnergyMatchCode     string  `json:"energy_match_code,omitempty"`
	PeriodIdentifier    string  `json:"period_identifier,omitempty"`
	CreatedAt           string  `json:"created_at,omitempty"`
}

type Charges struct {
	MaintenanceMonthly *float64 `json:"maintenance_monthly,omitempty"`
	TotalMonthly       *float64 `json:"total_monthly,omitempty"`
	Water              *float64 `json:"water,omitempty"`
	Parking            *float64 `json:"parking,omitempty"`
	Sauna              *float64 `json:"sauna,omitempty"`
	Electricity        string   `json:"electricity,omitempty"`
	Heating            string   `json:"heating,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

type TextSections struct {
	Description        string `json:"description,omitempty"`
	Availability       string `json:"availability,omitempty"`
	RenovationsDone    string `json:"renovations_done,omitempty"`
	RenovationsPlanned string `json:"renovations_planned,omitempty"`
	AdditionalInfo     string `json:"additional_info,omitempty"`
	Area               string `json:"area,omitempty"`
	Building           string `json:"building,omitempty"`
	Transport          string `json:"transport,omitempty"`
	Amenities          string `json:"amenities,omitempty"`
	Charges            string `json:"charges,omitempty"`
	Kitchen            string `json:"kitchen,omitempty"`
	Bathroom           string `json:"bathroom,omitempty"`
	Storage            string `json:"storage,omitempty"`
	Materials          string `json:"materials,omitempty"`
}

type Media struct {
	MainImage *Image  `json:"main_image,omitempty"`
	Images    []Image `json:"images,omitempty"`
}

type Image struct {
	ID          string            `json:"id,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	ProviderID  string            `json:"provider_id,omitempty"`
	URL         string            `json:"url"`
	Variants    map[string]string `json:"variants,omitempty"`
	Description string            `json:"description,omitempty"`
	Role        string            `json:"role,omitempty"`
	Ordinal     *int32            `json:"ordinal,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

type Contact struct {
	Name       string `json:"name,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	OfficeName string `json:"office_name,omitempty"`
	Title      string `json:"title,omitempty"`
}

type Showing struct {
	StartAt *time.Time `json:"start_at,omitempty"`
	EndAt   *time.Time `json:"end_at,omitempty"`
	Info    string     `json:"info,omitempty"`
}

type Link struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Type  string `json:"type,omitempty"`
}

type ListingInsights struct {
	Items []Insight `json:"items,omitempty"`
}

type BuildingInsights struct {
	Items []Insight `json:"items,omitempty"`
}

type Insight struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence,omitempty"`
	Source     string  `json:"source,omitempty"`
}

type SaleListingSummary struct {
	ID              string            `json:"id"`
	Source          ListingSource     `json:"source"`
	SourceProviders []string          `json:"source_providers,omitempty"`
	Headline        string            `json:"headline"`
	Unit            UnitDetails       `json:"unit"`
	Building        BuildingDetails   `json:"building"`
	Site            SiteDetails       `json:"site,omitempty"`
	Commercial      CommercialDetails `json:"commercial"`
	Media           Media             `json:"media,omitempty"`
}

type RentalSummary struct {
	ID         string            `json:"id"`
	Source     ListingSource     `json:"source"`
	Headline   string            `json:"headline"`
	Unit       UnitDetails       `json:"unit"`
	Building   BuildingDetails   `json:"building"`
	Site       SiteDetails       `json:"site,omitempty"`
	Commercial CommercialDetails `json:"commercial"`
	Media      Media             `json:"media,omitempty"`
}

type SaleListing struct {
	ID            string                 `json:"id"`
	Canonical     CanonicalOffering      `json:"canonical"`
	Source        ListingSource          `json:"source"`
	SourceRecords []OfferingSourceRecord `json:"source_records,omitempty"`
	Headline      string                 `json:"headline"`
	Unit          UnitDetails            `json:"unit"`
	Building      BuildingDetails        `json:"building"`
	Site          SiteDetails            `json:"site,omitempty"`
	Commercial    CommercialDetails      `json:"commercial"`
	Texts         TextSections           `json:"texts,omitempty"`
	Media         Media                  `json:"media,omitempty"`
	Contacts      []Contact              `json:"contacts,omitempty"`
	Showings      []Showing              `json:"showings,omitempty"`
	Links         []Link                 `json:"links,omitempty"`
	Insights      ListingInsights        `json:"insights,omitempty"`
}

type Rental struct {
	ID         string            `json:"id"`
	Source     ListingSource     `json:"source"`
	Headline   string            `json:"headline"`
	Unit       UnitDetails       `json:"unit"`
	Building   BuildingDetails   `json:"building"`
	Site       SiteDetails       `json:"site,omitempty"`
	Commercial CommercialDetails `json:"commercial"`
	Texts      TextSections      `json:"texts,omitempty"`
	Media      Media             `json:"media,omitempty"`
	Contacts   []Contact         `json:"contacts,omitempty"`
	Showings   []Showing         `json:"showings,omitempty"`
	Links      []Link            `json:"links,omitempty"`
	Insights   ListingInsights   `json:"insights,omitempty"`
}

type Building struct {
	ID            string           `json:"id"`
	Details       BuildingDetails  `json:"details"`
	Site          SiteDetails      `json:"site,omitempty"`
	SourceRecords []ListingSource  `json:"source_records"`
	Texts         TextSections     `json:"texts,omitempty"`
	Related       RelatedListings  `json:"related,omitempty"`
	Insights      BuildingInsights `json:"insights,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
}

type RelatedListings struct {
	Items []RelatedListing `json:"items,omitempty"`
}

type RelatedListing struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind,omitempty"`
	FriendlyID string     `json:"friendly_id,omitempty"`
	Address    string     `json:"address,omitempty"`
	RoomLayout string     `json:"room_layout,omitempty"`
	AreaM2     *float64   `json:"area_m2,omitempty"`
	Price      *int64     `json:"price,omitempty"`
	PricePerM2 *float64   `json:"price_per_m2,omitempty"`
	SoldPrice  *int64     `json:"sold_price,omitempty"`
	SoldAt     *time.Time `json:"sold_at,omitempty"`
	BuildYear  *int32     `json:"build_year,omitempty"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	Providers  []string   `json:"providers,omitempty"`
	Kinds      []string   `json:"kinds,omitempty"`
	RentPeriod string     `json:"rent_period,omitempty"`
	Published  *bool      `json:"published,omitempty"`
	MainImage  *Image     `json:"main_image,omitempty"`
}

type BuildingRenovation struct {
	Kind string `json:"kind"`
	Done *bool  `json:"done,omitempty"`
	Year *int32 `json:"year,omitempty"`
}

type rawMap map[string]any

func parseShortcutRaw(payload json.RawMessage) rawMap {
	_, out, err := shortcutpayload.DecodeStoredAd(payload)
	if err == nil {
		return rawMap(out)
	}
	return parseRawMap(payload)
}

func parseFrontdoorRaw(payload json.RawMessage) rawMap {
	_, out, err := frontdoorpayload.DecodeStoredAd(payload)
	if err == nil {
		return rawMap(out)
	}
	return parseRawMap(payload)
}

func parseRawMap(payload json.RawMessage) rawMap {
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return rawMap(out)
}
