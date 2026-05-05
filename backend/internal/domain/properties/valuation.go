package properties

import "koditon/internal/domain/valuation"

func valuationSaleListing(listing SaleListing) valuation.SaleListing {
	return valuation.SaleListing{Unit: valuationUnit(listing.Unit), Building: valuationBuilding(listing.Building), Site: valuationSite(listing.Site), Commercial: valuationCommercial(listing.Commercial), Texts: valuation.TextSections{RenovationsDone: listing.Texts.RenovationsDone, RenovationsPlanned: listing.Texts.RenovationsPlanned}, Insights: valuationInsights(listing.Insights), Inputs: listing.ValuationInputs}
}

func valuationUnit(unit UnitDetails) valuation.UnitDetails {
	return valuation.UnitDetails{Location: valuationLocation(unit.Location), PropertyType: unit.PropertyType, PropertySubtype: unit.PropertySubtype, RoomLayout: unit.RoomLayout, RoomsCount: unit.RoomsCount, BedroomsCount: unit.BedroomsCount, AreaM2: unit.AreaM2, LivingAreaM2: unit.LivingAreaM2, TotalAreaM2: unit.TotalAreaM2, OtherAreaM2: unit.OtherAreaM2, FloorLevel: unit.FloorLevel, Condition: unit.Condition, Sauna: unit.Sauna, Balcony: unit.Balcony, Parking: unit.Parking, KitchenDescription: unit.KitchenDescription, BathroomDescription: unit.BathroomDescription, StorageDescription: unit.StorageDescription, FloorMaterialsDescription: unit.FloorMaterialsDescription, WallMaterialsDescription: unit.WallMaterialsDescription, BalconyDescription: unit.BalconyDescription, SaunaDescription: unit.SaunaDescription, ViewsDescription: unit.ViewsDescription, Appliances: unit.Appliances, Features: unit.Features}
}

func valuationBuilding(building BuildingDetails) valuation.BuildingDetails {
	return valuation.BuildingDetails{Location: valuationLocation(building.Location), HousingCompany: building.HousingCompany, BusinessID: building.BusinessID, BuildingType: building.BuildingType, BuildingSubtype: building.BuildingSubtype, BuildYear: building.BuildYear, ConstructionYear: building.ConstructionYear, FloorCount: building.FloorCount, ApartmentCount: building.ApartmentCount, EnergyClass: building.EnergyClass, Heating: building.Heating, HeatingFuel: building.HeatingFuel, BuildingMaterial: building.BuildingMaterial, WallStructure: building.WallStructure, RoofType: building.RoofType, RoofMaterial: building.RoofMaterial, CommonAreas: building.CommonAreas, CarStorage: building.CarStorage, OtherInfo: building.OtherInfo, Elevator: building.Elevator, Sauna: building.Sauna, Renovations: valuationRenovations(building.Renovations)}
}

func valuationLocation(location Location) valuation.Location {
	return valuation.Location{StreetAddress: location.StreetAddress, City: location.City, Postal: location.Postal}
}

func valuationSite(site SiteDetails) valuation.SiteDetails {
	return valuation.SiteDetails{PlotOwnershipType: site.PlotOwnershipType, PlotType: site.PlotType, PlotAreaM2: site.PlotAreaM2, LotRedemptionInfo: site.LotRedemptionInfo, LotRentalAgreement: site.LotRentalAgreement, Services: site.Services, Transport: site.Transport}
}

func valuationCommercial(commercial CommercialDetails) valuation.CommercialDetails {
	return valuation.CommercialDetails{AskingPrice: commercial.AskingPrice, DebtFreePrice: commercial.DebtFreePrice, DebtShareAmount: commercial.DebtShareAmount, PricePerSquareMeter: commercial.PricePerSquareMeter, Charges: valuation.Charges{MaintenanceMonthly: commercial.Charges.MaintenanceMonthly, TotalMonthly: commercial.Charges.TotalMonthly, Water: commercial.Charges.Water, Parking: commercial.Charges.Parking, Sauna: commercial.Charges.Sauna, Electricity: commercial.Charges.Electricity, Heating: commercial.Charges.Heating, Notes: commercial.Charges.Notes}, MatchedTransaction: valuationTransaction(commercial.MatchedTransaction)}
}

func valuationTransaction(transaction *PriceTransactionMatch) *valuation.PriceTransactionMatch {
	if transaction == nil {
		return nil
	}
	return &valuation.PriceTransactionMatch{ID: transaction.ID, FirstSeenAt: transaction.FirstSeenAt, UpdatedAt: transaction.UpdatedAt, Description: transaction.Description, Type: transaction.Type, Category: transaction.Category, AreaM2: transaction.AreaM2, Price: transaction.Price, PricePerSquareMeter: transaction.PricePerSquareMeter, BuildYear: transaction.BuildYear, Floor: transaction.Floor, Elevator: transaction.Elevator, Condition: transaction.Condition, Plot: transaction.Plot, PlotOwned: transaction.PlotOwned, EnergyClass: transaction.EnergyClass, PeriodIdentifier: transaction.PeriodIdentifier, City: transaction.City, Neighborhood: transaction.Neighborhood, PostalCode: transaction.PostalCode, MatchStatus: transaction.MatchStatus, MatchScore: transaction.MatchScore, MatchConfidence: transaction.MatchConfidence}
}

func valuationRenovations(renovations []BuildingRenovation) []valuation.BuildingRenovation {
	out := make([]valuation.BuildingRenovation, 0, len(renovations))
	for _, renovation := range renovations {
		out = append(out, valuation.BuildingRenovation{Kind: renovation.Kind, Component: renovation.Component, Done: renovation.Done, Year: renovation.Year, Scope: renovation.Scope, Stage: renovation.Stage, Responsibility: renovation.Responsibility, CostEstimateEUR: renovation.CostEstimateEUR, Text: renovation.Text, Confidence: renovation.Confidence, Source: renovation.Source})
	}
	return out
}

func valuationInsights(insights ListingInsights) valuation.ListingInsights {
	out := valuation.ListingInsights{Items: make([]valuation.Insight, 0, len(insights.Items))}
	for _, insight := range insights.Items {
		out.Items = append(out.Items, valuation.Insight{Key: insight.Key, Value: insight.Value, Direction: insight.Direction, Severity: insight.Severity, Confidence: insight.Confidence, Source: insight.Source, Explanation: insight.Explanation})
	}
	return out
}
