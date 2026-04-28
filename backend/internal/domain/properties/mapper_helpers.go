package properties

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
)

func shortcutProperty(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) PropertyDetails {
	return PropertyDetails{PropertyType: firstNonEmpty(valueAtPath(payload, "adData", "habitationType"), valueAtPath(payload, "adData", "listingTypes")), PropertySubtype: valueAtPath(payload, "adData", "buildingOverrideBuildingSubtype"), RoomLayout: firstNonEmpty(valueAtPath(payload, "adData", "roomConfiguration"), row.AdRoomLayout), RoomsCount: firstInt32(row.ShortcutAdRoomsCount, int32Path(payload, "adData", "rooms")), BedroomsCount: int32Path(payload, "adData", "bedrooms"), AreaM2: firstFloat64(row.AdArea, float64Path(payload, "adData", "size"), float64Path(payload, "adData", "sizeTotal"), float64Path(payload, "adData", "sizeLiving")), LivingAreaM2: float64Path(payload, "adData", "sizeLiving"), TotalAreaM2: float64Path(payload, "adData", "sizeTotal"), FloorLevel: firstInt32(row.ShortcutAdFloorLevel, int32Path(payload, "adData", "floor")), TotalFloors: firstInt32(row.ShortcutAdTotalFloors, int32Path(payload, "adData", "totalFloors"), int32Path(payload, "buildingData", "floors")), BuildYear: firstInt32(row.ShortcutAdBuildYear, int32Path(payload, "buildingData", "year"), int32Path(payload, "adData", "constructionYear")), Condition: firstNonEmpty(valueOrEmpty(row.ShortcutAdCondition), valueAtPath(payload, "adData", "condition"), valueAtPath(payload, "adData", "apartmentCondition"), valueAtPath(payload, "property", "condition")), EnergyClass: firstNonEmpty(valueOrEmpty(row.ShortcutAdEnergyClass), valueAtPath(payload, "adData", "energyClass"), valueAtPath(payload, "adData", "buildingOverrideEnergyClass")), Elevator: firstBool(row.ShortcutAdElevator, boolPath(payload, "adData", "hasElevator"), boolPath(payload, "adData", "elevator")), Sauna: firstBool(row.ShortcutAdSauna, boolPath(payload, "adData", "hasSauna"), boolPath(payload, "adData", "sauna")), Balcony: boolPath(payload, "adData", "balcony"), Parking: firstNonEmpty(valueAtPath(payload, "adData", "parkingSpaceInfo"), valueAtPath(payload, "adData", "carStorageInfo")), Features: compactStrings(stringSlicePath(payload, "adData", "equipment"))}
}

func frontdoorProperty(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) PropertyDetails {
	return PropertyDetails{PropertyType: firstNonEmpty(valueAtPath(payload, "property", "propertyType"), row.AdPropertyType), PropertySubtype: firstNonEmpty(valueAtPath(payload, "property", "specificType"), valueAtPath(payload, "property", "residentialPropertyType")), RoomLayout: firstNonEmpty(valueAtPath(payload, "residenceDetailsDTO", "roomStructure"), row.AdRoomLayout), RoomsCount: firstInt32(row.FrontdoorAdRoomsCount, int32Path(payload, "residenceDetailsDTO", "totalRoomCount")), BedroomsCount: int32Path(payload, "residenceDetailsDTO", "bedroomCount"), AreaM2: firstFloat64(row.AdArea, float64Path(payload, "preparsed", "area"), float64Path(payload, "residenceDetailsDTO", "livingArea"), float64Path(payload, "property", "livingArea")), LivingAreaM2: float64Path(payload, "residenceDetailsDTO", "livingArea"), TotalAreaM2: float64Path(payload, "residenceDetailsDTO", "totalArea"), FloorLevel: firstInt32(row.FrontdoorAdFloorLevel, int32Path(payload, "residenceDetailsDTO", "housingCompanyApartmentInformationDTO", "floorLevel")), TotalFloors: firstInt32(row.FrontdoorAdTotalFloors, int32Path(payload, "property", "housingCompany", "floorCount"), int32Path(payload, "residenceDetailsDTO", "floorCount")), BuildYear: firstInt32(row.FrontdoorAdBuildYear, int32Path(payload, "residenceDetailsDTO", "constructionFinishedYear"), int32Path(payload, "property", "housingCompany", "usageStartYear")), Condition: firstNonEmpty(strings.TrimSpace(row.AdCondition), valueAtPath(payload, "residenceDetailsDTO", "inspection", "overallCondition"), valueAtPath(payload, "property", "condition")), EnergyClass: firstNonEmpty(valueOrEmpty(row.FrontdoorAdEnergyClass), valueAtPath(payload, "property", "housingCompany", "energyCertificate", "energyCertificateType")), Elevator: firstBool(row.FrontdoorAdElevator, boolPath(payload, "property", "housingCompany", "hasElevator")), Sauna: firstBool(row.FrontdoorAdSauna, boolPath(payload, "property", "housingCompany", "hasSauna")), Parking: valueAtPath(payload, "property", "carParkingInformation"), Features: compactStrings(stringSlicePath(payload, "residenceDetailsDTO", "generalDwellingFeatures"))}
}

func shortcutCharges(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) Charges {
	return Charges{MaintenanceMonthly: firstFloat64(row.ShortcutAdMaintenanceChargeMonthly, float64Path(payload, "priceData", "maintenanceCharge"), float64Path(payload, "priceData", "monthlyFee")), TotalMonthly: firstFloat64(row.ShortcutAdTotalChargeMonthly, float64Path(payload, "priceData", "totalCharge")), Water: firstFloat64(row.ShortcutAdWaterCharge, float64Path(payload, "priceData", "waterFee"), float64Path(payload, "adData", "waterFee")), Parking: float64Path(payload, "adData", "parkingFee"), Sauna: float64Path(payload, "adData", "saunaFee"), Electricity: valueAtPath(payload, "adData", "electricFee"), Heating: valueAtPath(payload, "adData", "heatingCost"), Notes: firstNonEmpty(valueOrEmpty(row.ShortcutAdChargesText), valueAtPath(payload, "priceData", "chargesText"), valueAtPath(payload, "adData", "feesInfo"))}
}

func frontdoorCharges(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) Charges {
	return Charges{MaintenanceMonthly: firstFloat64(row.FrontdoorAdMaintenanceChargeMonthly, periodicCharge(payload, "HOUSING_COMPANY_MAINTENANCE_CHARGE"), periodicCharge(payload, "MAINTENANCE_CHARGE")), TotalMonthly: firstFloat64(row.FrontdoorAdTotalChargeMonthly, periodicCharge(payload, "HOUSING_COMPANY_TOTAL_CHARGE")), Water: firstFloat64(row.FrontdoorAdWaterCharge, periodicCharge(payload, "WATER")), Notes: firstNonEmpty(valueOrEmpty(row.FrontdoorAdChargesText), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"), valueAtPath(payload, "property", "managementChargesAdditionalInfo"))}
}

func shortcutTexts(payload rawMap, row db.GetShortcutAdUnifiedDetailRow) TextSections {
	return TextSections{Description: firstNonEmpty(valueOrEmpty(row.ShortcutAdDescriptionText), valueAtPath(payload, "adData", "description"), valueAtPath(payload, "description"), valueAtPath(payload, "text")), Availability: firstNonEmpty(valueOrEmpty(row.ShortcutAdAvailabilityText), valueAtPath(payload, "adData", "availabilityDescription"), valueAtPath(payload, "adData", "availableFrom")), RenovationsDone: firstNonEmpty(valueOrEmpty(row.ShortcutAdRenovationsDoneText), valueAtPath(payload, "adData", "renovationInfo")), RenovationsPlanned: firstNonEmpty(valueOrEmpty(row.ShortcutAdRenovationsPlannedText), valueAtPath(payload, "adData", "renovationFutureInfo")), AdditionalInfo: firstNonEmpty(valueOrEmpty(row.ShortcutAdAdditionalInfoText), valueAtPath(payload, "adData", "moreInfoText"), valueAtPath(payload, "adData", "additionalInfo")), Area: valueAtPath(payload, "adData", "areaInfo"), Building: firstNonEmpty(valueAtPath(payload, "adData", "buildingExtraInfo"), valueAtPath(payload, "adData", "housingCompanyInformation")), Transport: valueAtPath(payload, "adData", "connectionsInfo"), Amenities: valueAtPath(payload, "adData", "servicesInfo"), Charges: firstNonEmpty(valueOrEmpty(row.ShortcutAdChargesText), valueAtPath(payload, "adData", "feesInfo"))}
}

func frontdoorTexts(payload rawMap, row db.GetFrontdoorAdUnifiedDetailRow) TextSections {
	return TextSections{Description: firstNonEmpty(valueOrEmpty(row.FrontdoorAdDescriptionText), valueAtPath(payload, "text"), valueAtPath(payload, "property", "description")), Availability: firstNonEmpty(valueOrEmpty(row.FrontdoorAdAvailabilityText), valueAtPath(payload, "availabilityDescription")), RenovationsDone: firstNonEmpty(valueOrEmpty(row.FrontdoorAdRenovationsDoneText), valueAtPath(payload, "property", "renovationsDoneDescription")), RenovationsPlanned: firstNonEmpty(valueOrEmpty(row.FrontdoorAdRenovationsPlannedText), valueAtPath(payload, "property", "renovationsPlannedDescription")), AdditionalInfo: firstNonEmpty(valueOrEmpty(row.FrontdoorAdAdditionalInfoText), valueAtPath(payload, "moreInformationAvailableFrom"), valueAtPath(payload, "additionalItemsIncludedInSale")), Area: valueAtPath(payload, "property", "additionalAreaMeasurementInformation"), Building: valueAtPath(payload, "property", "housingCompany", "otherInfo"), Transport: valueAtPath(payload, "property", "transportationServicesDescription"), Amenities: valueAtPath(payload, "property", "nearbyAmenitiesDescription"), Charges: firstNonEmpty(valueOrEmpty(row.FrontdoorAdChargesText), valueAtPath(payload, "property", "periodicChargesAdditionalInfo"))}
}

func shortcutAdBuildingSummary(row db.GetShortcutAdUnifiedDetailRow, location Location) BuildingSummary {
	identity := computedBuildingIdentity("shortcut", "building", ptrUUIDString(row.ShortcutBuildingID), location, valueOrEmpty(row.ShortcutBuildingHousingCompany), "", formatInt64(row.ShortcutBuildingExternalID))
	return BuildingSummary{Identity: identity, HousingCompany: valueOrEmpty(row.ShortcutBuildingHousingCompany), Address: valueOrEmpty(row.ShortcutBuildingAddress), BuildYear: row.ShortcutAdBuildYear}
}

func frontdoorAnnouncementLocation(row db.GetFrontdoorAnnouncementUnifiedDetailRow) Location {
	address := strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine1), valueOrEmpty(row.FrontdoorBuildingAnnouncementAddressLine2)}, " "))
	return Location{StreetAddress: address, City: firstNonEmpty(valueOrEmpty(row.FrontdoorBuildingAnnouncementLocation), valueOrEmpty(row.FrontdoorBuildingMunicipality), valueOrEmpty(row.FrontdoorBuildingPostArea)), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode)}
}

func frontdoorAnnouncementBuildingSummary(row db.GetFrontdoorAnnouncementUnifiedDetailRow, location Location) BuildingSummary {
	identity := computedBuildingIdentity("frontdoor", "building", row.FrontdoorBuildingID.String(), location, valueOrEmpty(row.FrontdoorBuildingCompanyName), "", formatInt64(row.FrontdoorBuildingHousingCompanyID))
	address := strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " "))
	return BuildingSummary{Identity: identity, HousingCompany: valueOrEmpty(row.FrontdoorBuildingCompanyName), Address: address, City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode)}
}

func shortcutMedia(payload rawMap) Media {
	images := imagesFromArray(payload, "media")
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
		uri := strings.TrimSpace(fmt.Sprint(image["uri"]))
		if uri == "" {
			continue
		}
		images = append(images, Image{URL: uri, Description: strings.TrimSpace(fmt.Sprint(image["description"])), Kind: strings.TrimSpace(fmt.Sprint(item["propertyImageType"]))})
	}
	var main *Image
	if len(images) > 0 {
		main = &images[0]
	}
	return Media{MainImage: main, Images: images}
}

func shortcutContacts(payload rawMap) []Contact {
	contact, ok := payload["contact"].(map[string]any)
	if !ok {
		return nil
	}
	out := Contact{Name: strings.TrimSpace(fmt.Sprint(contact["name"])), Phone: strings.TrimSpace(fmt.Sprint(contact["phone"])), Email: strings.TrimSpace(fmt.Sprint(contact["email"])), OfficeName: strings.TrimSpace(fmt.Sprint(contact["officeName"]))}
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
	out := Contact{Name: strings.TrimSpace(fmt.Sprint(info["name"])), Phone: firstNonEmpty(strings.TrimSpace(fmt.Sprint(info["phone"])), strings.TrimSpace(fmt.Sprint(info["mobilePhone"]))), OfficeName: strings.TrimSpace(fmt.Sprint(info["officeName"])), Title: strings.TrimSpace(fmt.Sprint(info["title"]))}
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
		out = append(out, Image{URL: url, Description: strings.TrimSpace(fmt.Sprint(m["description"])), Kind: strings.TrimSpace(fmt.Sprint(m["type"]))})
	}
	return out
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
