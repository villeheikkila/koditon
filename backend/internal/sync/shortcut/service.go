package shortcut

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	client "koditon/internal/clients/shortcut"
	"koditon/internal/db"
	"koditon/internal/platform/httpratelimit"
	"koditon/internal/platform/logging"
	shortcutpayload "koditon/internal/providers/shortcut"
	"koditon/internal/sync/sourcejson"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	docsBaseURL string,
	adBaseURL string,
	userAgent string,
	sitemapBase string,
	rateLimit httpratelimit.Config,
) *Service {
	queries := db.New(dbtx)
	tokenLoad := func(ctx context.Context) (*client.Tokens, error) {
		dbToken, err := queries.GetValidShortcutToken(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("no valid token found")
			}
			return nil, err
		}
		tokens := &client.Tokens{
			CUID:   dbToken.ShortcutTokenCuid,
			Token:  dbToken.ShortcutTokenToken,
			Loaded: dbToken.ShortcutTokenLoaded,
		}
		return tokens, nil
	}
	tokenStore := func(ctx context.Context, tokens *client.Tokens, expiresAt time.Time) error {
		_, err := queries.InsertShortcutToken(ctx, db.InsertShortcutTokenParams{
			ShortcutTokenCuid:      &tokens.CUID,
			ShortcutTokenToken:     &tokens.Token,
			ShortcutTokenLoaded:    &tokens.Loaded,
			ShortcutTokenExpiresAt: &expiresAt,
		})
		return err
	}
	shortcutClient := client.NewClient(
		logger,
		tokenLoad,
		tokenStore,
		baseURL,
		docsBaseURL,
		adBaseURL,
		userAgent,
		sitemapBase,
		rateLimit,
	)
	return &Service{
		client:  shortcutClient,
		queries: queries,
		logger:  logger.With("component", "shortcut"),
	}
}

func (s *Service) SyncSitemap(ctx context.Context) (buildingIDs []string, adIDs []string, err error) {
	logger := logging.With(s.logger, logging.Op("shortcut.sync_sitemap"))
	allEntries, fetchErr := s.client.GetSitemapEntries(ctx)
	if fetchErr != nil {
		return nil, nil, fmt.Errorf("fetch sitemap entries: %w", fetchErr)
	}
	var buildingEntries, listingEntries, rentalEntries []client.ShortcutSitemapEntry
	for _, entry := range allEntries {
		switch entry.Type {
		case client.SitemapURLTypeBuilding:
			buildingEntries = append(buildingEntries, entry)
		case client.SitemapURLTypeListing:
			listingEntries = append(listingEntries, entry)
		case client.SitemapURLTypeRental:
			rentalEntries = append(rentalEntries, entry)
		}
	}
	if len(buildingEntries) > 0 {
		seenBuildingIDs := make(map[int]struct{})
		validBuildingEntries := make([]client.ShortcutSitemapEntry, 0, len(buildingEntries))
		for _, entry := range buildingEntries {
			if _, seen := seenBuildingIDs[entry.ID]; seen {
				logger.DebugContext(ctx, "duplicate building id in sitemap, skipping", "building_id", entry.ID)
				continue
			}
			seenBuildingIDs[entry.ID] = struct{}{}
			validBuildingEntries = append(validBuildingEntries, entry)
		}
		params := mapBatchUpsertBuildingsFromSitemapParams(validBuildingEntries)
		buildings, upsertErr := s.queries.BatchUpsertShortcutBuildingsFromSitemap(ctx, params)
		if upsertErr != nil {
			return nil, nil, fmt.Errorf("batch upsert buildings: %w", upsertErr)
		}
		buildingIDs = make([]string, len(buildings))
		for i, building := range buildings {
			buildingIDs[i] = fmt.Sprintf("building:%s", building.ShortcutBuildingID.String())
		}
	}
	adEntries := append(listingEntries, rentalEntries...)
	if len(adEntries) > 0 {
		seenAdIDs := make(map[int]struct{})
		validEntries := make([]client.ShortcutSitemapEntry, 0, len(adEntries))
		validAdTypes := make([]shortcutpayload.AdType, 0, len(adEntries))
		for _, entry := range adEntries {
			if _, seen := seenAdIDs[entry.ID]; seen {
				logger.DebugContext(ctx, "duplicate ad id in sitemap, skipping", "ad_id", entry.ID)
				continue
			}
			seenAdIDs[entry.ID] = struct{}{}
			var adType shortcutpayload.AdType
			switch entry.Type {
			case client.SitemapURLTypeListing:
				adType = shortcutpayload.AdTypeListing
			case client.SitemapURLTypeRental:
				adType = shortcutpayload.AdTypeRental
			default:
				logger.WarnContext(ctx, "unknown ad type from sitemap, skipping", "ad_id", entry.ID, "type", entry.Type)
				continue
			}
			validEntries = append(validEntries, entry)
			validAdTypes = append(validAdTypes, adType)
		}
		if len(validEntries) > 0 {
			params := mapBatchUpsertAdsFromSitemapParams(validEntries, validAdTypes)
			ads, upsertErr := s.queries.BatchUpsertShortcutAdsFromSitemap(ctx, params)
			if upsertErr != nil {
				return nil, nil, fmt.Errorf("batch upsert ads: %w", upsertErr)
			}
			adIDs = make([]string, len(ads))
			for i, ad := range ads {
				adIDs[i] = fmt.Sprintf("ad:%d", ad.ShortcutAdID)
			}
		}
	}
	return buildingIDs, adIDs, nil
}

func (s *Service) SyncAd(ctx context.Context, adID int64) error {
	adData, err := s.client.GetAdByID(ctx, int(adID))
	if err != nil {
		return fmt.Errorf("fetch ad data (ad_id=%d): %w", adID, err)
	}
	payload, err := shortcutpayload.ValidateShortcutAdPayloadV1(adData, adID)
	if err != nil {
		return fmt.Errorf("validate ad data (ad_id=%d): %w", adID, err)
	}
	var shortcutBuildingID *uuid.UUID
	if payload.BuildingExternalID != nil {
		building, err := s.queries.GetShortcutBuildingByExternalID(ctx, payload.BuildingExternalID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("get linked building (ad_id=%d, building_external_id=%d): %w", adID, *payload.BuildingExternalID, err)
		}
		if err == nil {
			shortcutBuildingID = &building.ShortcutBuildingID
		}
	}
	existingAd, err := s.queries.GetShortcutAdByID(ctx, &adID)
	if err != nil {
		return fmt.Errorf("get existing ad (ad_id=%d): %w", adID, err)
	}
	params, err := mapUpsertAdParams(adID, existingAd.ShortcutAdUrl, string(payload.AdType), payload.Raw, payload.SchemaVersion, shortcutBuildingID)
	if err != nil {
		return fmt.Errorf("prepare ad data (ad_id=%d): %w", adID, err)
	}
	if _, err = s.queries.UpsertShortcutAd(ctx, params); err != nil {
		return fmt.Errorf("upsert ad data (ad_id=%d): %w", adID, err)
	}
	return nil
}

func (s *Service) BackfillAdDataHashes(ctx context.Context, limit int32) (sourcejson.BackfillResult, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.queries.ListShortcutAdsMissingDataHash(ctx, new(int64(limit)))
	if err != nil {
		return sourcejson.BackfillResult{}, fmt.Errorf("list shortcut ads missing data hash: %w", err)
	}
	result := sourcejson.BackfillResult{Scanned: len(rows)}
	if len(rows) > 0 {
		result.Batches = 1
	}
	for _, row := range rows {
		canonical, hash, err := sourcejson.CanonicalizeAndHash(row.ShortcutAdData)
		if err != nil {
			return result, fmt.Errorf("hash shortcut ad payload (ad_id=%d): %w", row.ShortcutAdID, err)
		}
		params := db.BackfillShortcutAdDataHashParams{ShortcutAdData: canonical, ShortcutAdDataHash: &hash, ShortcutAdDataHashAlgorithm: ptr(sourcejson.HashAlgorithmSHA256), ShortcutAdID: &row.ShortcutAdID}
		if err := s.queries.BackfillShortcutAdDataHash(ctx, params); err != nil {
			return result, fmt.Errorf("backfill shortcut ad data hash (ad_id=%d): %w", row.ShortcutAdID, err)
		}
		result.Updated++
	}
	return result, nil
}

func (s *Service) SyncBuilding(ctx context.Context, buildingID uuid.UUID) error {
	building, err := s.queries.GetShortcutBuildingByID(ctx, &buildingID)
	if err != nil {
		return fmt.Errorf("get building (building_id=%s): %w", buildingID, err)
	}
	if building.ShortcutBuildingPageNotFound != nil && *building.ShortcutBuildingPageNotFound {
		return nil
	}
	scrapedBuilding, listings, rentals, err := s.client.ScrapeBuildingPage(ctx, int(building.ShortcutBuildingExternalID), building.ShortcutBuildingUrl)
	if err != nil {
		if errors.Is(err, client.ErrScraperErrorPage) {
			if markErr := s.queries.MarkShortcutBuildingPageNotFound(ctx, &buildingID); markErr != nil {
				return fmt.Errorf("mark building page not found (building_id=%s): %w", buildingID, markErr)
			}
			return nil
		}
		if errors.Is(err, client.ErrScraperForbidden) {
			return fmt.Errorf("scraping forbidden (building_id=%s, url=%s): %w", buildingID, building.ShortcutBuildingUrl, err)
		}
		return fmt.Errorf("scrape building page (building_id=%s, url=%s): %w", buildingID, building.ShortcutBuildingUrl, err)
	}
	params := mapScrapedBuildingParams(int64(scrapedBuilding.ShortcutBuildingID), building.ShortcutBuildingUrl, scrapedBuilding)
	if _, err = s.queries.UpsertShortcutBuilding(ctx, params); err != nil {
		return fmt.Errorf("update building (building_id=%s): %w", buildingID, err)
	}
	var upsertErrors []error
	for _, listing := range listings {
		params := mapListingParams(buildingID, &listing)
		if _, err := s.queries.UpsertShortcutBuildingListing(ctx, params); err != nil {
			upsertErrors = append(upsertErrors, fmt.Errorf("upsert listing %d: %w", listing.Index, err))
		}
	}
	for _, rental := range rentals {
		params := mapRentalParams(buildingID, &rental)
		if _, err := s.queries.UpsertShortcutBuildingRental(ctx, params); err != nil {
			upsertErrors = append(upsertErrors, fmt.Errorf("upsert rental %d: %w", rental.Index, err))
		}
	}
	if err := s.queries.MarkShortcutBuildingProcessed(ctx, &buildingID); err != nil {
		return fmt.Errorf("mark building processed (building_id=%s): %w", buildingID, err)
	}
	if len(upsertErrors) > 0 && len(listings)+len(rentals) == len(upsertErrors) {
		return fmt.Errorf("all listing/rental upserts failed (building_id=%s): %w", buildingID, errors.Join(upsertErrors...))
	}
	return nil
}

func (s *Service) DescribeAd(ctx context.Context, adID int64) (string, error) {
	ad, err := s.queries.GetShortcutAdByID(ctx, &adID)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, 4)
	if title := extractAdTitle(ad.ShortcutAdData); title != "" {
		lines = append(lines, "title: "+title)
	}
	if ad.ShortcutBuildingID != nil {
		building, err := s.queries.GetShortcutBuildingByID(ctx, ad.ShortcutBuildingID)
		if err == nil {
			if addr := firstNonEmptyString(building.ShortcutBuildingAddress, building.ShortcutBuildingHousingCompany); addr != "" {
				lines = append(lines, "address: "+addr)
			}
		}
	}
	if ad.ShortcutAdUrl != "" {
		lines = append(lines, "url: "+ad.ShortcutAdUrl)
	}
	return strings.Join(uniqueStrings(lines), "\n"), nil
}

func (s *Service) DescribeBuilding(ctx context.Context, buildingID uuid.UUID) (string, error) {
	building, err := s.queries.GetShortcutBuildingByID(ctx, &buildingID)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, 4)
	if addr := firstNonEmptyString(building.ShortcutBuildingAddress, building.ShortcutBuildingHousingCompany); addr != "" {
		lines = append(lines, "address: "+addr)
	}
	if building.ShortcutBuildingUrl != "" {
		lines = append(lines, "url: "+building.ShortcutBuildingUrl)
	}
	if building.ShortcutBuildingExternalID > 0 {
		lines = append(lines, fmt.Sprintf("building_external_id: %d", building.ShortcutBuildingExternalID))
	}
	return strings.Join(uniqueStrings(lines), "\n"), nil
}

func extractAdTitle(data []byte) string {
	payload, err := shortcutpayload.DecodeAdRaw(data)
	if err != nil {
		return ""
	}
	candidates := []string{
		"address",
		"formattedAddress",
		"name",
		"title",
		"description",
	}
	for _, key := range candidates {
		if value := nestedString(map[string]any(payload), key); value != "" {
			return value
		}
	}
	return ""
}

func nestedString(root map[string]any, key string) string {
	if root == nil {
		return ""
	}
	if v, ok := root[key].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	for _, raw := range root {
		switch typed := raw.(type) {
		case map[string]any:
			if v := nestedString(typed, key); v != "" {
				return v
			}
		case []any:
			for _, item := range typed {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if v := nestedString(obj, key); v != "" {
					return v
				}
			}
		}
	}
	return ""
}

func firstNonEmptyString(values ...*string) string {
	for _, v := range values {
		if v == nil {
			continue
		}
		trimmed := strings.TrimSpace(*v)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
