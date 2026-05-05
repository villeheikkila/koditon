package valuation

import "testing"

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
