package frontdoor

import (
	"encoding/json"
	"fmt"

	client "koditon/internal/clients/frontdoor"
	"koditon/internal/db"
	"koditon/internal/platform/util"
	frontdoorpayload "koditon/internal/providers/frontdoor"
	"koditon/internal/sync/sourcejson"

	"github.com/google/uuid"
)

func mapBatchUpsertAdsFromSitemapParams(entries []client.SitemapEntry) db.BatchUpsertFrontdoorAdsFromSitemapParams {
	externalIDs := make([]string, len(entries))
	urls := make([]string, len(entries))
	for i, entry := range entries {
		externalIDs[i] = entry.ID
		urls[i] = entry.URL.String()
	}
	return db.BatchUpsertFrontdoorAdsFromSitemapParams{
		Column1: externalIDs,
		Column2: urls,
	}
}

func mapBatchUpsertBuildingsFromSitemapParams(entries []client.SitemapEntry) []string {
	urls := make([]string, len(entries))
	for i, entry := range entries {
		urls[i] = entry.URL.String()
	}
	return urls
}

func mapAdParams(friendlyID string, ad *frontdoorpayload.AdResponse) (db.UpdateFrontdoorAdDataParams, error) {
	params := db.UpdateFrontdoorAdDataParams{
		FrontdoorAdExternalID: &friendlyID,
	}
	jsonData, err := json.Marshal(ad)
	if err != nil {
		return db.UpdateFrontdoorAdDataParams{}, fmt.Errorf("marshal frontdoor ad payload: %w", err)
	}
	canonical, hash, err := sourcejson.CanonicalizeAndHash(jsonData)
	if err != nil {
		return db.UpdateFrontdoorAdDataParams{}, fmt.Errorf("hash frontdoor ad payload: %w", err)
	}
	params.FrontdoorAdData = canonical
	params.FrontdoorAdDataHash = &hash
	params.FrontdoorAdDataHashAlgorithm = ptr(sourcejson.HashAlgorithmSHA256)
	return params, nil
}

func mapBuildingParams(housingCompanyID int64, data *frontdoorpayload.HousingCompanyResponse) db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams {
	p := db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams{
		FrontdoorBuildingHousingCompanyID: new(housingCompanyID),
	}
	if page := data.HousingCompanyPage; page != nil && page.Response != nil {
		if hca := page.Response.HousingCompanyAnnouncement; hca != nil {
			p.FrontdoorBuildingDescription = hca.Text
			if ci := hca.ContactInfo; ci != nil {
				p.FrontdoorBuildingContactPhone = ci.Phone
				p.FrontdoorBuildingContactOfficeName = ci.OfficeName
				p.FrontdoorBuildingContactOfficeID = util.Int32Ptr(ci.OfficeID)
			}
			if hc := hca.HousingCompany; hc != nil {
				p.FrontdoorBuildingCompanyName = hc.Name
				p.FrontdoorBuildingApartmentCount = util.Int32Ptr(hc.ApartmentCount)
				p.FrontdoorBuildingFloorCount = util.FloatToInt32Ptr(hc.FloorCount)
				p.FrontdoorBuildingConstructionEndYear = util.Int32Ptr(hc.ConstructionEndYear)
				p.FrontdoorBuildingOtherInfo = hc.OtherInfo
				p.FrontdoorBuildingHouseNumber = hc.HouseNumber
				if hc.PostCode != nil {
					p.FrontdoorBuildingPostArea = hc.PostCode.PostArea
				}
				if hc.Municipality != nil {
					p.FrontdoorBuildingMunicipality = hc.Municipality.DefaultName
				}
				if hc.District != nil {
					p.FrontdoorBuildingDistrict = hc.District.DefaultName
				}
				if gc := hc.GeoCode; gc != nil {
					p.FrontdoorBuildingLatitude = gc.Latitude
					p.FrontdoorBuildingLongitude = gc.Longitude
				}
			}
		}
	}
	if ksa := data.KsaHousingCompanyPage; ksa != nil && ksa.Response != nil {
		resp := ksa.Response
		p.FrontdoorBuildingBusinessID = resp.BusinessID
		if p.FrontdoorBuildingCompanyName == nil {
			p.FrontdoorBuildingCompanyName = resp.CompanyName
		}
		if info := resp.AdHousingCompanyInfo; info != nil {
			if p.FrontdoorBuildingApartmentCount == nil {
				p.FrontdoorBuildingApartmentCount = util.Int32Ptr(info.ApartmentCount)
			}
			if p.FrontdoorBuildingFloorCount == nil {
				p.FrontdoorBuildingFloorCount = util.FloatToInt32Ptr(info.FloorCount)
			}
			p.FrontdoorBuildingHasElevator = info.HasElevator
			p.FrontdoorBuildingHasSauna = info.HasSauna
			p.FrontdoorBuildingEnergyCertificateCode = info.EnergyCertificateCode
			p.FrontdoorBuildingPlotHoldingType = info.PlotHoldingType
			p.FrontdoorBuildingOuterRoofMaterial = info.OuterRoofMaterial
			p.FrontdoorBuildingOuterRoofType = info.OuterRoofType
			p.FrontdoorBuildingCarStorageDescription = info.CarStorageDescription
			if r := info.ClassifiedPastRenovationsDetected; r != nil {
				p.FrontdoorBuildingElevatorRenovated = r.ElevatorRenovated
				p.FrontdoorBuildingElevatorRenovatedYear = util.Int32Ptr(r.ElevatorRenovatedYear)
				p.FrontdoorBuildingFacadeRenovated = r.FacadeRenovated
				p.FrontdoorBuildingFacadeRenovatedYear = util.Int32Ptr(r.FacadeRenovatedYear)
				p.FrontdoorBuildingWindowRenovated = r.WindowRenovated
				p.FrontdoorBuildingWindowRenovatedYear = util.Int32Ptr(r.WindowRenovatedYear)
				p.FrontdoorBuildingRoofRenovated = r.RoofRenovated
				p.FrontdoorBuildingRoofRenovatedYear = util.Int32Ptr(r.RoofRenovatedYear)
				p.FrontdoorBuildingPipeRenovated = r.PipeRenovated
				p.FrontdoorBuildingPipeRenovatedYear = util.Int32Ptr(r.PipeRenovatedYear)
				p.FrontdoorBuildingBalconyRenovated = r.BalconyRenovated
				p.FrontdoorBuildingBalconyRenovatedYear = util.Int32Ptr(r.BalconyRenovatedYear)
				p.FrontdoorBuildingElectricityRenovated = r.ElectricityRenovated
				p.FrontdoorBuildingElectricityRenovatedYear = util.Int32Ptr(r.ElectricityRenovatedYear)
			}
		}
		if len(resp.HouseAddresses) > 0 {
			addr := resp.HouseAddresses[0]
			p.FrontdoorBuildingStreetAddress = addr.StreetAddress
			p.FrontdoorBuildingPostcode = addr.Postcode
			if p.FrontdoorBuildingMunicipality == nil {
				p.FrontdoorBuildingMunicipality = addr.Municipality
			}
			if p.FrontdoorBuildingDistrict == nil {
				p.FrontdoorBuildingDistrict = addr.District
			}
			if p.FrontdoorBuildingLatitude == nil {
				p.FrontdoorBuildingLatitude = addr.Latitude
			}
			if p.FrontdoorBuildingLongitude == nil {
				p.FrontdoorBuildingLongitude = addr.Longitude
			}
		}
		if bp := resp.BuildingsGroupedByPurpose; bp != nil && len(bp.ResidentialOrBusinessPremises) > 0 {
			bldg := bp.ResidentialOrBusinessPremises[0]
			p.FrontdoorBuildingBuildYear = util.Int32Ptr(bldg.BuildYear)
			if bldg.Heating != nil {
				p.FrontdoorBuildingHeating = bldg.Heating
				p.FrontdoorBuildingHeatingFuel = []string{*bldg.Heating}
			}
		}
	}
	if jsonData, err := json.Marshal(data); err == nil {
		p.FrontdoorBuildingData = jsonData
	}
	return p
}

func mapAnnouncementParams(ann frontdoorpayload.Announcement, buildingID uuid.UUID) db.UpsertFrontdoorBuildingAnnouncementParams {
	return db.UpsertFrontdoorBuildingAnnouncementParams{
		FrontdoorBuildingAnnouncementExternalID:               util.Int32Ptr(ann.ID),
		FrontdoorBuildingAnnouncementFriendlyID:               ann.FriendlyID,
		FrontdoorBuildingAnnouncementUnpublishingTime:         ann.UnpublishingTime,
		FrontdoorBuildingAnnouncementAddressLine1:             ann.AddressLine1,
		FrontdoorBuildingAnnouncementAddressLine2:             ann.AddressLine2,
		FrontdoorBuildingAnnouncementLocation:                 ann.Location,
		FrontdoorBuildingAnnouncementSearchPrice:              ann.SearchPrice,
		FrontdoorBuildingAnnouncementNotifyPriceChanged:       ann.NotifyPriceChanged,
		FrontdoorBuildingAnnouncementPropertyType:             ann.PropertyType,
		FrontdoorBuildingAnnouncementPropertySubtype:          ann.PropertySubtype,
		FrontdoorBuildingAnnouncementConstructionFinishedYear: util.Int32Ptr(ann.ConstructionFinishedYear),
		FrontdoorBuildingAnnouncementMainImageUri:             ann.MainImageURI,
		FrontdoorBuildingAnnouncementHasOpenBidding:           ann.HasOpenBidding,
		FrontdoorBuildingAnnouncementRoomStructure:            ann.RoomStructure,
		FrontdoorBuildingAnnouncementArea:                     ann.Area,
		FrontdoorBuildingAnnouncementTotalArea:                ann.TotalArea,
		FrontdoorBuildingAnnouncementPricePerSquare:           ann.PricePerSquare,
		FrontdoorBuildingAnnouncementDaysOnMarket:             util.Int32Ptr(ann.DaysOnMarket),
		FrontdoorBuildingAnnouncementNewBuilding:              ann.NewBuilding,
		FrontdoorBuildingAnnouncementMainImageHidden:          ann.MainImageHidden,
		FrontdoorBuildingAnnouncementIsCompanyAnnouncement:    ann.IsCompanyAnnouncement,
		FrontdoorBuildingAnnouncementShowBiddingIndicators:    ann.ShowBiddingIndicators,
		FrontdoorBuildingAnnouncementPublished:                ann.Published,
		FrontdoorBuildingAnnouncementRentPeriod:               ann.RentPeriod,
		FrontdoorBuildingAnnouncementRentalUniqueNo:           util.Int32Ptr(ann.RentalUniqueNo),
		FrontdoorBuildingID:                                   &buildingID,
	}
}

//go:fix inline
func ptr[T any](value T) *T {
	return new(value)
}
