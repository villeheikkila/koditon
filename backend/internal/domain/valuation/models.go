package valuation

import "time"

type SaleListing struct {
	Unit       UnitDetails
	Building   BuildingDetails
	Site       SiteDetails
	Commercial CommercialDetails
	Texts      TextSections
	Insights   ListingInsights
	Inputs     ValuationInputs
}

type UnitDetails struct {
	Location                  Location
	PropertyType              string
	PropertySubtype           string
	RoomLayout                string
	RoomsCount                *int32
	BedroomsCount             *int32
	AreaM2                    *float64
	LivingAreaM2              *float64
	TotalAreaM2               *float64
	OtherAreaM2               *float64
	FloorLevel                *int32
	Condition                 string
	Sauna                     *bool
	Balcony                   *bool
	Parking                   string
	KitchenDescription        string
	BathroomDescription       string
	StorageDescription        string
	FloorMaterialsDescription string
	WallMaterialsDescription  string
	BalconyDescription        string
	SaunaDescription          string
	ViewsDescription          string
	Appliances                []string
	Features                  []string
}

type BuildingDetails struct {
	Location         Location
	HousingCompany   string
	BusinessID       string
	BuildingType     string
	BuildingSubtype  string
	BuildYear        *int32
	ConstructionYear *int32
	FloorCount       *int32
	ApartmentCount   *int32
	EnergyClass      string
	Heating          string
	HeatingFuel      string
	BuildingMaterial string
	WallStructure    string
	RoofType         string
	RoofMaterial     string
	CommonAreas      string
	CarStorage       string
	OtherInfo        string
	Elevator         *bool
	Sauna            *bool
	Renovations      []BuildingRenovation
}

type SiteDetails struct {
	PlotOwnershipType  string
	PlotType           string
	PlotAreaM2         *float64
	LotRedemptionInfo  string
	LotRentalAgreement string
	Services           string
	Transport          string
}

type CommercialDetails struct {
	AskingPrice         *int64
	DebtFreePrice       *int64
	DebtShareAmount     *int64
	PricePerSquareMeter *float64
	Charges             Charges
	MatchedTransaction  *PriceTransactionMatch
}

type PriceTransactionMatch struct {
	ID                  string
	FirstSeenAt         *time.Time
	UpdatedAt           *time.Time
	Description         string
	Type                string
	Category            string
	AreaM2              *float64
	Price               *int64
	PricePerSquareMeter *int64
	BuildYear           *int32
	Floor               string
	Elevator            *bool
	Condition           string
	Plot                string
	PlotOwned           *bool
	EnergyClass         string
	PeriodIdentifier    string
	City                string
	Neighborhood        string
	PostalCode          string
	MatchStatus         string
	MatchScore          *int32
	MatchConfidence     string
}

type Charges struct {
	MaintenanceMonthly *float64
	TotalMonthly       *float64
	Water              *float64
	Parking            *float64
	Sauna              *float64
	Electricity        string
	Heating            string
	Notes              string
}

type TextSections struct {
	RenovationsDone    string
	RenovationsPlanned string
}

type ListingInsights struct {
	Items []Insight
}

type Insight struct {
	Key         string
	Value       string
	Direction   string
	Severity    string
	Confidence  float64
	Source      string
	Explanation string
}

type ValuationInputs struct {
	Unit        UnitInput           `json:"unit,omitempty"`
	Layout      LayoutInput         `json:"layout,omitempty"`
	Floor       FloorInput          `json:"floor,omitempty"`
	Building    BuildingInput       `json:"building,omitempty"`
	Site        SiteInput           `json:"site,omitempty"`
	Charges     ChargesInput        `json:"charges,omitempty"`
	Renovations RenovationsInput    `json:"renovations,omitempty"`
	Market      MarketInput         `json:"market,omitempty"`
	Documents   DocumentsInput      `json:"documents,omitempty"`
	Facts       []ValuationFact     `json:"facts,omitempty"`
	ExtraFacts  []ValuationFact     `json:"extra_facts,omitempty"`
	Conflicts   []ValuationConflict `json:"conflicts,omitempty"`
	Missing     []string            `json:"missing,omitempty"`
}

type UnitInput struct {
	AreaM2                *float64 `json:"area_m2,omitempty"`
	LivingAreaM2          *float64 `json:"living_area_m2,omitempty"`
	TotalAreaM2           *float64 `json:"total_area_m2,omitempty"`
	OtherAreaM2           *float64 `json:"other_area_m2,omitempty"`
	Condition             string   `json:"condition,omitempty"`
	Sauna                 *bool    `json:"sauna,omitempty"`
	Balcony               *bool    `json:"balcony,omitempty"`
	BalconyGlazing        *bool    `json:"balcony_glazing,omitempty"`
	StorageQuality        string   `json:"storage_quality,omitempty"`
	Parking               string   `json:"parking,omitempty"`
	ViewQuality           string   `json:"view_quality,omitempty"`
	NoiseRisk             *bool    `json:"noise_risk,omitempty"`
	Accessibility         string   `json:"accessibility,omitempty"`
	SurfaceRenovationNeed *bool    `json:"surface_renovation_need,omitempty"`
	ModernizationNeed     *bool    `json:"modernization_need,omitempty"`
	KitchenRenovated      *bool    `json:"kitchen_renovated,omitempty"`
	BathroomRenovated     *bool    `json:"bathroom_renovated,omitempty"`
}

type LayoutInput struct {
	RoomLayout      string `json:"room_layout,omitempty"`
	RoomCount       *int32 `json:"room_count,omitempty"`
	BedroomCount    *int32 `json:"bedroom_count,omitempty"`
	KitchenType     string `json:"kitchen_type,omitempty"`
	SeparateKitchen *bool  `json:"separate_kitchen,omitempty"`
	OpenKitchen     *bool  `json:"open_kitchen,omitempty"`
	Alcove          *bool  `json:"alcove,omitempty"`
	SeparateWCCount *int32 `json:"separate_wc_count,omitempty"`
	AwkwardLayout   *bool  `json:"awkward_layout,omitempty"`
	LayoutQuality   string `json:"layout_quality,omitempty"`
}

type FloorInput struct {
	FloorLevel        *int32 `json:"floor_level,omitempty"`
	TotalFloors       *int32 `json:"total_floors,omitempty"`
	GroundFloor       *bool  `json:"ground_floor,omitempty"`
	TopFloor          *bool  `json:"top_floor,omitempty"`
	HighFloor         *bool  `json:"high_floor,omitempty"`
	Elevator          *bool  `json:"elevator,omitempty"`
	ElevatorRelevance string `json:"elevator_relevance,omitempty"`
}

type BuildingInput struct {
	BuildYear         *int32 `json:"build_year,omitempty"`
	BuildingType      string `json:"building_type,omitempty"`
	EnergyClass       string `json:"energy_class,omitempty"`
	HeatingMethod     string `json:"heating_method,omitempty"`
	BuildingMaterial  string `json:"building_material,omitempty"`
	RoofType          string `json:"roof_type,omitempty"`
	RoofMaterial      string `json:"roof_material,omitempty"`
	Elevator          *bool  `json:"elevator,omitempty"`
	ApartmentCount    *int32 `json:"apartment_count,omitempty"`
	CommonAreaQuality string `json:"common_area_quality,omitempty"`
}

type SiteInput struct {
	PlotOwnershipType  string   `json:"plot_ownership_type,omitempty"`
	PlotType           string   `json:"plot_type,omitempty"`
	PlotAreaM2         *float64 `json:"plot_area_m2,omitempty"`
	LotRedemptionInfo  string   `json:"lot_redemption_info,omitempty"`
	LotRentalAgreement string   `json:"lot_rental_agreement,omitempty"`
	Services           string   `json:"services,omitempty"`
	Transport          string   `json:"transport,omitempty"`
}

type ChargesInput struct {
	MaintenanceMonthly *float64 `json:"maintenance_monthly,omitempty"`
	TotalMonthly       *float64 `json:"total_monthly,omitempty"`
	Water              *float64 `json:"water,omitempty"`
	Parking            *float64 `json:"parking,omitempty"`
	Sauna              *float64 `json:"sauna,omitempty"`
	Electricity        string   `json:"electricity,omitempty"`
	Heating            string   `json:"heating,omitempty"`
	Notes              string   `json:"notes,omitempty"`
	ChargeRisk         string   `json:"charge_risk,omitempty"`
}

type RenovationsInput struct {
	Completed []BuildingRenovation      `json:"completed,omitempty"`
	Planned   []BuildingRenovation      `json:"planned,omitempty"`
	Forecast  []ApartmentRenovationNeed `json:"forecast,omitempty"`
}

type MarketInput struct {
	AskingPrice         *int64                 `json:"asking_price,omitempty"`
	DebtFreePrice       *int64                 `json:"debt_free_price,omitempty"`
	DebtShareAmount     *int64                 `json:"debt_share_amount,omitempty"`
	PricePerSquareMeter *float64               `json:"price_per_m2,omitempty"`
	MatchedTransaction  *PriceTransactionMatch `json:"matched_transaction,omitempty"`
}

type DocumentsInput struct {
	ManagerCertificateLoaded bool `json:"manager_certificate_loaded,omitempty"`
	FinancialStatementLoaded bool `json:"financial_statement_loaded,omitempty"`
}

type ValuationConflict struct {
	Path     string        `json:"path"`
	Chosen   ValuationFact `json:"chosen"`
	Rejected ValuationFact `json:"rejected"`
	Reason   string        `json:"reason"`
}

type ValuationFact struct {
	Section     string   `json:"section"`
	Key         string   `json:"key"`
	ValueKind   string   `json:"value_kind"`
	ValueText   string   `json:"value_text,omitempty"`
	ValueNumber *float64 `json:"value_number,omitempty"`
	ValueBool   *bool    `json:"value_bool,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	Source      string   `json:"source,omitempty"`
	Evidence    string   `json:"evidence,omitempty"`
	Model       string   `json:"model,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
}

type Location struct {
	StreetAddress string
	City          string
	Postal        string
}

type BuildingRenovation struct {
	Kind            string
	Component       string
	Done            *bool
	Year            *int32
	Scope           string
	Stage           string
	Responsibility  string
	CostEstimateEUR *int64
	Text            string
	Confidence      *int32
	Source          string
}

type ApartmentValuation struct {
	Subject         ApartmentValuationSubject     `json:"subject"`
	Price           ApartmentValuationPrice       `json:"price"`
	Renovations     ApartmentValuationRenovations `json:"renovations"`
	Input           ValuationInputs               `json:"input,omitempty"`
	Brief           ValuationBrief                `json:"brief,omitempty"`
	OfferAssessment ApartmentOfferAssessment      `json:"offer_assessment"`
	Signals         []ApartmentValuationSignal    `json:"signals,omitempty"`
	Explanation     string                        `json:"explanation"`
	Confidence      string                        `json:"confidence"`
	Missing         []string                      `json:"missing,omitempty"`
}

type ApartmentValuationSubject struct {
	Address        string   `json:"address,omitempty"`
	City           string   `json:"city,omitempty"`
	Postal         string   `json:"postal,omitempty"`
	RoomLayout     string   `json:"room_layout,omitempty"`
	AreaM2         *float64 `json:"area_m2,omitempty"`
	BuildYear      *int32   `json:"build_year,omitempty"`
	Condition      string   `json:"condition,omitempty"`
	EnergyClass    string   `json:"energy_class,omitempty"`
	PlotOwnership  string   `json:"plot_ownership,omitempty"`
	HousingCompany string   `json:"housing_company,omitempty"`
}

type ApartmentValuationPrice struct {
	AskingPrice                   *int64                 `json:"asking_price,omitempty"`
	DebtFreePrice                 *int64                 `json:"debt_free_price,omitempty"`
	ListingPricePerM2             *float64               `json:"listing_price_per_m2,omitempty"`
	MatchedTransaction            *PriceTransactionMatch `json:"matched_transaction,omitempty"`
	TransactionPriceDelta         *int64                 `json:"transaction_price_delta,omitempty"`
	TransactionPriceDeltaPct      *float64               `json:"transaction_price_delta_pct,omitempty"`
	TransactionPricePerM2Delta    *float64               `json:"transaction_price_per_m2_delta,omitempty"`
	TransactionPricePerM2DeltaPct *float64               `json:"transaction_price_per_m2_delta_pct,omitempty"`
}

type ApartmentValuationRenovations struct {
	Completed            []ApartmentRenovationItem `json:"completed,omitempty"`
	Upcoming             []ApartmentRenovationItem `json:"upcoming,omitempty"`
	Next40Years          []ApartmentRenovationNeed `json:"next_40_years,omitempty"`
	RawCompleted         string                    `json:"raw_completed,omitempty"`
	RawPlanned           string                    `json:"raw_planned,omitempty"`
	ExtractionModel      string                    `json:"extraction_model,omitempty"`
	ForecastStartYear    int32                     `json:"forecast_start_year,omitempty"`
	ForecastHorizonYears int32                     `json:"forecast_horizon_years,omitempty"`
}

type ApartmentRenovationItem struct {
	Category    string `json:"category"`
	Status      string `json:"status"`
	Year        *int32 `json:"year,omitempty"`
	Source      string `json:"source,omitempty"`
	PriceEffect string `json:"price_effect"`
	Explanation string `json:"explanation"`
}

type ApartmentRenovationNeed struct {
	Category        string   `json:"category"`
	Component       string   `json:"component,omitempty"`
	Status          string   `json:"status"`
	Scope           string   `json:"scope,omitempty"`
	Stage           string   `json:"stage,omitempty"`
	Responsibility  string   `json:"responsibility,omitempty"`
	Year            *int32   `json:"year,omitempty"`
	YearRange       string   `json:"year_range,omitempty"`
	WindowStartYear *int32   `json:"window_start_year,omitempty"`
	WindowEndYear   *int32   `json:"window_end_year,omitempty"`
	BasisYear       *int32   `json:"basis_year,omitempty"`
	CycleYears      *int32   `json:"cycle_years,omitempty"`
	Severity        string   `json:"severity"`
	Confidence      string   `json:"confidence,omitempty"`
	CostEstimateEUR *int64   `json:"cost_estimate_eur,omitempty"`
	PriceEffect     string   `json:"price_effect"`
	Source          string   `json:"source"`
	Basis           []string `json:"basis,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	PriceMechanisms []string `json:"price_mechanisms,omitempty"`
	Explanation     string   `json:"explanation"`
}

type ValuationBrief struct {
	Verdict          string                `json:"verdict"`
	Label            string                `json:"label,omitempty"`
	BuildingRisk     string                `json:"building_risk,omitempty"`
	ExpensiveWindows []OwnershipCostWindow `json:"expensive_windows,omitempty"`
	KeyRenovations   []KeyRenovationStatus `json:"key_renovations,omitempty"`
	TopRisks         []BriefSignal         `json:"top_risks,omitempty"`
	TopPositives     []BriefSignal         `json:"top_positives,omitempty"`
	MissingEvidence  []string              `json:"missing_evidence,omitempty"`
	Confidence       string                `json:"confidence"`
	Explanation      string                `json:"explanation,omitempty"`
}

type OwnershipCostWindow struct {
	StartYear *int32   `json:"start_year,omitempty"`
	EndYear   *int32   `json:"end_year,omitempty"`
	Severity  string   `json:"severity"`
	Label     string   `json:"label"`
	Reasons   []string `json:"reasons,omitempty"`
}

type KeyRenovationStatus struct {
	Category        string `json:"category"`
	Status          string `json:"status"`
	Year            *int32 `json:"year,omitempty"`
	WindowStartYear *int32 `json:"window_start_year,omitempty"`
	WindowEndYear   *int32 `json:"window_end_year,omitempty"`
	Severity        string `json:"severity,omitempty"`
	Confidence      string `json:"confidence,omitempty"`
	Explanation     string `json:"explanation,omitempty"`
}

type BriefSignal struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Severity    string `json:"severity"`
	Direction   string `json:"direction"`
	Explanation string `json:"explanation,omitempty"`
}

type ApartmentOfferAssessment struct {
	Verdict                    string                         `json:"verdict"`
	AskingPrice                *int64                         `json:"asking_price,omitempty"`
	DebtFreePrice              *int64                         `json:"debt_free_price,omitempty"`
	MarketValueRange           ApartmentValueRange            `json:"market_value_range,omitempty"`
	RiskAdjustedValueRange     ApartmentValueRange            `json:"risk_adjusted_value_range,omitempty"`
	RecommendedOfferRange      ApartmentValueRange            `json:"recommended_offer_range,omitempty"`
	RenovationRiskReserve      ApartmentValueRange            `json:"renovation_risk_reserve,omitempty"`
	RenovationRiskReservePerM2 ApartmentValueRange            `json:"renovation_risk_reserve_per_m2,omitempty"`
	EstimatedOwnershipCost     ApartmentOwnershipCostEstimate `json:"estimated_ownership_cost,omitempty"`
	Confidence                 string                         `json:"confidence"`
	MainReasons                []ApartmentOfferReason         `json:"main_reasons,omitempty"`
	Missing                    []string                       `json:"missing,omitempty"`
	Explanation                string                         `json:"explanation"`
}

type ApartmentValueRange struct {
	Low  *int64 `json:"low,omitempty"`
	High *int64 `json:"high,omitempty"`
}

type ApartmentOwnershipCostEstimate struct {
	CurrentMonthlyCharges      *float64 `json:"current_monthly_charges,omitempty"`
	CurrentMonthlyChargesPerM2 *float64 `json:"current_monthly_charges_per_m2,omitempty"`
	StressMonthlyCharges       *float64 `json:"stress_monthly_charges,omitempty"`
	StressMonthlyChargesPerM2  *float64 `json:"stress_monthly_charges_per_m2,omitempty"`
	StressAssumption           string   `json:"stress_assumption,omitempty"`
	CurrentDebtShare           *int64   `json:"current_debt_share,omitempty"`
	FinancingMissing           bool     `json:"financing_missing,omitempty"`
	CompanyFinancialsMissing   bool     `json:"company_financials_missing,omitempty"`
	ManagerCertificateMissing  bool     `json:"manager_certificate_missing,omitempty"`
}

type ApartmentOfferReason struct {
	Key         string `json:"key"`
	Direction   string `json:"direction"`
	Severity    string `json:"severity"`
	Explanation string `json:"explanation"`
}

type ApartmentValuationSignal struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Direction   string `json:"direction"`
	Severity    string `json:"severity"`
	Explanation string `json:"explanation"`
	Source      string `json:"source,omitempty"`
}
