package valuation

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func Assess(listing SaleListing) *ApartmentValuation {
	return apartmentValuation(listing)
}

func ForecastRenovations(building BuildingDetails, renovations []BuildingRenovation, startYear int32, horizonYears int32) []ApartmentRenovationNeed {
	return apartmentRenovationNeeds(building, renovations, startYear, horizonYears)
}

func RenovationSeverity(category string) string {
	return renovationSeverity(category)
}

func apartmentValuation(listing SaleListing) *ApartmentValuation {
	valuation := &ApartmentValuation{Subject: apartmentValuationSubject(listing), Price: apartmentValuationPrice(listing), Renovations: apartmentValuationRenovations(listing)}
	valuation.Signals = apartmentValuationSignals(listing, valuation)
	valuation.Missing = apartmentValuationMissingInputs(listing)
	valuation.Confidence = apartmentValuationConfidence(listing, valuation)
	valuation.OfferAssessment = apartmentOfferAssessment(listing, valuation)
	valuation.Explanation = apartmentValuationExplanation(listing, valuation)
	return valuation
}

func apartmentValuationSubject(listing SaleListing) ApartmentValuationSubject {
	return ApartmentValuationSubject{Address: listing.Unit.Location.StreetAddress, City: listing.Unit.Location.City, Postal: listing.Unit.Location.Postal, RoomLayout: listing.Unit.RoomLayout, AreaM2: listing.Unit.AreaM2, BuildYear: listing.Building.BuildYear, Condition: listing.Unit.Condition, EnergyClass: listing.Building.EnergyClass, PlotOwnership: listing.Site.PlotOwnershipType, HousingCompany: listing.Building.HousingCompany}
}

func apartmentValuationPrice(listing SaleListing) ApartmentValuationPrice {
	price := ApartmentValuationPrice{AskingPrice: listing.Commercial.AskingPrice, DebtFreePrice: listing.Commercial.DebtFreePrice, ListingPricePerM2: listing.Commercial.PricePerSquareMeter, MatchedTransaction: listing.Commercial.MatchedTransaction}
	if listing.Commercial.MatchedTransaction != nil {
		if listing.Commercial.AskingPrice != nil && listing.Commercial.MatchedTransaction.Price != nil && *listing.Commercial.MatchedTransaction.Price > 0 {
			delta := *listing.Commercial.AskingPrice - *listing.Commercial.MatchedTransaction.Price
			pct := roundPercent(float64(delta) / float64(*listing.Commercial.MatchedTransaction.Price) * 100)
			price.TransactionPriceDelta = &delta
			price.TransactionPriceDeltaPct = &pct
		}
		if listing.Commercial.PricePerSquareMeter != nil && listing.Commercial.MatchedTransaction.PricePerSquareMeter != nil && *listing.Commercial.MatchedTransaction.PricePerSquareMeter > 0 {
			delta := *listing.Commercial.PricePerSquareMeter - float64(*listing.Commercial.MatchedTransaction.PricePerSquareMeter)
			pct := roundPercent(delta / float64(*listing.Commercial.MatchedTransaction.PricePerSquareMeter) * 100)
			price.TransactionPricePerM2Delta = &delta
			price.TransactionPricePerM2DeltaPct = &pct
		}
	}
	return price
}

func apartmentValuationRenovations(listing SaleListing) ApartmentValuationRenovations {
	renovations := compactRenovations(listing.Building.Renovations)
	startYear := int32(time.Now().Year())
	out := ApartmentValuationRenovations{RawCompleted: cleanDisplayString(listing.Texts.RenovationsDone), RawPlanned: cleanDisplayString(listing.Texts.RenovationsPlanned), ExtractionModel: "source_structured_fields", ForecastStartYear: startYear, ForecastHorizonYears: 40, Next40Years: apartmentRenovationNeeds(listing.Building, renovations, startYear, 40)}
	for _, renovation := range renovations {
		item := apartmentRenovationItem(renovation)
		if renovation.Done != nil && !*renovation.Done {
			out.Upcoming = append(out.Upcoming, item)
			continue
		}
		out.Completed = append(out.Completed, item)
	}
	if out.RawPlanned != "" && len(out.Upcoming) == 0 {
		out.Upcoming = append(out.Upcoming, ApartmentRenovationItem{Category: "planned", Status: "planned", Source: "listing_text", PriceEffect: "risk", Explanation: "The listing contains planned renovation text that should be structured by the LLM extractor before relying on the valuation."})
	}
	return out
}

func apartmentRenovationItem(renovation BuildingRenovation) ApartmentRenovationItem {
	status := "unknown"
	source := "structured_renovation"
	if renovation.Done != nil && *renovation.Done {
		status = "done"
	}
	if renovation.Done != nil && !*renovation.Done {
		status = "planned"
	}
	effect := renovationPriceEffect(renovation.Kind, status)
	return ApartmentRenovationItem{Category: renovation.Kind, Status: status, Year: renovation.Year, Source: source, PriceEffect: effect, Explanation: renovationExplanation(renovation.Kind, status, renovation.Year)}
}

func apartmentRenovationNeeds(building BuildingDetails, renovations []BuildingRenovation, startYear int32, horizonYears int32) []ApartmentRenovationNeed {
	endYear := startYear + horizonYears
	latestCompleted := map[string]BuildingRenovation{}
	plannedCategories := map[string]struct{}{}
	var plannedRenovations []BuildingRenovation
	for _, renovation := range renovations {
		category := normalizeRenovationCategory(renovation.Kind)
		if category == "" {
			continue
		}
		renovation.Kind = category
		if renovation.Scope == "" {
			renovation.Scope = firstNonEmpty(renovationScopeFromStage(renovation.Stage), inferRenovationScope(renovation.Text+" "+renovation.Kind+" "+renovation.Component))
		}
		renovation.Stage = firstNonEmpty(renovation.Stage, inferRenovationStage(renovation.Text))
		renovation.Responsibility = firstNonEmpty(renovation.Responsibility, inferRenovationResponsibility(renovation.Text), "unknown")
		if renovation.Done != nil && !*renovation.Done {
			plannedCategories[category] = struct{}{}
			plannedRenovations = append(plannedRenovations, renovation)
			continue
		}
		if renovation.Done == nil || renovation.Year == nil {
			continue
		}
		current, ok := latestCompleted[category]
		if !ok || current.Year == nil || *renovation.Year > *current.Year {
			latestCompleted[category] = renovation
		}
	}
	needs := make([]ApartmentRenovationNeed, 0, len(plannedRenovations)+len(latestCompleted))
	seenPlanned := map[string]struct{}{}
	for _, renovation := range plannedRenovations {
		if renovation.Year != nil && *renovation.Year > endYear {
			continue
		}
		key := renovation.Kind + ":" + fmt.Sprint(renovationNeedSortYear(renovation, startYear))
		if _, ok := seenPlanned[key]; ok {
			continue
		}
		seenPlanned[key] = struct{}{}
		needs = append(needs, plannedRenovationNeed(renovation, startYear, endYear))
		if followUp, ok := plannedRenovationFollowUpNeed(building, renovation, startYear, endYear); ok {
			needs = append(needs, followUp)
		}
	}
	for category, renovation := range latestCompleted {
		if _, ok := plannedCategories[category]; ok {
			continue
		}
		rule, ok := renovationRuleForBuilding(category, building)
		if !ok || renovation.Year == nil {
			continue
		}
		lifecycle := renovationLifecycleForEvidence(rule, renovation)
		dueYear := nextRenovationDueYear(*renovation.Year, lifecycle.LifespanYears, startYear)
		if dueYear > endYear {
			continue
		}
		year := dueYear
		basisYear := *renovation.Year
		needs = append(needs, expectedRenovationNeed(rule, renovation, year, basisYear, "cycle_from_completed_renovation"))
	}
	for _, rule := range renovationRulesForBuilding(building) {
		if _, ok := plannedCategories[rule.Category]; ok {
			continue
		}
		if _, ok := latestCompleted[rule.Category]; ok {
			continue
		}
		if building.BuildYear == nil || !rule.ForecastFromBuildYear {
			continue
		}
		renovation := BuildingRenovation{Kind: rule.Category, Scope: "unknown"}
		lifecycle := renovationLifecycleForEvidence(rule, renovation)
		dueYear := nextRenovationDueYear(*building.BuildYear, lifecycle.LifespanYears, startYear)
		if dueYear > endYear {
			continue
		}
		year := dueYear
		basisYear := *building.BuildYear
		needs = append(needs, expectedRenovationNeed(rule, renovation, year, basisYear, "lifecycle_from_build_year"))
	}
	sortRenovationNeeds(needs, startYear)
	return needs
}

func plannedRenovationNeed(renovation BuildingRenovation, startYear int32, endYear int32) ApartmentRenovationNeed {
	category := normalizeRenovationCategory(renovation.Kind)
	status := "planned"
	source := "structured_planned_renovation"
	yearRange := ""
	scope := firstNonEmpty(renovation.Scope, renovationScopeFromStage(renovation.Stage), inferRenovationScope(renovation.Text+" "+renovation.Kind+" "+renovation.Component), "unknown")
	stage := firstNonEmpty(renovation.Stage, inferRenovationStage(renovation.Text), "unknown")
	responsibility := firstNonEmpty(renovation.Responsibility, inferRenovationResponsibility(renovation.Text), "unknown")
	var year *int32
	var windowStart, windowEnd *int32
	if renovation.Year != nil {
		value := *renovation.Year
		year = &value
		start, end := renovationWindow(value, 2)
		windowStart = &start
		windowEnd = &end
		if value < startYear {
			status = "verify_status"
			yearRange = fmt.Sprintf("%d-now", value)
		}
		if value > endYear {
			status = "long_term_planned"
		}
	}
	dependsOn, mechanisms := plannedRenovationRuleContext(category)
	return ApartmentRenovationNeed{Category: category, Component: renovation.Component, Status: status, Scope: scope, Stage: stage, Responsibility: responsibility, Year: year, YearRange: yearRange, WindowStartYear: windowStart, WindowEndYear: windowEnd, Severity: renovationSeverity(category), Confidence: plannedRenovationConfidence(renovation), CostEstimateEUR: renovation.CostEstimateEUR, PriceEffect: renovationPriceEffect(category, "planned"), Source: source, Basis: plannedRenovationBasis(renovation), DependsOn: dependsOn, PriceMechanisms: mechanisms, Explanation: renovationNeedExplanation(category, status, year)}
}

func plannedRenovationFollowUpNeed(building BuildingDetails, renovation BuildingRenovation, startYear int32, endYear int32) (ApartmentRenovationNeed, bool) {
	if renovation.Year == nil {
		return ApartmentRenovationNeed{}, false
	}
	rule, ok := renovationRuleForBuilding(renovation.Kind, building)
	if !ok {
		return ApartmentRenovationNeed{}, false
	}
	lifecycle := renovationLifecycleForEvidence(rule, renovation)
	if !lifecycle.CreatesFollowUp || lifecycle.FollowUpYears <= 0 {
		return ApartmentRenovationNeed{}, false
	}
	year := *renovation.Year + lifecycle.FollowUpYears
	if year < startYear || year > endYear {
		return ApartmentRenovationNeed{}, false
	}
	windowStart, windowEnd := renovationWindow(year, rule.WindowYears)
	cycleYears := lifecycle.FollowUpYears
	basisYear := *renovation.Year
	basis := plannedRenovationBasis(renovation)
	basis = append(basis, fmt.Sprintf("%s %s creates follow-up execution risk", rule.ID, lifecycle.Scope))
	return ApartmentRenovationNeed{Category: rule.Category, Component: renovation.Component, Status: "follow_up", Scope: "full", Stage: "execution", Responsibility: firstNonEmpty(renovation.Responsibility, "unknown"), Year: &year, WindowStartYear: &windowStart, WindowEndYear: &windowEnd, BasisYear: &basisYear, CycleYears: &cycleYears, Severity: rule.Severity, Confidence: "medium", CostEstimateEUR: renovation.CostEstimateEUR, PriceEffect: renovationPriceEffect(rule.Category, "planned"), Source: "follow_up_from_planned_survey", Basis: basis, DependsOn: rule.DependsOn, PriceMechanisms: rule.PriceMechanisms, Explanation: fmt.Sprintf("%s is in %s stage in %d. The rule catalog treats that as a staging signal, so execution risk is forecast around %d.", rule.Category, lifecycle.Scope, *renovation.Year, year)}, true
}

func plannedRenovationConfidence(renovation BuildingRenovation) string {
	if renovation.Confidence == nil {
		return "medium"
	}
	if *renovation.Confidence >= 80 {
		return "high"
	}
	if *renovation.Confidence < 60 {
		return "low"
	}
	return "medium"
}

func plannedRenovationBasis(renovation BuildingRenovation) []string {
	var basis []string
	if renovation.Text != "" {
		basis = append(basis, renovation.Text)
	}
	if renovation.Source != "" {
		basis = append(basis, "source "+renovation.Source)
	}
	if renovation.Component != "" {
		basis = append(basis, "component "+renovation.Component)
	}
	if renovation.Stage != "" && renovation.Stage != "unknown" {
		basis = append(basis, "stage "+renovation.Stage)
	}
	return basis
}

func plannedRenovationRuleContext(category string) ([]string, []string) {
	if rule, ok := renovationRuleForBuilding(category, BuildingDetails{}); ok {
		return rule.DependsOn, rule.PriceMechanisms
	}
	return nil, []string{"housing company debt", "maintenance charge pressure", "buyer demand"}
}

func renovationNeedExplanation(category string, status string, year *int32) string {
	if status == "verify_status" && year != nil {
		return fmt.Sprintf("%s was listed as planned for %d. Verify whether it has been completed, because unresolved planned work can still affect debt share and charges.", category, *year)
	}
	if year != nil {
		return fmt.Sprintf("%s is already listed as planned for %d, so it should be treated as an explicit future price risk.", category, *year)
	}
	return fmt.Sprintf("%s is listed as planned without a year, so timing needs review before pricing.", category)
}

func renovationNeedSortYear(renovation BuildingRenovation, fallback int32) int32 {
	if renovation.Year == nil {
		return fallback
	}
	return *renovation.Year
}

func sortRenovationNeeds(needs []ApartmentRenovationNeed, startYear int32) {
	severityRank := map[string]int{"high": 0, "medium": 1, "low": 2}
	sort.SliceStable(needs, func(i int, j int) bool {
		left := renovationNeedYear(needs[i], startYear)
		right := renovationNeedYear(needs[j], startYear)
		if left != right {
			return left < right
		}
		if severityRank[needs[i].Severity] != severityRank[needs[j].Severity] {
			return severityRank[needs[i].Severity] < severityRank[needs[j].Severity]
		}
		return needs[i].Category < needs[j].Category
	})
}

func renovationNeedYear(need ApartmentRenovationNeed, fallback int32) int32 {
	if need.Year == nil {
		return fallback
	}
	return *need.Year
}

func apartmentOfferAssessment(listing SaleListing, valuation *ApartmentValuation) ApartmentOfferAssessment {
	area := listing.Unit.AreaM2
	assessment := ApartmentOfferAssessment{AskingPrice: listing.Commercial.AskingPrice, DebtFreePrice: listing.Commercial.DebtFreePrice, Confidence: "low", EstimatedOwnershipCost: apartmentOwnershipCostEstimate(listing)}
	assessment.MarketValueRange = apartmentMarketValueRange(listing)
	assessment.RenovationRiskReservePerM2 = apartmentRenovationRiskReservePerM2(valuation.Renovations.Next40Years)
	assessment.RenovationRiskReserve = apartmentRangeMultiply(assessment.RenovationRiskReservePerM2, area)
	assessment.RiskAdjustedValueRange = apartmentRangeSubtract(assessment.MarketValueRange, assessment.RenovationRiskReserve)
	assessment.RecommendedOfferRange = apartmentRecommendedOfferRange(assessment.RiskAdjustedValueRange)
	assessment.MainReasons = apartmentOfferReasons(listing, valuation, assessment)
	assessment.Missing = apartmentOfferAssessmentMissing(listing)
	assessment.Confidence = apartmentOfferAssessmentConfidence(listing, assessment)
	assessment.Verdict = apartmentOfferVerdict(listing, assessment)
	assessment.Explanation = apartmentOfferExplanation(assessment)
	return assessment
}

func apartmentMarketValueRange(listing SaleListing) ApartmentValueRange {
	if listing.Commercial.MatchedTransaction != nil && listing.Commercial.MatchedTransaction.Price != nil {
		price := *listing.Commercial.MatchedTransaction.Price
		return apartmentIntRange(roundInt64(float64(price)*0.96), roundInt64(float64(price)*1.04))
	}
	if listing.Commercial.MatchedTransaction != nil && listing.Commercial.MatchedTransaction.PricePerSquareMeter != nil && listing.Unit.AreaM2 != nil {
		anchor := float64(*listing.Commercial.MatchedTransaction.PricePerSquareMeter) * *listing.Unit.AreaM2
		return apartmentIntRange(roundInt64(anchor*0.94), roundInt64(anchor*1.06))
	}
	if listing.Commercial.DebtFreePrice != nil {
		price := *listing.Commercial.DebtFreePrice
		return apartmentIntRange(roundInt64(float64(price)*0.92), roundInt64(float64(price)*1.04))
	}
	if listing.Commercial.AskingPrice != nil {
		price := *listing.Commercial.AskingPrice
		return apartmentIntRange(roundInt64(float64(price)*0.90), roundInt64(float64(price)*1.03))
	}
	return ApartmentValueRange{}
}

func apartmentRenovationRiskReservePerM2(needs []ApartmentRenovationNeed) ApartmentValueRange {
	var low, high int64
	for _, need := range needs {
		itemLow, itemHigh := renovationNeedReservePerM2(need)
		low += itemLow
		high += itemHigh
	}
	if high > 1800 {
		high = 1800
	}
	if low > 1200 {
		low = 1200
	}
	return apartmentIntRange(low, high)
}

func renovationNeedReservePerM2(need ApartmentRenovationNeed) (int64, int64) {
	if need.Year != nil && *need.Year > int32(time.Now().Year()+12) && need.Status != "planned" && need.Status != "follow_up" {
		return 0, 0
	}
	switch need.Category {
	case "pipe", "sewer", "water_supply":
		switch need.Status {
		case "planned", "follow_up":
			return reserveByScope(need.Scope, 300, 1100, 150, 450, 75, 250)
		default:
			return reserveByScope(need.Scope, 180, 800, 90, 350, 50, 180)
		}
	case "facade", "roof", "electricity", "elevator":
		return reserveByScope(need.Scope, 120, 550, 60, 240, 30, 120)
	case "heating", "ventilation", "drainage", "balcony", "window":
		return reserveByScope(need.Scope, 60, 300, 30, 160, 20, 80)
	default:
		if need.Severity == "high" {
			return 80, 350
		}
		if need.Severity == "medium" {
			return 30, 160
		}
		return 0, 60
	}
}

func reserveByScope(scope string, fullLow int64, fullHigh int64, partialLow int64, partialHigh int64, planningLow int64, planningHigh int64) (int64, int64) {
	switch scope {
	case "full":
		return fullLow, fullHigh
	case "partial", "maintenance":
		return partialLow, partialHigh
	case "survey", "planning":
		return planningLow, planningHigh
	default:
		return (partialLow + planningLow) / 2, (fullHigh + partialHigh) / 2
	}
}

func apartmentOwnershipCostEstimate(listing SaleListing) ApartmentOwnershipCostEstimate {
	out := ApartmentOwnershipCostEstimate{CurrentDebtShare: listing.Commercial.DebtShareAmount, FinancingMissing: true, CompanyFinancialsMissing: true, ManagerCertificateMissing: true}
	current := listing.Commercial.Charges.TotalMonthly
	if current == nil {
		current = listing.Commercial.Charges.MaintenanceMonthly
	}
	out.CurrentMonthlyCharges = current
	if current != nil && listing.Unit.AreaM2 != nil && *listing.Unit.AreaM2 > 0 {
		perM2 := roundCurrency(*current / *listing.Unit.AreaM2)
		out.CurrentMonthlyChargesPerM2 = &perM2
	}
	if current != nil {
		stress := roundCurrency(*current * 1.25)
		out.StressMonthlyCharges = &stress
		out.StressAssumption = "Current known monthly charges stressed by 25% because housing-company finances are not loaded yet."
		if listing.Unit.AreaM2 != nil && *listing.Unit.AreaM2 > 0 {
			perM2 := roundCurrency(stress / *listing.Unit.AreaM2)
			out.StressMonthlyChargesPerM2 = &perM2
		}
	}
	return out
}

func apartmentOfferReasons(listing SaleListing, valuation *ApartmentValuation, assessment ApartmentOfferAssessment) []ApartmentOfferReason {
	var reasons []ApartmentOfferReason
	if valuation.Price.TransactionPriceDeltaPct != nil {
		severity := "low"
		direction := "neutral"
		if *valuation.Price.TransactionPriceDeltaPct > 8 {
			severity = "high"
			direction = "negative"
		} else if *valuation.Price.TransactionPriceDeltaPct > 3 {
			severity = "medium"
			direction = "negative"
		} else if *valuation.Price.TransactionPriceDeltaPct < -5 {
			severity = "medium"
			direction = "positive"
		}
		reasons = append(reasons, ApartmentOfferReason{Key: "matched_transaction_delta", Direction: direction, Severity: severity, Explanation: fmt.Sprintf("Asking price is %.1f%% from the matched transaction anchor.", *valuation.Price.TransactionPriceDeltaPct)})
	}
	if assessment.RenovationRiskReserve.High != nil && *assessment.RenovationRiskReserve.High > 0 {
		severity := "medium"
		if *assessment.RenovationRiskReserve.High > 60000 {
			severity = "high"
		}
		reasons = append(reasons, ApartmentOfferReason{Key: "renovation_risk_reserve", Direction: "negative", Severity: severity, Explanation: fmt.Sprintf("Known and forecast renovations imply a reserve of roughly %s-%s before housing-company financials are available.", formatEURPtr(assessment.RenovationRiskReserve.Low), formatEURPtr(assessment.RenovationRiskReserve.High))})
	}
	if assessment.EstimatedOwnershipCost.CurrentMonthlyChargesPerM2 != nil && *assessment.EstimatedOwnershipCost.CurrentMonthlyChargesPerM2 > 6 {
		reasons = append(reasons, ApartmentOfferReason{Key: "high_charges", Direction: "negative", Severity: "medium", Explanation: fmt.Sprintf("Current monthly charges are %.2f EUR/m2/month, which is already elevated before future financing is modeled.", *assessment.EstimatedOwnershipCost.CurrentMonthlyChargesPerM2)})
	}
	if strings.EqualFold(listing.Site.PlotOwnershipType, "rented") {
		reasons = append(reasons, ApartmentOfferReason{Key: "rented_plot", Direction: "negative", Severity: "medium", Explanation: "Rented plot increases renewal and rent-increase uncertainty."})
	}
	if listing.Commercial.DebtShareAmount != nil && *listing.Commercial.DebtShareAmount > 0 {
		reasons = append(reasons, ApartmentOfferReason{Key: "existing_debt_share", Direction: "negative", Severity: "medium", Explanation: fmt.Sprintf("Listing already includes %s of debt share before future renovation financing is modeled.", formatEUR(*listing.Commercial.DebtShareAmount))})
	}
	for _, insight := range listing.Insights.Items {
		if insight.Direction == "" || insight.Direction == "neutral" {
			continue
		}
		reasons = append(reasons, ApartmentOfferReason{Key: "description_" + insight.Key, Direction: insight.Direction, Severity: firstNonEmpty(insight.Severity, "low"), Explanation: fmt.Sprintf("%s: %s", insight.Value, firstNonEmpty(insight.Explanation, "extracted from listing description"))})
	}
	return reasons
}

func apartmentOfferAssessmentMissing(listing SaleListing) []string {
	missing := []string{"manager_certificate", "housing_company_financials"}
	if listing.Commercial.MatchedTransaction == nil {
		missing = append(missing, "matched_transaction")
	}
	if listing.Commercial.Charges.MaintenanceMonthly == nil && listing.Commercial.Charges.TotalMonthly == nil {
		missing = append(missing, "monthly_charges")
	}
	if listing.Unit.AreaM2 == nil {
		missing = append(missing, "area_m2")
	}
	return missing
}

func apartmentOfferAssessmentConfidence(listing SaleListing, assessment ApartmentOfferAssessment) string {
	score := 0
	if listing.Commercial.MatchedTransaction != nil {
		score += 3
	}
	if assessment.MarketValueRange.Low != nil && assessment.MarketValueRange.High != nil {
		score += 2
	}
	if assessment.RenovationRiskReserve.High != nil {
		score += 1
	}
	if listing.Commercial.Charges.MaintenanceMonthly != nil || listing.Commercial.Charges.TotalMonthly != nil {
		score += 1
	}
	if score >= 6 {
		return "medium"
	}
	return "low"
}

func apartmentOfferVerdict(listing SaleListing, assessment ApartmentOfferAssessment) string {
	price := listing.Commercial.DebtFreePrice
	if price == nil {
		price = listing.Commercial.AskingPrice
	}
	if price == nil || assessment.RiskAdjustedValueRange.Low == nil || assessment.RiskAdjustedValueRange.High == nil {
		return "insufficient_data"
	}
	if listing.Commercial.MatchedTransaction == nil {
		if assessment.RenovationRiskReserve.High != nil && *assessment.RenovationRiskReserve.High > 40000 {
			return "high_risk_needs_financials"
		}
		return "insufficient_market_data"
	}
	if *price <= *assessment.RecommendedOfferRange.High && *price >= *assessment.RecommendedOfferRange.Low {
		if assessment.RenovationRiskReserve.High != nil && *assessment.RenovationRiskReserve.High > 40000 {
			return "fair_price_high_risk"
		}
		return "fair_price"
	}
	if *price < *assessment.RecommendedOfferRange.Low {
		if assessment.RenovationRiskReserve.High != nil && *assessment.RenovationRiskReserve.High > 60000 {
			return "cheap_but_high_risk"
		}
		return "good_offer"
	}
	if *price <= *assessment.RiskAdjustedValueRange.High {
		return "slightly_high"
	}
	return "overpriced"
}

func apartmentOfferExplanation(assessment ApartmentOfferAssessment) string {
	switch assessment.Verdict {
	case "good_offer":
		return "Asking price is below the current risk-adjusted offer range using available listing, transaction, charge, and renovation data."
	case "cheap_but_high_risk":
		return "Asking price screens cheap, but renovation risk is large enough that housing-company documents should drive the final offer."
	case "fair_price_high_risk":
		return "Price is near the suggested range, but future renovation financing could materially change total ownership cost."
	case "fair_price":
		return "Price is close to the suggested range using current data."
	case "high_risk_needs_financials":
		return "Renovation risk is material, but there is no matched transaction anchor, so housing-company financials and comparables are needed before judging the offer."
	case "insufficient_market_data":
		return "There is not enough market evidence yet to judge whether the asking price is good."
	case "slightly_high":
		return "Price is above the suggested range but still within the broader risk-adjusted value range."
	case "overpriced":
		return "Price is above the current risk-adjusted value range based on available data."
	default:
		return "There is not enough data for a reliable offer verdict."
	}
}

func apartmentRecommendedOfferRange(value ApartmentValueRange) ApartmentValueRange {
	if value.Low == nil || value.High == nil {
		return ApartmentValueRange{}
	}
	return apartmentIntRange(roundInt64(float64(*value.Low)*0.97), roundInt64(float64(*value.High)*0.99))
}

func apartmentRangeSubtract(value ApartmentValueRange, reserve ApartmentValueRange) ApartmentValueRange {
	if value.Low == nil || value.High == nil || reserve.Low == nil || reserve.High == nil {
		return value
	}
	return apartmentIntRange(*value.Low-*reserve.High, *value.High-*reserve.Low)
}

func apartmentRangeMultiply(value ApartmentValueRange, multiplier *float64) ApartmentValueRange {
	if value.Low == nil || value.High == nil || multiplier == nil {
		return ApartmentValueRange{}
	}
	return apartmentIntRange(roundInt64(float64(*value.Low)**multiplier), roundInt64(float64(*value.High)**multiplier))
}

func apartmentIntRange(low int64, high int64) ApartmentValueRange {
	if low < 0 {
		low = 0
	}
	if high < low {
		high = low
	}
	return ApartmentValueRange{Low: &low, High: &high}
}

func roundInt64(value float64) int64 {
	return int64(math.Round(value/100) * 100)
}

func roundCurrency(value float64) float64 {
	return math.Round(value*100) / 100
}

func formatEUR(value int64) string {
	return fmt.Sprintf("%d EUR", value)
}

func formatEURPtr(value *int64) string {
	if value == nil {
		return "unknown"
	}
	return formatEUR(*value)
}

func apartmentValuationSignals(listing SaleListing, valuation *ApartmentValuation) []ApartmentValuationSignal {
	var signals []ApartmentValuationSignal
	if valuation.Price.TransactionPriceDeltaPct != nil {
		direction := "neutral"
		severity := "low"
		label := "Asking price close to matched transaction"
		explanation := fmt.Sprintf("Asking price is %.1f%% from the matched transaction price.", *valuation.Price.TransactionPriceDeltaPct)
		if *valuation.Price.TransactionPriceDeltaPct > 8 {
			direction = "reduces"
			severity = "high"
			label = "Asking price above matched transaction"
		} else if *valuation.Price.TransactionPriceDeltaPct > 3 {
			direction = "reduces"
			severity = "medium"
			label = "Asking price somewhat above matched transaction"
		} else if *valuation.Price.TransactionPriceDeltaPct < -5 {
			direction = "supports"
			severity = "medium"
			label = "Asking price below matched transaction"
		}
		signals = append(signals, ApartmentValuationSignal{Key: "transaction_price_delta", Label: label, Direction: direction, Severity: severity, Explanation: explanation, Source: "matched_transaction"})
	}
	for _, renovation := range valuation.Renovations.Upcoming {
		severity := renovationSeverity(renovation.Category)
		signals = append(signals, ApartmentValuationSignal{Key: "upcoming_renovation." + renovation.Category, Label: "Upcoming " + renovation.Category + " renovation", Direction: "risk", Severity: severity, Explanation: renovation.Explanation, Source: renovation.Source})
	}
	if listing.Commercial.Charges.MaintenanceMonthly != nil && listing.Unit.AreaM2 != nil && *listing.Unit.AreaM2 > 0 {
		chargePerM2 := *listing.Commercial.Charges.MaintenanceMonthly / *listing.Unit.AreaM2
		if chargePerM2 > 6 {
			signals = append(signals, ApartmentValuationSignal{Key: "maintenance_charge", Label: "High maintenance charge", Direction: "reduces", Severity: "medium", Explanation: fmt.Sprintf("Maintenance charge is %.2f EUR/m2/month, which can reduce buyer affordability.", chargePerM2), Source: "listing_charges"})
		}
	}
	if strings.EqualFold(listing.Site.PlotOwnershipType, "rented") {
		signals = append(signals, ApartmentValuationSignal{Key: "rented_plot", Label: "Rented plot", Direction: "risk", Severity: "medium", Explanation: "A rented plot can add renewal and rent-increase risk compared with owned plot listings.", Source: "listing_site"})
	}
	if listing.Building.Elevator != nil && !*listing.Building.Elevator && listing.Unit.FloorLevel != nil && *listing.Unit.FloorLevel >= 3 {
		signals = append(signals, ApartmentValuationSignal{Key: "no_elevator", Label: "No elevator on higher floor", Direction: "reduces", Severity: "medium", Explanation: "A higher-floor apartment without an elevator narrows the buyer pool.", Source: "listing_building"})
	}
	return signals
}

func apartmentValuationMissingInputs(listing SaleListing) []string {
	var missing []string
	if listing.Commercial.MatchedTransaction == nil {
		missing = append(missing, "matched_transaction")
	}
	if listing.Texts.RenovationsDone == "" && listing.Texts.RenovationsPlanned == "" && len(listing.Building.Renovations) == 0 {
		missing = append(missing, "renovation_history")
	}
	if listing.Commercial.AskingPrice == nil {
		missing = append(missing, "asking_price")
	}
	if listing.Unit.AreaM2 == nil {
		missing = append(missing, "area_m2")
	}
	return missing
}

func apartmentValuationConfidence(listing SaleListing, valuation *ApartmentValuation) string {
	score := 0
	if listing.Commercial.MatchedTransaction != nil {
		score += 3
	}
	if valuation.Price.TransactionPriceDeltaPct != nil {
		score += 2
	}
	if len(valuation.Renovations.Completed)+len(valuation.Renovations.Upcoming) > 0 || valuation.Renovations.RawCompleted != "" || valuation.Renovations.RawPlanned != "" {
		score += 2
	}
	if listing.Unit.AreaM2 != nil && listing.Commercial.AskingPrice != nil {
		score++
	}
	if score >= 6 {
		return "high"
	}
	if score >= 3 {
		return "medium"
	}
	return "low"
}

func apartmentValuationExplanation(listing SaleListing, valuation *ApartmentValuation) string {
	parts := []string{"Valuation is based on the listing price, matched transaction data, housing-company facts, and renovation text already attached to the offering."}
	if valuation.Price.TransactionPriceDeltaPct != nil {
		parts = append(parts, fmt.Sprintf("The asking price is %.1f%% from the matched transaction, so the transaction is the strongest current anchor.", *valuation.Price.TransactionPriceDeltaPct))
	}
	if len(valuation.Renovations.Upcoming) > 0 {
		parts = append(parts, fmt.Sprintf("%d upcoming or planned renovation item(s) require price risk review because future housing-company work can affect debt share, monthly charges, and buyer demand.", len(valuation.Renovations.Upcoming)))
	}
	if len(valuation.Renovations.Completed) > 0 {
		parts = append(parts, fmt.Sprintf("%d completed renovation item(s) reduce uncertainty around already-finished building work.", len(valuation.Renovations.Completed)))
	}
	if len(valuation.Missing) > 0 {
		parts = append(parts, "Missing inputs lower confidence: "+strings.Join(valuation.Missing, ", ")+".")
	}
	return strings.Join(parts, " ")
}

func renovationPriceEffect(category string, status string) string {
	if status == "done" {
		return "supports"
	}
	switch renovationSeverity(category) {
	case "high":
		return "major_risk"
	case "medium":
		return "risk"
	default:
		return "minor_risk"
	}
}

func renovationSeverity(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "pipe", "sewer", "facade", "roof", "electricity", "elevator", "water_supply":
		return "high"
	case "window", "balcony", "heating", "yard", "ventilation", "drainage":
		return "medium"
	default:
		return "low"
	}
}

func renovationExplanation(category string, status string, year *int32) string {
	when := ""
	if year != nil {
		when = fmt.Sprintf(" around %d", *year)
	}
	if status == "done" {
		return fmt.Sprintf("%s renovation is marked completed%s, which usually reduces near-term uncertainty.", category, when)
	}
	if status == "planned" {
		return fmt.Sprintf("%s renovation is planned%s and can affect future debt share, monthly charges, and buyer demand.", category, when)
	}
	return fmt.Sprintf("%s renovation has unclear timing or status and needs structured review before pricing.", category)
}

func roundPercent(value float64) float64 {
	return math.Round(value*10) / 10
}
