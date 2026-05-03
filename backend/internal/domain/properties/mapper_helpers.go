package properties

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
)

func shortcutUnit(payload rawMap, row db.GetShortcutAdUnifiedDetailRow, location Location) UnitDetails {
	return UnitDetails{Location: location, PropertyType: firstNonEmpty(valueAtPath(payload, "adData", "habitationType"), valueAtPath(payload, "adData", "listingTypes")), PropertySubtype: valueAtPath(payload, "adData", "buildingOverrideBuildingSubtype"), RoomLayout: firstNonEmpty(valueAtPath(payload, "adData", "roomConfiguration"), row.AdRoomLayout), RoomsCount: firstInt32(row.ShortcutAdRoomsCount, int32Path(payload, "adData", "rooms")), BedroomsCount: int32Path(payload, "adData", "bedrooms"), AreaM2: firstFloat64(row.AdArea, float64Path(payload, "adData", "size"), float64Path(payload, "adData", "sizeTotal"), float64Path(payload, "adData", "sizeLiving")), LivingAreaM2: float64Path(payload, "adData", "sizeLiving"), TotalAreaM2: float64Path(payload, "adData", "sizeTotal"), OtherAreaM2: float64Path(payload, "adData", "sizeOther"), FloorLevel: firstInt32(row.ShortcutAdFloorLevel, int32Path(payload, "adData", "floor")), Condition: firstNonEmpty(valueOrEmpty(row.ShortcutAdCondition), valueAtPath(payload, "adData", "condition"), valueAtPath(payload, "adData", "apartmentCondition"), valueAtPath(payload, "property", "condition")), Sauna: firstBool(row.ShortcutAdSauna, boolPath(payload, "adData", "hasSauna"), boolPath(payload, "adData", "sauna")), Balcony: boolPath(payload, "adData", "balcony"), Parking: firstNonEmpty(valueAtPath(payload, "adData", "parkingSpaceInfo"), valueAtPath(payload, "adData", "carStorageInfo")), Availability: firstNonEmpty(valueAtPath(payload, "adData", "availabilityInfo"), valueAtPath(payload, "adData", "availabilityDescription"), valueAtPath(payload, "adData", "availableFrom"), valueAtPath(payload, "adData", "availabilityDate"), valueOrEmpty(row.ShortcutAdAvailabilityText)), KitchenDescription: valueAtPath(payload, "adData", "kitchenApplianceInfo"), BathroomDescription: valueAtPath(payload, "adData", "bathroomApplianceInfo"), StorageDescription: firstNonEmpty(valueAtPath(payload, "adData", "storageInfo"), valueAtPath(payload, "adData", "commonAreaInfo")), FloorMaterialsDescription: valueAtPath(payload, "adData", "floorMaterialInfo"), WallMaterialsDescription: valueAtPath(payload, "adData", "wallMaterialInfo"), BalconyDescription: valueAtPath(payload, "adData", "balconyInfo"), SaunaDescription: valueAtPath(payload, "adData", "saunaInfo"), Appliances: compactStrings(stringSlicePath(payload, "adData", "equipment")), Features: compactStrings(stringSlicePath(payload, "adData", "features"))}
}

func frontdoorUnit(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow, location Location) UnitDetails {
	return UnitDetails{Location: location, PropertyType: firstNonEmpty(valueAtPath(payload, "property", "propertyType"), row.AdPropertyType), PropertySubtype: firstNonEmpty(valueAtPath(payload, "property", "specificType"), valueAtPath(payload, "property", "residentialPropertyType")), RoomLayout: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "roomStructure"), row.AdRoomLayout), RoomsCount: firstInt32(row.FrontdoorAdRoomsCount, int32Path(payload, "residenceDetailsDTO", "totalRoomCount")), BedroomsCount: int32Path(payload, "residenceDetailsDTO", "bedroomCount"), AreaM2: firstFloat64(row.AdArea, float64Path(payload, "preparsed", "area"), float64Path(payload, "residenceDetailsDTO", "livingArea"), float64Path(payload, "property", "livingArea")), LivingAreaM2: float64Path(payload, "residenceDetailsDTO", "livingArea"), TotalAreaM2: float64Path(payload, "residenceDetailsDTO", "totalArea"), OtherAreaM2: float64Path(payload, "residenceDetailsDTO", "otherArea"), FloorLevel: firstInt32(row.FrontdoorAdFloorLevel, int32Path(payload, "residenceDetailsDTO", "housingCompanyApartmentInformationDTO", "floorLevel")), Condition: firstNonEmpty(strings.TrimSpace(row.AdCondition), valueAtPath(payload, "residenceDetailsDTO", "inspection", "overallCondition"), valueAtPath(payload, "property", "condition")), Sauna: row.FrontdoorAdSauna, Parking: valueAtPath(payload, "property", "carParkingInformation"), Availability: firstNonEmpty(valueOrEmpty(row.FrontdoorAdAvailabilityText), valueAtPath(payload, "availabilityDescription"), valueAtPath(payload, "property", "availabilityDescription")), KitchenDescription: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "kitchenDescription"), valueAtPath(payload, "property", "kitchenDescription")), BathroomDescription: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "bathroomDescription"), valueAtPath(payload, "property", "bathroomDescription")), StorageDescription: firstNonEmpty(valueAtPath(payload, "property", "storageSpacesDescription"), valueAtPath(payload, "residenceDetailsDTO", "storageSpacesDescription")), FloorMaterialsDescription: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "floorMaterialDescription"), valueAtPath(payload, "property", "floorMaterialDescription")), WallMaterialsDescription: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "wallMaterialDescription"), valueAtPath(payload, "property", "wallMaterialDescription")), BalconyDescription: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "balconyDescription"), valueAtPath(payload, "property", "balconyDescription")), SaunaDescription: valueAtPath(payload, "residenceDetailsDTO", "saunaDescription"), ViewsDescription: valueAtPath(payload, "residenceDetailsDTO", "viewsDescription"), Features: compactStrings(stringSlicePath(payload, "residenceDetailsDTO", "generalDwellingFeatures"))}
}

func shortcutCharges(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) Charges {
	return Charges{MaintenanceMonthly: firstFloat64(row.ShortcutAdMaintenanceChargeMonthly, float64Path(payload, "priceData", "maintenanceCharge"), float64Path(payload, "priceData", "monthlyFee")), TotalMonthly: firstFloat64(row.ShortcutAdTotalChargeMonthly, float64Path(payload, "priceData", "totalCharge")), Water: firstFloat64(row.ShortcutAdWaterCharge, float64Path(payload, "priceData", "waterFee"), float64Path(payload, "adData", "waterFee")), Parking: float64Path(payload, "adData", "parkingFee"), Sauna: float64Path(payload, "adData", "saunaFee"), Electricity: valueAtPath(payload, "adData", "electricFee"), Heating: valueAtPath(payload, "adData", "heatingCost"), Notes: firstNonEmpty(valueOrEmpty(row.ShortcutAdChargesText), valueAtPath(payload, "priceData", "chargesText"), valueAtPath(payload, "adData", "feesInfo"))}
}

func frontdoorCharges(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) Charges {
	return Charges{MaintenanceMonthly: firstFloat64(row.FrontdoorAdMaintenanceChargeMonthly, periodicCharge(payload, "HOUSING_COMPANY_MAINTENANCE_CHARGE"), periodicCharge(payload, "MAINTENANCE_CHARGE")), TotalMonthly: firstFloat64(row.FrontdoorAdTotalChargeMonthly, periodicCharge(payload, "HOUSING_COMPANY_TOTAL_CHARGE")), Water: firstFloat64(row.FrontdoorAdWaterCharge, periodicCharge(payload, "WATER")), Notes: firstNonEmpty(valueOrEmpty(row.FrontdoorAdChargesText), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), valueAtPath(payload, "property", "managementChargesAdditionalInfo"))}
}

func shortcutTexts(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) TextSections {
	return TextSections{Description: firstNonEmpty(valueOrEmpty(row.ShortcutAdDescriptionText), valueAtPath(payload, "adData", "description"), jsonTextAtPath(row.ShortcutAdData, "adData", "description"), valueAtPath(payload, "description"), jsonTextAtPath(row.ShortcutAdData, "description"), valueAtPath(payload, "text"), jsonTextAtPath(row.ShortcutAdData, "text")), Availability: firstNonEmpty(valueOrEmpty(row.ShortcutAdAvailabilityText), valueAtPath(payload, "adData", "availabilityDescription"), jsonTextAtPath(row.ShortcutAdData, "adData", "availabilityDescription"), valueAtPath(payload, "adData", "availableFrom"), jsonTextAtPath(row.ShortcutAdData, "adData", "availableFrom")), RenovationsDone: firstNonEmpty(valueOrEmpty(row.ShortcutAdRenovationsDoneText), valueAtPath(payload, "adData", "renovationInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "renovationInfo")), RenovationsPlanned: firstNonEmpty(valueOrEmpty(row.ShortcutAdRenovationsPlannedText), valueAtPath(payload, "adData", "renovationFutureInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "renovationFutureInfo")), AdditionalInfo: firstNonEmpty(valueOrEmpty(row.ShortcutAdAdditionalInfoText), valueAtPath(payload, "adData", "moreInfoText"), jsonTextAtPath(row.ShortcutAdData, "adData", "moreInfoText"), valueAtPath(payload, "adData", "additionalInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "additionalInfo")), Area: firstNonEmpty(valueAtPath(payload, "adData", "areaInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "areaInfo")), Building: firstNonEmpty(valueAtPath(payload, "adData", "buildingExtraInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "buildingExtraInfo"), valueAtPath(payload, "adData", "housingCompanyInformation"), jsonTextAtPath(row.ShortcutAdData, "adData", "housingCompanyInformation")), Transport: firstNonEmpty(valueAtPath(payload, "adData", "connectionsInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "connectionsInfo")), Amenities: firstNonEmpty(valueAtPath(payload, "adData", "servicesInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "servicesInfo")), Charges: firstNonEmpty(valueOrEmpty(row.ShortcutAdChargesText), valueAtPath(payload, "adData", "feesInfo"), jsonTextAtPath(row.ShortcutAdData, "adData", "feesInfo"))}
}

func frontdoorTexts(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) TextSections {
	return TextSections{Description: firstNonEmpty(valueOrEmpty(row.FrontdoorAdDescriptionText), valueAtPath(payload, "text"), jsonTextAtPath(row.FrontdoorAdData, "text"), valueAtPath(payload, "property", "description"), jsonTextAtPath(row.FrontdoorAdData, "property", "description")), Availability: firstNonEmpty(valueOrEmpty(row.FrontdoorAdAvailabilityText), valueAtPath(payload, "availabilityDescription"), jsonTextAtPath(row.FrontdoorAdData, "availabilityDescription")), RenovationsDone: firstNonEmpty(valueOrEmpty(row.FrontdoorAdRenovationsDoneText), valueAtPath(payload, "property", "renovationsDoneDescription"), jsonTextAtPath(row.FrontdoorAdData, "property", "renovationsDoneDescription")), RenovationsPlanned: firstNonEmpty(valueOrEmpty(row.FrontdoorAdRenovationsPlannedText), valueAtPath(payload, "property", "renovationsPlannedDescription"), jsonTextAtPath(row.FrontdoorAdData, "property", "renovationsPlannedDescription")), AdditionalInfo: firstNonEmpty(valueOrEmpty(row.FrontdoorAdAdditionalInfoText), valueAtPath(payload, "moreInformationAvailableFrom"), jsonTextAtPath(row.FrontdoorAdData, "moreInformationAvailableFrom"), valueAtPath(payload, "additionalItemsIncludedInSale"), jsonTextAtPath(row.FrontdoorAdData, "additionalItemsIncludedInSale")), Area: firstNonEmpty(valueAtPath(payload, "property", "additionalAreaMeasurementInformation"), jsonTextAtPath(row.FrontdoorAdData, "property", "additionalAreaMeasurementInformation")), Building: firstNonEmpty(valueAtPath(payload, "property", "housingCompany", "otherInfo"), jsonTextAtPath(row.FrontdoorAdData, "property", "housingCompany", "otherInfo")), Transport: firstNonEmpty(valueAtPath(payload, "property", "transportationServicesDescription"), jsonTextAtPath(row.FrontdoorAdData, "property", "transportationServicesDescription")), Amenities: firstNonEmpty(valueAtPath(payload, "property", "nearbyAmenitiesDescription"), jsonTextAtPath(row.FrontdoorAdData, "property", "nearbyAmenitiesDescription")), Charges: firstNonEmpty(valueOrEmpty(row.FrontdoorAdChargesText), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), jsonTextAtPath(row.FrontdoorAdData, "property", "periodicChargesAdditionalInfo"))}
}

func jsonTextAtPath(payload []byte, path ...string) string {
	if len(payload) == 0 {
		return ""
	}
	var value map[string]any
	if err := json.Unmarshal(payload, &value); err != nil {
		return ""
	}
	return valueAtPath(value, path...)
}

func shortcutAdBuilding(row db.GetShortcutAdUnifiedDetailRow, payload rawMap, location Location) BuildingDetails {
	identity := computedBuildingIdentity("shortcut", "building", ptrUUIDString(row.ShortcutBuildingID), location, valueOrEmpty(row.ShortcutBuildingHousingCompany), "", formatInt64(row.ShortcutBuildingExternalID))
	buildingLocation := Location{StreetAddress: firstNonEmpty(valueOrEmpty(row.ShortcutBuildingAddress), location.StreetAddress), City: location.City, Postal: location.Postal}
	return BuildingDetails{Identity: identity, Location: buildingLocation, HousingCompany: valueOrEmpty(row.ShortcutBuildingHousingCompany), BuildingType: valueAtPath(payload, "buildingData", "buildingType"), BuildingSubtype: valueAtPath(payload, "adData", "buildingOverrideBuildingSubtype"), BuildYear: firstInt32(row.ShortcutAdBuildYear, int32Path(payload, "buildingData", "year"), int32Path(payload, "adData", "constructionYear")), FloorCount: firstInt32(row.ShortcutAdTotalFloors, int32Path(payload, "adData", "totalFloors"), int32Path(payload, "buildingData", "floors")), EnergyClass: firstNonEmpty(valueOrEmpty(row.ShortcutAdEnergyClass), valueAtPath(payload, "adData", "energyClass"), valueAtPath(payload, "adData", "buildingOverrideEnergyClass")), EnergyEfficiencyLabel: firstNonEmpty(valueAtPath(payload, "adData", "buildingOverrideEnergyClass"), valueAtPath(payload, "adData", "energyClass"), valueAtPath(payload, "property", "energyClass"), valueOrEmpty(row.ShortcutAdEnergyClass)), Heating: firstNonEmpty(valueAtPath(payload, "adData", "heatingMethods"), valueAtPath(payload, "adData", "heatingDistributionMethods")), HeatingDescription: firstNonEmpty(valueAtPath(payload, "adData", "heatingInfo"), valueAtPath(payload, "buildingData", "heatingSystem")), BuildingMaterial: valueAtPath(payload, "buildingData", "buildingMaterial"), WallStructure: valueAtPath(payload, "adData", "wallStructure"), FrameConstructionMethod: valueAtPath(payload, "adData", "frameConstructionMethod"), CommonAreas: firstNonEmpty(valueAtPath(payload, "adData", "commonAreaInfo"), valueAtPath(payload, "adData", "commonAreas")), CarStorage: valueAtPath(payload, "adData", "carStorageInfo"), Elevator: firstBool(row.ShortcutAdElevator, boolPath(payload, "adData", "hasElevator"), boolPath(payload, "adData", "elevator")), Sauna: firstBool(row.ShortcutAdSauna, boolPath(payload, "adData", "hasSauna"), boolPath(payload, "adData", "sauna")), ManagementMethod: valueAtPath(payload, "adData", "managementMethod"), PropertyManager: valueAtPath(payload, "adData", "propertyManager")}
}

func frontdoorAdBuilding(row db.GetFrontdoorAdUnifiedDetailRow, payload rawMap, location Location, nativeID string) BuildingDetails {
	identity := computedBuildingIdentity("frontdoor", "building", nativeID, location, valueAtPath(payload, "property", "housingCompany", "name"), valueAtPath(payload, "property", "housingCompany", "businessId"), valueAtPath(payload, "property", "housingCompany", "id"))
	return BuildingDetails{Identity: identity, Location: location, HousingCompany: valueAtPath(payload, "property", "housingCompany", "name"), BusinessID: valueAtPath(payload, "property", "housingCompany", "businessId"), BuildingType: valueAtPath(payload, "property", "buildingType"), BuildingSubtype: valueAtPath(payload, "property", "specificType"), BuildYear: firstInt32(row.FrontdoorAdBuildYear, int32Path(payload, "residenceDetailsDTO", "constructionFinishedYear"), int32Path(payload, "property", "housingCompany", "usageStartYear")), FloorCount: firstInt32(row.FrontdoorAdTotalFloors, int32Path(payload, "property", "housingCompany", "floorCount"), int32Path(payload, "residenceDetailsDTO", "floorCount")), ApartmentCount: int32Path(payload, "property", "housingCompany", "apartmentsInHousingCompany"), BusinessPremiseCount: int32Path(payload, "property", "housingCompany", "businessPremisesInHousingCompany"), EnergyClass: firstNonEmpty(valueOrEmpty(row.FrontdoorAdEnergyClass), valueAtPath(payload, "property", "housingCompany", "energyCertificate", "energyCertificateType")), EnergyEfficiencyLabel: firstNonEmpty(valueAtPath(payload, "property", "housingCompany", "energyCertificate", "energyCertificateDescription"), valueAtPath(payload, "property", "energyCertificate", "energyCertificateDescription"), valueAtPath(payload, "property", "housingCompany", "energyCertificate", "energyCertificateType"), valueAtPath(payload, "property", "energyCertificate", "energyCertificateType"), valueOrEmpty(row.FrontdoorAdEnergyClass)), Heating: valueAtPath(payload, "property", "heating"), HeatingDescription: valueAtPath(payload, "property", "heatingDescription"), HeatingFuel: valueAtPath(payload, "property", "heatingFuel"), BuildingMaterial: valueAtPath(payload, "property", "buildingMaterial"), RoofType: valueAtPath(payload, "property", "roofType"), RoofMaterial: valueAtPath(payload, "property", "roofMaterial"), CommonAreas: valueAtPath(payload, "property", "commonAreas"), CarStorage: valueAtPath(payload, "property", "carParkingInformation"), Connectivity: valueAtPath(payload, "property", "telecommunicationConnections"), OtherInfo: valueAtPath(payload, "property", "housingCompany", "otherInfo"), Elevator: firstBool(row.FrontdoorAdElevator, boolPath(payload, "property", "housingCompany", "hasElevator")), Sauna: firstBool(row.FrontdoorAdSauna, boolPath(payload, "property", "housingCompany", "hasSauna")), MaintenanceResponsibility: valueAtPath(payload, "property", "maintenanceResponsibilityDescription")}
}

func frontdoorAnnouncementLocation(row db.GetFrontdoorAnnouncementUnifiedDetailRow) Location {
	address := strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2)}, " "))
	return Location{StreetAddress: address, City: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation), valueOrEmpty(row.FrontdoorBuildingMunicipality), valueOrEmpty(row.FrontdoorBuildingPostArea)), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode)}
}

func frontdoorAnnouncementBuilding(row db.GetFrontdoorAnnouncementUnifiedDetailRow, location Location) BuildingDetails {
	identity := computedBuildingIdentity("frontdoor", "building", row.FrontdoorBuildingID.String(), location, valueOrEmpty(row.FrontdoorBuildingCompanyName), "", formatInt64(row.FrontdoorBuildingHousingCompanyID))
	address := strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " "))
	buildingLocation := Location{StreetAddress: address, City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode)}
	return BuildingDetails{Identity: identity, Location: buildingLocation, HousingCompany: valueOrEmpty(row.FrontdoorBuildingCompanyName), EnergyClass: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode), EnergyEfficiencyLabel: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode)}
}

func shortcutSite(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) SiteDetails {
	return SiteDetails{PlotType: firstNonEmpty(valueOrEmpty(row.ShortcutAdPlotType), valueAtPath(payload, "adData", "plotType"), valueAtPath(payload, "property", "plotType")), PlotOwnershipType: firstNonEmpty(valueAtPath(payload, "adData", "estateOwnershipType"), valueAtPath(payload, "adData", "buildingOverrideLotOwnership")), PlotAreaM2: firstFloat64(float64Path(payload, "adData", "plotArea"), float64Path(payload, "buildingData", "plotArea")), Services: valueAtPath(payload, "adData", "servicesInfo"), Transport: valueAtPath(payload, "adData", "connectionsInfo")}
}

func frontdoorSite(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) SiteDetails {
	return SiteDetails{PlotType: firstNonEmpty(valueOrEmpty(row.FrontdoorAdPlotType), valueAtPath(payload, "property", "plot", "plotType")), PlotOwnershipType: firstNonEmpty(valueAtPath(payload, "property", "plot", "holdingType"), valueAtPath(payload, "property", "plot", "ownershipType")), PlotAreaM2: float64Path(payload, "property", "plot", "area"), LotRedemptionInfo: valueAtPath(payload, "property", "lotRedemptionInfo"), LotRentalAgreement: valueAtPath(payload, "property", "lotRentalAgreement"), Yard: valueAtPath(payload, "property", "yardDescription"), Shore: valueAtPath(payload, "property", "shoreDescription"), WaterSupply: valueAtPath(payload, "property", "waterSupplyDescription"), Sewer: valueAtPath(payload, "property", "sewerDescription"), RoadAccess: valueAtPath(payload, "property", "roadAccessDescription"), Zoning: valueAtPath(payload, "property", "zoningInformation"), DrivingDirections: valueAtPath(payload, "property", "drivingInstructions"), Services: valueAtPath(payload, "property", "nearbyAmenitiesDescription"), Transport: valueAtPath(payload, "property", "transportationServicesDescription"), WaterSupplyTypes: compactStrings(stringSlicePath(payload, "property", "waterSupplyTypes"))}
}

func shortcutCommercial(payload rawMap, row db.GetShortcutAdUnifiedDetailRow, source ListingSource) CommercialDetails {
	return CommercialDetails{Status: valueAtPath(payload, "status"), FirstSeenAt: source.FirstSeenAt, LastSeenAt: source.LastSeenAt, AskingPrice: row.AdPrice, DebtFreePrice: firstInt64(row.ShortcutAdDebtFreePrice, int64Path(payload, "priceData", "priceDebtFree")), DebtShareAmount: firstInt64(row.ShortcutAdDebtShareAmount, int64Path(payload, "priceData", "debtShare")), PricePerSquareMeter: firstFloat64(row.ShortcutAdPricePerM2, float64Path(payload, "priceData", "pricePerSqm"), float64Path(payload, "priceData", "pricePerSquareMeter")), Rent: row.AdPrice, RentPeriod: "month", SecurityDeposit: valueAtPath(payload, "adData", "securityDeposit"), AvailableFrom: firstNonEmpty(valueAtPath(payload, "adData", "availableFrom"), valueAtPath(payload, "adData", "availabilityDate"), valueOrEmpty(row.ShortcutAdAvailabilityText)), MinimumTermMonths: int32Path(payload, "adData", "minRentTimeMonths"), FixedTerm: boolPath(payload, "adData", "fixedTerm"), Furnished: boolPath(payload, "adData", "rentFurnished"), PetsAllowed: firstBool(boolPath(payload, "adData", "petsAllowed"), boolPath(payload, "adData", "allowedPets")), OwnershipType: valueAtPath(payload, "adData", "ownershipType"), FeesInfo: valueAtPath(payload, "adData", "feesInfo"), OtherTerms: valueAtPath(payload, "adData", "otherTerms"), NewDevelopment: boolPath(payload, "adData", "newDevelopment"), Charges: shortcutCharges(payload, row)}
}

func frontdoorCommercial(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow, source ListingSource) CommercialDetails {
	return CommercialDetails{Status: firstNonEmpty(source.Status, valueAtPath(payload, "status")), BookingStatus: valueAtPath(payload, "bookingStatus"), PublishedAt: source.PublishedAt, FirstSeenAt: source.FirstSeenAt, LastSeenAt: source.LastSeenAt, MapVisible: boolPath(payload, "mapVisible"), CanReceiveLeads: boolPath(payload, "canReceiveLeads"), AskingPrice: row.AdPrice, DebtFreePrice: firstInt64(row.FrontdoorAdDebtFreePrice, int64Path(payload, "debfFreePrice"), int64Path(payload, "debtFreePrice")), DebtShareAmount: firstInt64(row.FrontdoorAdDebtShareAmount, int64Path(payload, "debtShareAmount")), PreviousAskingPrice: int64Path(payload, "previousPrice"), PreviousDebtFreePrice: int64Path(payload, "previousDebtFreePrice"), PricePerSquareMeter: firstFloat64(row.FrontdoorAdPricePerM2, float64Path(payload, "pricePerSquareMeter")), OwnershipType: valueAtPath(payload, "property", "ownershipType"), DebtShareAdditionalInfo: valueAtPath(payload, "debtShareAdditionalInfo"), FeesInfo: firstNonEmpty(valueOrEmpty(row.FrontdoorAdChargesText), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), valueAtPath(payload, "property", "managementChargesAdditionalInfo")), FinancingFeeInterestOnlyPeriod: valueAtPath(payload, "financingFeeInterestOnlyPeriod"), FinancingFeeInterestOnlyStartDate: valueAtPath(payload, "financingFeeInterestOnlyPeriodStartDate"), FinancingFeeInterestOnlyEndDate: valueAtPath(payload, "financingFeeInterestOnlyPeriodEndDate"), OpenBiddingInUse: boolPath(payload, "openBiddingInUse"), OpenBiddingStartingSellingPrice: int64Path(payload, "openBiddingStartingSellingPrice"), OpenBiddingStartingDebtFreePrice: int64Path(payload, "openBiddingStartingDebtFreePrice"), OpenBiddingLatestOffer: int64Path(payload, "openBiddingLatestOffer"), OpenBiddingTargetURL: valueAtPath(payload, "openBiddingTargetUrl"), DevelopmentPhase: valueAtPath(payload, "hcaDevelopmentPhase"), NewDevelopment: boolPath(payload, "newProperty"), NotifyPriceChanged: boolPath(payload, "notifyPriceChanged"), Charges: frontdoorCharges(payload, row)}
}

func shortcutMedia(payload rawMap) Media {
	images := shortcutImagesFromArray(payload, "media")
	if len(images) == 0 {
		images = imagesFromArray(payload, "links")
	}
	var main *Image
	if len(images) > 0 {
		main = &images[0]
	}
	return Media{MainImage: main, Images: images}
}

func frontdoorMedia(payload rawMap) Media {
	property, ok := payload["property"].(map[string]any)
	if !ok {
		return Media{}
	}
	rawImages, ok := property["images"].(map[string]any)
	if !ok {
		return Media{}
	}
	images := make([]Image, 0, len(rawImages))
	for _, raw := range rawImages {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		image, ok := item["image"].(map[string]any)
		if !ok {
			continue
		}
		uri := cleanAnyString(image["uri"])
		if uri == "" {
			continue
		}
		images = append(images, frontdoorImageFromTemplate(uri, cleanAnyString(item["id"]), cleanAnyString(image["id"]), cleanAnyString(image["description"]), cleanAnyString(item["propertyImageType"]), int32FromAny(item["ordinal"])))
	}
	sortImages(images)
	var main *Image
	for i := range images {
		if images[i].Role == "MAIN" {
			main = &images[i]
			break
		}
	}
	if main == nil && len(images) > 0 {
		main = &images[0]
	}
	return Media{MainImage: main, Images: images}
}

func frontdoorAnnouncementMedia(uri string) Media {
	image := frontdoorImageFromTemplate(uri, "", "", "", "MAIN", ptrInt32(0))
	if image.URL == "" {
		return Media{}
	}
	return Media{MainImage: &image, Images: []Image{image}}
}

func shortcutContacts(payload rawMap) []Contact {
	contact, ok := payload["contact"].(map[string]any)
	if !ok {
		return nil
	}
	out := Contact{Name: cleanAnyString(contact["name"]), Phone: cleanAnyString(contact["phone"]), Email: cleanAnyString(contact["email"]), OfficeName: cleanAnyString(contact["officeName"])}
	if out == (Contact{}) {
		return nil
	}
	return []Contact{out}
}

func frontdoorContacts(payload rawMap) []Contact {
	info, ok := payload["announcementContactInfo"].(map[string]any)
	if !ok {
		return nil
	}
	out := Contact{Name: cleanAnyString(info["name"]), Phone: firstNonEmpty(cleanAnyString(info["phone"]), cleanAnyString(info["mobilePhone"])), OfficeName: cleanAnyString(info["officeName"]), Title: cleanAnyString(info["title"])}
	if out == (Contact{}) {
		return nil
	}
	return []Contact{out}
}

func shortcutLinks(payload rawMap) []Link {
	return linksFromArray(payload, "links")
}

func frontdoorLinks(payload rawMap) []Link {
	return linksFromArray(payload, "links")
}

func frontdoorShowings(payload rawMap) []Showing {
	items, ok := payload["showings"].([]any)
	if !ok {
		return nil
	}
	out := make([]Showing, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		start := anyMillisToTime(m["startTime"])
		end := anyMillisToTime(m["endTime"])
		info := strings.TrimSpace(fmt.Sprint(m["info"]))
		if start != nil || end != nil || info != "" {
			out = append(out, Showing{StartAt: start, EndAt: end, Info: info})
		}
	}
	return out
}

func periodicCharge(value rawMap, charge string) *float64 {
	property, ok := value["property"].(map[string]any)
	if !ok {
		return nil
	}
	items, ok := property["periodicCharges"].([]any)
	if !ok {
		return nil
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok || strings.TrimSpace(fmt.Sprint(m["periodicCharge"])) != charge {
			continue
		}
		if value, ok := m["price"].(float64); ok {
			return &value
		}
	}
	return nil
}

func firstBool(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func buildingRenovation(kind string, done *bool, year *int32) BuildingRenovation {
	if year != nil && *year <= 0 {
		year = nil
	}
	if done != nil && !*done && year == nil {
		done = nil
	}
	if done == nil && year == nil {
		return BuildingRenovation{}
	}
	return BuildingRenovation{Kind: kind, Done: done, Year: year}
}

func compactRenovations(values []BuildingRenovation) []BuildingRenovation {
	out := make([]BuildingRenovation, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		kind := cleanDisplayString(value.Kind)
		if kind == "" || (value.Done == nil && value.Year == nil) {
			continue
		}
		year := ""
		if value.Year != nil {
			year = strconv.FormatInt(int64(*value.Year), 10)
		}
		done := ""
		if value.Done != nil && *value.Done {
			done = "true"
		}
		key := strings.ToLower(kind + ":" + year + ":" + done)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value.Kind = kind
		out = append(out, value)
	}
	return out
}

func boolTextPtr(value *string) *bool {
	if value == nil {
		return nil
	}
	raw := strings.ToLower(strings.TrimSpace(*value))
	switch raw {
	case "true", "yes", "kyllä", "kylla", "1", "on":
		v := true
		return &v
	case "false", "no", "ei", "0", "off":
		v := false
		return &v
	default:
		return nil
	}
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func ptrBool(value bool) *bool {
	return &value
}

func ptrUUIDString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func formatInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatInt32(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "<nil>" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func imagesFromArray(payload rawMap, key string) []Image {
	items, ok := payload[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Image, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		url := firstNonEmpty(fmt.Sprint(m["url"]), fmt.Sprint(m["uri"]), fmt.Sprint(m["imageUrl"]))
		if url == "" || url == "<nil>" {
			continue
		}
		out = append(out, Image{URL: url, Description: strings.TrimSpace(fmt.Sprint(m["description"])), Role: strings.TrimSpace(fmt.Sprint(m["type"]))})
	}
	return out
}

func shortcutImagesFromArray(payload rawMap, key string) []Image {
	items, ok := payload[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Image, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		variants := compactImageVariants(map[string]string{
			"full":        cleanAnyString(m["url_full"]),
			"large":       cleanAnyString(m["url_large"]),
			"thumb":       cleanAnyString(m["url_thumb"]),
			"contact":     cleanAnyString(m["url_contact"]),
			"logo":        cleanAnyString(m["url_logo"]),
			"logo_full":   cleanAnyString(m["url_logo_full"]),
			"search_logo": cleanAnyString(m["url_search_logo"]),
		})
		url := firstNonEmpty(variants["large"], variants["full"], variants["thumb"], cleanAnyString(m["url"]), cleanAnyString(m["uri"]), cleanAnyString(m["imageUrl"]))
		if url == "" {
			continue
		}
		tags := stringSliceValue(m["tags"])
		role := "OTHER"
		if containsFold(tags, "floorplan") {
			role = "FLOOR_PLAN"
		}
		ordinal := int32FromAny(m["ordernr"])
		out = append(out, Image{ID: firstNonEmpty(cleanAnyString(m["media_id"]), cleanAnyString(m["card_id"])), Provider: "shortcut", ProviderID: cleanAnyString(m["media_id"]), URL: url, Variants: variants, Description: cleanAnyString(m["description"]), Role: role, Ordinal: ordinal, Tags: tags})
	}
	sortImages(out)
	if len(out) > 0 && out[0].Role == "OTHER" {
		out[0].Role = "MAIN"
	}
	return out
}

func frontdoorImageFromTemplate(uri string, propertyImageID string, imageID string, description string, role string, ordinal *int32) Image {
	template := normalizeFrontdoorImageTemplate(uri)
	if template == "" {
		return Image{}
	}
	variants := compactImageVariants(map[string]string{
		"thumb":   strings.ReplaceAll(template, "{imageParameters}", "320x220,fit,q75,f=webp"),
		"card":    strings.ReplaceAll(template, "{imageParameters}", "640x440,fit,q80,f=webp"),
		"large":   strings.ReplaceAll(template, "{imageParameters}", "1280x854,fit,q80,f=webp"),
		"gallery": strings.ReplaceAll(template, "{imageParameters}", "2470x1710,fit,q80,f=webp"),
	})
	providerID := firstNonEmpty(imageID, propertyImageID)
	return Image{ID: providerID, Provider: "frontdoor", ProviderID: providerID, URL: variants["large"], Variants: variants, Description: cleanDescription(description), Role: role, Ordinal: ordinal}
}

func normalizeFrontdoorImageTemplate(uri string) string {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" || trimmed == "<nil>" {
		return ""
	}
	if strings.HasPrefix(trimmed, "//") {
		return "https:" + trimmed
	}
	if strings.HasPrefix(trimmed, "http://") {
		return "https://" + strings.TrimPrefix(trimmed, "http://")
	}
	return trimmed
}

func compactImageVariants(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		cleaned := cleanAnyString(value)
		if cleaned != "" {
			out[key] = cleaned
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sortImages(images []Image) {
	sort.SliceStable(images, func(i int, j int) bool {
		leftMain := images[i].Role == "MAIN"
		rightMain := images[j].Role == "MAIN"
		if leftMain != rightMain {
			return leftMain
		}
		leftOrdinal := int32(1<<31 - 1)
		rightOrdinal := int32(1<<31 - 1)
		if images[i].Ordinal != nil {
			leftOrdinal = *images[i].Ordinal
		}
		if images[j].Ordinal != nil {
			rightOrdinal = *images[j].Ordinal
		}
		return leftOrdinal < rightOrdinal
	})
}

func int32FromAny(value any) *int32 {
	switch v := value.(type) {
	case int:
		out := int32(v)
		return &out
	case int32:
		return &v
	case int64:
		out := int32(v)
		return &out
	case float64:
		out := int32(v)
		return &out
	default:
		return nil
	}
}

func ptrInt32(value int32) *int32 {
	return &value
}

func stringSliceValue(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, cleanAnyString(item))
	}
	return compactStrings(out)
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func cleanAnyString(value any) string {
	return cleanDescription(fmt.Sprint(value))
}

func cleanDescription(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func linksFromArray(payload rawMap, key string) []Link {
	items, ok := payload[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Link, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		url := strings.TrimSpace(fmt.Sprint(m["url"]))
		if url == "" || url == "<nil>" {
			continue
		}
		out = append(out, Link{URL: url, Title: strings.TrimSpace(fmt.Sprint(m["title"])), Type: strings.TrimSpace(fmt.Sprint(m["type"]))})
	}
	return out
}

func anyMillisToTime(value any) *time.Time {
	switch v := value.(type) {
	case float64:
		i := int64(v)
		return millisToTime(&i)
	case int64:
		return millisToTime(&v)
	default:
		return nil
	}
}
