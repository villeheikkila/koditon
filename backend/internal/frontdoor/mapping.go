package frontdoor

import (
	"encoding/json"

	"koditon-go/internal/frontdoor/client"
	"koditon-go/internal/frontdoor/db"
	"koditon-go/internal/util"

	"github.com/jackc/pgx/v5/pgtype"
)

func mapAdParams(friendlyID string, ad *client.AdResponse) *db.UpdateFrontdoorAdDataParams {
	params := &db.UpdateFrontdoorAdDataParams{
		FrontdoorAdsExternalID: friendlyID,
	}
	if jsonData, err := json.Marshal(ad); err == nil {
		params.Column2 = jsonData
	}
	return params
}

func mapBuildingParams(housingCompanyID int64, data *client.HousingCompanyResponse) *db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams {
	p := &db.UpdateFrontdoorBuildingDetailsByHousingCompanyIDParams{
		FrontdoorBuildingsHousingCompanyID: util.ToInt8(housingCompanyID),
	}
	if page := data.HousingCompanyPage; page != nil && page.Response != nil {
		if hca := page.Response.HousingCompanyAnnouncement; hca != nil {
			p.FrontdoorBuildingsDescription = hca.Text
			if ci := hca.ContactInfo; ci != nil {
				p.FrontdoorBuildingsContactPhone = ci.Phone
				p.FrontdoorBuildingsContactOfficeName = ci.OfficeName
				p.FrontdoorBuildingsContactOfficeID = util.ToInt4(ci.OfficeID)
			}
			if hc := hca.HousingCompany; hc != nil {
				p.FrontdoorBuildingsCompanyName = hc.Name
				p.FrontdoorBuildingsApartmentCount = util.ToInt4(hc.ApartmentCount)
				p.FrontdoorBuildingsFloorCount = util.FloatToInt4(hc.FloorCount)
				p.FrontdoorBuildingsConstructionEndYear = util.ToInt4(hc.ConstructionEndYear)
				p.FrontdoorBuildingsOtherInfo = hc.OtherInfo
				p.FrontdoorBuildingsHouseNumber = hc.HouseNumber
				if hc.PostCode != nil {
					p.FrontdoorBuildingsPostArea = hc.PostCode.PostArea
				}
				if hc.Municipality != nil {
					p.FrontdoorBuildingsMunicipality = hc.Municipality.DefaultName
				}
				if hc.District != nil {
					p.FrontdoorBuildingsDistrict = hc.District.DefaultName
				}
				if gc := hc.GeoCode; gc != nil {
					p.FrontdoorBuildingsLatitude = util.ToFloat8(gc.Latitude)
					p.FrontdoorBuildingsLongitude = util.ToFloat8(gc.Longitude)
				}
			}
		}
	}
	if ksa := data.KsaHousingCompanyPage; ksa != nil && ksa.Response != nil {
		resp := ksa.Response
		p.FrontdoorBuildingsBusinessID = resp.BusinessID
		if p.FrontdoorBuildingsCompanyName == nil {
			p.FrontdoorBuildingsCompanyName = resp.CompanyName
		}
		if info := resp.AdHousingCompanyInfo; info != nil {
			if !p.FrontdoorBuildingsApartmentCount.Valid {
				p.FrontdoorBuildingsApartmentCount = util.ToInt4(info.ApartmentCount)
			}
			if !p.FrontdoorBuildingsFloorCount.Valid {
				p.FrontdoorBuildingsFloorCount = util.FloatToInt4(info.FloorCount)
			}
			p.FrontdoorBuildingsHasElevator = util.ToBoolean(info.HasElevator)
			p.FrontdoorBuildingsHasSauna = util.ToBoolean(info.HasSauna)
			p.FrontdoorBuildingsEnergyCertificateCode = info.EnergyCertificateCode
			p.FrontdoorBuildingsPlotHoldingType = info.PlotHoldingType
			p.FrontdoorBuildingsOuterRoofMaterial = info.OuterRoofMaterial
			p.FrontdoorBuildingsOuterRoofType = info.OuterRoofType
			p.FrontdoorBuildingsCarStorageDescription = info.CarStorageDescription
			if r := info.ClassifiedPastRenovationsDetected; r != nil {
				p.FrontdoorBuildingsElevatorRenovated = util.ToBoolean(r.ElevatorRenovated)
				p.FrontdoorBuildingsElevatorRenovatedYear = util.ToInt4(r.ElevatorRenovatedYear)
				p.FrontdoorBuildingsFacadeRenovated = util.ToBoolean(r.FacadeRenovated)
				p.FrontdoorBuildingsFacadeRenovatedYear = util.ToInt4(r.FacadeRenovatedYear)
				p.FrontdoorBuildingsWindowRenovated = util.ToBoolean(r.WindowRenovated)
				p.FrontdoorBuildingsWindowRenovatedYear = util.ToInt4(r.WindowRenovatedYear)
				p.FrontdoorBuildingsRoofRenovated = util.ToBoolean(r.RoofRenovated)
				p.FrontdoorBuildingsRoofRenovatedYear = util.ToInt4(r.RoofRenovatedYear)
				p.FrontdoorBuildingsPipeRenovated = util.ToBoolean(r.PipeRenovated)
				p.FrontdoorBuildingsPipeRenovatedYear = util.ToInt4(r.PipeRenovatedYear)
				p.FrontdoorBuildingsBalconyRenovated = util.ToBoolean(r.BalconyRenovated)
				p.FrontdoorBuildingsBalconyRenovatedYear = util.ToInt4(r.BalconyRenovatedYear)
				p.FrontdoorBuildingsElectricityRenovated = util.ToBoolean(r.ElectricityRenovated)
				p.FrontdoorBuildingsElectricityRenovatedYear = util.ToInt4(r.ElectricityRenovatedYear)
			}
		}
		if len(resp.HouseAddresses) > 0 {
			addr := resp.HouseAddresses[0]
			p.FrontdoorBuildingsStreetAddress = addr.StreetAddress
			p.FrontdoorBuildingsPostcode = addr.Postcode
			if p.FrontdoorBuildingsMunicipality == nil {
				p.FrontdoorBuildingsMunicipality = addr.Municipality
			}
			if p.FrontdoorBuildingsDistrict == nil {
				p.FrontdoorBuildingsDistrict = addr.District
			}
			if !p.FrontdoorBuildingsLatitude.Valid {
				p.FrontdoorBuildingsLatitude = util.ToFloat8(addr.Latitude)
			}
			if !p.FrontdoorBuildingsLongitude.Valid {
				p.FrontdoorBuildingsLongitude = util.ToFloat8(addr.Longitude)
			}
		}
		if bp := resp.BuildingsGroupedByPurpose; bp != nil && len(bp.ResidentialOrBusinessPremises) > 0 {
			bldg := bp.ResidentialOrBusinessPremises[0]
			p.FrontdoorBuildingsBuildYear = util.ToInt4(bldg.BuildYear)
			if bldg.Heating != nil {
				p.FrontdoorBuildingsHeating = bldg.Heating
				p.FrontdoorBuildingsHeatingFuel = []string{*bldg.Heating}
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
		FrontdoorBuildingAnnouncementsExternalID:               util.ToInt4(ann.ID),
		FrontdoorBuildingAnnouncementsFriendlyID:               ann.FriendlyID,
		FrontdoorBuildingAnnouncementsUnpublishingTime:         util.ToFloat8(ann.UnpublishingTime),
		FrontdoorBuildingAnnouncementsAddressLine1:             ann.AddressLine1,
		FrontdoorBuildingAnnouncementsAddressLine2:             ann.AddressLine2,
		FrontdoorBuildingAnnouncementsLocation:                 ann.Location,
		FrontdoorBuildingAnnouncementsSearchPrice:              util.ToFloat8(ann.SearchPrice),
		FrontdoorBuildingAnnouncementsNotifyPriceChanged:       util.ToBoolean(ann.NotifyPriceChanged),
		FrontdoorBuildingAnnouncementsPropertyType:             ann.PropertyType,
		FrontdoorBuildingAnnouncementsPropertySubtype:          ann.PropertySubtype,
		FrontdoorBuildingAnnouncementsConstructionFinishedYear: util.ToInt4(ann.ConstructionFinishedYear),
		FrontdoorBuildingAnnouncementsMainImageUri:             ann.MainImageURI,
		FrontdoorBuildingAnnouncementsHasOpenBidding:           util.ToBoolean(ann.HasOpenBidding),
		FrontdoorBuildingAnnouncementsRoomStructure:            ann.RoomStructure,
		FrontdoorBuildingAnnouncementsArea:                     util.ToFloat8(ann.Area),
		FrontdoorBuildingAnnouncementsTotalArea:                util.ToFloat8(ann.TotalArea),
		FrontdoorBuildingAnnouncementsPricePerSquare:           util.ToFloat8(ann.PricePerSquare),
		FrontdoorBuildingAnnouncementsDaysOnMarket:             util.ToInt4(ann.DaysOnMarket),
		FrontdoorBuildingAnnouncementsNewBuilding:              util.ToBoolean(ann.NewBuilding),
		FrontdoorBuildingAnnouncementsMainImageHidden:          util.ToBoolean(ann.MainImageHidden),
		FrontdoorBuildingAnnouncementsIsCompanyAnnouncement:    util.ToBoolean(ann.IsCompanyAnnouncement),
		FrontdoorBuildingAnnouncementsShowBiddingIndicators:    util.ToBoolean(ann.ShowBiddingIndicators),
		FrontdoorBuildingAnnouncementsPublished:                util.ToBoolean(ann.Published),
		FrontdoorBuildingAnnouncementsRentPeriod:               ann.RentPeriod,
		FrontdoorBuildingAnnouncementsRentalUniqueNo:           util.ToInt4(ann.RentalUniqueNo),
		FrontdoorBuildingAnnouncementsBuildingID:               buildingID,
	}
}
