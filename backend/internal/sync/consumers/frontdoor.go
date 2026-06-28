package consumers

const (
	TaskTypeFrontdoorSitemapSync          = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorBuildingsSitemapSync = "frontdoor_buildings_sitemap_sync"
	TaskTypeFrontdoorSync                 = "frontdoor_sync"
	TaskTypeFrontdoorAdDataHashBackfill   = "frontdoor_ad_data_hash_backfill"
)

type sourceAdDataHashBackfillPayload struct {
	Limit int32 `json:"limit,omitempty"`
}
