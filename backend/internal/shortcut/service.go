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

type AdType string

const (
	AdTypeListing AdType = "listing"
	AdTypeRental  AdType = "rental"
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
	if len(buildingEntries) > 0 {
		params := mapBatchUpsertBuildingsFromSitemapParams(buildingEntries)
		buildings, upsertErr := s.queries.BatchUpsertShortcutBuildingsFromSitemap(ctx, params)
		if upsertErr != nil {
			return nil, nil, fmt.Errorf("batch upsert buildings: %w", upsertErr)
		}
		buildingIDs = make([]string, len(buildings))
		for i, building := range buildings {
			buildingIDs[i] = fmt.Sprintf("building:%s", building.ShortcutBuildingsID.String())
		}
	}
	adEntries := append(listingEntries, rentalEntries...)
	if len(adEntries) > 0 {
		seenAdIDs := make(map[int]struct{})
		validEntries := make([]client.ShortcutSitemapEntry, 0, len(adEntries))
		validAdTypes := make([]AdType, 0, len(adEntries))
		for _, entry := range adEntries {
			if _, seen := seenAdIDs[entry.ID]; seen {
				s.logger.Debug("duplicate ad ID in sitemap, skipping", "ad_id", entry.ID)
				continue
			}
			seenAdIDs[entry.ID] = struct{}{}
			var adType AdType
			switch entry.Type {
			case client.SitemapURLTypeListing:
				adType = AdTypeListing
			case client.SitemapURLTypeRental:
				adType = AdTypeRental
			default:
				s.logger.Warn("unknown ad type from sitemap, skipping", "ad_id", entry.ID, "type", entry.Type)
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
				adIDs[i] = fmt.Sprintf("ad:%d", ad.ShortcutAdsID)
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
	var adDataMap map[string]any
	if err := json.Unmarshal(adData, &adDataMap); err != nil {
		return fmt.Errorf("unmarshal ad data (ad_id=%d): %w", adID, err)
	}
	var adType AdType
	if cardType, ok := adDataMap["cardType"].(float64); ok {
		switch int(cardType) {
		case 100:
			adType = AdTypeListing
		case 101:
			adType = AdTypeRental
		default:
			s.logger.Warn("unknown cardType from ad data, skipping sync", "ad_id", adID, "card_type", int(cardType))
			return nil
		}
	} else {
		s.logger.Warn("missing cardType in ad data, skipping sync", "ad_id", adID)
		return nil
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
	params := mapUpsertAdParams(adID, existingAd.ShortcutAdsUrl, string(adType), adData, shortcutBuildingID)
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
