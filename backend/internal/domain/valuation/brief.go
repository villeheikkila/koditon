package valuation

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

var keyRenovationCategories = []string{"pipe", "roof", "facade", "window", "electricity", "elevator", "heating", "ventilation", "drainage"}

func BuildBrief(valuation *ApartmentValuation) ValuationBrief {
	if valuation == nil {
		return ValuationBrief{Verdict: "insufficient_data", Label: "Insufficient data", Confidence: "low"}
	}
	brief := ValuationBrief{Verdict: briefVerdict(valuation.OfferAssessment.Verdict), Confidence: briefConfidence(valuation), BuildingRisk: briefBuildingRisk(valuation), MissingEvidence: briefMissingEvidence(valuation)}
	brief.Label = briefVerdictLabel(brief.Verdict)
	brief.ExpensiveWindows = briefExpensiveWindows(valuation)
	brief.KeyRenovations = briefKeyRenovations(valuation)
	brief.TopRisks = briefTopRisks(valuation)
	brief.TopPositives = briefTopPositives(valuation)
	brief.Explanation = briefExplanation(brief)
	return brief
}

func briefVerdict(verdict string) string {
	switch verdict {
	case "good_offer":
		return "good"
	case "fair_price", "fair_price_high_risk":
		return "fair"
	case "cheap_but_high_risk", "high_risk_needs_financials", "slightly_high", "insufficient_market_data":
		return "risky"
	case "overpriced":
		return "avoid"
	default:
		return "insufficient_data"
	}
}

func briefVerdictLabel(verdict string) string {
	switch verdict {
	case "good":
		return "Looks good"
	case "fair":
		return "Fair with current evidence"
	case "risky":
		return "Needs caution"
	case "avoid":
		return "Avoid at current price"
	default:
		return "Not enough evidence"
	}
}

func briefBuildingRisk(valuation *ApartmentValuation) string {
	startYear := int32(time.Now().Year())
	for _, need := range valuation.Renovations.Next40Years {
		year := renovationNeedBriefYear(need, startYear)
		if need.Severity == "high" && year <= startYear+12 {
			return "high"
		}
		if (need.Status == "planned" || need.Status == "follow_up") && year <= startYear+12 && need.Severity != "low" {
			return "high"
		}
	}
	if valuation.OfferAssessment.RenovationRiskReserve.High != nil && *valuation.OfferAssessment.RenovationRiskReserve.High > 60000 {
		return "high"
	}
	for _, need := range valuation.Renovations.Next40Years {
		year := renovationNeedBriefYear(need, startYear)
		if (need.Severity == "high" || need.Severity == "medium") && year <= startYear+20 {
			return "medium"
		}
	}
	if valuation.OfferAssessment.RenovationRiskReserve.High != nil && *valuation.OfferAssessment.RenovationRiskReserve.High > 25000 {
		return "medium"
	}
	return "low"
}

func briefExpensiveWindows(valuation *ApartmentValuation) []OwnershipCostWindow {
	startYear := int32(time.Now().Year())
	type bucket struct {
		start    int32
		end      int32
		severity string
		reasons  []string
	}
	buckets := map[int32]*bucket{}
	for _, need := range valuation.Renovations.Next40Years {
		if need.Severity == "low" {
			continue
		}
		year := renovationNeedBriefYear(need, startYear)
		if year > startYear+20 {
			continue
		}
		bucketStart := startYear + ((year - startYear) / 5 * 5)
		bucketEnd := bucketStart + 4
		current := buckets[bucketStart]
		if current == nil {
			current = &bucket{start: bucketStart, end: bucketEnd, severity: need.Severity}
			buckets[bucketStart] = current
		}
		if severityRank(need.Severity) > severityRank(current.severity) {
			current.severity = need.Severity
		}
		current.reasons = appendUnique(current.reasons, renovationStatusLabel(need.Category))
	}
	keys := make([]int32, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	windows := make([]OwnershipCostWindow, 0, len(keys))
	for _, key := range keys {
		current := buckets[key]
		start := current.start
		end := current.end
		windows = append(windows, OwnershipCostWindow{StartYear: &start, EndYear: &end, Severity: current.severity, Label: fmt.Sprintf("%d-%d", start, end), Reasons: current.reasons})
	}
	return windows
}

func briefKeyRenovations(valuation *ApartmentValuation) []KeyRenovationStatus {
	out := make([]KeyRenovationStatus, 0, len(keyRenovationCategories))
	for _, category := range keyRenovationCategories {
		if need, ok := briefBestNeed(category, valuation.Renovations.Next40Years); ok {
			out = append(out, KeyRenovationStatus{Category: category, Status: need.Status, Year: need.Year, WindowStartYear: need.WindowStartYear, WindowEndYear: need.WindowEndYear, Severity: need.Severity, Confidence: need.Confidence, Explanation: need.Explanation})
			continue
		}
		if item, ok := briefCompletedRenovation(category, valuation.Renovations.Completed); ok {
			out = append(out, KeyRenovationStatus{Category: category, Status: "done", Year: item.Year, Severity: renovationSeverity(category), Confidence: "medium", Explanation: item.Explanation})
			continue
		}
		out = append(out, KeyRenovationStatus{Category: category, Status: "unknown", Severity: renovationSeverity(category), Confidence: "low", Explanation: "No structured evidence for this renovation category yet."})
	}
	return out
}

func briefBestNeed(category string, needs []ApartmentRenovationNeed) (ApartmentRenovationNeed, bool) {
	startYear := int32(time.Now().Year())
	var best ApartmentRenovationNeed
	found := false
	for _, need := range needs {
		if need.Category != category {
			continue
		}
		if !found || renovationNeedBriefYear(need, startYear) < renovationNeedBriefYear(best, startYear) || severityRank(need.Severity) > severityRank(best.Severity) {
			best = need
			found = true
		}
	}
	return best, found
}

func briefCompletedRenovation(category string, items []ApartmentRenovationItem) (ApartmentRenovationItem, bool) {
	var best ApartmentRenovationItem
	found := false
	for _, item := range items {
		if normalizeRenovationCategory(item.Category) != category {
			continue
		}
		if !found || item.Year != nil && (best.Year == nil || *item.Year > *best.Year) {
			best = item
			found = true
		}
	}
	return best, found
}

func briefTopRisks(valuation *ApartmentValuation) []BriefSignal {
	var risks []BriefSignal
	for _, reason := range valuation.OfferAssessment.MainReasons {
		if reason.Direction != "negative" && reason.Direction != "risk" {
			continue
		}
		risks = append(risks, BriefSignal{Key: reason.Key, Label: renovationStatusLabel(reason.Key), Severity: reason.Severity, Direction: "negative", Explanation: reason.Explanation})
	}
	for _, need := range valuation.Renovations.Next40Years {
		if need.Severity == "low" {
			continue
		}
		risks = append(risks, BriefSignal{Key: "renovation_" + need.Category, Label: "Renovation: " + renovationStatusLabel(need.Category), Severity: need.Severity, Direction: "negative", Explanation: need.Explanation})
	}
	if len(valuation.Input.Missing) > 0 {
		risks = append(risks, BriefSignal{Key: "missing_evidence", Label: "Missing evidence", Severity: "medium", Direction: "negative", Explanation: "Some valuation dimensions still depend on assumptions: " + strings.Join(briefHumanMissing(valuation.Input.Missing), ", ") + "."})
	}
	sortBriefSignals(risks)
	return limitBriefSignals(dedupeBriefSignals(risks), 5)
}

func briefTopPositives(valuation *ApartmentValuation) []BriefSignal {
	var positives []BriefSignal
	for _, reason := range valuation.OfferAssessment.MainReasons {
		if reason.Direction != "positive" {
			continue
		}
		positives = append(positives, BriefSignal{Key: reason.Key, Label: renovationStatusLabel(reason.Key), Severity: reason.Severity, Direction: "positive", Explanation: reason.Explanation})
	}
	input := valuation.Input
	if input.Unit.BalconyGlazing != nil && *input.Unit.BalconyGlazing {
		positives = append(positives, BriefSignal{Key: "balcony_glazing", Label: "Glazed balcony", Severity: "low", Direction: "positive", Explanation: "The balcony appears glazed, improving usability."})
	}
	if input.Unit.Sauna != nil && *input.Unit.Sauna {
		positives = append(positives, BriefSignal{Key: "private_sauna", Label: "Private sauna", Severity: "low", Direction: "positive", Explanation: "The apartment has a private sauna."})
	}
	if input.Floor.Elevator != nil && *input.Floor.Elevator && input.Floor.HighFloor != nil && *input.Floor.HighFloor {
		positives = append(positives, BriefSignal{Key: "high_floor_elevator", Label: "High floor with elevator", Severity: "low", Direction: "positive", Explanation: "A higher floor is easier to sell when the building has an elevator."})
	}
	if input.Unit.KitchenRenovated != nil && *input.Unit.KitchenRenovated {
		positives = append(positives, BriefSignal{Key: "kitchen_renovated", Label: "Renovated kitchen", Severity: "low", Direction: "positive", Explanation: "Canonical inputs indicate a renovated kitchen."})
	}
	if input.Unit.BathroomRenovated != nil && *input.Unit.BathroomRenovated {
		positives = append(positives, BriefSignal{Key: "bathroom_renovated", Label: "Renovated bathroom", Severity: "low", Direction: "positive", Explanation: "Canonical inputs indicate a renovated bathroom."})
	}
	sortBriefSignals(positives)
	return limitBriefSignals(dedupeBriefSignals(positives), 5)
}

func briefMissingEvidence(valuation *ApartmentValuation) []string {
	var missing []string
	missing = append(missing, valuation.Missing...)
	missing = append(missing, valuation.OfferAssessment.Missing...)
	missing = append(missing, valuation.Input.Missing...)
	return briefHumanMissing(dedupeStrings(missing))
}

func briefHumanMissing(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch item {
		case "documents.manager_certificate", "manager_certificate":
			out = appendUnique(out, "manager certificate")
		case "documents.financial_statement", "housing_company_financials":
			out = appendUnique(out, "housing company financials")
		case "market.matched_transaction", "matched_transaction":
			out = appendUnique(out, "matched transaction")
		case "renovations.history", "renovation_history":
			out = appendUnique(out, "renovation history")
		case "charges.monthly_charges", "monthly_charges":
			out = appendUnique(out, "monthly charges")
		default:
			out = appendUnique(out, renovationStatusLabel(item))
		}
	}
	return out
}

func briefConfidence(valuation *ApartmentValuation) string {
	if valuation.OfferAssessment.Confidence == "low" || len(valuation.Input.Missing) >= 4 {
		return "low"
	}
	if valuation.Confidence == "high" && valuation.OfferAssessment.Confidence == "medium" && len(valuation.Input.Missing) <= 2 {
		return "high"
	}
	return "medium"
}

func briefExplanation(brief ValuationBrief) string {
	parts := []string{brief.Label}
	if brief.BuildingRisk != "" {
		parts = append(parts, "building risk is "+brief.BuildingRisk)
	}
	if len(brief.ExpensiveWindows) > 0 {
		parts = append(parts, "main cost window "+brief.ExpensiveWindows[0].Label)
	}
	if len(brief.TopRisks) > 0 {
		parts = append(parts, "top risk: "+brief.TopRisks[0].Label)
	}
	return strings.Join(parts, "; ") + "."
}

func renovationNeedBriefYear(need ApartmentRenovationNeed, fallback int32) int32 {
	if need.Year != nil {
		return *need.Year
	}
	if need.WindowStartYear != nil {
		return *need.WindowStartYear
	}
	return fallback
}

func severityRank(severity string) int {
	switch severity {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func sortBriefSignals(signals []BriefSignal) {
	sort.SliceStable(signals, func(i int, j int) bool {
		if severityRank(signals[i].Severity) != severityRank(signals[j].Severity) {
			return severityRank(signals[i].Severity) > severityRank(signals[j].Severity)
		}
		return signals[i].Label < signals[j].Label
	})
}

func dedupeBriefSignals(signals []BriefSignal) []BriefSignal {
	seen := map[string]struct{}{}
	out := make([]BriefSignal, 0, len(signals))
	for _, signal := range signals {
		if _, ok := seen[signal.Key]; ok {
			continue
		}
		seen[signal.Key] = struct{}{}
		out = append(out, signal)
	}
	return out
}

func limitBriefSignals(signals []BriefSignal, limit int) []BriefSignal {
	if len(signals) <= limit {
		return signals
	}
	return signals[:limit]
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func appendUnique(items []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return items
	}
	if slices.Contains(items, item) {
		return items
	}
	return append(items, item)
}

func renovationStatusLabel(value string) string {
	value = strings.ReplaceAll(value, ".", " ")
	value = strings.ReplaceAll(value, "_", " ")
	return strings.TrimSpace(value)
}
