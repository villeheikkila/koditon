package properties

import (
	"strconv"
	"strings"

	"koditon/internal/db"
)

func saleFromShortcutAd(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) SaleListing {
	payload := parseRaw(row.ShortcutAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal)}
	source := shortcutAdSource(canonicalID, nativeID, row)
	property := shortcutProperty(payload, row)
	building := shortcutAdBuildingSummary(row, location)
	identity := building.Identity
	sale := SaleListing{ID: canonicalID, Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Location: location, Property: property, SaleTerms: SaleTerms{AskingPrice: row.AdPrice, DebtFreePrice: firstInt64(row.ShortcutAdDebtFreePrice, int64Path(payload, "priceData", "priceDebtFree")), DebtShareAmount: firstInt64(row.ShortcutAdDebtShareAmount, int64Path(payload, "priceData", "debtShare")), PricePerSquareMeter: firstFloat64(row.ShortcutAdPricePerM2, float64Path(payload, "priceData", "pricePerSqm"), float64Path(payload, "priceData", "pricePerSquareMeter")), OwnershipType: valueAtPath(payload, "adData", "ownershipType"), PlotType: firstNonEmpty(valueOrEmpty(row.ShortcutAdPlotType), valueAtPath(payload, "adData", "plotType"), valueAtPath(payload, "property", "plotType"))}, Charges: shortcutCharges(payload, row), Texts: shortcutTexts(payload, row), BuildingIdentity: identity, Building: &building}
	sale.Media = shortcutMedia(payload)
	sale.Contacts = shortcutContacts(payload)
	sale.Links = shortcutLinks(payload)
	return sale
}

func rentalFromShortcutAd(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) Rental {
	payload := parseRaw(row.ShortcutAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal)}
	source := shortcutAdSource(canonicalID, nativeID, row)
	property := shortcutProperty(payload, row)
	building := shortcutAdBuildingSummary(row, location)
	identity := building.Identity
	rental := Rental{ID: canonicalID, Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Location: location, Property: property, RentalTerms: RentalTerms{Rent: row.AdPrice, RentPeriod: "month", SecurityDeposit: valueAtPath(payload, "adData", "securityDeposit"), AvailableFrom: firstNonEmpty(valueAtPath(payload, "adData", "availableFrom"), valueOrEmpty(row.ShortcutAdAvailabilityText)), MinimumTermMonths: int32Path(payload, "adData", "minRentTimeMonths"), FixedTerm: boolPath(payload, "adData", "fixedTerm"), Furnished: boolPath(payload, "adData", "rentFurnished"), PetsAllowed: firstBool(boolPath(payload, "adData", "petsAllowed"), boolPath(payload, "adData", "allowedPets")), PricePerSquareMeter: row.ShortcutAdPricePerM2}, Charges: shortcutCharges(payload, row), Texts: shortcutTexts(payload, row), BuildingIdentity: identity, Building: &building}
	rental.Media = shortcutMedia(payload)
	rental.Contacts = shortcutContacts(payload)
	rental.Links = shortcutLinks(payload)
	return rental
}

func saleFromFrontdoorAd(canonicalID string, nativeID string, row db.GetFrontdoorAdUnifiedDetailRow) SaleListing {
	payload := parseRaw(row.FrontdoorAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Latitude: float64Path(payload, "property", "geoCode", "latitude"), Longitude: float64Path(payload, "property", "geoCode", "longitude")}
	source := frontdoorAdSource(canonicalID, nativeID, row, payload)
	property := frontdoorProperty(payload, row)
	identity := computedBuildingIdentity("frontdoor", "ad", nativeID, location, valueAtPath(payload, "property", "housingCompany", "name"), valueAtPath(payload, "property", "housingCompany", "businessId"), valueAtPath(payload, "property", "housingCompany", "id"))
	sale := SaleListing{ID: canonicalID, Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Location: location, Property: property, SaleTerms: SaleTerms{AskingPrice: row.AdPrice, DebtFreePrice: firstInt64(row.FrontdoorAdDebtFreePrice, int64Path(payload, "debfFreePrice")), DebtShareAmount: firstInt64(row.FrontdoorAdDebtShareAmount, int64Path(payload, "debtShareAmount")), PricePerSquareMeter: firstFloat64(row.FrontdoorAdPricePerM2, float64Path(payload, "pricePerSquareMeter")), OwnershipType: valueAtPath(payload, "property", "ownershipType"), PlotType: firstNonEmpty(valueOrEmpty(row.FrontdoorAdPlotType), valueAtPath(payload, "property", "plot", "plotType"), valueAtPath(payload, "property", "plot", "holdingType"))}, Charges: frontdoorCharges(payload, row), Texts: frontdoorTexts(payload, row), BuildingIdentity: identity}
	sale.Media = frontdoorMedia(payload)
	sale.Contacts = frontdoorContacts(payload)
	sale.Showings = frontdoorShowings(payload)
	sale.Links = frontdoorLinks(payload)
	return sale
}

func saleFromFrontdoorAnnouncement(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) SaleListing {
	location := frontdoorAnnouncementLocation(row)
	source := frontdoorAnnouncementSource(canonicalID, nativeID, row)
	building := frontdoorAnnouncementBuildingSummary(row, location)
	sale := SaleListing{ID: canonicalID, Source: source, Headline: firstNonEmpty(location.StreetAddress, valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), nativeID), Location: location, Property: PropertyDetails{PropertyType: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType), PropertySubtype: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype), RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), AreaM2: row.FrontdoorBuildingAnnouncementArea}, SaleTerms: SaleTerms{AskingPrice: float64ToInt64(row.FrontdoorBuildingAnnouncementSearchPrice)}, BuildingIdentity: building.Identity, Building: &building}
	return sale
}

func rentalFromFrontdoorAnnouncement(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) Rental {
	location := frontdoorAnnouncementLocation(row)
	source := frontdoorAnnouncementSource(canonicalID, nativeID, row)
	building := frontdoorAnnouncementBuildingSummary(row, location)
	rental := Rental{ID: canonicalID, Source: source, Headline: firstNonEmpty(location.StreetAddress, valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), nativeID), Location: location, Property: PropertyDetails{PropertyType: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType), PropertySubtype: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype), RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), AreaM2: row.FrontdoorBuildingAnnouncementArea}, RentalTerms: RentalTerms{Rent: float64ToInt64(row.FrontdoorBuildingAnnouncementSearchPrice), RentPeriod: valueOrEmpty(row.FrontdoorBuildingAnnouncementRentPeriod)}, BuildingIdentity: building.Identity, Building: &building}
	return rental
}

func buildingFromShortcut(canonicalID string, nativeID string, row db.GetShortcutBuildingUnifiedDetailRow) Building {
	location := Location{StreetAddress: valueOrEmpty(row.ShortcutBuildingAddress), Latitude: row.ShortcutBuildingLatitude, Longitude: row.ShortcutBuildingLongitude}
	source := ListingSource{Provider: "shortcut", Kind: "building", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: strconv.FormatInt(row.ShortcutBuildingExternalID, 10), URL: row.ShortcutBuildingUrl, OriginalURL: row.ShortcutBuildingUrl, LastSeenAt: timePtr(row.ShortcutBuildingUpdatedAt), Flags: map[string]bool{"page_not_found": boolPtrValue(row.ShortcutBuildingPageNotFound)}}
	identity := computedBuildingIdentity("shortcut", "building", nativeID, location, valueOrEmpty(row.ShortcutBuildingHousingCompany), "", strconv.FormatInt(row.ShortcutBuildingExternalID, 10))
	return Building{ID: identity.Key, Identity: identity, SourceRecords: []ListingSource{source}, Location: location, HousingCompany: valueOrEmpty(row.ShortcutBuildingHousingCompany), BuildingType: valueOrEmpty(row.ShortcutBuildingBuildingType), BuildingSubtype: valueOrEmpty(row.ShortcutBuildingBuildingSubtype), ConstructionYear: row.ShortcutBuildingConstructionYear, FloorCount: row.ShortcutBuildingFloorCount, ApartmentCount: row.ShortcutBuildingApartmentCount, Heating: valueOrEmpty(row.ShortcutBuildingHeatingSystem), PlotType: valueOrEmpty(row.ShortcutBuildingPlotType), Elevator: boolTextPtr(row.ShortcutBuildingHasElevator), Sauna: boolTextPtr(row.ShortcutBuildingHasSauna)}
}

func buildingFromFrontdoor(canonicalID string, nativeID string, row db.GetFrontdoorBuildingUnifiedDetailRow) Building {
	location := Location{StreetAddress: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Latitude: row.FrontdoorBuildingLatitude, Longitude: row.FrontdoorBuildingLongitude}
	source := ListingSource{Provider: "frontdoor", Kind: "building", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: formatInt64(row.FrontdoorBuildingHousingCompanyID), FriendlyID: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID), URL: valueOrEmpty(row.FrontdoorBuildingUrl), OriginalURL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: timePtr(row.FrontdoorBuildingLastSeenAt)}
	identity := computedBuildingIdentity("frontdoor", "building", nativeID, location, valueOrEmpty(row.FrontdoorBuildingCompanyName), valueOrEmpty(row.FrontdoorBuildingBusinessID), formatInt64(row.FrontdoorBuildingHousingCompanyID))
	return Building{ID: identity.Key, Identity: identity, SourceRecords: []ListingSource{source}, Location: location, HousingCompany: valueOrEmpty(row.FrontdoorBuildingCompanyName), BusinessID: valueOrEmpty(row.FrontdoorBuildingBusinessID), BuildYear: row.FrontdoorBuildingBuildYear, FloorCount: row.FrontdoorBuildingFloorCount, ApartmentCount: row.FrontdoorBuildingApartmentCount, EnergyClass: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode), Heating: valueOrEmpty(row.FrontdoorBuildingHeating), Elevator: row.FrontdoorBuildingHasElevator, Sauna: row.FrontdoorBuildingHasSauna, Texts: TextSections{Description: string(row.FrontdoorBuildingData)}}
}

func shortcutAdSource(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) ListingSource {
	return ListingSource{Provider: "shortcut", Kind: "ad", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: nativeID, URL: row.ShortcutAdUrl, OriginalURL: row.ShortcutAdUrl, LastSeenAt: timePtr(row.ShortcutAdLastSeenAt), Metadata: sourceMetadata(map[string]any{"ad_type": row.ShortcutAdType, "building_id": ptrUUIDString(row.ShortcutBuildingID), "building_external_id": formatInt64(row.ShortcutBuildingExternalID)})}
}

func frontdoorAdSource(canonicalID string, nativeID string, row db.GetFrontdoorAdUnifiedDetailRow, payload rawMap) ListingSource {
	return ListingSource{Provider: "frontdoor", Kind: "ad", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: row.FrontdoorAdExternalID, FriendlyID: valueAtPath(payload, "friendlyId"), URL: row.FrontdoorAdUrl, OriginalURL: row.FrontdoorAdUrl, LastSeenAt: timePtr(row.FrontdoorAdLastSeenAt), PublishedAt: millisToTime(int64Path(payload, "publishingTime")), Status: valueAtPath(payload, "status"), Flags: map[string]bool{"page_not_found": row.FrontdoorAdPageNotFound}}
}

func frontdoorAnnouncementSource(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) ListingSource {
	return ListingSource{Provider: "frontdoor", Kind: "announcement", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: formatInt32(row.FrontdoorBuildingAnnouncementExternalID), FriendlyID: valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), URL: valueOrEmpty(row.FrontdoorBuildingUrl), OriginalURL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), Flags: map[string]bool{"published": boolPtrValue(row.FrontdoorBuildingAnnouncementPublished)}, Metadata: sourceMetadata(map[string]any{"building_id": row.FrontdoorBuildingID.String(), "housing_company_id": formatInt64(row.FrontdoorBuildingHousingCompanyID), "rental_unique_no": formatInt32(row.FrontdoorBuildingAnnouncementRentalUniqueNo)})}
}
