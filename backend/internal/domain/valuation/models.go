package valuation

import "time"

type SaleListing struct {
	Unit       UnitDetails
	Building   BuildingDetails
	Site       SiteDetails
	Commercial CommercialDetails
	Texts      TextSections
	Insights   ListingInsights
}

type UnitDetails struct {
	Location   Location
	RoomLayout string
	AreaM2     *float64
	FloorLevel *int32
	Condition  string
}

type BuildingDetails struct {
	Location        Location
	HousingCompany  string
	BuildingType    string
	BuildingSubtype string
	BuildYear       *int32
	EnergyClass     string
	Elevator        *bool
	Renovations     []BuildingRenovation
}

type SiteDetails struct {
	PlotOwnershipType string
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
