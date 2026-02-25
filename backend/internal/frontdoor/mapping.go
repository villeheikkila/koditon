package frontdoor

import (
	"encoding/json"

	"koditon-go/internal/db"
	"koditon-go/internal/frontdoor/client"
	"koditon-go/internal/util"

	"github.com/jackc/pgx/v5/pgtype"
)

func mapBatchUpsertAdsFromSitemapParams(entries []client.SitemapEntry) *db.BatchUpsertFrontdoorAdsFromSitemapParams {
	externalIDs := make([]string, len(entries))
	urls := make([]string, len(entries))
	for i, entry := range entries {
		externalIDs[i] = entry.ID
		urls[i] = entry.URL.String()
	}
	return &db.BatchUpsertFrontdoorAdsFromSitemapParams{
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

func mapAdParams(friendlyID string, ad *client.AdResponse) *db.UpdateFrontdoorAdDataParams {
	params := &db.UpdateFrontdoorAdDataParams{
		FrontdoorAdExternalID: friendlyID,
	}
	if jsonData, err := json.Marshal(ad); err == nil {
		params.Column2 = jsonData
	}
	return params
}

func mapBuildingParams(housingCompanyID int64, data *client.HousingCompanyResponse) *db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams {
	p := &db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams{
		FrontdoorBuildingHousingCompanyID: util.Int64Ptr(housingCompanyID),
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
					p.FrontdoorBuildingLatitude = util.ToFloat8(gc.Latitude)
					p.FrontdoorBuildingLongitude = util.ToFloat8(gc.Longitude)
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
			p.FrontdoorBuildingHasElevator = util.ToBoolean(info.HasElevator)
			p.FrontdoorBuildingHasSauna = util.ToBoolean(info.HasSauna)
			p.FrontdoorBuildingEnergyCertificateCode = info.EnergyCertificateCode
			p.FrontdoorBuildingPlotHoldingType = info.PlotHoldingType
			p.FrontdoorBuildingOuterRoofMaterial = info.OuterRoofMaterial
			p.FrontdoorBuildingOuterRoofType = info.OuterRoofType
			p.FrontdoorBuildingCarStorageDescription = info.CarStorageDescription
			if r := info.ClassifiedPastRenovationsDetected; r != nil {
				p.FrontdoorBuildingElevatorRenovated = util.ToBoolean(r.ElevatorRenovated)
				p.FrontdoorBuildingElevatorRenovatedYear = util.Int32Ptr(r.ElevatorRenovatedYear)
				p.FrontdoorBuildingFacadeRenovated = util.ToBoolean(r.FacadeRenovated)
				p.FrontdoorBuildingFacadeRenovatedYear = util.Int32Ptr(r.FacadeRenovatedYear)
				p.FrontdoorBuildingWindowRenovated = util.ToBoolean(r.WindowRenovated)
				p.FrontdoorBuildingWindowRenovatedYear = util.Int32Ptr(r.WindowRenovatedYear)
				p.FrontdoorBuildingRoofRenovated = util.ToBoolean(r.RoofRenovated)
				p.FrontdoorBuildingRoofRenovatedYear = util.Int32Ptr(r.RoofRenovatedYear)
				p.FrontdoorBuildingPipeRenovated = util.ToBoolean(r.PipeRenovated)
				p.FrontdoorBuildingPipeRenovatedYear = util.Int32Ptr(r.PipeRenovatedYear)
				p.FrontdoorBuildingBalconyRenovated = util.ToBoolean(r.BalconyRenovated)
				p.FrontdoorBuildingBalconyRenovatedYear = util.Int32Ptr(r.BalconyRenovatedYear)
				p.FrontdoorBuildingElectricityRenovated = util.ToBoolean(r.ElectricityRenovated)
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
			if !p.FrontdoorBuildingLatitude.Valid { //nolint:govet
				p.FrontdoorBuildingLatitude = util.ToFloat8(addr.Latitude)
			}
			if !p.FrontdoorBuildingLongitude.Valid {
				p.FrontdoorBuildingLongitude = util.ToFloat8(addr.Longitude)
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
		p.Column44 = jsonData
	}
	return p
}

func mapAnnouncementParams(ann client.Announcement, buildingID pgtype.UUID) *db.UpsertFrontdoorBuildingAnnouncementParams {
	return &db.UpsertFrontdoorBuildingAnnouncementParams{
		FrontdoorBuildingAnnouncementExternalID:               util.Int32Ptr(ann.ID),
		FrontdoorBuildingAnnouncementFriendlyID:               ann.FriendlyID,
		FrontdoorBuildingAnnouncementUnpublishingTime:         util.ToFloat8(ann.UnpublishingTime),
		FrontdoorBuildingAnnouncementAddressLine1:             ann.AddressLine1,
		FrontdoorBuildingAnnouncementAddressLine2:             ann.AddressLine2,
		FrontdoorBuildingAnnouncementLocation:                 ann.Location,
		FrontdoorBuildingAnnouncementSearchPrice:              util.ToFloat8(ann.SearchPrice),
		FrontdoorBuildingAnnouncementNotifyPriceChanged:       util.ToBoolean(ann.NotifyPriceChanged),
		FrontdoorBuildingAnnouncementPropertyType:             ann.PropertyType,
		FrontdoorBuildingAnnouncementPropertySubtype:          ann.PropertySubtype,
		FrontdoorBuildingAnnouncementConstructionFinishedYear: util.Int32Ptr(ann.ConstructionFinishedYear),
		FrontdoorBuildingAnnouncementMainImageUri:             ann.MainImageURI,
		FrontdoorBuildingAnnouncementHasOpenBidding:           util.ToBoolean(ann.HasOpenBidding),
		FrontdoorBuildingAnnouncementRoomStructure:            ann.RoomStructure,
		FrontdoorBuildingAnnouncementArea:                     util.ToFloat8(ann.Area),
		FrontdoorBuildingAnnouncementTotalArea:                util.ToFloat8(ann.TotalArea),
		FrontdoorBuildingAnnouncementPricePerSquare:           util.ToFloat8(ann.PricePerSquare),
		FrontdoorBuildingAnnouncementDaysOnMarket:             util.Int32Ptr(ann.DaysOnMarket),
		FrontdoorBuildingAnnouncementNewBuilding:              util.ToBoolean(ann.NewBuilding),
		FrontdoorBuildingAnnouncementMainImageHidden:          util.ToBoolean(ann.MainImageHidden),
		FrontdoorBuildingAnnouncementIsCompanyAnnouncement:    util.ToBoolean(ann.IsCompanyAnnouncement),
		FrontdoorBuildingAnnouncementShowBiddingIndicators:    util.ToBoolean(ann.ShowBiddingIndicators),
		FrontdoorBuildingAnnouncementPublished:                util.ToBoolean(ann.Published),
		FrontdoorBuildingAnnouncementRentPeriod:               ann.RentPeriod,
		FrontdoorBuildingAnnouncementRentalUniqueNo:           util.Int32Ptr(ann.RentalUniqueNo),
		FrontdoorBuildingID:                                   buildingID,
	}
}
