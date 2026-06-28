package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
	"koditon/internal/sync/workflows"
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

func (c *Consumer) projectTypedHousingCompanyProfileForSaleListing(ctx context.Context, saleListingID string) error {
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return fmt.Errorf("parse sale listing id for typed projection: %w", err)
	}
	if _, err := c.queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: id, Reason: "source_link_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty: %w", err)
	}
	if err := c.enqueueDimensionLayerListing(ctx, id, "source_link_changed", nil); err != nil {
		return fmt.Errorf("enqueue dimension layer listing: %w", err)
	}
	return nil
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

func (c *Consumer) enqueueCanonicalSourceMatchSaleListing(ctx context.Context, saleListingID string, attempt int32) error {
	payload, err := json.Marshal(canonicalMatchSaleListingPayload{SaleListingID: saleListingID, Attempt: attempt})
	if err != nil {
		return fmt.Errorf("marshal canonical sale listing source match payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalMatchSaleListingSource,
		Params:   payload,
	})
	return err
}
