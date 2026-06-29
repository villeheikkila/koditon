package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

var pricesWorkflowKinds = []string{
	TaskTypePricesMatchSaleListingsBackfill,
	TaskTypePricesMatchSaleListingsFanout,
	TaskTypePricesMatchSaleListing,
}

type pricesFanoutResult struct {
	Enqueued int `json:"enqueued"`
}

type pricesMatchListingWorkflowResult struct {
	SaleListingID string                 `json:"sale_listing_id"`
	Status        string                 `json:"status"`
	Run           *pricesMatchRunSummary `json:"run,omitempty"`
}

type pricesMatchLoadedListing struct {
	Found bool                      `json:"found"`
	Row   pricesMatchSaleListingRow `json:"row"`
}

func (c *Consumer) startPricesWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.pricesWorkflowClient == nil {
		return errors.New("prices absurd workflow client is not configured")
	}
	for _, kind := range pricesWorkflowKinds {
		def, ok := workflows.FindDefinition(kind)
		if !ok {
			return fmt.Errorf("missing prices workflow definition: %s", kind)
		}
		task := absurd.Task[json.RawMessage, json.RawMessage](
			kind,
			c.handlePricesWorkflow,
			absurd.TaskOptions{QueueName: workflows.QueuePrices, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation},
		)
		if err := c.pricesWorkflowClient.Register(task); err != nil {
			return fmt.Errorf("register prices workflow %s: %w", kind, err)
		}
	}
	logger := logging.With(c.logger, logging.Op("consumer.prices.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.pricesWorkflowCancel = cancel
	c.pricesWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.pricesWorkflowDone)
		logger.InfoContext(workerCtx, "prices absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueuePrices)
		err := c.pricesWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "prices",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "prices absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "prices absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
	return nil
}

func (c *Consumer) handlePricesWorkflow(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	taskName := absurd.MustTaskContext(ctx).TaskName()
	logger := logging.With(c.logger,
		logging.Op("consumer.prices.workflow"),
		slog.String("task_type", taskName),
	)
	var result any
	var err error
	switch taskName {
	case TaskTypePricesMatchSaleListingsBackfill:
		result, err = c.runPricesMatchSaleListingsBackfillWorkflow(ctx, logger, params)
	case TaskTypePricesMatchSaleListingsFanout:
		result, err = c.runPricesMatchSaleListingsFanoutWorkflow(ctx, logger, params)
	case TaskTypePricesMatchSaleListing:
		result, err = c.runPricesMatchSaleListingWorkflow(ctx, logger, params)
	default:
		return nil, fmt.Errorf("unknown prices workflow kind: %s", taskName)
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal prices workflow result: %w", err)
	}
	return raw, nil
}

func (c *Consumer) runPricesMatchSaleListingsBackfillWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (pricesMatchRunSummary, error) {
	payload, err := decodePricesMatchBackfillWorkflowPayload(params)
	if err != nil {
		return pricesMatchRunSummary{}, err
	}
	run, err := absurd.Step(ctx, "run-sale-listing-match-backfill", func(ctx context.Context) (pricesMatchRunSummary, error) {
		return c.runPricesMatchBackfill(ctx, int(payload.ScoreThreshold), int(payload.CompetitorMargin))
	})
	if err != nil {
		return pricesMatchRunSummary{}, err
	}
	logger.InfoContext(ctx, "prices sale listing backfill matched", "run_id", run.RunID, "candidates", run.Candidates, "auto_linked", run.AutoLinked, "ambiguous", run.Ambiguous, "outcome", logging.OutcomeSuccess)
	return run, nil
}

func (c *Consumer) runPricesMatchSaleListingsFanoutWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (pricesFanoutResult, error) {
	payload, err := decodePricesMatchFanoutWorkflowPayload(params)
	if err != nil {
		return pricesFanoutResult{}, err
	}
	return absurd.Step(ctx, "scan-and-spawn-listings", func(ctx context.Context) (pricesFanoutResult, error) {
		rows, err := c.pool.Query(ctx, `
SELECT sale_listing_id::text, COALESCE(sale_listing_prices_match_attempt_count, 0)
FROM public.property_source_offerings
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
			return pricesFanoutResult{}, fmt.Errorf("list sale listings for prices matching: %w", err)
		}
		defer rows.Close()
		enqueued := 0
		for rows.Next() {
			var saleListingID string
			var attempt int32
			if err := rows.Scan(&saleListingID, &attempt); err != nil {
				return pricesFanoutResult{}, fmt.Errorf("scan sale listing match fanout row: %w", err)
			}
			if err := c.spawnPricesMatchSaleListing(ctx, saleListingID, attempt+1); err != nil {
				return pricesFanoutResult{}, err
			}
			enqueued++
		}
		if err := rows.Err(); err != nil {
			return pricesFanoutResult{}, fmt.Errorf("iterate sale listing match fanout rows: %w", err)
		}
		logger.InfoContext(ctx, "prices sale listing match tasks spawned", "count", enqueued, "outcome", logging.OutcomeSuccess)
		return pricesFanoutResult{Enqueued: enqueued}, nil
	})
}

func (c *Consumer) runPricesMatchSaleListingWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (pricesMatchListingWorkflowResult, error) {
	payload, err := decodePricesMatchSaleListingWorkflowPayload(params)
	if err != nil {
		return pricesMatchListingWorkflowResult{}, err
	}
	for {
		loaded, err := absurd.Step(ctx, "load-listing", func(ctx context.Context) (pricesMatchLoadedListing, error) {
			row, err := c.loadPricesMatchSaleListing(ctx, payload.SaleListingID)
			if errors.Is(err, pgx.ErrNoRows) {
				return pricesMatchLoadedListing{}, nil
			}
			return pricesMatchLoadedListing{Found: true, Row: row}, err
		})
		if err != nil {
			return pricesMatchListingWorkflowResult{}, err
		}
		if !loaded.Found {
			return pricesMatchListingWorkflowResult{SaleListingID: payload.SaleListingID, Status: "not_found"}, nil
		}
		row := loaded.Row
		if row.LastSeenAt == nil && row.TransactionID == nil {
			if _, err := c.deferPricesMatchSaleListingWorkflow(ctx, row.ID, "defer-missing-last-seen", time.Now().UTC().Add(pricesMatchRetryDelay), nil); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			continue
		}
		now := time.Now().UTC()
		if row.TransactionID != nil {
			if err := c.updatePricesMatchStateStep(ctx, "mark-already-auto-linked", row.ID, "auto_linked", nil, nil, nil); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			return pricesMatchListingWorkflowResult{SaleListingID: row.ID, Status: "auto_linked"}, nil
		}
		firstEligible := row.LastSeenAt.Add(pricesMatchInitialDelay)
		expiresAt := row.LastSeenAt.Add(pricesMatchMaxAge)
		if now.Before(firstEligible) {
			result, err := c.deferPricesMatchSaleListingWorkflow(ctx, row.ID, "defer-until-eligible", firstEligible, &expiresAt)
			if err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			if result.Status == "deferred" {
				continue
			}
			return result, nil
		}
		if now.After(expiresAt) {
			if err := c.updatePricesMatchStateStep(ctx, "mark-expired", row.ID, "expired", nil, nil, &expiresAt); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			return pricesMatchListingWorkflowResult{SaleListingID: row.ID, Status: "expired"}, nil
		}
		run, err := absurd.Step(ctx, "run-sale-listing-match", func(ctx context.Context) (pricesMatchRunSummary, error) {
			return c.runPricesMatchForSaleListing(ctx, row.ID)
		})
		if err != nil {
			return pricesMatchListingWorkflowResult{}, err
		}
		if run.AutoLinked > 0 {
			logger.InfoContext(ctx, "prices sale listing auto-linked", "sale_listing_id", row.ID, "run_id", run.RunID, "outcome", logging.OutcomeSuccess)
			if err := c.updatePricesMatchStateStep(ctx, "mark-auto-linked", row.ID, "auto_linked", nil, &run.RunID, &expiresAt); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			return pricesMatchListingWorkflowResult{SaleListingID: row.ID, Status: "auto_linked", Run: &run}, nil
		}
		if run.Ambiguous > 0 {
			logger.InfoContext(ctx, "prices sale listing needs review", "sale_listing_id", row.ID, "run_id", run.RunID, "candidates", run.Ambiguous)
			if err := c.updatePricesMatchStateStep(ctx, "mark-needs-review", row.ID, "needs_review", nil, &run.RunID, &expiresAt); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			return pricesMatchListingWorkflowResult{SaleListingID: row.ID, Status: "needs_review", Run: &run}, nil
		}
		next := now.Add(pricesMatchRetryDelay)
		if next.After(expiresAt) {
			if err := c.updatePricesMatchStateStep(ctx, "mark-expired-after-run", row.ID, "expired", nil, &run.RunID, &expiresAt); err != nil {
				return pricesMatchListingWorkflowResult{}, err
			}
			return pricesMatchListingWorkflowResult{SaleListingID: row.ID, Status: "expired", Run: &run}, nil
		}
		if err := c.updatePricesMatchStateStep(ctx, "mark-deferred-after-run", row.ID, "deferred", &next, &run.RunID, &expiresAt); err != nil {
			return pricesMatchListingWorkflowResult{}, err
		}
		logger.InfoContext(ctx, "prices sale listing match sleeping until retry", "sale_listing_id", row.ID, "next_attempt_at", next)
		if err := absurd.SleepUntil(ctx, "sleep-until-retry", next); err != nil {
			return pricesMatchListingWorkflowResult{}, err
		}
	}
}

func (c *Consumer) deferPricesMatchSaleListingWorkflow(ctx context.Context, saleListingID string, stepName string, wakeAt time.Time, expiresAt *time.Time) (pricesMatchListingWorkflowResult, error) {
	if err := c.updatePricesMatchStateStep(ctx, "mark-"+stepName, saleListingID, "deferred", &wakeAt, nil, expiresAt); err != nil {
		return pricesMatchListingWorkflowResult{}, err
	}
	if err := absurd.SleepUntil(ctx, stepName, wakeAt); err != nil {
		return pricesMatchListingWorkflowResult{}, err
	}
	return pricesMatchListingWorkflowResult{SaleListingID: saleListingID, Status: "deferred"}, nil
}

func (c *Consumer) updatePricesMatchStateStep(ctx context.Context, stepName string, saleListingID, status string, nextAttemptAt *time.Time, runID *string, expiresAt *time.Time) error {
	_, err := absurd.Step(ctx, stepName, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.updatePricesMatchState(ctx, saleListingID, status, nextAttemptAt, runID, expiresAt)
	})
	return err
}

func decodePricesMatchBackfillWorkflowPayload(raw json.RawMessage) (pricesMatchBackfillPayload, error) {
	payload := pricesMatchBackfillPayload{ScoreThreshold: 90, CompetitorMargin: 15}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return pricesMatchBackfillPayload{}, fmt.Errorf("decode prices match backfill payload: %w", err)
		}
	}
	if payload.ScoreThreshold <= 0 {
		payload.ScoreThreshold = 90
	}
	if payload.CompetitorMargin < 0 {
		payload.CompetitorMargin = 15
	}
	return payload, nil
}

func decodePricesMatchFanoutWorkflowPayload(raw json.RawMessage) (pricesMatchFanoutPayload, error) {
	payload := pricesMatchFanoutPayload{Limit: 5000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return pricesMatchFanoutPayload{}, fmt.Errorf("decode prices match fanout payload: %w", err)
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 5000
	}
	return payload, nil
}

func decodePricesMatchSaleListingWorkflowPayload(raw json.RawMessage) (pricesMatchSaleListingPayload, error) {
	var payload pricesMatchSaleListingPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return pricesMatchSaleListingPayload{}, fmt.Errorf("decode prices sale listing match payload: %w", err)
		}
	}
	payload.SaleListingID = strings.TrimSpace(payload.SaleListingID)
	if payload.SaleListingID == "" {
		return pricesMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id is required")
	}
	if _, err := uuid.Parse(payload.SaleListingID); err != nil {
		return pricesMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id must be a uuid: %w", err)
	}
	return payload, nil
}

func (c *Consumer) spawnPricesMatchSaleListing(ctx context.Context, saleListingID string, attempt int32) error {
	payload, err := json.Marshal(pricesMatchSaleListingPayload{SaleListingID: saleListingID, Attempt: attempt})
	if err != nil {
		return fmt.Errorf("marshal prices sale listing match payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.pricesWorkflowClient, workflows.SpawnTaskRequest{
		TaskName:     TaskTypePricesMatchSaleListing,
		Params:       payload,
		Cancellation: &absurd.CancellationPolicy{MaxDuration: int64((pricesMatchMaxAge + pricesMatchRetryDelay).Seconds())},
	})
	return err
}
