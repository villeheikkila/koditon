package shortcut

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"koditon-go/internal/shortcut/client"
	"koditon-go/internal/shortcut/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
) *Service {
	queries := db.New(dbtx)
	// Token management: We store tokens with a long expiry (1 year) and rely on the API
	// returning 401 to trigger token refresh. This allows tokens to be reused as long as
	// they're valid according to the API, rather than trying to predict expiry times.
	tokenLoad := func(ctx context.Context) (*client.Tokens, error) {
		dbToken, err := queries.GetValidShortcutToken(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("no valid token found")
			}
			return nil, err
		}
		tokens := &client.Tokens{
			CUID:   dbToken.ShortcutTokensCuid,
			Token:  dbToken.ShortcutTokensToken,
			Loaded: dbToken.ShortcutTokensLoaded,
		}
		return tokens, nil
	}
	tokenStore := func(ctx context.Context, tokens *client.Tokens, expiresAt time.Time) error {
		_, err := queries.InsertShortcutToken(ctx, &db.InsertShortcutTokenParams{
			ShortcutTokensCuid:      tokens.CUID,
			ShortcutTokensToken:     tokens.Token,
			ShortcutTokensLoaded:    tokens.Loaded,
			ShortcutTokensExpiresAt: expiresAt,
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
	)
	return &Service{
		client:  shortcutClient,
		queries: queries,
		logger:  logger.With("component", "shortcut"),
	}
}

func (s *Service) SyncSitemap(ctx context.Context) (buildingIDs []string, adIDs []string, err error) {
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
	adEntries := append(listingEntries, rentalEntries...)
	var upsertErrors []error
	if len(buildingEntries) > 0 {
		buildingIDs = make([]string, 0, len(buildingEntries))
		for _, entry := range buildingEntries {
			params := mapUpsertBuildingFromSitemapParams(entry)
			building, upsertErr := s.queries.UpsertShortcutBuildingFromSitemap(ctx, params)
			if upsertErr != nil {
				upsertErrors = append(upsertErrors, fmt.Errorf("upsert building %d: %w", entry.ID, upsertErr))
				continue
			}
			entityID := fmt.Sprintf("building:%s", building.ShortcutBuildingsID.String())
			buildingIDs = append(buildingIDs, entityID)
		}
	}
	if len(adEntries) > 0 {
		adIDs = make([]string, 0, len(adEntries))
		for _, entry := range adEntries {
			adID := int64(entry.ID)
			params := mapUpsertAdParams(adID, entry.URL.String(), "unknown", nil, pgtype.UUID{Valid: false})
			ad, upsertErr := s.queries.UpsertShortcutAd(ctx, params)
			if upsertErr != nil {
				upsertErrors = append(upsertErrors, fmt.Errorf("upsert ad %d: %w", entry.ID, upsertErr))
				continue
			}
			entityID := fmt.Sprintf("ad:%d", ad.ShortcutAdsID)
			adIDs = append(adIDs, entityID)
		}
	}
	if len(buildingIDs) == 0 && len(adIDs) == 0 && len(upsertErrors) > 0 {
		return nil, nil, fmt.Errorf("all upserts failed: %w", errors.Join(upsertErrors...))
	}
	return buildingIDs, adIDs, nil
}

func (s *Service) SyncAd(ctx context.Context, adID int64) error {
	adData, err := s.client.GetAdByID(ctx, int(adID))
	if err != nil {
		return fmt.Errorf("fetch ad data (ad_id=%d): %w", adID, err)
	}
	var adDataMap map[string]any
	if err := json.Unmarshal(adData, &adDataMap); err != nil {
		return fmt.Errorf("unmarshal ad data (ad_id=%d): %w", adID, err)
	}
	adType := "unknown"
	if cardType, ok := adDataMap["cardType"].(float64); ok {
		switch int(cardType) {
		case 100:
			adType = "sale"
		case 101:
			adType = "rent"
		default:
			adType = fmt.Sprintf("type_%d", int(cardType))
		}
	}
	var shortcutBuildingID pgtype.UUID
	if buildingData, ok := adDataMap["buildingData"].(map[string]any); ok {
		if buildingIDFloat, ok := buildingData["buildingId"].(float64); ok {
			buildingIDInt := int64(buildingIDFloat)
			building, err := s.queries.GetShortcutBuildingByExternalID(ctx, buildingIDInt)
			if err == nil {
				shortcutBuildingID = building.ShortcutBuildingsID
			}
		}
	}
	existingAd, err := s.queries.GetShortcutAdByID(ctx, adID)
	if err != nil {
		return fmt.Errorf("get existing ad (ad_id=%d): %w", adID, err)
	}
	params := mapUpsertAdParams(adID, existingAd.ShortcutAdsUrl, adType, adData, shortcutBuildingID)
	if _, err = s.queries.UpsertShortcutAd(ctx, params); err != nil {
		return fmt.Errorf("upsert ad data (ad_id=%d): %w", adID, err)
	}
	return nil
}

func (s *Service) SyncBuilding(ctx context.Context, buildingID uuid.UUID) error {
	building, err := s.queries.GetShortcutBuildingByID(ctx, pgtype.UUID{Bytes: buildingID, Valid: true})
	if err != nil {
		return fmt.Errorf("get building (building_id=%s): %w", buildingID, err)
	}
	if building.ShortcutBuildingsPageNotFound != nil && *building.ShortcutBuildingsPageNotFound {
		return nil
	}
	scrapedBuilding, listings, rentals, err := s.client.ScrapeBuildingPage(ctx, int(building.ShortcutBuildingsExternalID), building.ShortcutBuildingsUrl)
	if err != nil {
		if errors.Is(err, client.ErrScraperErrorPage) {
			if markErr := s.queries.MarkShortcutBuildingPageNotFound(ctx, pgtype.UUID{Bytes: buildingID, Valid: true}); markErr != nil {
				return fmt.Errorf("mark building page not found (building_id=%s): %w", buildingID, markErr)
			}
			return nil
		}
		if errors.Is(err, client.ErrScraperForbidden) {
			return fmt.Errorf("scraping forbidden (building_id=%s, url=%s): %w", buildingID, building.ShortcutBuildingsUrl, err)
		}
		return fmt.Errorf("scrape building page (building_id=%s, url=%s): %w", buildingID, building.ShortcutBuildingsUrl, err)
	}
	params := mapScrapedBuildingParams(int64(scrapedBuilding.ShortcutBuildingID), building.ShortcutBuildingsUrl, scrapedBuilding)
	if _, err = s.queries.UpsertShortcutBuilding(ctx, params); err != nil {
		return fmt.Errorf("update building (building_id=%s): %w", buildingID, err)
	}
	var upsertErrors []error
	for _, listing := range listings {
		params := mapListingParams(pgtype.UUID{Bytes: buildingID, Valid: true}, &listing)
		if _, err := s.queries.UpsertShortcutBuildingListing(ctx, params); err != nil {
			upsertErrors = append(upsertErrors, fmt.Errorf("upsert listing %d: %w", listing.Index, err))
		}
	}
	for _, rental := range rentals {
		params := mapRentalParams(pgtype.UUID{Bytes: buildingID, Valid: true}, &rental)
		if _, err := s.queries.UpsertShortcutBuildingRental(ctx, params); err != nil {
			upsertErrors = append(upsertErrors, fmt.Errorf("upsert rental %d: %w", rental.Index, err))
		}
	}
	if err := s.queries.MarkShortcutBuildingProcessed(ctx, pgtype.UUID{Bytes: buildingID, Valid: true}); err != nil {
		return fmt.Errorf("mark building processed (building_id=%s): %w", buildingID, err)
	}
	if len(upsertErrors) > 0 && len(listings)+len(rentals) == len(upsertErrors) {
		return fmt.Errorf("all listing/rental upserts failed (building_id=%s): %w", buildingID, errors.Join(upsertErrors...))
	}
	return nil
}
