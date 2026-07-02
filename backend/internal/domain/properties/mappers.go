package properties

import (
	"strconv"
	"strings"

	"koditon/internal/db"
)

func saleFromShortcutAd(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) SaleListing {
	payload := parseShortcutRaw(row.ShortcutAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal)}
	source := shortcutAdSource(canonicalID, nativeID, row)
	commercial := shortcutCommercial(payload, row, source)
	commercial.Rent = nil
	commercial.RentPeriod = ""
	commercial.SecurityDeposit = ""
	commercial.AvailableFrom = ""
	commercial.MinimumTermMonths = nil
	commercial.FixedTerm = nil
	commercial.Furnished = nil
	commercial.PetsAllowed = nil
	sale := SaleListing{ID: publicID("l", canonicalID), Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Unit: shortcutUnit(payload, row, location), Building: shortcutAdBuilding(row, payload, location), Site: shortcutSite(payload, row), Commercial: commercial, Texts: shortcutTexts(payload, row)}
	sale.Media = shortcutMedia(payload)
	sale.Contacts = shortcutContacts(payload)
	sale.Links = shortcutLinks(payload)
	return sale
}

func rentalFromShortcutAd(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) Rental {
	payload := parseShortcutRaw(row.ShortcutAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal)}
	source := shortcutAdSource(canonicalID, nativeID, row)
	commercial := shortcutCommercial(payload, row, source)
	commercial.AskingPrice = nil
	commercial.DebtFreePrice = nil
	commercial.DebtShareAmount = nil
	rental := Rental{ID: publicID("r", canonicalID), Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Unit: shortcutUnit(payload, row, location), Building: shortcutAdBuilding(row, payload, location), Site: shortcutSite(payload, row), Commercial: commercial, Texts: shortcutTexts(payload, row)}
	rental.Media = shortcutMedia(payload)
	rental.Contacts = shortcutContacts(payload)
	rental.Links = shortcutLinks(payload)
	return rental
}

func saleFromFrontdoorAd(canonicalID string, nativeID string, row db.GetFrontdoorAdUnifiedDetailRow) SaleListing {
	payload := parseFrontdoorRaw(row.FrontdoorAdData)
	location := Location{StreetAddress: valueOrEmpty(row.AdAddress), City: valueOrEmpty(row.AdCity), Postal: valueOrEmpty(row.AdPostal), Latitude: float64Path(payload, "property", "geoCode", "latitude"), Longitude: float64Path(payload, "property", "geoCode", "longitude")}
	source := frontdoorAdSource(canonicalID, nativeID, row, payload)
	sale := SaleListing{ID: publicID("l", canonicalID), Source: source, Headline: firstNonEmpty(location.StreetAddress, nativeID), Unit: frontdoorUnit(payload, row, location), Building: frontdoorAdBuilding(row, payload, location, nativeID), Site: frontdoorSite(payload, row), Commercial: frontdoorCommercial(payload, row, source), Texts: frontdoorTexts(payload, row)}
	sale.Media = frontdoorMedia(payload)
	sale.Contacts = frontdoorContacts(payload)
	sale.Showings = frontdoorShowings(payload)
	sale.Links = frontdoorLinks(payload)
	return sale
}

func saleFromFrontdoorAnnouncement(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) SaleListing {
	location := frontdoorAnnouncementLocation(row)
	source := frontdoorAnnouncementSource(canonicalID, nativeID, row)
	building := frontdoorAnnouncementBuilding(row, location)
	sale := SaleListing{ID: publicID("l", canonicalID), Source: source, Headline: firstNonEmpty(location.StreetAddress, valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), nativeID), Unit: UnitDetails{Location: location, PropertyType: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType), PropertySubtype: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype), RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), AreaM2: row.FrontdoorBuildingAnnouncementArea}, Building: building, Commercial: CommercialDetails{AskingPrice: float64ToInt64(row.FrontdoorBuildingAnnouncementSearchPrice), LastSeenAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), PublishedAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), IsCompanyAnnouncement: new(true)}}
	sale.Media = frontdoorAnnouncementMedia(valueOrEmpty(row.FrontdoorBuildingAnnouncementMainImageUri))
	return sale
}

func rentalFromFrontdoorAnnouncement(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) Rental {
	location := frontdoorAnnouncementLocation(row)
	source := frontdoorAnnouncementSource(canonicalID, nativeID, row)
	building := frontdoorAnnouncementBuilding(row, location)
	rental := Rental{ID: publicID("r", canonicalID), Source: source, Headline: firstNonEmpty(location.StreetAddress, valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), nativeID), Unit: UnitDetails{Location: location, PropertyType: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertyType), PropertySubtype: valueOrEmpty(row.FrontdoorBuildingAnnouncementPropertySubtype), RoomLayout: valueOrEmpty(row.FrontdoorBuildingAnnouncementRoomStructure), AreaM2: row.FrontdoorBuildingAnnouncementArea}, Building: building, Commercial: CommercialDetails{Rent: float64ToInt64(row.FrontdoorBuildingAnnouncementSearchPrice), RentPeriod: valueOrEmpty(row.FrontdoorBuildingAnnouncementRentPeriod), LastSeenAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), PublishedAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), IsCompanyAnnouncement: new(true)}}
	rental.Media = frontdoorAnnouncementMedia(valueOrEmpty(row.FrontdoorBuildingAnnouncementMainImageUri))
	return rental
}

func buildingFromShortcut(canonicalID string, nativeID string, row db.GetShortcutBuildingUnifiedDetailRow) Building {
	location := Location{StreetAddress: valueOrEmpty(row.ShortcutBuildingAddress), Latitude: row.ShortcutBuildingLatitude, Longitude: row.ShortcutBuildingLongitude}
	source := ListingSource{Provider: "shortcut", Kind: "building", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: strconv.FormatInt(row.ShortcutBuildingExternalID, 10), URL: row.ShortcutBuildingUrl, OriginalURL: row.ShortcutBuildingUrl, LastSeenAt: timePtr(row.ShortcutBuildingUpdatedAt), Flags: map[string]bool{"page_not_found": boolPtrValue(row.ShortcutBuildingPageNotFound)}}
	identity := computedBuildingIdentity("shortcut", "building", nativeID, location, valueOrEmpty(row.ShortcutBuildingHousingCompany), "", strconv.FormatInt(row.ShortcutBuildingExternalID, 10))
	details := BuildingDetails{Identity: identity, Location: location, HousingCompany: valueOrEmpty(row.ShortcutBuildingHousingCompany), BuildingType: valueOrEmpty(row.ShortcutBuildingBuildingType), BuildingSubtype: valueOrEmpty(row.ShortcutBuildingBuildingSubtype), ConstructionYear: row.ShortcutBuildingConstructionYear, FloorCount: row.ShortcutBuildingFloorCount, ApartmentCount: row.ShortcutBuildingApartmentCount, Heating: valueOrEmpty(row.ShortcutBuildingHeatingSystem), Elevator: boolTextPtr(row.ShortcutBuildingHasElevator), Sauna: boolTextPtr(row.ShortcutBuildingHasSauna)}
	return Building{ID: publicID("b", canonicalID), Details: details, Site: SiteDetails{PlotType: valueOrEmpty(row.ShortcutBuildingPlotType)}, SourceRecords: []ListingSource{source}}
}

func buildingFromFrontdoor(canonicalID string, nativeID string, row db.GetFrontdoorBuildingUnifiedDetailRow) Building {
	location := Location{StreetAddress: strings.TrimSpace(strings.Join([]string{valueOrEmpty(row.FrontdoorBuildingStreetAddress), valueOrEmpty(row.FrontdoorBuildingHouseNumber)}, " ")), City: valueOrEmpty(row.FrontdoorBuildingMunicipality), Postal: valueOrEmpty(row.FrontdoorBuildingPostcode), Latitude: row.FrontdoorBuildingLatitude, Longitude: row.FrontdoorBuildingLongitude}
	source := ListingSource{Provider: "frontdoor", Kind: "building", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: formatInt64(row.FrontdoorBuildingHousingCompanyID), FriendlyID: valueOrEmpty(row.FrontdoorBuildingHousingCompanyFriendlyID), URL: valueOrEmpty(row.FrontdoorBuildingUrl), OriginalURL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: timePtr(row.FrontdoorBuildingLastSeenAt)}
	identity := computedBuildingIdentity("frontdoor", "building", nativeID, location, valueOrEmpty(row.FrontdoorBuildingCompanyName), valueOrEmpty(row.FrontdoorBuildingBusinessID), formatInt64(row.FrontdoorBuildingHousingCompanyID))
	details := BuildingDetails{Identity: identity, Location: location, HousingCompany: valueOrEmpty(row.FrontdoorBuildingCompanyName), BusinessID: valueOrEmpty(row.FrontdoorBuildingBusinessID), BuildYear: row.FrontdoorBuildingBuildYear, FloorCount: row.FrontdoorBuildingFloorCount, ApartmentCount: row.FrontdoorBuildingApartmentCount, EnergyClass: valueOrEmpty(row.FrontdoorBuildingEnergyCertificateCode), Heating: valueOrEmpty(row.FrontdoorBuildingHeating), Elevator: row.FrontdoorBuildingHasElevator, Sauna: row.FrontdoorBuildingHasSauna}
	return Building{ID: publicID("b", canonicalID), Details: details, SourceRecords: []ListingSource{source}}
}

func shortcutAdSource(canonicalID string, nativeID string, row db.GetShortcutAdUnifiedDetailRow) ListingSource {
	return ListingSource{Provider: "shortcut", Kind: "ad", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: nativeID, URL: row.ShortcutAdUrl, OriginalURL: row.ShortcutAdUrl, LastSeenAt: timePtr(row.ShortcutAdLastSeenAt), Metadata: sourceMetadata(map[string]any{"ad_type": row.ShortcutAdType, "building_id": ptrUUIDString(row.ShortcutBuildingID), "building_external_id": strconv.FormatInt(row.ShortcutBuildingExternalID, 10)})}
}

func frontdoorAdSource(canonicalID string, nativeID string, row db.GetFrontdoorAdUnifiedDetailRow, payload rawMap) ListingSource {
	return ListingSource{Provider: "frontdoor", Kind: "ad", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: row.FrontdoorAdExternalID, FriendlyID: valueAtPath(payload, "friendlyId"), URL: row.FrontdoorAdUrl, OriginalURL: row.FrontdoorAdUrl, LastSeenAt: timePtr(row.FrontdoorAdLastSeenAt), PublishedAt: millisToTime(int64Path(payload, "publishingTime")), Status: valueAtPath(payload, "status"), Flags: map[string]bool{"page_not_found": row.FrontdoorAdPageNotFound}}
}

func frontdoorAnnouncementSource(canonicalID string, nativeID string, row db.GetFrontdoorAnnouncementUnifiedDetailRow) ListingSource {
	return ListingSource{Provider: "frontdoor", Kind: "announcement", CanonicalID: canonicalID, NativeID: nativeID, ExternalID: formatInt32(row.FrontdoorBuildingAnnouncementExternalID), FriendlyID: valueOrEmpty(row.FrontdoorBuildingAnnouncementFriendlyID), URL: valueOrEmpty(row.FrontdoorBuildingUrl), OriginalURL: valueOrEmpty(row.FrontdoorBuildingUrl), LastSeenAt: timePtr(row.FrontdoorBuildingAnnouncementLastSeenAt), Flags: map[string]bool{"published": boolPtrValue(row.FrontdoorBuildingAnnouncementPublished)}, Metadata: sourceMetadata(map[string]any{"building_id": row.FrontdoorBuildingID.String(), "housing_company_id": formatInt64(row.FrontdoorBuildingHousingCompanyID), "rental_unique_no": formatInt32(row.FrontdoorBuildingAnnouncementRentalUniqueNo)})}
}
