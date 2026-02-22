package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"koditon-go/internal/prices"
	"koditon-go/internal/syncflows"
)

type action struct {
	Title       string
	Description string
	NeedsInput  bool
	Prompt      string
	Run         func(ctx context.Context, runner *syncflows.Runner, input string) (string, error)
}

func buildActions() []action {
	return []action{
		{
			Title:       "Frontdoor: Full Sitemap Sync",
			Description: "Fetch sitemap and sync all frontdoor ad/building entities now",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				adIDs, buildingIDs, err := runner.FrontdoorSitemap(ctx)
				if err != nil {
					return "", err
				}
				entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
				entityIDs = append(entityIDs, adIDs...)
				entityIDs = append(entityIDs, buildingIDs...)
				result := runEntityBatch(ctx, entityIDs, runner.FrontdoorSyncEntity)
				return fmt.Sprintf("discovered ads=%d buildings=%d total=%d success=%d failed=%d duration=%s", len(adIDs), len(buildingIDs), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond)), joinResultErrors(result)
			},
		},
		{
			Title:       "Shortcut: Full Sitemap Sync",
			Description: "Fetch sitemap and sync all shortcut ad/building entities now",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				buildingIDs, adIDs, err := runner.ShortcutSitemap(ctx)
				if err != nil {
					return "", err
				}
				entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
				entityIDs = append(entityIDs, buildingIDs...)
				entityIDs = append(entityIDs, adIDs...)
				result := runEntityBatch(ctx, entityIDs, runner.ShortcutSyncEntity)
				return fmt.Sprintf("discovered buildings=%d ads=%d total=%d success=%d failed=%d duration=%s", len(buildingIDs), len(adIDs), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond)), joinResultErrors(result)
			},
		},
		{
			Title:       "Prices: Cities Init (Full Sync Now)",
			Description: "Fetch prices cities and sync each city in-process",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				cities, err := runner.PricesFetchCities(ctx)
				if err != nil {
					return "", err
				}
				entityIDs := make([]string, 0, len(cities))
				for _, city := range cities {
					entityIDs = append(entityIDs, "city:"+city)
				}
				result := runEntityBatch(ctx, entityIDs, runner.PricesSyncCityEntity)
				return fmt.Sprintf("cities=%d total=%d success=%d failed=%d duration=%s", len(cities), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond)), joinResultErrors(result)
			},
		},
		{
			Title:       "Prices: Sync All",
			Description: "Run prices sync-all flow",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				cfg := prices.DefaultSyncAllConfig()
				cfg.Concurrency = 1
				res, err := runner.PricesSyncAll(ctx, cfg)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("cities=%d postal_codes=%d neighborhoods=%d transactions=%d errors=%d", res.CitiesProcessed, res.PostalCodesProcessed, res.NeighborhoodsUpdated, res.TransactionsProcessed, len(res.Errors)), nil
			},
		},
		{
			Title:       "Prices: Neighborhood Postal Code Sync",
			Description: "Run neighborhood->postal code mapping sync",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				if err := runner.PricesSyncNeighborhoodPostalCodes(ctx, nil); err != nil {
					return "", err
				}
				return "completed prices neighborhood postal code sync", nil
			},
		},
		{
			Title:       "Postal: Sync",
			Description: "Run postal data sync",
			Run: func(ctx context.Context, runner *syncflows.Runner, _ string) (string, error) {
				res, err := runner.PostalSync(ctx, nil)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("total=%d ad_areas=%d municipalities=%d postal_codes=%d skipped=%d", res.TotalRecords, res.AdAreasUpserted, res.MunicipalitiesUpserted, res.PostalCodesUpserted, res.SkippedRecords), nil
			},
		},
		{
			Title:       "Frontdoor: Sync Ad by ID",
			Description: "Sync one frontdoor ad by friendly ID",
			NeedsInput:  true,
			Prompt:      "friendly ad id",
			Run: func(ctx context.Context, runner *syncflows.Runner, input string) (string, error) {
				entityID := "ad:" + strings.TrimSpace(input)
				if err := runner.FrontdoorSyncEntity(ctx, entityID); err != nil {
					return "", err
				}
				return "synced " + entityID, nil
			},
		},
		{
			Title:       "Frontdoor: Sync Building by ID",
			Description: "Sync one frontdoor building by housing company ID",
			NeedsInput:  true,
			Prompt:      "building housing company id",
			Run: func(ctx context.Context, runner *syncflows.Runner, input string) (string, error) {
				entityID := "building:" + strings.TrimSpace(input)
				if err := runner.FrontdoorSyncEntity(ctx, entityID); err != nil {
					return "", err
				}
				return "synced " + entityID, nil
			},
		},
		{
			Title:       "Shortcut: Sync Ad by ID",
			Description: "Sync one shortcut ad by numeric ID",
			NeedsInput:  true,
			Prompt:      "ad id",
			Run: func(ctx context.Context, runner *syncflows.Runner, input string) (string, error) {
				entityID := "ad:" + strings.TrimSpace(input)
				if err := runner.ShortcutSyncEntity(ctx, entityID); err != nil {
					return "", err
				}
				return "synced " + entityID, nil
			},
		},
		{
			Title:       "Shortcut: Sync Building by UUID",
			Description: "Sync one shortcut building by UUID",
			NeedsInput:  true,
			Prompt:      "building uuid",
			Run: func(ctx context.Context, runner *syncflows.Runner, input string) (string, error) {
				entityID := "building:" + strings.TrimSpace(input)
				if err := runner.ShortcutSyncEntity(ctx, entityID); err != nil {
					return "", err
				}
				return "synced " + entityID, nil
			},
		},
		{
			Title:       "Prices: Sync City by Name",
			Description: "Sync prices data for one city",
			NeedsInput:  true,
			Prompt:      "city name",
			Run: func(ctx context.Context, runner *syncflows.Runner, input string) (string, error) {
				entityID := "city:" + strings.TrimSpace(input)
				if err := runner.PricesSyncCityEntity(ctx, entityID); err != nil {
					return "", err
				}
				return "synced " + entityID, nil
			},
		},
	}
}

func runEntityBatch(ctx context.Context, entityIDs []string, syncFn func(context.Context, string) error) *syncflows.BatchRunResult {
	start := time.Now()
	result := &syncflows.BatchRunResult{Total: len(entityIDs), Errors: make([]error, 0)}
	for _, entityID := range entityIDs {
		if ctx.Err() != nil {
			result.Errors = append(result.Errors, ctx.Err())
			result.Failed++
			continue
		}
		if err := syncFn(ctx, entityID); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", entityID, err))
			result.Failed++
			continue
		}
		result.Success++
	}
	result.Duration = time.Since(start)
	return result
}

func joinResultErrors(result *syncflows.BatchRunResult) error {
	if result == nil || len(result.Errors) == 0 {
		return nil
	}
	return errorsJoin(result.Errors)
}

func errorsJoin(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
