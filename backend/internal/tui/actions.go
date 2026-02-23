package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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

type batchRunOptions struct {
	Limit int
	Delay time.Duration
}

type batchRunReport struct {
	Result *syncflows.BatchRunResult
	Loaded []string
	Failed []string
}

type entityDetailFn func(context.Context, string) string

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
	BuildInput    func(ctx *appContext, action action, values []string, breadcrumb string) Screen
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
			Title:       "Ads",
			Description: "Unified ads and announcement report search",
			Actions: []action{
				{
					Title:       "Search Reports",
					Description: "Browse shortcut/frontdoor ads and frontdoor announcements with filters",
					BuildInput:  newAdsReportFormScreen,
					Run: func(_ context.Context, _ *syncflows.Runner, _ []string, _ reportFn) (actionResult, error) {
						return actionResult{Output: "ads report browser opened"}, nil
					},
				},
			},
		},
		{
			Title:       "Frontdoor",
			Description: "Frontdoor sitemap and entity sync",
			Actions: []action{
				{
					Title:       "Full Sitemap Sync",
					Description: "Fetch sitemap and sync all frontdoor ad/building entities now",
					BuildInput:  newBatchSyncSettingsScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						opts, err := parseBatchRunOptions(inputs)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: "Fetching frontdoor sitemap..."})
						adIDs, buildingIDs, err := runner.FrontdoorSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
						entityIDs = append(entityIDs, adIDs...)
						entityIDs = append(entityIDs, buildingIDs...)
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d ads and %d buildings", len(adIDs), len(buildingIDs))})
						batch := runEntityBatch(ctx, entityIDs, runner.FrontdoorSyncEntity, nil, report, opts)
						return actionResult{Output: fmt.Sprintf("discovered ads=%d buildings=%d total=%d success=%d failed=%d loaded=%s failed_items=%s duration=%s", len(adIDs), len(buildingIDs), batch.Result.Total, batch.Result.Success, batch.Result.Failed, summarizeEntityIDs(batch.Loaded, 5), summarizeEntityIDs(batch.Failed, 3), batch.Result.Duration.Round(time.Millisecond))}, joinResultErrors(batch.Result)
					},
				},
				{
					Title:	   "Sitemap Discover",
					Description: "Fetch frontdoor sitemap and report discovered entities without syncing",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching frontdoor sitemap..."})
						adIDs, buildingIDs, err := runner.FrontdoorSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: fmt.Sprintf("discovered ads=%d buildings=%d total=%d", len(adIDs), len(buildingIDs), len(adIDs)+len(buildingIDs))}, nil
					},
				},
				{
					Title:	   "Sync Buildings",
					Description: "Fetch frontdoor sitemap and sync only buildings in batch",
					BuildInput:  newBatchSyncSettingsScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						opts, err := parseBatchRunOptions(inputs)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: "Fetching frontdoor sitemap..."})
						_, buildingIDs, err := runner.FrontdoorSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d buildings", len(buildingIDs))})
						batch := runEntityBatch(ctx, buildingIDs, runner.FrontdoorSyncEntity, nil, report, opts)
						return actionResult{Output: fmt.Sprintf("buildings=%d success=%d failed=%d loaded=%s failed_items=%s duration=%s", len(buildingIDs), batch.Result.Success, batch.Result.Failed, summarizeEntityIDs(batch.Loaded, 5), summarizeEntityIDs(batch.Failed, 3), batch.Result.Duration.Round(time.Millisecond))}, joinResultErrors(batch.Result)
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
					BuildInput:  newBatchSyncSettingsScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						opts, err := parseBatchRunOptions(inputs)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: "Fetching shortcut sitemap..."})
						buildingIDs, adIDs, err := runner.ShortcutSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						entityIDs := make([]string, 0, len(adIDs)+len(buildingIDs))
						entityIDs = append(entityIDs, buildingIDs...)
						entityIDs = append(entityIDs, adIDs...)
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d buildings and %d ads", len(buildingIDs), len(adIDs))})
						batch := runEntityBatch(ctx, entityIDs, runner.ShortcutSyncEntity, shortcutDetailFn(runner), report, opts)
						return actionResult{Output: fmt.Sprintf("discovered buildings=%d ads=%d total=%d success=%d failed=%d loaded=%s failed_items=%s duration=%s", len(buildingIDs), len(adIDs), batch.Result.Total, batch.Result.Success, batch.Result.Failed, summarizeEntityIDs(batch.Loaded, 5), summarizeEntityIDs(batch.Failed, 3), batch.Result.Duration.Round(time.Millisecond))}, joinResultErrors(batch.Result)
					},
				},
				{
					Title:	   "Sitemap Discover",
					Description: "Fetch shortcut sitemap and report discovered entities without syncing",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching shortcut sitemap..."})
						buildingIDs, adIDs, err := runner.ShortcutSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: fmt.Sprintf("discovered buildings=%d ads=%d total=%d", len(buildingIDs), len(adIDs), len(buildingIDs)+len(adIDs))}, nil
					},
				},
				{
					Title:	   "Sync Buildings",
					Description: "Fetch shortcut sitemap and sync only buildings in batch",
					BuildInput:  newBatchSyncSettingsScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						opts, err := parseBatchRunOptions(inputs)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: "Fetching shortcut sitemap..."})
						buildingIDs, _, err := runner.ShortcutSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d buildings", len(buildingIDs))})
						batch := runEntityBatch(ctx, buildingIDs, runner.ShortcutSyncEntity, shortcutDetailFn(runner), report, opts)
						return actionResult{Output: fmt.Sprintf("buildings=%d success=%d failed=%d loaded=%s failed_items=%s duration=%s", len(buildingIDs), batch.Result.Success, batch.Result.Failed, summarizeEntityIDs(batch.Loaded, 5), summarizeEntityIDs(batch.Failed, 3), batch.Result.Duration.Round(time.Millisecond))}, joinResultErrors(batch.Result)
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
					BuildInput:  newBatchSyncSettingsScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						opts, err := parseBatchRunOptions(inputs)
						if err != nil {
							return actionResult{}, err
						}
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
						batch := runEntityBatch(ctx, entityIDs, runner.PricesSyncCityEntity, nil, report, opts)
						return actionResult{Output: fmt.Sprintf("cities=%d total=%d success=%d failed=%d loaded=%s failed_items=%s duration=%s", len(cities), batch.Result.Total, batch.Result.Success, batch.Result.Failed, summarizeEntityIDs(batch.Loaded, 5), summarizeEntityIDs(batch.Failed, 3), batch.Result.Duration.Round(time.Millisecond))}, joinResultErrors(batch.Result)
					},
				},
				{
					Title:	   "Cities Discover",
					Description: "Fetch available prices cities without syncing",
					Run: func(ctx context.Context, runner *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching prices cities..."})
						cities, err := runner.PricesFetchCities(ctx)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d cities", len(cities))})
						return actionResult{Output: fmt.Sprintf("cities=%d list=%s", len(cities), strings.Join(cities, ", "))}, nil
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
					Description:   "Targeted search with city/postal query, area filters and sorting",
					Prompts:       []string{"city name"},
					UseCityPicker: true,
					BuildInput:    newTransactionsSearchFormScreen,
					Run: func(ctx context.Context, runner *syncflows.Runner, inputs []string, report reportFn) (actionResult, error) {
						city := safeInput(inputs, 0)
						search := safeInput(inputs, 1)
						minArea, err := parseOptionalFloat64(safeInput(inputs, 2))
						if err != nil {
							return actionResult{}, fmt.Errorf("parse min area: %w", err)
						}
						maxArea, err := parseOptionalFloat64(safeInput(inputs, 3))
						if err != nil {
							return actionResult{}, fmt.Errorf("parse max area: %w", err)
						}
						if minArea != nil && maxArea != nil && *minArea > *maxArea {
							return actionResult{}, fmt.Errorf("min area cannot be greater than max area")
						}
						sortMode := parseSortMode(safeInput(inputs, 4))
						limit, err := parseOptionalInt32Default(safeInput(inputs, 5), 500, 1, 5000)
						if err != nil {
							return actionResult{}, fmt.Errorf("parse limit: %w", err)
						}
						report(progressUpdate{Message: fmt.Sprintf("Searching city=%s query=%s min_area=%s max_area=%s sort=%s limit=%d", city, search, formatOptionalFloat(minArea), formatOptionalFloat(maxArea), sortMode, limit)})
						rows, err := runner.PricesSearchTransactionsByCityAndAddress(ctx, city, search, limit)
						if err != nil {
							return actionResult{}, err
						}
						rows = filterRowsByArea(rows, minArea, maxArea)
						sortRows(rows, sortMode)
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
							Output: fmt.Sprintf("city=%s query=%s area=%s-%s sort=%s rows=%d", city, search, formatOptionalFloat(minArea), formatOptionalFloat(maxArea), sortMode, len(tableRows)),
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

func safeInput(inputs []string, idx int) string {
	if idx < 0 || idx >= len(inputs) {
		return ""
	}
	return strings.TrimSpace(inputs[idx])
}

func parseOptionalFloat64(v string) (*float64, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalInt32Default(v string, fallback, minValue, maxValue int32) (int32, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil {
		return 0, err
	}
	value := int32(parsed)
	if value < minValue || value > maxValue {
		return 0, fmt.Errorf("must be between %d and %d", minValue, maxValue)
	}
	return value, nil
}

func formatOptionalFloat(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *v)
}

func filterRowsByArea(rows []prices.SearchTransactionsRow, minArea, maxArea *float64) []prices.SearchTransactionsRow {
	if minArea == nil && maxArea == nil {
		return rows
	}
	filtered := make([]prices.SearchTransactionsRow, 0, len(rows))
	for _, row := range rows {
		if minArea != nil && row.Area < *minArea {
			continue
		}
		if maxArea != nil && row.Area > *maxArea {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func parseSortMode(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "price_asc", "price_desc", "date_asc", "date_desc":
		return strings.TrimSpace(strings.ToLower(v))
	default:
		return "price_asc"
	}
}

func sortRows(rows []prices.SearchTransactionsRow, sortMode string) {
	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		switch sortMode {
		case "price_desc":
			if left.Price == right.Price {
				return left.CreatedAt.After(right.CreatedAt)
			}
			return left.Price > right.Price
		case "date_asc":
			if left.CreatedAt.Equal(right.CreatedAt) {
				return left.Price < right.Price
			}
			return left.CreatedAt.Before(right.CreatedAt)
		case "date_desc":
			if left.CreatedAt.Equal(right.CreatedAt) {
				return left.Price < right.Price
			}
			return left.CreatedAt.After(right.CreatedAt)
		default:
			if left.Price == right.Price {
				return left.CreatedAt.After(right.CreatedAt)
			}
			return left.Price < right.Price
		}
	})
}

func parseBatchRunOptions(inputs []string) (batchRunOptions, error) {
	opts := batchRunOptions{}
	limitRaw := safeInput(inputs, 0)
	delayRaw := safeInput(inputs, 1)
	if limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil {
			return batchRunOptions{}, fmt.Errorf("max entries must be numeric")
		}
		if limit < 1 {
			return batchRunOptions{}, fmt.Errorf("max entries must be at least 1")
		}
		opts.Limit = limit
	}
	if delayRaw != "" {
		delay, err := time.ParseDuration(delayRaw)
		if err != nil {
			return batchRunOptions{}, fmt.Errorf("delay must be a valid duration (example: 1s)")
		}
		if delay < 0 {
			return batchRunOptions{}, fmt.Errorf("delay cannot be negative")
		}
		opts.Delay = delay
	}
	return opts, nil
}

func boolToYN(v bool) string {
	if v {
		return "Y"
	}
	return "N"
}

func summarizeEntityIDs(ids []string, maxCount int) string {
	if len(ids) == 0 {
		return "-"
	}
	limit := maxCount
	if limit <= 0 || limit > len(ids) {
		limit = len(ids)
	}
	short := strings.Join(ids[:limit], ",")
	if limit == len(ids) {
		return short
	}
	return fmt.Sprintf("%s,+%d", short, len(ids)-limit)
}

func shortcutDetailFn(runner *syncflows.Runner) entityDetailFn {
	return func(ctx context.Context, entityID string) string {
		detail, err := runner.ShortcutDescribeEntity(ctx, entityID)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(detail)
	}
}

func trimForProgress(value string, maxLen int) string {
	v := strings.TrimSpace(value)
	if v == "" || maxLen <= 0 {
		return ""
	}
	if len(v) <= maxLen {
		return v
	}
	return v[:maxLen-3] + "..."
}

func formatErr(err error) string {
	if err == nil {
		return ""
	}
	parts := strings.Split(err.Error(), ": ")
	if len(parts) <= 1 {
		return err.Error()
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteString("\n")
			for range i {
				b.WriteString("  ")
			}
		}
		b.WriteString(part)
	}
	return b.String()
}

func runEntityBatch(ctx context.Context, entityIDs []string, syncFn func(context.Context, string) error, detailFn entityDetailFn, report reportFn, opts batchRunOptions) *batchRunReport {
	if opts.Limit > 0 && len(entityIDs) > opts.Limit {
		entityIDs = entityIDs[:opts.Limit]
	}
	start := time.Now()
	result := &syncflows.BatchRunResult{Total: len(entityIDs), Errors: make([]error, 0)}
	batch := &batchRunReport{Result: result, Loaded: make([]string, 0), Failed: make([]string, 0)}
	for i, entityID := range entityIDs {
		if i > 0 && opts.Delay > 0 {
			report(progressUpdate{Message: fmt.Sprintf("Waiting %s before next sync", opts.Delay)})
			timer := time.NewTimer(opts.Delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				result.Errors = append(result.Errors, ctx.Err())
				result.Failed++
				batch.Failed = append(batch.Failed, entityID)
				result.Duration = time.Since(start)
				return batch
			case <-timer.C:
			}
		}
		report(progressUpdate{Message: fmt.Sprintf("Syncing %s (%d/%d, ok=%d, fail=%d)", entityID, i+1, len(entityIDs), result.Success, result.Failed), Current: i + 1, Total: len(entityIDs)})
		if ctx.Err() != nil {
			result.Errors = append(result.Errors, ctx.Err())
			result.Failed++
			continue
		}
		if err := syncFn(ctx, entityID); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", entityID, err))
			result.Failed++
			batch.Failed = append(batch.Failed, entityID)
			report(progressUpdate{Message: fmt.Sprintf("Failed %s (%d/%d): %s", entityID, i+1, len(entityIDs), formatErr(err)), Current: i + 1, Total: len(entityIDs)})
			continue
		}
		result.Success++
		batch.Loaded = append(batch.Loaded, entityID)
		detail := ""
		if detailFn != nil {
			detail = detailFn(ctx, entityID)
		}
		if detail != "" {
			report(progressUpdate{Message: fmt.Sprintf("Done %s (%d/%d)\n%s", entityID, i+1, len(entityIDs), indentLines(detail, "  ")), Current: i + 1, Total: len(entityIDs)})
			continue
		}
		report(progressUpdate{Message: fmt.Sprintf("Done %s (%d/%d)", entityID, i+1, len(entityIDs)), Current: i + 1, Total: len(entityIDs)})
	}
	result.Duration = time.Since(start)
	return batch
}

func indentLines(value string, prefix string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for i := range lines {
		lines[i] = prefix + strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

func joinResultErrors(result *syncflows.BatchRunResult) error {
	if result == nil || len(result.Errors) == 0 {
		return nil
	}
	return errors.Join(result.Errors...)
}
