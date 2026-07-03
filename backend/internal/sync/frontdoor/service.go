package frontdoor

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"

	client "koditon/internal/clients/frontdoor"
	"koditon/internal/db"
	"koditon/internal/platform/httpratelimit"
	frontdoorpayload "koditon/internal/providers/frontdoor"
	"koditon/internal/sync/sourcejson"
)

type Service struct {
	client  *client.Client
	queries *db.Queries
	logger  *slog.Logger
}

func NewService(
	dbtx db.DBTX,
	logger *slog.Logger,
	baseURL string,
	userAgent string,
	cookie string,
	sitemapBase string,
	rateLimit httpratelimit.Config,
) *Service {
	frontdoorClient := client.New(
		baseURL,
		userAgent,
		cookie,
		sitemapBase,
		rateLimit,
	)
	return &Service{
		client:  frontdoorClient,
		queries: db.New(dbtx),
		logger:  logger.With("component", "frontdoor"),
	}
}

func (s *Service) SyncSitemap(ctx context.Context) (adIDs []string, buildingIDs []string, err error) {
	entries, fetchErr := s.client.GetSitemapEntries(ctx)
	if fetchErr != nil {
		return nil, nil, fmt.Errorf("fetch sitemap entries: %w", fetchErr)
	}
	var adEntries []client.SitemapEntry
	var buildingEntries []client.SitemapEntry
	for _, entry := range entries {
		switch entry.Type {
		case client.EntryTypeAd:
			adEntries = append(adEntries, entry)
		case client.EntryTypeBuilding:
			buildingEntries = append(buildingEntries, entry)
		default:
		}
	}
	if len(adEntries) > 0 {
		params := mapBatchUpsertAdsFromSitemapParams(adEntries)
		ads, upsertErr := s.queries.BatchUpsertFrontdoorAdsFromSitemap(ctx, params)
		if upsertErr != nil {
			return nil, nil, fmt.Errorf("batch upsert ads: %w", upsertErr)
		}
		adIDs = make([]string, len(ads))
		for i, ad := range ads {
			adIDs[i] = fmt.Sprintf("ad:%s", ad.FrontdoorAdExternalID)
		}
	}
	if len(buildingEntries) > 0 {
		params := mapBatchUpsertBuildingsFromSitemapParams(buildingEntries)
		buildings, upsertErr := s.queries.BatchUpsertFrontdoorBuildingsFromSitemap(ctx, params)
		if upsertErr != nil {
			return nil, nil, fmt.Errorf("batch upsert buildings: %w", upsertErr)
		}
		buildingIDs = make([]string, len(buildings))
		for i, building := range buildings {
			buildingIDs[i] = fmt.Sprintf("building:%s", building.FrontdoorBuildingID.String())
		}
	}
	return adIDs, buildingIDs, nil
}

func (s *Service) SyncAd(ctx context.Context, friendlyID string) error {
	ad, err := s.client.GetAdByFriendlyID(ctx, friendlyID)
	if err != nil {
		if httpErr, ok := client.IsHTTPStatusError(err); ok && httpErr.IsNotFound() {
			if markErr := s.queries.MarkFrontdoorAdNotFoundByExternalID(ctx, &friendlyID); markErr != nil {
				return fmt.Errorf("mark ad not found (friendly_id=%s): %w", friendlyID, markErr)
			}
			return nil
		}
		return fmt.Errorf("fetch ad data (friendly_id=%s): %w", friendlyID, err)
	}
	params, err := mapAdParams(friendlyID, ad)
	if err != nil {
		return fmt.Errorf("prepare ad data (friendly_id=%s): %w", friendlyID, err)
	}
	if err := s.queries.UpdateFrontdoorAdData(ctx, params); err != nil {
		return fmt.Errorf("update ad data (friendly_id=%s): %w", friendlyID, err)
	}
	return nil
}

func (s *Service) BackfillAdDataHashes(ctx context.Context, limit int32) (sourcejson.BackfillResult, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.queries.ListFrontdoorAdsMissingDataHash(ctx, new(int64(limit)))
	if err != nil {
		return sourcejson.BackfillResult{}, fmt.Errorf("list frontdoor ads missing data hash: %w", err)
	}
	result := sourcejson.BackfillResult{Scanned: len(rows)}
	if len(rows) > 0 {
		result.Batches = 1
	}
	for _, row := range rows {
		canonical, hash, err := sourcejson.CanonicalizeAndHash(row.FrontdoorAdData)
		if err != nil {
			return result, fmt.Errorf("hash frontdoor ad payload (friendly_id=%s): %w", row.FrontdoorAdExternalID, err)
		}
		params := db.BackfillFrontdoorAdDataHashParams{FrontdoorAdData: canonical, FrontdoorAdDataHash: &hash, FrontdoorAdDataHashAlgorithm: ptr(sourcejson.HashAlgorithmSHA256), FrontdoorAdExternalID: &row.FrontdoorAdExternalID}
		if err := s.queries.BackfillFrontdoorAdDataHash(ctx, params); err != nil {
			return result, fmt.Errorf("backfill frontdoor ad data hash (friendly_id=%s): %w", row.FrontdoorAdExternalID, err)
		}
		result.Updated++
	}
	return result, nil
}

func (s *Service) SyncBuilding(ctx context.Context, externalID string) error {
	buildingURL, housingCompanyID, err := s.resolveBuildingURL(ctx, externalID)
	if err != nil {
		return err
	}
	buildingData, err := s.client.GetBuildingPageData(ctx, buildingURL)
	if err != nil {
		return fmt.Errorf("fetch building data (id=%s, url=%s): %w", externalID, buildingURL, err)
	}
	if housingCompanyID == 0 {
		housingCompanyID = extractHousingCompanyID(buildingData)
		if housingCompanyID == 0 {
			return fmt.Errorf("cannot determine housing company ID for building %s", externalID)
		}
	}
	if err := s.upsertBuildingData(ctx, housingCompanyID, buildingData); err != nil {
		return fmt.Errorf("upsert building data (housing_company_id=%d): %w", housingCompanyID, err)
	}
	announcements := extractAnnouncements(buildingData)
	if len(announcements) > 0 {
		if err := s.upsertBuildingAnnouncements(ctx, housingCompanyID, announcements); err != nil {
			return fmt.Errorf("upsert building announcements (housing_company_id=%d): %w", housingCompanyID, err)
		}
	}
	return nil
}

func (s *Service) resolveBuildingURL(ctx context.Context, externalID string) (url string, housingCompanyID int64, err error) {
	if id, parseErr := strconv.ParseInt(externalID, 10, 64); parseErr == nil {
		idPtr := new(id)
		u, lookupErr := s.queries.GetFrontdoorBuildingURLByHousingCompanyID(ctx, idPtr)
		if lookupErr != nil {
			return "", 0, fmt.Errorf("get building url (housing_company_id=%d): %w", id, lookupErr)
		}
		if u == nil {
			return "", 0, fmt.Errorf("building url is null (housing_company_id=%d)", id)
		}
		return *u, id, nil
	}
	buildingUUID, parseErr := uuid.Parse(externalID)
	if parseErr != nil {
		return "", 0, fmt.Errorf("invalid building identifier %q: not a housing company ID or UUID", externalID)
	}
	building, lookupErr := s.queries.GetFrontdoorBuildingByID(ctx, &buildingUUID)
	if lookupErr != nil {
		return "", 0, fmt.Errorf("lookup building by UUID %s: %w", externalID, lookupErr)
	}
	if building.FrontdoorBuildingUrl == nil {
		return "", 0, fmt.Errorf("building %s has no URL", externalID)
	}
	var hcID int64
	if building.FrontdoorBuildingHousingCompanyID != nil {
		hcID = *building.FrontdoorBuildingHousingCompanyID
	}
	return *building.FrontdoorBuildingUrl, hcID, nil
}

func extractHousingCompanyID(data *frontdoorpayload.HousingCompanyResponse) int64 {
	if data == nil || data.HousingCompanyPage == nil || data.HousingCompanyPage.Response == nil {
		return 0
	}
	hca := data.HousingCompanyPage.Response.HousingCompanyAnnouncement
	if hca == nil || hca.HousingCompany == nil || hca.HousingCompany.ID == nil {
		return 0
	}
	return int64(*hca.HousingCompany.ID)
}

func (s *Service) upsertBuildingData(ctx context.Context, housingCompanyID int64, buildingData *frontdoorpayload.HousingCompanyResponse) error {
	params := mapBuildingParams(housingCompanyID, buildingData)
	if err := s.queries.UpdateFrontdoorBuildingDetailsByHousingCompanyID(ctx, params); err != nil {
		return fmt.Errorf("update building details: %w", err)
	}
	return nil
}

func (s *Service) upsertBuildingAnnouncements(ctx context.Context, housingCompanyID int64, announcements []frontdoorpayload.Announcement) error {
	if len(announcements) == 0 {
		return nil
	}
	idPtr := new(housingCompanyID)
	buildingID, err := s.queries.GetFrontdoorBuildingIDByHousingCompanyID(ctx, idPtr)
	if err != nil {
		return fmt.Errorf("get building id: %w", err)
	}
	for _, ann := range announcements {
		params := mapAnnouncementParams(ann, buildingID)
		if _, err := s.queries.UpsertFrontdoorBuildingAnnouncement(ctx, params); err != nil {
			annID := 0
			if ann.ID != nil {
				annID = *ann.ID
			}
			return fmt.Errorf("upsert announcement (id=%d, friendly_id=%s): %w", annID, valueOrEmpty(ann.FriendlyID), err)
		}
	}
	return nil
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func extractAnnouncements(building *frontdoorpayload.HousingCompanyResponse) []frontdoorpayload.Announcement {
	if building == nil || building.KsaHousingCompanyPage == nil || building.KsaHousingCompanyPage.Response == nil {
		return nil
	}
	resp := building.KsaHousingCompanyPage.Response
	var announcements []frontdoorpayload.Announcement
	announcements = append(announcements, resp.UnpublishedAnnouncements...)
	announcements = append(announcements, resp.UnpublishedRentalAnnouncements...)
	announcements = append(announcements, resp.PublishedAnnouncements...)
	announcements = append(announcements, resp.PublishedRentalAnnouncements...)
	return filterUniqueAnnouncements(announcements)
}

func filterUniqueAnnouncements(announcements []frontdoorpayload.Announcement) []frontdoorpayload.Announcement {
	seen := make(map[string]bool)
	var unique []frontdoorpayload.Announcement
	for _, announcement := range announcements {
		id := int64(0)
		if announcement.ID != nil {
			id = int64(*announcement.ID)
		}
		unpublishingTime := int64(0)
		if announcement.UnpublishingTime != nil {
			unpublishingTime = int64(*announcement.UnpublishingTime)
		}
		searchPrice := int64(0)
		if announcement.SearchPrice != nil {
			searchPrice = int64(*announcement.SearchPrice)
		}
		key := fmt.Sprintf("%d_%d_%d", id, unpublishingTime, searchPrice)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, announcement)
		}
	}
	return unique
}
