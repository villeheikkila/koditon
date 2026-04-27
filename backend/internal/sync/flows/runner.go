package syncflows

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"

	"koditon/internal/domain/ads"
	"koditon/internal/sync/frontdoor"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
)

type Runner struct {
	logger           *slog.Logger
	adsService       *ads.Service
	pricesService    *prices.Service
	shortcutService  *shortcut.Service
	frontdoorService *frontdoor.Service
	postalService    *postal.Service
}

func NewRunner(logger *slog.Logger, adsService *ads.Service, pricesService *prices.Service, shortcutService *shortcut.Service, frontdoorService *frontdoor.Service, postalService *postal.Service) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{logger: logger, adsService: adsService, pricesService: pricesService, shortcutService: shortcutService, frontdoorService: frontdoorService, postalService: postalService}
}

func (r *Runner) FrontdoorSitemap(ctx context.Context) ([]string, []string, error) {
	adIDs, buildingIDs, err := r.frontdoorService.SyncSitemap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("frontdoor sitemap sync: %w", err)
	}
	return adIDs, buildingIDs, nil
}

func (r *Runner) ShortcutSitemap(ctx context.Context) ([]string, []string, error) {
	buildingIDs, adIDs, err := r.shortcutService.SyncSitemap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("shortcut sitemap sync: %w", err)
	}
	return buildingIDs, adIDs, nil
}

func (r *Runner) PricesFetchCities(ctx context.Context) ([]string, error) {
	cities, err := r.pricesService.FetchCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cities: %w", err)
	}
	return cities, nil
}

func (r *Runner) FrontdoorSyncEntity(ctx context.Context, entityID string) error {
	entityType, externalID, err := parseEntityID(entityID)
	if err != nil {
		return err
	}
	switch entityType {
	case "ad":
		if err := r.frontdoorService.SyncAd(ctx, externalID); err != nil {
			return fmt.Errorf("sync frontdoor ad %s: %w", externalID, err)
		}
		return nil
	case "building":
		if err := r.frontdoorService.SyncBuilding(ctx, externalID); err != nil {
			return fmt.Errorf("sync frontdoor building %s: %w", externalID, err)
		}
		return nil
	default:
		return &EntityParseError{EntityID: entityID, Reason: fmt.Sprintf("unknown frontdoor entity type: %s", entityType)}
	}
}

func (r *Runner) ShortcutSyncEntity(ctx context.Context, entityID string) error {
	entityType, externalID, err := parseEntityID(entityID)
	if err != nil {
		return err
	}
	switch entityType {
	case "ad":
		adID, err := strconv.ParseInt(externalID, 10, 64)
		if err != nil {
			return &EntityParseError{EntityID: entityID, Reason: "invalid ad ID", Err: err}
		}
		if err := r.shortcutService.SyncAd(ctx, adID); err != nil {
			return fmt.Errorf("sync shortcut ad %d: %w", adID, err)
		}
		return nil
	case "building":
		buildingID, err := uuid.Parse(externalID)
		if err != nil {
			return &EntityParseError{EntityID: entityID, Reason: "invalid building UUID", Err: err}
		}
		if err := r.shortcutService.SyncBuilding(ctx, buildingID); err != nil {
			return fmt.Errorf("sync shortcut building %s: %w", buildingID, err)
		}
		return nil
	default:
		return &EntityParseError{EntityID: entityID, Reason: fmt.Sprintf("expected ad/building entity type for shortcut sync, got: %s", entityType)}
	}
}

func (r *Runner) PricesSyncCityEntity(ctx context.Context, entityID string) error {
	return r.PricesSyncCityEntityWithProgress(ctx, entityID, nil)
}

func (r *Runner) PricesSyncCityEntityWithProgress(ctx context.Context, entityID string, progressFn func(prices.SyncCityProgress)) error {
	entityType, cityName, err := parseEntityID(entityID)
	if err != nil {
		return err
	}
	if entityType != "city" {
		return &EntityParseError{EntityID: entityID, Reason: fmt.Sprintf("expected city entity type for prices sync, got: %s", entityType)}
	}
	if err := r.pricesService.SyncCityWithProgress(ctx, cityName, progressFn); err != nil {
		return fmt.Errorf("sync prices city %s: %w", cityName, err)
	}
	return nil
}

func (r *Runner) PricesSyncCityIndex(ctx context.Context, cityName string, progressFn func(prices.SyncCityProgress)) (*prices.SyncCityIndexResult, error) {
	result, err := r.pricesService.SyncCityIndex(ctx, cityName, progressFn)
	if err != nil {
		return nil, fmt.Errorf("sync prices city index %s: %w", cityName, err)
	}
	return result, nil
}

func (r *Runner) PricesSyncPostalCodeTransactions(ctx context.Context, cityName, postalCode string, progressFn func(prices.SyncPostalCodeProgress)) error {
	if err := r.pricesService.SyncPostalCodeTransactions(ctx, cityName, postalCode, progressFn); err != nil {
		return fmt.Errorf("sync prices postal code %s/%s: %w", cityName, postalCode, err)
	}
	return nil
}

func (r *Runner) PricesSyncPostalCodeTransactionPage(ctx context.Context, cityName, postalCode string, page int, progressFn func(prices.SyncPostalCodeProgress)) (*prices.SyncPostalCodePageResult, error) {
	result, err := r.pricesService.SyncPostalCodeTransactionPage(ctx, cityName, postalCode, page, progressFn)
	if err != nil {
		return nil, fmt.Errorf("sync prices postal code page %s/%s/%d: %w", cityName, postalCode, page, err)
	}
	return result, nil
}

func (r *Runner) PricesSyncAll(ctx context.Context, cfg prices.SyncAllConfig) (*prices.SyncAllResult, error) {
	result, err := r.pricesService.SyncAll(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("sync all prices: %w", err)
	}
	return result, nil
}

func (r *Runner) PricesSyncNeighborhoodPostalCodes(ctx context.Context, progressFn func(prices.SyncNeighborhoodPostalCodesProgress)) error {
	if err := r.pricesService.SyncNeighborhoodPostalCodes(ctx, progressFn); err != nil {
		return fmt.Errorf("sync neighborhood postal codes: %w", err)
	}
	return nil
}

func (r *Runner) PricesSearchTransactionsByCityAndAddress(ctx context.Context, cityName, searchTerm string, limit int32) ([]prices.SearchTransactionsRow, error) {
	rows, err := r.pricesService.SearchTransactionsByCityAndAddress(ctx, cityName, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("search prices transactions: %w", err)
	}
	return rows, nil
}

func (r *Runner) ShortcutDescribeEntity(ctx context.Context, entityID string) (string, error) {
	entityType, externalID, err := parseEntityID(entityID)
	if err != nil {
		return "", err
	}
	switch entityType {
	case "ad":
		adID, err := strconv.ParseInt(externalID, 10, 64)
		if err != nil {
			return "", &EntityParseError{EntityID: entityID, Reason: "invalid ad ID", Err: err}
		}
		desc, err := r.shortcutService.DescribeAd(ctx, adID)
		if err != nil {
			return "", fmt.Errorf("describe shortcut ad %d: %w", adID, err)
		}
		return desc, nil
	case "building":
		buildingID, err := uuid.Parse(externalID)
		if err != nil {
			return "", &EntityParseError{EntityID: entityID, Reason: "invalid building UUID", Err: err}
		}
		desc, err := r.shortcutService.DescribeBuilding(ctx, buildingID)
		if err != nil {
			return "", fmt.Errorf("describe shortcut building %s: %w", buildingID, err)
		}
		return desc, nil
	default:
		return "", &EntityParseError{EntityID: entityID, Reason: fmt.Sprintf("unknown shortcut entity type: %s", entityType)}
	}
}

func (r *Runner) PostalSync(ctx context.Context, logger *slog.Logger) (*postal.SyncResult, error) {
	if logger == nil {
		logger = r.logger
	}
	result, err := r.postalService.Sync(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("sync postal codes: %w", err)
	}
	return result, nil
}

func (r *Runner) AdsSearchReports(ctx context.Context, params ads.SearchParams) (ads.ReportPage, error) {
	if r.adsService == nil {
		return ads.ReportPage{}, fmt.Errorf("ads service unavailable")
	}
	page, err := r.adsService.Search(ctx, params)
	if err != nil {
		return ads.ReportPage{}, fmt.Errorf("search ads reports: %w", err)
	}
	return page, nil
}

func (r *Runner) AdsReportDetail(ctx context.Context, canonicalID string) (ads.UnifiedEntityDetail, error) {
	if r.adsService == nil {
		return ads.UnifiedEntityDetail{}, fmt.Errorf("ads service unavailable")
	}
	detail, err := r.adsService.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		return ads.UnifiedEntityDetail{}, fmt.Errorf("load ads report detail: %w", err)
	}
	return detail, nil
}
