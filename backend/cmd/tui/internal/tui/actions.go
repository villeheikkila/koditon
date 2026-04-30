package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	syncflows "koditon/internal/sync/flows"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/prices"
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
	Run           func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error)
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
					Run: func(_ context.Context, _ *appContext, _ []string, _ reportFn) (actionResult, error) {
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
					Description: "Queue frontdoor sitemap sync and watch the durable job",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "frontdoor", Kind: "frontdoor_sitemap_sync", EntityID: "frontdoor:sitemap"}})
					},
				},
				{
					Title:       "Sitemap Discover",
					Description: "Fetch frontdoor sitemap and report discovered entities without syncing",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching frontdoor sitemap..."})
						adIDs, buildingIDs, err := app.runner.FrontdoorSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: fmt.Sprintf("discovered ads=%d buildings=%d total=%d", len(adIDs), len(buildingIDs), len(adIDs)+len(buildingIDs))}, nil
					},
				},
				{
					Title:       "Sync Buildings",
					Description: "Queue frontdoor building-only sitemap fanout",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "frontdoor", Kind: "frontdoor_buildings_sitemap_sync", EntityID: "frontdoor:buildings_sitemap"}})
					},
				},
				{
					Title:       "Sync Ad by ID",
					Description: "Queue one frontdoor ad sync by friendly ID",
					Prompts:     []string{"friendly ad id"},
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
						entityID := "ad:" + strings.TrimSpace(inputs[0])
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "frontdoor", Kind: "frontdoor_sync", EntityID: entityID}})
					},
				},
				{
					Title:       "Sync Building by ID",
					Description: "Queue one frontdoor building sync by housing company ID",
					Prompts:     []string{"building housing company id"},
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
						entityID := "building:" + strings.TrimSpace(inputs[0])
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "frontdoor", Kind: "frontdoor_sync", EntityID: entityID}})
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
					Description: "Queue shortcut sitemap sync and watch the durable job",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "shortcut", Kind: "shortcut_sitemap_sync", EntityID: "shortcut:sitemap"}})
					},
				},
				{
					Title:       "Sitemap Discover",
					Description: "Fetch shortcut sitemap and report discovered entities without syncing",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching shortcut sitemap..."})
						buildingIDs, adIDs, err := app.runner.ShortcutSitemap(ctx)
						if err != nil {
							return actionResult{}, err
						}
						return actionResult{Output: fmt.Sprintf("discovered buildings=%d ads=%d total=%d", len(buildingIDs), len(adIDs), len(buildingIDs)+len(adIDs))}, nil
					},
				},
				{
					Title:       "Sync Buildings",
					Description: "Queue shortcut building-only sitemap fanout",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "shortcut", Kind: "shortcut_buildings_sitemap_sync", EntityID: "shortcut:buildings_sitemap"}})
					},
				},
				{
					Title:       "Sync Ad by ID",
					Description: "Queue one shortcut ad sync by numeric ID",
					Prompts:     []string{"ad id"},
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
						entityID := "ad:" + strings.TrimSpace(inputs[0])
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "shortcut", Kind: "shortcut_scraper_sync", EntityID: entityID}})
					},
				},
				{
					Title:       "Sync Building by UUID",
					Description: "Queue one shortcut building sync by UUID",
					Prompts:     []string{"building uuid"},
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
						entityID := "building:" + strings.TrimSpace(inputs[0])
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "shortcut", Kind: "shortcut_scraper_sync", EntityID: entityID}})
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
					Description: "Queue prices city initialization and watch the durable job",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "prices", Kind: "prices_cities_init", EntityID: "prices:cities"}})
					},
				},
				{
					Title:       "Cities Discover",
					Description: "Fetch available prices cities without syncing",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						report(progressUpdate{Message: "Fetching prices cities..."})
						cities, err := app.runner.PricesFetchCities(ctx)
						if err != nil {
							return actionResult{}, err
						}
						report(progressUpdate{Message: fmt.Sprintf("Discovered %d cities", len(cities))})
						return actionResult{Output: fmt.Sprintf("cities=%d list=%s", len(cities), strings.Join(cities, ", "))}, nil
					},
				},
				{
					Title:       "Sync All",
					Description: "Queue prices sync-all fanout",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "prices", Kind: "prices_sync_all", EntityID: "prices:sync_all"}})
					},
				},
				{
					Title:       "Neighborhood Postal Code Sync",
					Description: "Queue neighborhood->postal code mapping sync",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "prices", Kind: "prices_neighborhood_postal_code_sync", EntityID: "prices:neighborhood_postal_codes"}})
					},
				},
				{
					Title:       "Queue Sale Listing Match Fanout",
					Description: "Enqueue weekly transaction matching jobs for closed sale listings",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "prices", Kind: "prices_match_sale_listings_fanout", EntityID: "prices:match_sale_listings"}})
					},
				},
				{
					Title:         "Sync City by Name",
					Description:   "Queue prices data sync for one city",
					Prompts:       []string{"city name"},
					UseCityPicker: true,
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
						entityID := "city:" + strings.TrimSpace(inputs[0])
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "prices", Kind: "prices_sync", EntityID: entityID}})
					},
				},
				{
					Title:         "Search Transactions",
					Description:   "Targeted search with city/postal query, area filters and sorting",
					Prompts:       []string{"city name"},
					UseCityPicker: true,
					BuildInput:    newTransactionsSearchFormScreen,
					Run: func(ctx context.Context, app *appContext, inputs []string, report reportFn) (actionResult, error) {
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
						rows, err := app.runner.PricesSearchTransactionsByCityAndAddress(ctx, city, search, limit)
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
				{
					Title:       "Review Sale Listing Matches",
					Description: "Auto-link obvious transaction matches, then resolve ambiguous sale listings",
					BuildInput: func(ctx *appContext, _ action, _ []string, breadcrumb string) Screen {
						return newPricesMatchReviewScreen(ctx, breadcrumb+" > Match Review")
					},
					Run: func(_ context.Context, _ *appContext, _ []string, _ reportFn) (actionResult, error) {
						return actionResult{Output: "prices match review opened"}, nil
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
					Description: "Queue postal data sync",
					Run: func(ctx context.Context, app *appContext, _ []string, report reportFn) (actionResult, error) {
						return enqueueAndWatchSyncJobs(ctx, app, report, []syncjobs.EnqueueRequest{{Provider: "postal", Kind: "postal_sync", EntityID: "postal:all"}})
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
