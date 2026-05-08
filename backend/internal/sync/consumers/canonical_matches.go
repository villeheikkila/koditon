package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

const (
	TaskTypeCanonicalMatchSaleListingSourcesBackfill = "canonical_match_sale_listing_sources_backfill"
	TaskTypeCanonicalMatchSaleListingSourcesFanout   = "canonical_match_sale_listing_sources_fanout"
	TaskTypeCanonicalMatchSaleListingSource          = "canonical_match_sale_listing_source"
)

type canonicalMatchFanoutPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type canonicalMatchBackfillPayload struct {
	ScoreThreshold   int32 `json:"score_threshold,omitempty"`
	CompetitorMargin int32 `json:"competitor_margin,omitempty"`
}

type canonicalMatchSaleListingPayload struct {
	SaleListingID string `json:"sale_listing_id"`
	Attempt       int32  `json:"attempt,omitempty"`
}

type canonicalMatchSaleListingRow struct {
	ID           string
	LinkMethod   *string
	LinkStatus   *string
	Status       *string
	AttemptCount int32
}

type canonicalMatchRunSummary struct {
	RunID      string
	Candidates int32
	AutoLinked int32
	Ambiguous  int32
}

func (c *Consumer) handleCanonicalMatchSaleListingSourcesBackfill(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.match_sale_listing_sources_backfill"))
	payload := canonicalMatchBackfillPayload{ScoreThreshold: 95, CompetitorMargin: 10}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode canonical source match backfill payload: %w", err), "invalid payload")
		}
	}
	if payload.ScoreThreshold <= 0 {
		payload.ScoreThreshold = 95
	}
	if payload.CompetitorMargin < 0 {
		payload.CompetitorMargin = 10
	}
	run, err := c.runCanonicalSourceMatchBackfill(ctx, int(payload.ScoreThreshold), int(payload.CompetitorMargin))
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
	logger.InfoContext(ctx, "canonical sale listing source backfill matched", "run_id", run.RunID, "candidates", run.Candidates, "auto_linked", run.AutoLinked, "ambiguous", run.Ambiguous, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalMatchSaleListingSourcesFanout(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.match_sale_listing_sources_fanout"))
	payload := canonicalMatchFanoutPayload{Limit: 5000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode canonical source match fanout payload: %w", err), "invalid payload")
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 5000
	}
	rows, err := c.pool.Query(ctx, `
SELECT sl.sale_listing_id::text, COALESCE(sl.sale_listing_source_match_attempt_count, 0)
FROM public.property_source_offerings sl
JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
WHERE sl.sale_listing_source_kind = 'ad'
    AND pos.property_offering_source_link_status <> 'rejected'
    AND pos.property_offering_source_link_method <> 'manual'
    AND COALESCE(sl.sale_listing_source_match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at) <= now()
ORDER BY COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at), sl.sale_listing_updated_at
LIMIT $1`, payload.Limit)
	if err != nil {
		return fmt.Errorf("list sale listings for canonical source matching: %w", err)
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var saleListingID string
		var attempt int32
		if err := rows.Scan(&saleListingID, &attempt); err != nil {
			return fmt.Errorf("scan canonical source match fanout row: %w", err)
		}
		if err := c.enqueueCanonicalSourceMatchSaleListing(ctx, saleListingID, attempt+1, time.Now()); err != nil {
			return err
		}
		enqueued++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate canonical source match fanout rows: %w", err)
	}
	logger.InfoContext(ctx, "canonical sale listing source match jobs enqueued", "count", enqueued, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalMatchSaleListingSource(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.match_sale_listing_source"))
	payload, err := decodeCanonicalMatchSaleListingPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	row, err := c.loadCanonicalMatchSaleListing(ctx, payload.SaleListingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if row.LinkMethod != nil && *row.LinkMethod == "manual" {
		if err := c.updateCanonicalSourceMatchState(ctx, row.ID, "manual_linked", nil, nil); err != nil {
			return err
		}
		return c.projectTypedHousingCompanyProfileForSaleListing(ctx, row.ID)
	}
	run, err := c.runCanonicalSourceMatchForSaleListing(ctx, row.ID)
	if err != nil {
		return err
	}
	if run.AutoLinked > 0 {
		logger.InfoContext(ctx, "canonical sale listing source auto-linked", "sale_listing_id", row.ID, "run_id", run.RunID, "outcome", logging.OutcomeSuccess)
		if err := c.updateCanonicalSourceMatchState(ctx, row.ID, "auto_linked", nil, &run.RunID); err != nil {
			return err
		}
		return c.projectTypedHousingCompanyProfileForSaleListing(ctx, row.ID)
	}
	if run.Ambiguous > 0 {
		logger.InfoContext(ctx, "canonical sale listing source needs review", "sale_listing_id", row.ID, "run_id", run.RunID, "candidates", run.Ambiguous)
		return c.updateCanonicalSourceMatchState(ctx, row.ID, "needs_review", nil, &run.RunID)
	}
	next := time.Now().UTC().Add(7 * 24 * time.Hour)
	if run.Candidates == 0 {
		return c.updateCanonicalSourceMatchState(ctx, row.ID, "noop", &next, &run.RunID)
	}
	return c.updateCanonicalSourceMatchState(ctx, row.ID, "deferred", &next, &run.RunID)
}

func (c *Consumer) projectTypedHousingCompanyProfileForSaleListing(ctx context.Context, saleListingID string) error {
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return fmt.Errorf("parse sale listing id for typed projection: %w", err)
	}
	if err := c.queries.EnsurePhysicalBuildingForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("ensure physical building: %w", err)
	}
	if err := c.queries.UpsertSaleListingProviderClaims(ctx, id); err != nil {
		return fmt.Errorf("upsert provider property claims: %w", err)
	}
	if err := c.queries.ProjectApartmentProfileForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project apartment profile: %w", err)
	}
	if err := c.queries.ProjectHousingCompanyRenovationsForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project housing company renovations: %w", err)
	}
	if err := c.queries.ProjectHousingCompanySystemsFromRenovationsForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project housing company systems from renovations: %w", err)
	}
	if err := c.queries.ProjectBuildingProfileForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project building profile: %w", err)
	}
	if err := c.queries.ProjectHousingCompanyProfileForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project housing company profile: %w", err)
	}
	if err := c.queries.ProjectQualityScoresForSaleListing(ctx, id); err != nil {
		return fmt.Errorf("project quality scores: %w", err)
	}
	return nil
}

func decodeCanonicalMatchSaleListingPayload(job db.SyncJob) (canonicalMatchSaleListingPayload, error) {
	var payload canonicalMatchSaleListingPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return canonicalMatchSaleListingPayload{}, fmt.Errorf("decode canonical sale listing source match payload: %w", err)
		}
	}
	if payload.SaleListingID == "" {
		_, value, err := parseJobEntity(job.SyncJobEntityID)
		if err != nil {
			return canonicalMatchSaleListingPayload{}, fmt.Errorf("parse sale listing entity: %w", err)
		}
		payload.SaleListingID, _, _ = strings.Cut(strings.TrimSpace(value), ":attempt:")
	}
	if payload.SaleListingID == "" {
		return canonicalMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id is required")
	}
	if _, err := uuid.Parse(payload.SaleListingID); err != nil {
		return canonicalMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id must be a uuid: %w", err)
	}
	return payload, nil
}

func (c *Consumer) loadCanonicalMatchSaleListing(ctx context.Context, saleListingID string) (canonicalMatchSaleListingRow, error) {
	var row canonicalMatchSaleListingRow
	err := c.pool.QueryRow(ctx, `
SELECT
    sl.sale_listing_id::text,
    pos.property_offering_source_link_method,
    pos.property_offering_source_link_status,
    sl.sale_listing_source_match_status,
    sl.sale_listing_source_match_attempt_count
FROM public.property_source_offerings sl
LEFT JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
    AND pos.property_offering_source_link_status <> 'rejected'
WHERE sl.sale_listing_id = $1::uuid`, saleListingID).Scan(&row.ID, &row.LinkMethod, &row.LinkStatus, &row.Status, &row.AttemptCount)
	return row, err
}

func (c *Consumer) runCanonicalSourceMatchForSaleListing(ctx context.Context, saleListingID string) (canonicalMatchRunSummary, error) {
	var runID string
	if err := c.pool.QueryRow(ctx, `SELECT public.fnc__refresh_property_offering_source_matches(true, 95, 10, $1::uuid)::text`, saleListingID).Scan(&runID); err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match: %w", err)
	}
	return c.loadCanonicalSourceMatchRun(ctx, runID)
}

func (c *Consumer) runCanonicalSourceMatchBackfill(ctx context.Context, scoreThreshold, competitorMargin int) (canonicalMatchRunSummary, error) {
	var runID string
	if err := c.pool.QueryRow(ctx, `SELECT public.fnc__refresh_property_offering_source_matches(true, $1, $2, NULL::uuid)::text`, scoreThreshold, competitorMargin).Scan(&runID); err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match backfill: %w", err)
	}
	return c.loadCanonicalSourceMatchRun(ctx, runID)
}

func (c *Consumer) loadCanonicalSourceMatchRun(ctx context.Context, runID string) (canonicalMatchRunSummary, error) {
	var summary canonicalMatchRunSummary
	err := c.pool.QueryRow(ctx, `
SELECT
    property_offering_source_match_run_id::text,
    property_offering_source_match_candidates_count,
    property_offering_source_match_auto_linked_count,
    property_offering_source_match_ambiguous_count
FROM public.property_offering_source_match_runs
WHERE property_offering_source_match_run_id = $1::uuid`, runID).Scan(&summary.RunID, &summary.Candidates, &summary.AutoLinked, &summary.Ambiguous)
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("load canonical sale listing source match run: %w", err)
	}
	return summary, nil
}

func (c *Consumer) updateCanonicalSourceMatchState(ctx context.Context, saleListingID, status string, nextAttemptAt *time.Time, runID *string) error {
	_, err := c.pool.Exec(ctx, `
UPDATE public.property_source_offerings
SET
    sale_listing_source_match_status = $2,
    sale_listing_source_match_next_attempt_at = $3,
    sale_listing_source_match_last_attempted_at = now(),
    sale_listing_source_match_attempt_count = sale_listing_source_match_attempt_count + 1,
    sale_listing_source_match_run_id = COALESCE($4::uuid, sale_listing_source_match_run_id),
    sale_listing_updated_at = now()
WHERE sale_listing_id = $1::uuid`, saleListingID, status, nextAttemptAt, runID)
	if err != nil {
		return fmt.Errorf("update canonical source match state: %w", err)
	}
	return nil
}

func (c *Consumer) enqueueCanonicalSourceMatchSaleListing(ctx context.Context, saleListingID string, attempt int32, runAfter time.Time) error {
	payload, err := json.Marshal(canonicalMatchSaleListingPayload{SaleListingID: saleListingID, Attempt: attempt})
	if err != nil {
		return fmt.Errorf("marshal canonical sale listing source match payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalMatchSaleListingSource,
		EntityID:    fmt.Sprintf("sale_listing:%s:attempt:%d", saleListingID, attempt),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}
