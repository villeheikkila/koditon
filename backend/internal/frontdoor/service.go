package frontdoor

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"koditon-go/internal/frontdoor/client"
	"koditon-go/internal/frontdoor/db"

	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	client  *client.Client
	queries *db.Queries
}

func NewService(
	dbtx db.DBTX,
	baseURL string,
	userAgent string,
	cookie string,
	sitemapBase string,
) *Service {
	frontdoorClient := client.New(
		baseURL,
		userAgent,
		cookie,
		sitemapBase,
	)
	return &Service{
		client:  frontdoorClient,
		queries: db.New(dbtx),
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
		}
	}
	var upsertErrors []error
	if len(adEntries) > 0 {
		successCount := 0
		adIDs = make([]string, 0, len(adEntries))
		for _, entry := range adEntries {
			params := &db.UpsertFrontdoorAdFromSitemapParams{
				FrontdoorAdsExternalID: entry.ID,
				FrontdoorAdsUrl:        entry.URL.String(),
			}
			if _, upsertErr := s.queries.UpsertFrontdoorAdFromSitemap(ctx, params); upsertErr != nil {
				upsertErrors = append(upsertErrors, fmt.Errorf("upsert ad %s: %w", entry.ID, upsertErr))
				continue
			}
			entityID := fmt.Sprintf("ad:%s", entry.ID)
			adIDs = append(adIDs, entityID)
			successCount++
		}
		_ = successCount
	}
	if len(buildingEntries) > 0 {
		successCount := 0
		buildingIDs = make([]string, 0, len(buildingEntries))
		for _, entry := range buildingEntries {
			housingCompanyID, parseErr := strconv.ParseInt(entry.ID, 10, 64)
			if parseErr != nil {
				upsertErrors = append(upsertErrors, fmt.Errorf("parse housing company ID %s: %w", entry.ID, parseErr))
				continue
			}
			url := entry.URL.String()
			params := &db.UpsertFrontdoorBuildingParams{
				FrontdoorBuildingsUrl: &url,
				FrontdoorBuildingsHousingCompanyID:         pgtype.Int8{Int64: housingCompanyID, Valid: true},
				FrontdoorBuildingsHousingCompanyFriendlyID: &entry.ID,
			}
			if _, upsertErr := s.queries.UpsertFrontdoorBuilding(ctx, params); upsertErr != nil {
				upsertErrors = append(upsertErrors, fmt.Errorf("upsert building %d: %w", housingCompanyID, upsertErr))
				continue
			}
			entityID := fmt.Sprintf("building:%d", housingCompanyID)
			buildingIDs = append(buildingIDs, entityID)
			successCount++
		}
		_ = successCount
	}
	if len(adIDs) == 0 && len(buildingIDs) == 0 && len(upsertErrors) > 0 {
		return nil, nil, fmt.Errorf("all upserts failed: %w", errors.Join(upsertErrors...))
	}
	return adIDs, buildingIDs, nil
}

func (s *Service) SyncAd(ctx context.Context, friendlyID string) error {
	ad, err := s.client.GetAdByFriendlyID(ctx, friendlyID)
	if err != nil {
		if httpErr, ok := client.IsHTTPStatusError(err); ok && httpErr.IsNotFound() {
			if markErr := s.queries.MarkFrontdoorAdNotFoundByExternalID(ctx, friendlyID); markErr != nil {
				return fmt.Errorf("mark ad not found (friendly_id=%s): %w", friendlyID, markErr)
			}
			return nil
		}
		return fmt.Errorf("fetch ad data (friendly_id=%s): %w", friendlyID, err)
	}
	if err := s.queries.UpdateFrontdoorAdData(ctx, mapAdParams(friendlyID, ad)); err != nil {
		return fmt.Errorf("update ad data (friendly_id=%s): %w", friendlyID, err)
	}
	return nil
}

func (s *Service) SyncBuilding(ctx context.Context, externalID string) error {
	housingCompanyID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid housing company ID %q: %w", externalID, err)
	}
	housingCompanyIDPg := pgtype.Int8{Int64: housingCompanyID, Valid: true}
	buildingURL, err := s.queries.GetFrontdoorBuildingURLByHousingCompanyID(ctx, housingCompanyIDPg)
	if err != nil {
		return fmt.Errorf("get building url (housing_company_id=%d): %w", housingCompanyID, err)
	}
	if buildingURL == nil {
		return fmt.Errorf("building url is null (housing_company_id=%d)", housingCompanyID)
	}
	buildingData, err := s.client.GetBuildingPageData(ctx, *buildingURL)
	if err != nil {
		return fmt.Errorf("fetch building data (housing_company_id=%d, url=%s): %w", housingCompanyID, *buildingURL, err)
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

func (s *Service) upsertBuildingData(ctx context.Context, housingCompanyID int64, buildingData *client.HousingCompanyResponse) error {
	params := mapBuildingParams(housingCompanyID, buildingData)
	if err := s.queries.UpdateFrontdoorBuildingDetailsByHousingCompanyID(ctx, params); err != nil {
		return fmt.Errorf("update building details: %w", err)
	}
	return nil
}

func (s *Service) upsertBuildingAnnouncements(ctx context.Context, housingCompanyID int64, announcements []client.Announcement) error {
	if len(announcements) == 0 {
		return nil
	}
	housingCompanyIDPg := pgtype.Int8{Int64: housingCompanyID, Valid: true}
	buildingID, err := s.queries.GetFrontdoorBuildingIDByHousingCompanyID(ctx, housingCompanyIDPg)
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

func extractAnnouncements(building *client.HousingCompanyResponse) []client.Announcement {
	if building == nil || building.KsaHousingCompanyPage == nil || building.KsaHousingCompanyPage.Response == nil {
		return nil
	}
	resp := building.KsaHousingCompanyPage.Response
	var announcements []client.Announcement
	announcements = append(announcements, resp.UnpublishedAnnouncements...)
	announcements = append(announcements, resp.UnpublishedRentalAnnouncements...)
	announcements = append(announcements, resp.PublishedAnnouncements...)
	announcements = append(announcements, resp.PublishedRentalAnnouncements...)
	return filterUniqueAnnouncements(announcements)
}

func filterUniqueAnnouncements(announcements []client.Announcement) []client.Announcement {
	seen := make(map[string]bool)
	var unique []client.Announcement
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
