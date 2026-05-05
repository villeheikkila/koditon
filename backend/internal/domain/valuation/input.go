package valuation

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func BuildInput(listing SaleListing) ValuationInputs {
	input := ValuationInputs{Facts: append([]ValuationFact(nil), listing.Inputs.Facts...)}
	input.Unit = UnitInput{AreaM2: listing.Unit.AreaM2, LivingAreaM2: listing.Unit.LivingAreaM2, TotalAreaM2: listing.Unit.TotalAreaM2, OtherAreaM2: listing.Unit.OtherAreaM2, Condition: listing.Unit.Condition, Sauna: listing.Unit.Sauna, Balcony: listing.Unit.Balcony, Parking: listing.Unit.Parking}
	input.Layout = LayoutInput{RoomLayout: listing.Unit.RoomLayout, RoomCount: listing.Unit.RoomsCount, BedroomCount: listing.Unit.BedroomsCount}
	input.Floor = FloorInput{FloorLevel: listing.Unit.FloorLevel, TotalFloors: listing.Building.FloorCount, Elevator: listing.Building.Elevator}
	input.Building = BuildingInput{BuildYear: listing.Building.BuildYear, BuildingType: firstNonEmpty(listing.Building.BuildingType, listing.Building.BuildingSubtype), EnergyClass: listing.Building.EnergyClass, HeatingMethod: firstNonEmpty(listing.Building.Heating, listing.Building.HeatingFuel), BuildingMaterial: listing.Building.BuildingMaterial, RoofType: listing.Building.RoofType, RoofMaterial: listing.Building.RoofMaterial, Elevator: listing.Building.Elevator, ApartmentCount: listing.Building.ApartmentCount}
	input.Site = SiteInput{PlotOwnershipType: listing.Site.PlotOwnershipType, PlotType: listing.Site.PlotType, PlotAreaM2: listing.Site.PlotAreaM2, LotRedemptionInfo: listing.Site.LotRedemptionInfo, LotRentalAgreement: listing.Site.LotRentalAgreement, Services: listing.Site.Services, Transport: listing.Site.Transport}
	input.Charges = ChargesInput{MaintenanceMonthly: listing.Commercial.Charges.MaintenanceMonthly, TotalMonthly: listing.Commercial.Charges.TotalMonthly, Water: listing.Commercial.Charges.Water, Parking: listing.Commercial.Charges.Parking, Sauna: listing.Commercial.Charges.Sauna, Electricity: listing.Commercial.Charges.Electricity, Heating: listing.Commercial.Charges.Heating, Notes: listing.Commercial.Charges.Notes}
	input.Market = MarketInput{AskingPrice: listing.Commercial.AskingPrice, DebtFreePrice: listing.Commercial.DebtFreePrice, DebtShareAmount: listing.Commercial.DebtShareAmount, PricePerSquareMeter: listing.Commercial.PricePerSquareMeter, MatchedTransaction: listing.Commercial.MatchedTransaction}
	for _, renovation := range listing.Building.Renovations {
		if renovation.Done != nil && *renovation.Done {
			input.Renovations.Completed = append(input.Renovations.Completed, renovation)
			continue
		}
		if renovation.Done != nil && !*renovation.Done {
			input.Renovations.Planned = append(input.Renovations.Planned, renovation)
		}
	}
	if input.Floor.FloorLevel != nil {
		input.Floor.GroundFloor = boolPtrFromValue(*input.Floor.FloorLevel <= 1)
		input.Floor.HighFloor = boolPtrFromValue(*input.Floor.FloorLevel >= 4)
	}
	if input.Floor.FloorLevel != nil && input.Floor.TotalFloors != nil {
		input.Floor.TopFloor = boolPtrFromValue(*input.Floor.FloorLevel == *input.Floor.TotalFloors)
	}
	input.Floor.ElevatorRelevance = elevatorRelevance(input.Floor)
	input.Layout.KitchenType = inferKitchenType(listing.Unit.RoomLayout + " " + listing.Unit.KitchenDescription)
	if input.Layout.KitchenType != "" {
		input.Layout.SeparateKitchen = boolPtrFromValue(input.Layout.KitchenType == "separate")
		input.Layout.OpenKitchen = boolPtrFromValue(input.Layout.KitchenType == "open")
	}
	if strings.Contains(strings.ToLower(listing.Unit.RoomLayout), "alk") {
		input.Layout.Alcove = boolPtrFromValue(true)
	}
	input = applyInputFacts(input)
	input.Missing = valuationInputMissing(input)
	return input
}

func applyInputFacts(input ValuationInputs) ValuationInputs {
	for _, fact := range input.Facts {
		applied := true
		if _, ok := inputDimensionFor(fact.Section, fact.Key); !ok {
			input.ExtraFacts = append(input.ExtraFacts, fact)
			continue
		}
		switch fact.Section + "." + fact.Key {
		case "balcony.glazing":
			input.Unit.BalconyGlazing = chooseBool(input.Unit.BalconyGlazing, fact, &input)
		case "unit.balcony", "balcony.has_balcony":
			input.Unit.Balcony = chooseBool(input.Unit.Balcony, fact, &input)
		case "unit.sauna", "sauna.has_sauna", "sauna.private_sauna":
			input.Unit.Sauna = chooseBool(input.Unit.Sauna, fact, &input)
		case "layout.kitchen_type":
			input.Layout.KitchenType = chooseText(input.Layout.KitchenType, fact, &input)
		case "layout.has_separate_kitchen":
			input.Layout.SeparateKitchen = chooseBool(input.Layout.SeparateKitchen, fact, &input)
		case "layout.has_open_kitchen":
			input.Layout.OpenKitchen = chooseBool(input.Layout.OpenKitchen, fact, &input)
		case "layout.has_alcove":
			input.Layout.Alcove = chooseBool(input.Layout.Alcove, fact, &input)
		case "layout.awkward_layout":
			input.Layout.AwkwardLayout = chooseBool(input.Layout.AwkwardLayout, fact, &input)
		case "layout.layout_quality":
			input.Layout.LayoutQuality = chooseText(input.Layout.LayoutQuality, fact, &input)
		case "rooms.separate_wc_count", "layout.separate_wc_count":
			input.Layout.SeparateWCCount = chooseInt32(input.Layout.SeparateWCCount, fact, &input)
		case "storage.storage_quality":
			input.Unit.StorageQuality = chooseText(input.Unit.StorageQuality, fact, &input)
		case "views.view_quality":
			input.Unit.ViewQuality = chooseText(input.Unit.ViewQuality, fact, &input)
		case "views.noise_risk":
			input.Unit.NoiseRisk = chooseBool(input.Unit.NoiseRisk, fact, &input)
		case "unit.accessibility", "building.accessibility":
			input.Unit.Accessibility = chooseText(input.Unit.Accessibility, fact, &input)
		case "condition.surface_renovation_need":
			input.Unit.SurfaceRenovationNeed = chooseBool(input.Unit.SurfaceRenovationNeed, fact, &input)
		case "condition.modernization_need":
			input.Unit.ModernizationNeed = chooseBool(input.Unit.ModernizationNeed, fact, &input)
		case "kitchen.renovated":
			input.Unit.KitchenRenovated = chooseBool(input.Unit.KitchenRenovated, fact, &input)
		case "bathroom.renovated":
			input.Unit.BathroomRenovated = chooseBool(input.Unit.BathroomRenovated, fact, &input)
		case "heating.heating_method":
			input.Building.HeatingMethod = chooseText(input.Building.HeatingMethod, fact, &input)
		case "building.common_area_quality":
			input.Building.CommonAreaQuality = chooseText(input.Building.CommonAreaQuality, fact, &input)
		case "charges.charge_risk":
			input.Charges.ChargeRisk = chooseText(input.Charges.ChargeRisk, fact, &input)
		default:
			applied = false
		}
		if !applied {
			input.ExtraFacts = append(input.ExtraFacts, fact)
		}
	}
	return input
}

func chooseText(current string, fact ValuationFact, input *ValuationInputs) string {
	value := factTextValue(fact)
	if value == "" {
		return current
	}
	if current == "" {
		return value
	}
	if !strings.EqualFold(current, value) {
		input.Conflicts = append(input.Conflicts, conflictFor(fact.Section+"."+fact.Key, providerTextFact(fact, current), fact, "provider value retained"))
	}
	return current
}

func chooseBool(current *bool, fact ValuationFact, input *ValuationInputs) *bool {
	value, ok := factBoolValue(fact)
	if !ok {
		return current
	}
	if current == nil {
		return &value
	}
	if *current != value {
		input.Conflicts = append(input.Conflicts, conflictFor(fact.Section+"."+fact.Key, providerBoolFact(fact, *current), fact, "provider value retained"))
	}
	return current
}

func chooseInt32(current *int32, fact ValuationFact, input *ValuationInputs) *int32 {
	value, ok := factInt32Value(fact)
	if !ok {
		return current
	}
	if current == nil {
		return &value
	}
	if *current != value {
		input.Conflicts = append(input.Conflicts, conflictFor(fact.Section+"."+fact.Key, providerNumberFact(fact, float64(*current)), fact, "provider value retained"))
	}
	return current
}

func factTextValue(fact ValuationFact) string {
	if fact.ValueKind == "text" {
		return strings.TrimSpace(fact.ValueText)
	}
	if fact.ValueBool != nil {
		if *fact.ValueBool {
			return "true"
		}
		return "false"
	}
	if fact.ValueNumber != nil {
		return fmt.Sprint(*fact.ValueNumber)
	}
	return ""
}

func factBoolValue(fact ValuationFact) (bool, bool) {
	if fact.ValueBool != nil {
		return *fact.ValueBool, true
	}
	switch strings.ToLower(strings.TrimSpace(fact.ValueText)) {
	case "true", "yes", "kyllä", "glazed", "has", "present":
		return true, true
	case "false", "no", "ei", "none", "missing":
		return false, true
	default:
		return false, false
	}
}

func factInt32Value(fact ValuationFact) (int32, bool) {
	if fact.ValueNumber == nil {
		return 0, false
	}
	return int32(math.Round(*fact.ValueNumber)), true
}

func conflictFor(path string, chosen ValuationFact, rejected ValuationFact, reason string) ValuationConflict {
	return ValuationConflict{Path: path, Chosen: chosen, Rejected: rejected, Reason: reason}
}

func providerTextFact(fact ValuationFact, value string) ValuationFact {
	return ValuationFact{Section: fact.Section, Key: fact.Key, ValueKind: "text", ValueText: value, Confidence: 1, Source: "provider_structured"}
}

func providerBoolFact(fact ValuationFact, value bool) ValuationFact {
	return ValuationFact{Section: fact.Section, Key: fact.Key, ValueKind: "bool", ValueBool: &value, Confidence: 1, Source: "provider_structured"}
}

func providerNumberFact(fact ValuationFact, value float64) ValuationFact {
	return ValuationFact{Section: fact.Section, Key: fact.Key, ValueKind: "number", ValueNumber: &value, Confidence: 1, Source: "provider_structured"}
}

func valuationInputMissing(input ValuationInputs) []string {
	var missing []string
	if input.Market.MatchedTransaction == nil {
		missing = append(missing, "market.matched_transaction")
	}
	if input.Charges.MaintenanceMonthly == nil && input.Charges.TotalMonthly == nil {
		missing = append(missing, "charges.monthly_charges")
	}
	if input.Unit.AreaM2 == nil {
		missing = append(missing, "unit.area_m2")
	}
	if input.Layout.KitchenType == "" {
		missing = append(missing, "layout.kitchen_type")
	}
	if len(input.Renovations.Completed)+len(input.Renovations.Planned) == 0 {
		missing = append(missing, "renovations.history")
	}
	if !input.Documents.ManagerCertificateLoaded {
		missing = append(missing, "documents.manager_certificate")
	}
	if !input.Documents.FinancialStatementLoaded {
		missing = append(missing, "documents.financial_statement")
	}
	return missing
}

func elevatorRelevance(floor FloorInput) string {
	if floor.FloorLevel == nil {
		return ""
	}
	if floor.Elevator != nil && *floor.Elevator {
		return "elevator_present"
	}
	if *floor.FloorLevel >= 3 {
		return "high_importance_missing"
	}
	return "low_importance"
}

func inferKitchenType(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "avok"), strings.Contains(value, "open kitchen"):
		return "open"
	case strings.Contains(value, "kk"), strings.Contains(value, "kitchenette"):
		return "kitchenette"
	case strings.Contains(value, "+k"), strings.Contains(value, " k"), strings.Contains(value, "keittiö"):
		return "separate"
	default:
		return ""
	}
}

func boolPtrFromValue(value bool) *bool {
	return &value
}

func attachRenovationForecast(input ValuationInputs, forecast []ApartmentRenovationNeed) ValuationInputs {
	input.Renovations.Forecast = append([]ApartmentRenovationNeed(nil), forecast...)
	return input
}

func currentYear() int32 {
	return int32(time.Now().Year())
}
