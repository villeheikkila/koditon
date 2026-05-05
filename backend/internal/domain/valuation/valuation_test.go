package valuation

import (
	"testing"
	"time"
)

func TestAssessExplainsTransactionAndUpcomingRenovations(t *testing.T) {
	area := 50.0
	asking := int64(230000)
	transactionPrice := int64(200000)
	transactionPriceM2 := int64(4000)
	listingPriceM2 := 4600.0
	planned := false
	year := int32(2027)
	listing := SaleListing{Unit: UnitDetails{Location: Location{StreetAddress: "Testikatu 1", City: "Helsinki", Postal: "00100"}, RoomLayout: "2h+k", AreaM2: &area}, Building: BuildingDetails{HousingCompany: "As Oy Testi", Renovations: []BuildingRenovation{{Kind: "pipe", Done: &planned, Year: &year}}}, Commercial: CommercialDetails{AskingPrice: &asking, PricePerSquareMeter: &listingPriceM2, MatchedTransaction: &PriceTransactionMatch{Price: &transactionPrice, PricePerSquareMeter: &transactionPriceM2}}, Texts: TextSections{RenovationsPlanned: "Putkiremontti suunnitteilla 2027"}}
	valuation := Assess(listing)
	if valuation.Price.TransactionPriceDeltaPct == nil || *valuation.Price.TransactionPriceDeltaPct != 15 {
		t.Fatalf("expected 15%% price delta, got %#v", valuation.Price.TransactionPriceDeltaPct)
	}
	if len(valuation.Renovations.Upcoming) != 1 || valuation.Renovations.Upcoming[0].Category != "pipe" {
		t.Fatalf("expected upcoming pipe renovation, got %#v", valuation.Renovations.Upcoming)
	}
	if valuation.Confidence != "high" {
		t.Fatalf("expected high confidence, got %q", valuation.Confidence)
	}
	if valuation.OfferAssessment.Verdict == "" || valuation.OfferAssessment.RiskAdjustedValueRange.High == nil {
		t.Fatalf("expected offer assessment with risk-adjusted range, got %#v", valuation.OfferAssessment)
	}
	var hasPriceRisk, hasRenovationRisk bool
	for _, signal := range valuation.Signals {
		hasPriceRisk = hasPriceRisk || signal.Key == "transaction_price_delta" && signal.Direction == "reduces"
		hasRenovationRisk = hasRenovationRisk || signal.Key == "upcoming_renovation.pipe" && signal.Direction == "risk"
	}
	if !hasPriceRisk || !hasRenovationRisk {
		t.Fatalf("expected price and renovation risk signals, got %#v", valuation.Signals)
	}
}

func TestAssessFlagsOverpricedRiskAdjustedOffer(t *testing.T) {
	area := 50.0
	asking := int64(260000)
	transactionPrice := int64(200000)
	transactionPriceM2 := int64(4000)
	planned := false
	year := int32(2027)
	listing := SaleListing{Unit: UnitDetails{AreaM2: &area}, Building: BuildingDetails{Renovations: []BuildingRenovation{{Kind: "pipe", Done: &planned, Year: &year, Scope: "full"}}}, Commercial: CommercialDetails{AskingPrice: &asking, DebtFreePrice: &asking, MatchedTransaction: &PriceTransactionMatch{Price: &transactionPrice, PricePerSquareMeter: &transactionPriceM2}}}
	assessment := Assess(listing).OfferAssessment
	if assessment.Verdict != "overpriced" {
		t.Fatalf("expected overpriced verdict, got %#v", assessment)
	}
	if assessment.RenovationRiskReserve.High == nil || *assessment.RenovationRiskReserve.High == 0 {
		t.Fatalf("expected renovation reserve, got %#v", assessment.RenovationRiskReserve)
	}
}

func TestBuildInputPromotesProviderAndExtractedFacts(t *testing.T) {
	area := 51.5
	rooms := int32(2)
	bedrooms := int32(1)
	floor := int32(5)
	totalFloors := int32(6)
	elevator := true
	balcony := true
	glazed := true
	renovated := true
	listing := SaleListing{
		Unit:     UnitDetails{RoomLayout: "2h+avok", RoomsCount: &rooms, BedroomsCount: &bedrooms, AreaM2: &area, FloorLevel: &floor, Balcony: &balcony},
		Building: BuildingDetails{FloorCount: &totalFloors, Elevator: &elevator},
		Inputs: ValuationInputs{Facts: []ValuationFact{
			{Section: "balcony", Key: "glazing", ValueKind: "bool", ValueBool: &glazed, Confidence: 0.82, Source: "llm_balcony_description_text"},
			{Section: "kitchen", Key: "renovated", ValueKind: "bool", ValueBool: &renovated, Confidence: 0.78, Source: "llm_kitchen_description_text"},
		}},
	}
	input := BuildInput(listing)
	if input.Unit.AreaM2 == nil || *input.Unit.AreaM2 != area {
		t.Fatalf("expected provider area, got %#v", input.Unit.AreaM2)
	}
	if input.Layout.RoomCount == nil || *input.Layout.RoomCount != rooms || input.Layout.KitchenType != "open" {
		t.Fatalf("expected parsed layout input, got %#v", input.Layout)
	}
	if input.Floor.HighFloor == nil || !*input.Floor.HighFloor || input.Floor.ElevatorRelevance != "elevator_present" {
		t.Fatalf("expected high-floor elevator input, got %#v", input.Floor)
	}
	if input.Unit.BalconyGlazing == nil || !*input.Unit.BalconyGlazing || input.Unit.KitchenRenovated == nil || !*input.Unit.KitchenRenovated {
		t.Fatalf("expected promoted extracted facts, got %#v", input.Unit)
	}
}

func TestBuildInputRetainsProviderValueAndReportsConflict(t *testing.T) {
	balcony := true
	noBalcony := false
	listing := SaleListing{Unit: UnitDetails{Balcony: &balcony}, Inputs: ValuationInputs{Facts: []ValuationFact{{Section: "balcony", Key: "glazing", ValueKind: "bool", ValueBool: &noBalcony, Confidence: 0.7, Source: "llm_balcony_description_text"}}}}
	input := BuildInput(listing)
	if input.Unit.BalconyGlazing == nil || *input.Unit.BalconyGlazing {
		t.Fatalf("expected extracted glazing fact to be false, got %#v", input.Unit.BalconyGlazing)
	}
	listing = SaleListing{Unit: UnitDetails{Balcony: &balcony}, Inputs: ValuationInputs{Facts: []ValuationFact{{Section: "unit", Key: "balcony", ValueKind: "bool", ValueBool: &noBalcony, Confidence: 0.7, Source: "llm_description_text"}}}}
	input = BuildInput(listing)
	if input.Unit.Balcony == nil || !*input.Unit.Balcony {
		t.Fatalf("expected provider balcony to remain true, got %#v", input.Unit.Balcony)
	}
	if len(input.Conflicts) != 1 || input.Conflicts[0].Path != "unit.balcony" {
		t.Fatalf("expected balcony conflict, got %#v", input.Conflicts)
	}
}

func TestAssessExposesCanonicalInput(t *testing.T) {
	area := 42.0
	listing := SaleListing{Unit: UnitDetails{RoomLayout: "1h+kk", AreaM2: &area}}
	result := Assess(listing)
	if result.Input.Unit.AreaM2 == nil || *result.Input.Unit.AreaM2 != area {
		t.Fatalf("expected canonical input in valuation result, got %#v", result.Input)
	}
	if result.Input.Layout.KitchenType != "kitchenette" {
		t.Fatalf("expected kitchenette layout input, got %#v", result.Input.Layout)
	}
}

func TestAssessIncludesBrief(t *testing.T) {
	area := 42.0
	listing := SaleListing{Unit: UnitDetails{AreaM2: &area}, Commercial: CommercialDetails{AskingPrice: ptrInt64(200000)}}
	result := Assess(listing)
	if result.Brief.Verdict == "" || result.Brief.Confidence == "" {
		t.Fatalf("expected valuation brief, got %#v", result.Brief)
	}
}

func TestBuildBriefFlagsExpensiveRenovationWindow(t *testing.T) {
	area := 60.0
	planned := false
	pipeYear := int32(time.Now().Year() + 1)
	roofYear := int32(time.Now().Year() + 2)
	listing := SaleListing{Unit: UnitDetails{AreaM2: &area}, Building: BuildingDetails{Renovations: []BuildingRenovation{{Kind: "pipe", Done: &planned, Year: &pipeYear, Scope: "full"}, {Kind: "roof", Done: &planned, Year: &roofYear, Scope: "full"}}}, Commercial: CommercialDetails{AskingPrice: ptrInt64(200000)}}
	result := Assess(listing)
	if result.Brief.BuildingRisk != "high" {
		t.Fatalf("expected high building risk, got %#v", result.Brief)
	}
	if len(result.Brief.ExpensiveWindows) == 0 || result.Brief.ExpensiveWindows[0].Severity != "high" {
		t.Fatalf("expected high expensive window, got %#v", result.Brief.ExpensiveWindows)
	}
}

func TestBuildBriefShowsMissingEvidence(t *testing.T) {
	result := Assess(SaleListing{})
	missing := map[string]bool{}
	for _, item := range result.Brief.MissingEvidence {
		missing[item] = true
	}
	if !missing["manager certificate"] || !missing["housing company financials"] || !missing["matched transaction"] {
		t.Fatalf("expected human missing evidence, got %#v", result.Brief.MissingEvidence)
	}
}

func TestBuildBriefPromotesPositiveInputs(t *testing.T) {
	glazed := true
	renovated := true
	listing := SaleListing{Inputs: ValuationInputs{Facts: []ValuationFact{{Section: "balcony", Key: "glazing", ValueKind: "bool", ValueBool: &glazed, Confidence: 0.8}, {Section: "kitchen", Key: "renovated", ValueKind: "bool", ValueBool: &renovated, Confidence: 0.8}}}}
	result := Assess(listing)
	positives := map[string]bool{}
	for _, item := range result.Brief.TopPositives {
		positives[item.Key] = true
	}
	if !positives["balcony_glazing"] || !positives["kitchen_renovated"] {
		t.Fatalf("expected canonical input positives, got %#v", result.Brief.TopPositives)
	}
}

func TestForecastRenovationsProjectsNextFortyYears(t *testing.T) {
	done := true
	planned := false
	pipeYear := int32(1987)
	roofYear := int32(1998)
	windowYear := int32(2025)
	electricityYear := int32(2026)
	renovations := []BuildingRenovation{{Kind: "Putkiremontti", Done: &done, Year: &pipeYear}, {Kind: "roof", Done: &done, Year: &roofYear}, {Kind: "window", Done: &planned, Year: &windowYear}, {Kind: "electricity", Done: &planned, Year: &electricityYear}}
	buildYear := int32(1934)
	needs := ForecastRenovations(BuildingDetails{BuildYear: &buildYear, BuildingType: "Apartment house"}, renovations, 2026, 40)
	assertRenovationNeed(t, needs, "window", "verify_status", 2025)
	assertRenovationNeed(t, needs, "electricity", "planned", 2026)
	assertRenovationNeed(t, needs, "roof", "expected", 2028)
	assertRenovationNeed(t, needs, "pipe", "expected", 2037)
	assertRenovationNeed(t, needs, "facade", "expected", 2054)
	assertRenovationNeed(t, needs, "heating", "expected", 2034)
}

func TestForecastRenovationsKeepsSeparatePlannedYears(t *testing.T) {
	done := true
	planned := false
	doneYear := int32(1987)
	studyYear := int32(2026)
	liningYear := int32(2033)
	renovations := []BuildingRenovation{{Kind: "pipe", Done: &done, Year: &doneYear}, {Kind: "pipe", Done: &planned, Year: &studyYear, Text: "Putkiston kuntotutkimus"}, {Kind: "pipe", Done: &planned, Year: &liningYear}}
	buildYear := int32(1934)
	needs := ForecastRenovations(BuildingDetails{BuildYear: &buildYear, BuildingType: "Apartment house"}, renovations, 2026, 40)
	assertRenovationNeed(t, needs, "pipe", "planned", 2026)
	assertRenovationNeed(t, needs, "pipe", "planned", 2033)
	assertRenovationNeed(t, needs, "pipe", "follow_up", 2029)
}

func TestForecastRenovationsUsesDeclarativeBuildingRules(t *testing.T) {
	done := true
	elevator := true
	buildYear := int32(1965)
	elevatorYear := int32(2001)
	renovations := []BuildingRenovation{{Kind: "elevator", Done: &done, Year: &elevatorYear}}
	needs := ForecastRenovations(BuildingDetails{BuildYear: &buildYear, BuildingType: "Apartment house", Elevator: &elevator}, renovations, 2026, 40)
	assertRenovationNeed(t, needs, "pipe", "expected", 2065)
	assertRenovationNeed(t, needs, "elevator", "expected", 2031)
	assertRenovationNeed(t, needs, "electricity", "expected", 2045)
	assertRenovationNeed(t, needs, "water_supply", "expected", 2065)
}

func TestForecastRenovationsCarriesScopeWindowAndEvidence(t *testing.T) {
	done := true
	buildYear := int32(1970)
	roofYear := int32(2020)
	confidence := int32(92)
	renovations := []BuildingRenovation{{Kind: "roof", Done: &done, Year: &roofYear, Text: "Vesikaton huoltomaalaus", Confidence: &confidence, Source: "llm_renovations_done_text"}}
	needs := ForecastRenovations(BuildingDetails{BuildYear: &buildYear, BuildingType: "Apartment house"}, renovations, 2026, 40)
	need := findRenovationNeed(t, needs, "roof", "expected", 2035)
	if need.Scope != "partial" || need.Confidence != "medium" {
		t.Fatalf("expected partial medium-confidence roof forecast, got %#v", need)
	}
	if need.WindowStartYear == nil || *need.WindowStartYear != 2033 || need.WindowEndYear == nil || *need.WindowEndYear != 2037 {
		t.Fatalf("expected 2033-2037 window, got %#v", need)
	}
	if len(need.Basis) == 0 || len(need.PriceMechanisms) == 0 {
		t.Fatalf("expected basis and price mechanisms, got %#v", need)
	}
}

func assertRenovationNeed(t *testing.T, needs []ApartmentRenovationNeed, category string, status string, year int32) {
	t.Helper()
	_ = findRenovationNeed(t, needs, category, status, year)
}

func findRenovationNeed(t *testing.T, needs []ApartmentRenovationNeed, category string, status string, year int32) ApartmentRenovationNeed {
	t.Helper()
	for _, need := range needs {
		if need.Category == category && need.Status == status && need.Year != nil && *need.Year == year {
			return need
		}
	}
	t.Fatalf("missing forecast row for %s %s %d in %#v", category, status, year, needs)
	return ApartmentRenovationNeed{}
}

func ptrInt64(value int64) *int64 {
	return &value
}
