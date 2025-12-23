package shortcut

import (
	"koditon-go/internal/shortcut/client"
	"koditon-go/internal/shortcut/db"
	"koditon-go/internal/util"

	"github.com/jackc/pgx/v5/pgtype"
)

func mapUpsertBuildingFromSitemapParams(entry client.ShortcutSitemapEntry) *db.UpsertShortcutBuildingFromSitemapParams {
	return &db.UpsertShortcutBuildingFromSitemapParams{
		ShortcutBuildingsExternalID: int64(entry.ID),
		ShortcutBuildingsUrl:        entry.URL.String(),
	}
}

func mapUpsertAdParams(adID int64, url string, adType string, data []byte, shortcutBuildingID pgtype.UUID) *db.UpsertShortcutAdParams {
	return &db.UpsertShortcutAdParams{
		ShortcutAdsID:         adID,
		ShortcutAdsUrl:        url,
		ShortcutAdsType:       adType,
		ShortcutAdsData:       data,
		ShortcutAdsBuildingID: shortcutBuildingID,
	}
}

func mapScrapedBuildingParams(shortcutBuildingID int64, url string, scraped *client.ScrapedBuilding) *db.UpsertShortcutBuildingParams {
	return &db.UpsertShortcutBuildingParams{
		ShortcutBuildingsExternalID:              shortcutBuildingID,
		ShortcutBuildingsBuildingID:              scraped.BuildingID,
		ShortcutBuildingsBuildingType:            scraped.BuildingType,
		ShortcutBuildingsBuildingSubtype:         scraped.BuildingSubtype,
		ShortcutBuildingsConstructionYear:        util.ToInt4(scraped.ConstructionYear),
		ShortcutBuildingsFloorCount:              util.ToInt4(scraped.FloorCount),
		ShortcutBuildingsApartmentCount:          util.ToInt4(scraped.ApartmentCount),
		ShortcutBuildingsHeatingSystem:           scraped.HeatingSystem,
		ShortcutBuildingsBuildingMaterial:        scraped.BuildingMaterial,
		ShortcutBuildingsPlotType:                scraped.PlotType,
		ShortcutBuildingsWallStructure:           scraped.WallStructure,
		ShortcutBuildingsHeatSource:              scraped.HeatSource,
		ShortcutBuildingsHasElevator:             scraped.HasElevator,
		ShortcutBuildingsHasSauna:                scraped.HasSauna,
		ShortcutBuildingsLatitude:                util.ToFloat8(scraped.Latitude),
		ShortcutBuildingsLongitude:               util.ToFloat8(scraped.Longitude),
		ShortcutBuildingsAdditionalAddresses:     scraped.AdditionalAddresses,
		ShortcutBuildingsUrl:                     url,
		ShortcutBuildingsAddress:                 &scraped.Address,
		ShortcutBuildingsFrameConstructionMethod: scraped.FrameConstructionMethod,
		ShortcutBuildingsHousingCompany:          scraped.HousingCompany,
	}
}

func mapListingParams(buildingID pgtype.UUID, listing *client.BuildingListing) *db.UpsertShortcutBuildingListingParams {
	return &db.UpsertShortcutBuildingListingParams{
		ShortcutBuildingListingsBuildingID:    buildingID,
		ShortcutBuildingListingsLayout:        listing.Layout,
		ShortcutBuildingListingsSize:          util.ToFloat8(listing.Size),
		ShortcutBuildingListingsPrice:         util.ToFloat8(listing.Price),
		ShortcutBuildingListingsPricePerSqm:   util.ToFloat8(listing.PricePerSqm),
		ShortcutBuildingListingsMarketingTime: listing.MarketingTime,
		ShortcutBuildingListingsIdx:           util.ToInt4(&listing.Index),
	}
}

func mapRentalParams(buildingID pgtype.UUID, rental *client.RentalListing) *db.UpsertShortcutBuildingRentalParams {
	return &db.UpsertShortcutBuildingRentalParams{
		ShortcutBuildingRentalsBuildingID:    buildingID,
		ShortcutBuildingRentalsLayout:        rental.Layout,
		ShortcutBuildingRentalsSize:          util.ToFloat8(rental.Size),
		ShortcutBuildingRentalsPrice:         util.ToFloat8(rental.Price),
		ShortcutBuildingRentalsMarketingTime: rental.MarketingTime,
		ShortcutBuildingRentalsIdx:           util.ToInt4(&rental.Index),
	}
}
