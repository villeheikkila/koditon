package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/prices"
)

const (
	TaskTypePricesCitiesInit                 = "prices_cities_init"
	TaskTypePricesSync                       = "prices_sync"
	TaskTypePricesPostalCodeSync             = "prices_postal_code_sync"
	TaskTypePricesPostalCodePageSync         = "prices_postal_code_page_sync"
	TaskTypePricesNeighborhoodPostalCodeSync = "prices_neighborhood_postal_code_sync"
	TaskTypePricesSyncAll                    = "prices_sync_all"
	TaskTypePricesMatchSaleListingsBackfill  = "prices_match_sale_listings_backfill"
	TaskTypePricesMatchSaleListingsFanout    = "prices_match_sale_listings_fanout"
	TaskTypePricesMatchSaleListing           = "prices_match_sale_listing"

	pricesMatchInitialDelay = 7 * 24 * time.Hour
	pricesMatchRetryDelay   = 7 * 24 * time.Hour
	pricesMatchMaxAge       = 4 * 30 * 24 * time.Hour
)

type pricesPostalCodePayload struct {
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Page       int    `json:"page,omitempty"`
}

type pricesMatchFanoutPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type pricesMatchBackfillPayload struct {
	ScoreThreshold   int32 `json:"score_threshold,omitempty"`
	CompetitorMargin int32 `json:"competitor_margin,omitempty"`
}

type pricesMatchSaleListingPayload struct {
	ListingPublicID string `json:"listing_public_id"`
	Attempt         int32  `json:"attempt,omitempty"`
}

type pricesMatchSaleListingRow struct {
	PublicID      string
	LastSeenAt    *time.Time
	TransactionID *string
	Status        *string
	AttemptCount  int32
	ExpiresAt     *time.Time
}

type pricesMatchRunSummary struct {
	RunID      string
	Candidates int32
	AutoLinked int32
	Ambiguous  int32
}

func (c *Consumer) handlePricesTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "prices", logger, msg, c.runPricesSyncJob)
}

func (c *Consumer) handlePricesCitiesInit(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.cities_init"))
	logger.InfoContext(ctx, "prices cities initialization started")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	if len(cities) > 0 {
		pricesQueue := taskqueue.NewQueue(c.pool, "prices")
		var enqueueErrors int
		for _, city := range cities {
			if enqErr := c.enqueuePricesTask(ctx, pricesQueue, taskqueue.EntityPrefixCity+city, TaskTypePricesSync); enqErr != nil {
				enqueueErrors++
			}
		}
		logger.InfoContext(ctx, "prices city entities enqueued", "count", len(cities), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	}
	return nil
}

func (c *Consumer) handlePricesSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.prices.city_sync"))
	logger.InfoContext(ctx, "prices city sync started")
	if err := c.syncRunner.PricesSyncCityEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices city sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesNeighborhoodPostalCodeSync(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.neighborhood_postal_code_sync"))
	logger.InfoContext(ctx, "prices neighborhood postal code sync started")
	err := c.syncRunner.PricesSyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
		if p.Page > 0 {
			logger.DebugContext(ctx, "postal code transactions fetch started", "city", p.City, "postal_code", p.PostalCode, "page", p.Page)
		} else if p.Updated > 0 {
			logger.InfoContext(ctx, "neighborhood postal code mappings updated", "city", p.City, "postal_code", p.PostalCode, "updated", p.Updated)
		}
	})
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices neighborhood postal code sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesSyncAll(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.sync_all"))
	logger.InfoContext(ctx, "prices sync all fanout started")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	var enqueueErrors int
	for _, city := range cities {
		if err := c.enqueuePricesTask(ctx, nil, taskqueue.EntityPrefixCity+city, TaskTypePricesSync); err != nil {
			enqueueErrors++
		}
	}
	if enqueueErrors > 0 {
		return fmt.Errorf("enqueue prices sync all city jobs: %d errors", enqueueErrors)
	}
	logger.InfoContext(ctx, "prices sync all fanout completed", "cities", len(cities), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueuePricesTask(ctx context.Context, _ *taskqueue.Queue, entityID, taskType string) error {
	_, err := c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "prices",
		Kind:        taskType,
		EntityID:    entityID,
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
	})
	return err
}

func (c *Consumer) runPricesSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	switch job.SyncJobKind {
	case TaskTypePricesCitiesInit:
		return c.handlePricesCitiesInit(ctx, logger)
	case TaskTypePricesSync:
		return c.handleDurablePricesCitySync(ctx, logger, job)
	case TaskTypePricesPostalCodeSync:
		return c.handleDurablePricesPostalCodeSync(ctx, logger, job)
	case TaskTypePricesPostalCodePageSync:
		return c.handleDurablePricesPostalCodePageSync(ctx, logger, job)
	case TaskTypePricesNeighborhoodPostalCodeSync:
		return c.handlePricesNeighborhoodPostalCodeSync(ctx, logger)
	case TaskTypePricesSyncAll:
		return c.handlePricesSyncAll(ctx, logger)
	case TaskTypePricesMatchSaleListingsBackfill:
		return c.handlePricesMatchSaleListingsBackfill(ctx, logger, job)
	case TaskTypePricesMatchSaleListingsFanout:
		return c.handlePricesMatchSaleListingsFanout(ctx, logger, job)
	case TaskTypePricesMatchSaleListing:
		return c.handlePricesMatchSaleListing(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSourcesBackfill:
		return c.handleCanonicalMatchSaleListingSourcesBackfill(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSourcesFanout:
		return c.handleCanonicalMatchSaleListingSourcesFanout(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSource:
		return c.handleCanonicalMatchSaleListingSource(ctx, logger, job)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown prices sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}

func (c *Consumer) handleDurablePricesCitySync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.city_index"))
	logger.InfoContext(ctx, "prices city index started")
	entityType, cityName, err := parseJobEntity(job.SyncJobEntityID)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid prices city entity")
	}
	if entityType != "city" {
		return taskqueue.NewPermanentError(fmt.Errorf("expected city entity, got %s", entityType), "invalid prices city entity")
	}
	result, err := c.syncRunner.PricesSyncCityIndex(ctx, cityName, func(p prices.SyncCityProgress) {
		c.updatePricesCheckpoint(ctx, job, map[string]any{
			"city":       p.City,
			"step":       p.Step,
			"count":      p.Count,
			"details":    p.Details,
			"updated_at": time.Now().UTC(),
		})
	})
	if err != nil {
		return err
	}
	var enqueueErrors int
	for _, postalCode := range result.PostalCodes {
		payload, err := json.Marshal(pricesPostalCodePayload{City: result.City, PostalCode: postalCode, Page: 0})
		if err != nil {
			return fmt.Errorf("marshal prices postal code page payload: %w", err)
		}
		_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
			Provider:    "prices",
			Kind:        TaskTypePricesPostalCodePageSync,
			EntityID:    pricesPostalCodePageEntityID(result.City, postalCode, 0),
			Priority:    int32(taskqueue.PriorityNormal),
			MaxAttempts: 3,
			Payload:     payload,
		})
		if err != nil {
			enqueueErrors++
		}
	}
	if enqueueErrors > 0 {
		return fmt.Errorf("enqueue prices postal code jobs: %d errors", enqueueErrors)
	}
	logger.InfoContext(ctx, "prices city index completed", "city", result.City, "postal_codes", len(result.PostalCodes), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleDurablePricesPostalCodeSync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	var payload pricesPostalCodePayload
	if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("decode prices postal code payload: %w", err), "invalid payload")
	}
	payload.Page = 0
	return c.enqueuePricesPostalCodePage(ctx, payload)
}

func (c *Consumer) handleDurablePricesPostalCodePageSync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.postal_code_sync"))
	var payload pricesPostalCodePayload
	if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("decode prices postal code page payload: %w", err), "invalid payload")
	}
	if payload.City == "" || payload.PostalCode == "" {
		return taskqueue.NewPermanentError(fmt.Errorf("city and postal_code are required"), "invalid payload")
	}
	logger.InfoContext(ctx, "prices postal code page sync started", "city", payload.City, "postal_code", payload.PostalCode, "page", payload.Page)
	result, err := c.syncRunner.PricesSyncPostalCodeTransactionPage(ctx, payload.City, payload.PostalCode, payload.Page, func(p prices.SyncPostalCodeProgress) {
		c.updatePricesCheckpoint(ctx, job, map[string]any{
			"city":         p.City,
			"postal_code":  p.PostalCode,
			"page":         p.Page,
			"transactions": p.Transactions,
			"upserted":     p.Upserted,
			"updated_at":   time.Now().UTC(),
		})
	})
	if err != nil {
		return err
	}
	if result.NextPage != nil {
		payload.Page = *result.NextPage
		if err := c.enqueuePricesPostalCodePage(ctx, payload); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "prices postal code page sync completed", "city", payload.City, "postal_code", payload.PostalCode, "page", result.Page, "next_page", result.NextPage, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueuePricesPostalCodePage(ctx context.Context, payload pricesPostalCodePayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal prices postal code page payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "prices",
		Kind:        TaskTypePricesPostalCodePageSync,
		EntityID:    pricesPostalCodePageEntityID(payload.City, payload.PostalCode, payload.Page),
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
		Payload:     raw,
	})
	return err
}

func (c *Consumer) handlePricesMatchSaleListingsBackfill(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.match_sale_listings_backfill"))
	payload := pricesMatchBackfillPayload{ScoreThreshold: 90, CompetitorMargin: 15}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode prices match backfill payload: %w", err), "invalid payload")
		}
	}
	if payload.ScoreThreshold <= 0 {
		payload.ScoreThreshold = 90
	}
	if payload.CompetitorMargin < 0 {
		payload.CompetitorMargin = 15
	}
	run, err := c.runPricesMatchBackfill(ctx, int(payload.ScoreThreshold), int(payload.CompetitorMargin))
	if err != nil {
		return err
	}
	result, err := json.Marshal(map[string]any{
		"run_id":      run.RunID,
		"candidates":  run.Candidates,
		"auto_linked": run.AutoLinked,
		"ambiguous":   run.Ambiguous,
	})
	if err == nil {
		c.updatePricesCheckpoint(ctx, job, map[string]any{
			"run_id":      run.RunID,
			"candidates":  run.Candidates,
			"auto_linked": run.AutoLinked,
			"ambiguous":   run.Ambiguous,
			"updated_at":  time.Now().UTC(),
			"result":      json.RawMessage(result),
		})
	}
	logger.InfoContext(ctx, "prices sale listing backfill matched", "run_id", run.RunID, "candidates", run.Candidates, "auto_linked", run.AutoLinked, "ambiguous", run.Ambiguous, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesMatchSaleListingsFanout(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.match_sale_listings_fanout"))
	payload := pricesMatchFanoutPayload{Limit: 5000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode prices match fanout payload: %w", err), "invalid payload")
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 5000
	}
	rows, err := c.pool.Query(ctx, `
SELECT sale_listing_public_id, COALESCE(sale_listing_prices_match_attempt_count, 0)
FROM public.sale_listings
WHERE sale_listing_source_kind = 'ad'
    AND prices_transaction_id IS NULL
    AND sale_listing_last_seen_at IS NOT NULL
    AND sale_listing_last_seen_at <= now() - interval '7 days'
    AND sale_listing_last_seen_at >= now() - interval '4 months'
    AND COALESCE(sale_listing_prices_match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(sale_listing_prices_match_next_attempt_at, sale_listing_last_seen_at + interval '7 days') <= now()
ORDER BY COALESCE(sale_listing_prices_match_next_attempt_at, sale_listing_last_seen_at + interval '7 days'), sale_listing_last_seen_at
LIMIT $1`, payload.Limit)
	if err != nil {
		return fmt.Errorf("list sale listings for prices matching: %w", err)
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var publicID string
		var attempt int32
		if err := rows.Scan(&publicID, &attempt); err != nil {
			return fmt.Errorf("scan sale listing match fanout row: %w", err)
		}
		if err := c.enqueuePricesMatchSaleListing(ctx, publicID, attempt+1, time.Now()); err != nil {
			return err
		}
		enqueued++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate sale listing match fanout rows: %w", err)
	}
	logger.InfoContext(ctx, "prices sale listing match jobs enqueued", "count", enqueued, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesMatchSaleListing(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.match_sale_listing"))
	payload, err := decodePricesMatchSaleListingPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	row, err := c.loadPricesMatchSaleListing(ctx, payload.ListingPublicID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	now := time.Now().UTC()
	if row.TransactionID != nil {
		return c.updatePricesMatchState(ctx, row.PublicID, "auto_linked", nil, nil, nil)
	}
	if row.LastSeenAt == nil {
		next := now.Add(pricesMatchRetryDelay)
		if err := c.updatePricesMatchState(ctx, row.PublicID, "deferred", &next, nil, nil); err != nil {
			return err
		}
		return c.enqueuePricesMatchSaleListing(ctx, row.PublicID, row.AttemptCount+2, next)
	}
	firstEligible := row.LastSeenAt.Add(pricesMatchInitialDelay)
	expiresAt := row.LastSeenAt.Add(pricesMatchMaxAge)
	if now.Before(firstEligible) {
		if err := c.updatePricesMatchState(ctx, row.PublicID, "deferred", &firstEligible, nil, &expiresAt); err != nil {
			return err
		}
		return c.enqueuePricesMatchSaleListing(ctx, row.PublicID, row.AttemptCount+2, firstEligible)
	}
	if now.After(expiresAt) {
		return c.updatePricesMatchState(ctx, row.PublicID, "expired", nil, nil, &expiresAt)
	}
	run, err := c.runPricesMatchForSaleListing(ctx, row.PublicID)
	if err != nil {
		return err
	}
	if run.AutoLinked > 0 {
		logger.InfoContext(ctx, "prices sale listing auto-linked", "listing", row.PublicID, "run_id", run.RunID, "outcome", logging.OutcomeSuccess)
		return c.updatePricesMatchState(ctx, row.PublicID, "auto_linked", nil, &run.RunID, &expiresAt)
	}
	if run.Ambiguous > 0 {
		logger.InfoContext(ctx, "prices sale listing needs review", "listing", row.PublicID, "run_id", run.RunID, "candidates", run.Ambiguous)
		return c.updatePricesMatchState(ctx, row.PublicID, "needs_review", nil, &run.RunID, &expiresAt)
	}
	next := now.Add(pricesMatchRetryDelay)
	if next.After(expiresAt) {
		return c.updatePricesMatchState(ctx, row.PublicID, "expired", nil, &run.RunID, &expiresAt)
	}
	if err := c.updatePricesMatchState(ctx, row.PublicID, "deferred", &next, &run.RunID, &expiresAt); err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices sale listing match deferred", "listing", row.PublicID, "next_attempt_at", next)
	return c.enqueuePricesMatchSaleListing(ctx, row.PublicID, row.AttemptCount+2, next)
}

func decodePricesMatchSaleListingPayload(job db.SyncJob) (pricesMatchSaleListingPayload, error) {
	var payload pricesMatchSaleListingPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return pricesMatchSaleListingPayload{}, fmt.Errorf("decode prices sale listing match payload: %w", err)
		}
	}
	if payload.ListingPublicID == "" {
		_, value, err := parseJobEntity(job.SyncJobEntityID)
		if err != nil {
			return pricesMatchSaleListingPayload{}, fmt.Errorf("parse listing entity: %w", err)
		}
		payload.ListingPublicID = strings.TrimSpace(value)
	}
	if payload.ListingPublicID == "" {
		return pricesMatchSaleListingPayload{}, fmt.Errorf("listing_public_id is required")
	}
	return payload, nil
}

func (c *Consumer) loadPricesMatchSaleListing(ctx context.Context, publicID string) (pricesMatchSaleListingRow, error) {
	var row pricesMatchSaleListingRow
	var transactionID *string
	err := c.pool.QueryRow(ctx, `
SELECT
    sale_listing_public_id,
    sale_listing_last_seen_at,
    prices_transaction_id::text,
    sale_listing_prices_match_status,
    sale_listing_prices_match_attempt_count,
    sale_listing_prices_match_expires_at
FROM public.sale_listings
WHERE sale_listing_public_id = $1`, publicID).Scan(&row.PublicID, &row.LastSeenAt, &transactionID, &row.Status, &row.AttemptCount, &row.ExpiresAt)
	row.TransactionID = transactionID
	return row, err
}

func (c *Consumer) runPricesMatchForSaleListing(ctx context.Context, publicID string) (pricesMatchRunSummary, error) {
	var runID string
	if err := c.pool.QueryRow(ctx, `SELECT public.fnc__refresh_sale_listing_prices_transaction_matches(true, 90, 15, $1)::text`, publicID).Scan(&runID); err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("run prices sale listing match: %w", err)
	}
	var summary pricesMatchRunSummary
	err := c.pool.QueryRow(ctx, `
SELECT
    sale_listing_prices_transaction_match_run_id::text,
    sale_listing_prices_transaction_match_candidates_count,
    sale_listing_prices_transaction_match_auto_linked_count,
    sale_listing_prices_transaction_match_ambiguous_count
FROM public.sale_listing_prices_transaction_match_runs
WHERE sale_listing_prices_transaction_match_run_id = $1::uuid`, runID).Scan(&summary.RunID, &summary.Candidates, &summary.AutoLinked, &summary.Ambiguous)
	if err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("load prices sale listing match run: %w", err)
	}
	return summary, nil
}

func (c *Consumer) runPricesMatchBackfill(ctx context.Context, scoreThreshold, competitorMargin int) (pricesMatchRunSummary, error) {
	var runID string
	if err := c.pool.QueryRow(ctx, `SELECT public.fnc__refresh_sale_listing_prices_transaction_matches(true, $1, $2, NULL)::text`, scoreThreshold, competitorMargin).Scan(&runID); err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("run prices sale listing match backfill: %w", err)
	}
	var summary pricesMatchRunSummary
	err := c.pool.QueryRow(ctx, `
SELECT
    sale_listing_prices_transaction_match_run_id::text,
    sale_listing_prices_transaction_match_candidates_count,
    sale_listing_prices_transaction_match_auto_linked_count,
    sale_listing_prices_transaction_match_ambiguous_count
FROM public.sale_listing_prices_transaction_match_runs
WHERE sale_listing_prices_transaction_match_run_id = $1::uuid`, runID).Scan(&summary.RunID, &summary.Candidates, &summary.AutoLinked, &summary.Ambiguous)
	if err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("load prices sale listing match backfill run: %w", err)
	}
	return summary, nil
}

func (c *Consumer) updatePricesMatchState(ctx context.Context, publicID, status string, nextAttemptAt *time.Time, runID *string, expiresAt *time.Time) error {
	_, err := c.pool.Exec(ctx, `
UPDATE public.sale_listings
SET
    sale_listing_prices_match_status = $2,
    sale_listing_prices_match_next_attempt_at = $3,
    sale_listing_prices_match_last_attempted_at = now(),
    sale_listing_prices_match_attempt_count = sale_listing_prices_match_attempt_count + 1,
    sale_listing_prices_match_run_id = COALESCE($4::uuid, sale_listing_prices_match_run_id),
    sale_listing_prices_match_expires_at = COALESCE($5, sale_listing_prices_match_expires_at),
    sale_listing_updated_at = now()
WHERE sale_listing_public_id = $1`, publicID, status, nextAttemptAt, runID, expiresAt)
	if err != nil {
		return fmt.Errorf("update prices match state: %w", err)
	}
	return nil
}

func (c *Consumer) enqueuePricesMatchSaleListing(ctx context.Context, publicID string, attempt int32, runAfter time.Time) error {
	payload, err := json.Marshal(pricesMatchSaleListingPayload{ListingPublicID: publicID, Attempt: attempt})
	if err != nil {
		return fmt.Errorf("marshal prices sale listing match payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "prices",
		Kind:        TaskTypePricesMatchSaleListing,
		EntityID:    fmt.Sprintf("listing:%s:attempt:%d", publicID, attempt),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}

func pricesPostalCodePageEntityID(city, postalCode string, page int) string {
	return fmt.Sprintf("city:%s:postal_code:%s:page:%d", city, postalCode, page)
}

func parseJobEntity(entityID string) (string, string, error) {
	entityType, value, ok := strings.Cut(entityID, ":")
	if !ok || entityType == "" || value == "" {
		return "", "", fmt.Errorf("expected type:value entity id")
	}
	return entityType, value, nil
}

func (c *Consumer) updatePricesCheckpoint(ctx context.Context, job db.SyncJob, value map[string]any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.syncJobs.UpdateCheckpoint(ctx, job, raw)
}
