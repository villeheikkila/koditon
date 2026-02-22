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

type progressUpdate struct {
	Message string
	Current int
	Total   int
}

type reportFn func(progressUpdate)

type actionTable struct {
	Title   string
	Columns []string
	Rows    [][]string
}

type actionResult struct {
	Output string
	Table  *actionTable
}

type action struct {
	Title         string
	Description   string
	Prompts       []string
	UseCityPicker bool
	Run           func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error)
}

type subsystem struct {
	Title       string
	Description string
	Actions     []action
}

func buildSubsystems() []subsystem {
	return []subsystem{
		{
			Title:       "Frontdoor",
			Description: "Frontdoor sitemap and entity sync",
			Actions: []action{
				{
					Title:       "Full Sitemap Sync",
					Description: "Fetch sitemap and sync all frontdoor ad/building entities now",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching frontdoor sitemap..."})
						adIDs, buildingIDs, err := runner.FrontdoorSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
						entityIDs = append(entityIDs, adIDs...)
						entityIDs = append(entityIDs, buildingIDs...)
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d ads and %d buildings", len(adIDs), len(buildingIDs))})
						result := runEntityBatch(ctx, entityIDs, runner.FrontdoorSyncEntity, report)
						return actionResult{Output: fmt.Sprintf("discovered ads=%d buildings=%d total=%d success=%d failed=%d duration=%s", len(adIDs), len(buildingIDs), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond))}, joinResultErrors(result)
					},
				},
				{
					Title:       "Sync Ad by ID",
					Description: "Sync one frontdoor ad by friendly ID",
					Prompts:     []string{"friendly ad id"},
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						entityID := "ad:" + strings.TrimSpace(inputs[0])
						report(progressUpdate{Message: "Syncing " + entityID})
						if err := runner.FrontdoorSyncEntity(ctx, entityID); err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "synced " + entityID}, nil
					},
				},
				{
					Title:       "Sync Building by ID",
					Description: "Sync one frontdoor building by housing company ID",
					Prompts:     []string{"building housing company id"},
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						entityID := "building:" + strings.TrimSpace(inputs[0])
						report(progressUpdate{Message: "Syncing " + entityID})
						if err := runner.FrontdoorSyncEntity(ctx, entityID); err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "synced " + entityID}, nil
					},
				},
			},
		},
		{
			Title:       "Shortcut",
			Description: "Shortcut sitemap and entity sync",
			Actions: []action{
				{
					Title:       "Full Sitemap Sync",
					Description: "Fetch sitemap and sync all shortcut ad/building entities now",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching shortcut sitemap..."})
						buildingIDs, adIDs, err := runner.ShortcutSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
						entityIDs = append(entityIDs, buildingIDs...)
						entityIDs = append(entityIDs, adIDs...)
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d buildings and %d ads", len(buildingIDs), len(adIDs))})
						result := runEntityBatch(ctx, entityIDs, runner.ShortcutSyncEntity, report)
						return actionResult{Output: fmt.Sprintf("discovered buildings=%d ads=%d total=%d success=%d failed=%d duration=%s", len(buildingIDs), len(adIDs), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond))}, joinResultErrors(result)
					},
				},
				{
					Title:       "Sync Ad by ID",
					Description: "Sync one shortcut ad by numeric ID",
					Prompts:     []string{"ad id"},
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						entityID := "ad:" + strings.TrimSpace(inputs[0])
						report(progressUpdate{Message: "Syncing " + entityID})
						if err := runner.ShortcutSyncEntity(ctx, entityID); err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "synced " + entityID}, nil
					},
				},
				{
					Title:       "Sync Building by UUID",
					Description: "Sync one shortcut building by UUID",
					Prompts:     []string{"building uuid"},
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						entityID := "building:" + strings.TrimSpace(inputs[0])
						report(progressUpdate{Message: "Syncing " + entityID})
						if err := runner.ShortcutSyncEntity(ctx, entityID); err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "synced " + entityID}, nil
					},
				},
			},
		},
		{
			Title:       "Prices",
			Description: "Prices batch and city-level sync",
			Actions: []action{
				{
					Title:       "Cities Init (Full Sync Now)",
					Description: "Fetch prices cities and sync each city in-process",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching prices cities..."})
						cities, err := runner.PricesFetchCities(ctx)
						if err != nil {
							return actionResult{}, err
						}
						entityIDs := make([]string, 0, len(cities))
						for _, city := range cities {
							entityIDs = append(entityIDs, "city:"+city)
						}
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d cities", len(cities))})
						result := runEntityBatch(ctx, entityIDs, runner.PricesSyncCityEntity, report)
						return actionResult{Output: fmt.Sprintf("cities=%d total=%d success=%d failed=%d duration=%s", len(cities), result.Total, result.Success, result.Failed, result.Duration.Round(time.Millisecond))}, joinResultErrors(result)
					},
				},
				{
					Title:       "Sync All",
					Description: "Run prices sync-all flow",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Running prices sync-all..."})
						cfg := prices.DefaultSyncAllConfig()
						cfg.Concurrency = 1
						cfg.Logger = newProgressLogger(report)
						res, err := runner.PricesSyncAll(ctx, cfg)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: "Prices sync-all completed"})
						return actionResult{Output: fmt.Sprintf("cities=%d postal_codes=%d neighborhoods=%d transactions=%d errors=%d", res.CitiesProcessed, res.PostalCodesProcessed, res.NeighborhoodsUpdated, res.TransactionsProcessed, len(res.Errors))}, nil
					},
				},
				{
					Title:       "Neighborhood Postal Code Sync",
					Description: "Run neighborhood->postal code mapping sync",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Running neighborhood postal code sync..."})
						err := runner.PricesSyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
							if p.Page > 0 {
								report(progressUpdate{Message: fmt.Sprintf("City=%s postal=%s page=%d", p.City, p.PostalCode, p.Page)})
								return
							}
							if p.Updated > 0 {
								report(progressUpdate{Message: fmt.Sprintf("Updated %d mappings for %s %s", p.Updated, p.City, p.PostalCode)})
							}
						})
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "completed prices neighborhood postal code sync"}, nil
					},
				},
				{
					Title:         "Sync City by Name",
					Description:   "Sync prices data for one city",
					Prompts:       []string{"city name"},
					UseCityPicker: true,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						entityID := "city:" + strings.TrimSpace(inputs[0])
						report(progressUpdate{Message: "Syncing " + entityID})
						if err := runner.PricesSyncCityEntityWithProgress(ctx, entityID, func(p prices.SyncCityProgress) {
							msg := "prices city " + p.Step
							switch p.Step {
							case "city_upsert_start":
								msg = fmt.Sprintf("%s upsert city row", p.City)
							case "city_upsert_done":
								msg = fmt.Sprintf("%s city upsert complete", p.City)
							case "postal_codes_fetch_start":
								msg = fmt.Sprintf("%s fetch postal codes", p.City)
							case "postal_codes_fetch_done":
								msg = fmt.Sprintf("%s fetched postal codes count=%d", p.City, p.Count)
							case "postal_codes_upsert_start":
								msg = fmt.Sprintf("%s upsert postal codes count=%d", p.City, p.Count)
							case "postal_codes_upsert_done":
								msg = fmt.Sprintf("%s postal codes upserted count=%d", p.City, p.Count)
							case "neighborhoods_fetch_start":
								msg = fmt.Sprintf("%s fetch neighborhoods", p.City)
							case "neighborhoods_fetch_done":
								msg = fmt.Sprintf("%s fetched neighborhoods count=%d", p.City, p.Count)
							case "transactions_fetch_start":
								msg = fmt.Sprintf("%s fetch transaction pages", p.City)
							case "transactions_page":
								msg = fmt.Sprintf("%s fetched transactions page=%d rows=%d %s", p.City, p.Page, p.Count, p.Details)
							case "transactions_fetch_done":
								msg = fmt.Sprintf("%s fetched all transactions count=%d", p.City, p.Count)
							case "neighborhoods_merge_done":
								msg = fmt.Sprintf("%s merged neighborhood set count=%d", p.City, p.Count)
							case "neighborhoods_upsert_start":
								msg = fmt.Sprintf("%s upsert neighborhoods count=%d", p.City, p.Count)
							case "neighborhoods_upsert_done":
								msg = fmt.Sprintf("%s neighborhoods upserted count=%d", p.City, p.Count)
							case "transactions_upsert_start":
								msg = fmt.Sprintf("%s upsert transactions count=%d period=%s", p.City, p.Count, p.Details)
							case "transactions_upsert_done":
								msg = fmt.Sprintf("%s transactions upserted count=%d period=%s", p.City, p.Count, p.Details)
							case "sync_city_done":
								msg = fmt.Sprintf("%s sync complete", p.City)
							}
							report(progressUpdate{Message: msg})
						}); err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: "synced " + entityID}, nil
					},
				},
				{
					Title:         "Search Transactions",
					Description:   "Search prices data by city and neighborhood / postal code",
					Prompts:       []string{"city name", "neighborhood name or postal code"},
					UseCityPicker: true,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						city := strings.TrimSpace(inputs[0])
						search := strings.TrimSpace(inputs[1])
						report(progressUpdate{Message: fmt.Sprintf("Searching city=%s query=%s", city, search)})
						rows, err := runner.PricesSearchTransactionsByCityAndAddress(ctx, city, search, 500)
						if err != nil {
							return actionResult{}, err
						}
						tableRows := make([][]string, 0, len(rows))
						for _, row := range rows {
							tableRows = append(tableRows, []string{
								row.CreatedAt.Format("2006-01-02"),
								row.City,
								row.Municipality,
								row.PostalCode,
								row.PostalArea,
								row.Neighborhood,
								row.Description,
								fmt.Sprintf("%d", row.Price),
								fmt.Sprintf("%d", row.PricePerSqm),
								fmt.Sprintf("%.1f", row.Area),
								row.Type,
								row.Category,
								fmt.Sprintf("%d", row.BuildYear),
								row.Floor,
								boolToYN(row.Elevator),
								row.Condition,
								row.EnergyClass,
								row.Plot,
								row.PeriodIdentifier,
							})
						}
						report(progressUpdate{Message: fmt.Sprintf("Found %d rows", len(tableRows))})
						return actionResult{
							Output: fmt.Sprintf("city=%s query=%s rows=%d", city, search, len(tableRows)),
							Table: &actionTable{
								Title: "Prices Transaction Matches",
								Columns: []string{
									"Date", "City", "Municipality", "Postal", "PostalArea", "Neighborhood", "Address", "Price", "EUR/m2", "Area", "Type", "Category", "Year", "Floor", "Elev", "Condition", "Energy", "Plot", "Period",
								},
								Rows: tableRows,
							},
						}, nil
					},
				},
			},
		},
		{
			Title:       "Postal",
			Description: "Postal dataset synchronization",
			Actions: []action{
				{
					Title:       "Sync",
					Description: "Run postal data sync",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Running postal sync..."})
						res, err := runner.PostalSync(ctx, newProgressLogger(report))
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: fmt.Sprintf("total=%d ad_areas=%d municipalities=%d postal_codes=%d skipped=%d", res.TotalRecords, res.AdAreasUpserted, res.MunicipalitiesUpserted, res.PostalCodesUpserted, res.SkippedRecords)}, nil
					},
				},
			},
		},
	}
}

func boolToYN(v bool) string {
	if v {
		return "Y"
	}
	return "N"
}

func runEntityBatch(ctx context.Context, entityIDs []string, syncFn func(context.Context, string) error, report reportFn) *syncflows.BatchRunResult {
	start := time.Now()
	result := &syncflows.BatchRunResult{Total: len(entityIDs), Errors: make([]error, 0)}
	for i, entityID := range entityIDs {
		report(progressUpdate{Message: "Syncing " + entityID, Current: i + 1, Total: len(entityIDs)})
		if ctx.Err() != nil {
			result.Errors = append(result.Errors, ctx.Err())
			result.Failed++
			continue
		}
		if err := syncFn(ctx, entityID); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", entityID, err))
			result.Failed++
			report(progressUpdate{Message: "Failed " + entityID, Current: i + 1, Total: len(entityIDs)})
			continue
		}
		result.Success++
		report(progressUpdate{Message: "Done " + entityID, Current: i + 1, Total: len(entityIDs)})
	}
	result.Duration = time.Since(start)
	return result
}

func joinResultErrors(result *syncflows.BatchRunResult) error {
	if result == nil || len(result.Errors) == 0 {
		return nil
	}
	return errors.Join(result.Errors...)
}
