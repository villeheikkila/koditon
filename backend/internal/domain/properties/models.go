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
	City            string
	Postal          string
	MinPrice        *int64
	MaxPrice        *int64
	MinArea         *float64
	MaxArea         *float64
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

type ListingSource struct {
	Provider    string          `json:"provider"`
	Kind        string          `json:"kind"`
	CanonicalID string          `json:"canonical_id"`
	NativeID    string          `json:"native_id"`
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

type PropertyDetails struct {
	PropertyType    string   `json:"property_type,omitempty"`
	PropertySubtype string   `json:"property_subtype,omitempty"`
	RoomLayout      string   `json:"room_layout,omitempty"`
	RoomsCount      *int32   `json:"rooms_count,omitempty"`
	BedroomsCount   *int32   `json:"bedrooms_count,omitempty"`
	AreaM2          *float64 `json:"area_m2,omitempty"`
	LivingAreaM2    *float64 `json:"living_area_m2,omitempty"`
	TotalAreaM2     *float64 `json:"total_area_m2,omitempty"`
	FloorLevel      *int32   `json:"floor_level,omitempty"`
	TotalFloors     *int32   `json:"total_floors,omitempty"`
	BuildYear       *int32   `json:"build_year,omitempty"`
	Condition       string   `json:"condition,omitempty"`
	EnergyClass     string   `json:"energy_class,omitempty"`
	Elevator        *bool    `json:"elevator,omitempty"`
	Sauna           *bool    `json:"sauna,omitempty"`
	Balcony         *bool    `json:"balcony,omitempty"`
	Parking         string   `json:"parking,omitempty"`
	Features        []string `json:"features,omitempty"`
}

type BuildingSummary struct {
	Identity       BuildingIdentity `json:"identity"`
	HousingCompany string           `json:"housing_company,omitempty"`
	Address        string           `json:"address,omitempty"`
	City           string           `json:"city,omitempty"`
	Postal         string           `json:"postal,omitempty"`
	BuildYear      *int32           `json:"build_year,omitempty"`
	ApartmentCount *int32           `json:"apartment_count,omitempty"`
	FloorCount     *int32           `json:"floor_count,omitempty"`
}

type SaleTerms struct {
	AskingPrice         *int64   `json:"asking_price,omitempty"`
	DebtFreePrice       *int64   `json:"debt_free_price,omitempty"`
	DebtShareAmount     *int64   `json:"debt_share_amount,omitempty"`
	PricePerSquareMeter *float64 `json:"price_per_m2,omitempty"`
	OwnershipType       string   `json:"ownership_type,omitempty"`
	PlotType            string   `json:"plot_type,omitempty"`
}

type RentalTerms struct {
	Rent                *int64   `json:"rent,omitempty"`
	RentPeriod          string   `json:"rent_period,omitempty"`
	SecurityDeposit     string   `json:"security_deposit,omitempty"`
	AvailableFrom       string   `json:"available_from,omitempty"`
	MinimumTermMonths   *int32   `json:"minimum_term_months,omitempty"`
	FixedTerm           *bool    `json:"fixed_term,omitempty"`
	Furnished           *bool    `json:"furnished,omitempty"`
	PetsAllowed         *bool    `json:"pets_allowed,omitempty"`
	PricePerSquareMeter *float64 `json:"price_per_m2,omitempty"`
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
}

type Media struct {
	MainImage *Image  `json:"main_image,omitempty"`
	Images    []Image `json:"images,omitempty"`
}

type Image struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind,omitempty"`
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
	ID               string           `json:"id"`
	Source           ListingSource    `json:"source"`
	Headline         string           `json:"headline"`
	Location         Location         `json:"location"`
	Property         PropertyDetails  `json:"property"`
	SaleTerms        SaleTerms        `json:"sale_terms"`
	BuildingIdentity BuildingIdentity `json:"building_identity"`
	MainImage        *Image           `json:"main_image,omitempty"`
	PublishedAt      *time.Time       `json:"published_at,omitempty"`
	LastSeenAt       *time.Time       `json:"last_seen_at,omitempty"`
}

type RentalSummary struct {
	ID               string           `json:"id"`
	Source           ListingSource    `json:"source"`
	Headline         string           `json:"headline"`
	Location         Location         `json:"location"`
	Property         PropertyDetails  `json:"property"`
	RentalTerms      RentalTerms      `json:"rental_terms"`
	BuildingIdentity BuildingIdentity `json:"building_identity"`
	MainImage        *Image           `json:"main_image,omitempty"`
	PublishedAt      *time.Time       `json:"published_at,omitempty"`
	LastSeenAt       *time.Time       `json:"last_seen_at,omitempty"`
}

type SaleListing struct {
	ID               string           `json:"id"`
	Source           ListingSource    `json:"source"`
	Headline         string           `json:"headline"`
	Location         Location         `json:"location"`
	Property         PropertyDetails  `json:"property"`
	SaleTerms        SaleTerms        `json:"sale_terms"`
	Charges          Charges          `json:"charges,omitempty"`
	Texts            TextSections     `json:"texts,omitempty"`
	Media            Media            `json:"media,omitempty"`
	Contacts         []Contact        `json:"contacts,omitempty"`
	Showings         []Showing        `json:"showings,omitempty"`
	Links            []Link           `json:"links,omitempty"`
	BuildingIdentity BuildingIdentity `json:"building_identity"`
	Building         *BuildingSummary `json:"building,omitempty"`
	Insights         ListingInsights  `json:"insights,omitempty"`
}

type Rental struct {
	ID               string           `json:"id"`
	Source           ListingSource    `json:"source"`
	Headline         string           `json:"headline"`
	Location         Location         `json:"location"`
	Property         PropertyDetails  `json:"property"`
	RentalTerms      RentalTerms      `json:"rental_terms"`
	Charges          Charges          `json:"charges,omitempty"`
	Texts            TextSections     `json:"texts,omitempty"`
	Media            Media            `json:"media,omitempty"`
	Contacts         []Contact        `json:"contacts,omitempty"`
	Showings         []Showing        `json:"showings,omitempty"`
	Links            []Link           `json:"links,omitempty"`
	BuildingIdentity BuildingIdentity `json:"building_identity"`
	Building         *BuildingSummary `json:"building,omitempty"`
	Insights         ListingInsights  `json:"insights,omitempty"`
}

type Building struct {
	ID               string               `json:"id"`
	Identity         BuildingIdentity     `json:"identity"`
	SourceRecords    []ListingSource      `json:"source_records"`
	Location         Location             `json:"location"`
	HousingCompany   string               `json:"housing_company,omitempty"`
	BusinessID       string               `json:"business_id,omitempty"`
	BuildingType     string               `json:"building_type,omitempty"`
	BuildingSubtype  string               `json:"building_subtype,omitempty"`
	BuildYear        *int32               `json:"build_year,omitempty"`
	ConstructionYear *int32               `json:"construction_year,omitempty"`
	FloorCount       *int32               `json:"floor_count,omitempty"`
	ApartmentCount   *int32               `json:"apartment_count,omitempty"`
	EnergyClass      string               `json:"energy_class,omitempty"`
	Heating          string               `json:"heating,omitempty"`
	PlotType         string               `json:"plot_type,omitempty"`
	Elevator         *bool                `json:"elevator,omitempty"`
	Sauna            *bool                `json:"sauna,omitempty"`
	Renovations      []BuildingRenovation `json:"renovations,omitempty"`
	Texts            TextSections         `json:"texts,omitempty"`
	Insights         BuildingInsights     `json:"insights,omitempty"`
	Metadata         map[string]any       `json:"metadata,omitempty"`
}

type BuildingRenovation struct {
	Kind string `json:"kind"`
	Done *bool  `json:"done,omitempty"`
	Year *int32 `json:"year,omitempty"`
}

type rawMap map[string]any

func parseShortcutRaw(payload json.RawMessage) rawMap {
	_, out, err := shortcutpayload.DecodeStoredAd(payload)
	if err != nil {
		return nil
	}
	return rawMap(out)
}

func parseFrontdoorRaw(payload json.RawMessage) rawMap {
	_, out, err := frontdoorpayload.DecodeStoredAd(payload)
	if err != nil {
		return nil
	}
	return rawMap(out)
}
