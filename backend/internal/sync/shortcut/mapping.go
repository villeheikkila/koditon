package shortcut

import (
	client "koditon/internal/clients/shortcut"
	"koditon/internal/db"
	"koditon/internal/platform/util"

	"github.com/google/uuid"
)

func mapBatchUpsertBuildingsFromSitemapParams(entries []client.ShortcutSitemapEntry) db.BatchUpsertShortcutBuildingsFromSitemapParams {
	externalIDs := make([]int64, len(entries))
	urls := make([]string, len(entries))
	for i, entry := range entries {
		externalIDs[i] = int64(entry.ID)
		urls[i] = entry.URL.String()
	}
	return db.BatchUpsertShortcutBuildingsFromSitemapParams{
		Column1: externalIDs,
		Column2: urls,
	}
}

func mapBatchUpsertAdsFromSitemapParams(entries []client.ShortcutSitemapEntry, adTypes []AdType) db.BatchUpsertShortcutAdsFromSitemapParams {
	ids := make([]int64, len(entries))
	urls := make([]string, len(entries))
	types := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = int64(entry.ID)
		urls[i] = entry.URL.String()
		types[i] = string(adTypes[i])
	}
	return db.BatchUpsertShortcutAdsFromSitemapParams{
		Column1: ids,
		Column2: urls,
		Column3: types,
	}
}

func mapUpsertAdParams(adID int64, url string, adType string, data []byte, schemaVersion int16, shortcutBuildingID *uuid.UUID) db.UpsertShortcutAdParams {
	return db.UpsertShortcutAdParams{
		ShortcutAdID:                adID,
		ShortcutAdUrl:               url,
		ShortcutAdType:              adType,
		ShortcutAdData:              data,
		ShortcutAdDataSchemaVersion: schemaVersion,
		ShortcutBuildingID:          shortcutBuildingID,
	}
}

func mapScrapedBuildingParams(shortcutBuildingID int64, url string, scraped *client.ScrapedBuilding) db.UpsertShortcutBuildingParams {
	return db.UpsertShortcutBuildingParams{
		ShortcutBuildingExternalID:              shortcutBuildingID,
		ShortcutBuildingBuildingID:              scraped.BuildingID,
		ShortcutBuildingBuildingType:            scraped.BuildingType,
		ShortcutBuildingBuildingSubtype:         scraped.BuildingSubtype,
		ShortcutBuildingConstructionYear:        util.Int32Ptr(scraped.ConstructionYear),
		ShortcutBuildingFloorCount:              util.Int32Ptr(scraped.FloorCount),
		ShortcutBuildingApartmentCount:          util.Int32Ptr(scraped.ApartmentCount),
		ShortcutBuildingHeatingSystem:           scraped.HeatingSystem,
		ShortcutBuildingBuildingMaterial:        scraped.BuildingMaterial,
		ShortcutBuildingPlotType:                scraped.PlotType,
		ShortcutBuildingWallStructure:           scraped.WallStructure,
		ShortcutBuildingHeatSource:              scraped.HeatSource,
		ShortcutBuildingHasElevator:             scraped.HasElevator,
		ShortcutBuildingHasSauna:                scraped.HasSauna,
		ShortcutBuildingLatitude:                scraped.Latitude,
		ShortcutBuildingLongitude:               scraped.Longitude,
		ShortcutBuildingAdditionalAddresses:     scraped.AdditionalAddresses,
		ShortcutBuildingUrl:                     url,
		ShortcutBuildingAddress:                 &scraped.Address,
		ShortcutBuildingFrameConstructionMethod: scraped.FrameConstructionMethod,
		ShortcutBuildingHousingCompany:          scraped.HousingCompany,
	}
}

func mapListingParams(buildingID uuid.UUID, listing *client.BuildingListing) db.UpsertShortcutBuildingListingParams {
	return db.UpsertShortcutBuildingListingParams{
		ShortcutBuildingID:                   buildingID,
		ShortcutBuildingListingLayout:        listing.Layout,
		ShortcutBuildingListingSize:          listing.Size,
		ShortcutBuildingListingPrice:         listing.Price,
		ShortcutBuildingListingPricePerSqm:   listing.PricePerSqm,
		ShortcutBuildingListingMarketingTime: listing.MarketingTime,
		ShortcutBuildingListingIdx:           util.Int32Ptr(&listing.Index),
	}
}

func mapRentalParams(buildingID uuid.UUID, rental *client.RentalListing) db.UpsertShortcutBuildingRentalParams {
	return db.UpsertShortcutBuildingRentalParams{
		ShortcutBuildingID:                  buildingID,
		ShortcutBuildingRentalLayout:        rental.Layout,
		ShortcutBuildingRentalSize:          rental.Size,
		ShortcutBuildingRentalPrice:         rental.Price,
		ShortcutBuildingRentalMarketingTime: rental.MarketingTime,
		ShortcutBuildingRentalIdx:           util.Int32Ptr(&rental.Index),
	}
}
