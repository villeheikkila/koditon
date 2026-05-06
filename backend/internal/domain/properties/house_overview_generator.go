package properties

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"charm.land/fantasy"
	fantasyobject "charm.land/fantasy/object"
)

type houseOverviewObject struct {
	Headline            string   `json:"headline" description:"Short, buyer-facing headline for the building and apartment situation."`
	Summary             string   `json:"summary" description:"Concise overview of the house/building and why it matters for the listing value."`
	RenovationReadiness string   `json:"renovation_readiness" enum:"strong,acceptable,unclear,risky" description:"How complete the key building renovations look from the available structured facts."`
	ExpensiveWindow     string   `json:"expensive_window" description:"Plain-language timing for when ownership is likely to become expensive, based only on supplied facts and forecast."`
	KeyStrengths        []string `json:"key_strengths" description:"Most important positives supported by the supplied facts."`
	KeyRisks            []string `json:"key_risks" description:"Most important risks supported by the supplied facts."`
	EvidenceGaps        []string `json:"evidence_gaps" description:"Important missing evidence that limits confidence."`
	Confidence          string   `json:"confidence" enum:"low,medium,high" description:"Confidence in the overview."`
}

func (s *Service) GenerateSaleListingHouseOverview(ctx context.Context, input string, modelName string) (HouseOverviewGenerationResult, error) {
	offeringID, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return HouseOverviewGenerationResult{}, ErrNotFound
	}
	listing, err := s.SaleListingByID(ctx, offeringID.String(), "", "")
	if err != nil {
		return HouseOverviewGenerationResult{}, err
	}
	if strings.TrimSpace(s.renovationExtractorAPIKey) == "" {
		return HouseOverviewGenerationResult{}, ErrRenovationExtractorNotConfigured
	}
	modelName = firstNonEmpty(modelName, s.renovationExtractorModelName, "~google/gemini-flash-latest")
	model, err := s.renovationExtractionModel(ctx, modelName)
	if err != nil {
		return HouseOverviewGenerationResult{}, err
	}
	objectResult, err := fantasyobject.Generate[houseOverviewObject](ctx, model, fantasy.ObjectCall{Prompt: fantasy.Prompt{fantasy.NewUserMessage(houseOverviewPrompt(listing))}, SchemaName: "generate_house_overview", SchemaDescription: "Generate a concise housing company overview from preprocessed structured facts", Temperature: ptrFloat64(0), MaxOutputTokens: ptrInt64(3500)})
	if err != nil {
		return HouseOverviewGenerationResult{}, fmt.Errorf("generate house overview with fantasy: %w", err)
	}
	overview := normalizeHouseOverview(objectResult.Object, modelName)
	return HouseOverviewGenerationResult{SaleListingID: listing.ID, Model: modelName, Overview: overview}, nil
}

func normalizeHouseOverview(in houseOverviewObject, modelName string) HouseOverview {
	return HouseOverview{
		Headline:            cleanDisplayString(in.Headline),
		Summary:             cleanDisplayString(in.Summary),
		RenovationReadiness: normalizeOverviewReadiness(in.RenovationReadiness),
		ExpensiveWindow:     cleanDisplayString(in.ExpensiveWindow),
		KeyStrengths:        cleanStringList(in.KeyStrengths, 5),
		KeyRisks:            cleanStringList(in.KeyRisks, 5),
		EvidenceGaps:        cleanStringList(in.EvidenceGaps, 5),
		Confidence:          normalizeOverviewConfidence(in.Confidence),
		GeneratedAt:         time.Now().Format(time.RFC3339),
		Model:               modelName,
	}
}

func houseOverviewPrompt(listing SaleListing) string {
	return fmt.Sprintf(`Generate a concise Finnish housing-company overview from preprocessed facts.

Rules:
- Use only the supplied facts. Do not invent missing renovations or financial data.
- Treat listing/ad facts as weaker evidence than manager certificate or financial statements. In this input we currently only have provider fields, LLM-extracted listing facts, transactions, and forecasts.
- Explain whether key building renovations look done, planned, expected, or missing.
- State why this affects value and when ownership may become expensive.
- Mention important unknowns as evidence gaps.
- Keep the summary practical for deciding whether the listed apartment is attractive.

listing:
%s

building:
%s

apartment_profile:
%s

renovation_facts:
%s

forecast_next_40_years:
%s

valuation_brief:
%s`, houseOverviewListingFacts(listing), houseOverviewBuildingFacts(listing), houseOverviewApartmentProfileFacts(listing.ApartmentProfile), houseOverviewRenovationFacts(listing.Building.Renovations), houseOverviewForecastFacts(listing), houseOverviewBriefFacts(listing))
}

func houseOverviewListingFacts(listing SaleListing) string {
	values := []string{
		"headline=" + listing.Headline,
		"address=" + listing.Unit.Location.StreetAddress,
		"city=" + listing.Unit.Location.City,
		"postal=" + listing.Unit.Location.Postal,
	}
	if listing.Commercial.AskingPrice != nil {
		values = append(values, fmt.Sprintf("asking_price_eur=%d", *listing.Commercial.AskingPrice))
	}
	if listing.Commercial.DebtFreePrice != nil {
		values = append(values, fmt.Sprintf("debt_free_price_eur=%d", *listing.Commercial.DebtFreePrice))
	}
	if listing.Commercial.PricePerSquareMeter != nil {
		values = append(values, fmt.Sprintf("price_per_m2=%.0f", *listing.Commercial.PricePerSquareMeter))
	}
	if listing.Commercial.MatchedTransaction != nil && listing.Commercial.MatchedTransaction.PricePerSquareMeter != nil {
		values = append(values, fmt.Sprintf("matched_transaction_price_per_m2=%d", *listing.Commercial.MatchedTransaction.PricePerSquareMeter))
	}
	return compactFactLines(values)
}

func houseOverviewBuildingFacts(listing SaleListing) string {
	building := listing.Building
	values := []string{
		"housing_company=" + building.HousingCompany,
		"business_id=" + building.BusinessID,
		"building_type=" + building.BuildingType,
		"building_subtype=" + building.BuildingSubtype,
		"energy_class=" + building.EnergyClass,
		"heating=" + building.Heating,
		"heating_fuel=" + building.HeatingFuel,
		"building_material=" + building.BuildingMaterial,
		"roof_type=" + building.RoofType,
		"roof_material=" + building.RoofMaterial,
	}
	if building.BuildYear != nil {
		values = append(values, fmt.Sprintf("build_year=%d", *building.BuildYear))
	}
	if building.FloorCount != nil {
		values = append(values, fmt.Sprintf("floors=%d", *building.FloorCount))
	}
	if building.ApartmentCount != nil {
		values = append(values, fmt.Sprintf("apartments=%d", *building.ApartmentCount))
	}
	if building.Elevator != nil {
		values = append(values, fmt.Sprintf("elevator=%t", *building.Elevator))
	}
	if listing.Site.PlotOwnershipType != "" || listing.Site.PlotType != "" {
		values = append(values, "plot="+plotSummaryText(listing.Site))
	}
	return compactFactLines(values)
}

func houseOverviewApartmentProfileFacts(profile ApartmentProfile) string {
	values := []string{
		"room_layout=" + profile.RoomLayout,
		"kitchen_type=" + profile.KitchenType,
		"layout_quality=" + profile.LayoutQuality,
		"condition=" + profile.Condition,
		"kitchen_condition=" + profile.KitchenCondition,
		"bathroom_condition=" + profile.BathroomCondition,
		"parking_type=" + profile.ParkingType,
		"storage_quality=" + profile.StorageQuality,
		"view_quality=" + profile.ViewQuality,
		"accessibility=" + profile.Accessibility,
		"confidence=" + profile.Confidence,
	}
	if profile.AreaM2 != nil {
		values = append(values, fmt.Sprintf("area_m2=%.1f", *profile.AreaM2))
	}
	if profile.FloorLevel != nil {
		values = append(values, fmt.Sprintf("floor_level=%d", *profile.FloorLevel))
	}
	if profile.TotalFloors != nil {
		values = append(values, fmt.Sprintf("total_floors=%d", *profile.TotalFloors))
	}
	values = appendBoolFact(values, "surface_renovation_need", profile.SurfaceRenovationNeed)
	values = appendBoolFact(values, "modernization_need", profile.ModernizationNeed)
	values = appendBoolFact(values, "sauna", profile.Sauna)
	values = appendBoolFact(values, "balcony", profile.Balcony)
	values = appendBoolFact(values, "balcony_glazing", profile.BalconyGlazing)
	values = appendBoolFact(values, "noise_risk", profile.NoiseRisk)
	return compactFactLines(values)
}

func houseOverviewRenovationFacts(renovations []BuildingRenovation) string {
	lines := make([]string, 0, len(renovations))
	for _, renovation := range renovations {
		status := "unknown"
		if renovation.Done != nil && *renovation.Done {
			status = "done"
		}
		if renovation.Done != nil && !*renovation.Done {
			status = "planned"
		}
		parts := []string{renovation.Kind, "status=" + status}
		if renovation.Year != nil {
			parts = append(parts, fmt.Sprintf("year=%d", *renovation.Year))
		}
		parts = append(parts, "scope="+renovation.Scope, "stage="+renovation.Stage, "responsibility="+renovation.Responsibility, "source="+renovation.Source)
		if renovation.Text != "" {
			parts = append(parts, "evidence="+renovation.Text)
		}
		lines = append(lines, compactFactLine(parts))
	}
	return strings.Join(lines, "\n")
}

func houseOverviewForecastFacts(listing SaleListing) string {
	if listing.Valuation == nil {
		return ""
	}
	lines := make([]string, 0, len(listing.Valuation.Renovations.Next40Years))
	for _, item := range listing.Valuation.Renovations.Next40Years {
		parts := []string{item.Category, "status=" + item.Status, "severity=" + item.Severity, "confidence=" + item.Confidence, "price_effect=" + item.PriceEffect}
		if item.Year != nil {
			parts = append(parts, fmt.Sprintf("year=%d", *item.Year))
		}
		if item.YearRange != "" {
			parts = append(parts, "year_range="+item.YearRange)
		}
		if item.Explanation != "" {
			parts = append(parts, "explanation="+item.Explanation)
		}
		lines = append(lines, compactFactLine(parts))
	}
	return strings.Join(lines, "\n")
}

func houseOverviewBriefFacts(listing SaleListing) string {
	if listing.Valuation == nil {
		return ""
	}
	brief := listing.Valuation.Brief
	values := []string{"verdict=" + brief.Verdict, "building_risk=" + brief.BuildingRisk, "confidence=" + brief.Confidence, "explanation=" + brief.Explanation}
	values = append(values, "offer_verdict="+listing.Valuation.OfferAssessment.Verdict, "offer_explanation="+listing.Valuation.OfferAssessment.Explanation)
	return compactFactLines(values)
}

func appendBoolFact(values []string, key string, value *bool) []string {
	if value == nil {
		return values
	}
	return append(values, fmt.Sprintf("%s=%t", key, *value))
}

func compactFactLines(values []string) string {
	return compactFactLine(values)
}

func compactFactLine(values []string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = cleanDisplayString(value)
		if value == "" || strings.HasSuffix(value, "=") {
			continue
		}
		out = append(out, value)
	}
	return strings.Join(out, "; ")
}

func cleanStringList(values []string, limit int) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = cleanDisplayString(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func normalizeOverviewReadiness(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strong", "acceptable", "unclear", "risky":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unclear"
	}
}

func normalizeOverviewConfidence(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "low"
	}
}

func plotSummaryText(site SiteDetails) string {
	return strings.Join([]string{site.PlotOwnershipType, site.PlotType}, " ")
}
