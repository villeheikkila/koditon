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
    source_link.link_method,
    source_link.link_status,
    sl.sale_listing_source_match_status,
    sl.sale_listing_source_match_attempt_count
FROM public.property_source_offerings sl
LEFT JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
    AND source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.link_status <> 'rejected'
WHERE sl.sale_listing_id = $1::uuid`, saleListingID).Scan(&row.ID, &row.LinkMethod, &row.LinkStatus, &row.Status, &row.AttemptCount)
	return row, err
}

func (c *Consumer) runCanonicalSourceMatchForSaleListing(ctx context.Context, saleListingID string) (canonicalMatchRunSummary, error) {
	runID := uuid.NewString()
	var summary canonicalMatchRunSummary
	err := c.pool.QueryRow(ctx, `
WITH base AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_unit_match_key,
        link.target_id
    FROM public.property_source_offerings sl
    JOIN public.target_sources link ON link.target_type = 'listing'
        AND link.source_type = 'source_listing'
        AND link.source_id = sl.sale_listing_id
        AND link.link_status <> 'rejected'
    WHERE sl.sale_listing_id = $1::uuid
        AND sl.sale_listing_source_kind = 'ad'
        AND COALESCE(sl.sale_listing_unit_match_key, '') <> ''
    ORDER BY CASE WHEN link.link_status = 'confirmed' THEN 0 ELSE 1 END, link.link_score DESC
    LIMIT 1
),
candidates AS (
    SELECT
        candidate.sale_listing_id,
        candidate.sale_listing_source_provider,
        candidate.sale_listing_source_kind,
        candidate.sale_listing_native_id,
        candidate.sale_listing_first_seen_at,
        candidate.sale_listing_last_seen_at,
        active_link.target_source_id AS active_target_source_id,
        active_link.target_id AS active_target_id,
        active_link.link_method AS active_link_method
    FROM base
    JOIN public.property_source_offerings candidate ON candidate.sale_listing_unit_match_key = base.sale_listing_unit_match_key
        AND candidate.sale_listing_id <> base.sale_listing_id
        AND candidate.sale_listing_source_kind = 'ad'
    LEFT JOIN public.target_sources active_link ON active_link.target_type = 'listing'
        AND active_link.source_type = 'source_listing'
        AND active_link.source_id = candidate.sale_listing_id
        AND active_link.link_status <> 'rejected'
    WHERE candidate.sale_listing_source_provider <> base.sale_listing_source_provider
),
linkable AS (
    SELECT candidates.*
    FROM candidates
    WHERE active_target_source_id IS NULL
        OR active_target_id = (SELECT target_id FROM base)
),
inserted AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        'listing',
        base.target_id,
        'source_listing',
        linkable.sale_listing_id,
        'confirmed',
        'source_match_auto',
        100,
        jsonb_build_object('method', 'unit_match_key_exact', 'matched_source_listing_id', base.sale_listing_id, 'provider', linkable.sale_listing_source_provider, 'native_id', linkable.sale_listing_native_id),
        linkable.sale_listing_first_seen_at,
        linkable.sale_listing_last_seen_at,
        now(),
        now()
    FROM base
    JOIN linkable ON true
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_status ELSE EXCLUDED.link_status END,
        link_method = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_method ELSE EXCLUDED.link_method END,
        link_score = GREATEST(public.target_sources.link_score, EXCLUDED.link_score),
        link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_source_id
)
SELECT
    $2::text,
    (SELECT count(*)::int4 FROM candidates),
    (SELECT count(*)::int4 FROM inserted),
    (SELECT count(*)::int4 FROM candidates WHERE active_target_source_id IS NOT NULL AND active_target_id <> (SELECT target_id FROM base))`, saleListingID, runID).Scan(&summary.RunID, &summary.Candidates, &summary.AutoLinked, &summary.Ambiguous)
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match: %w", err)
	}
	return summary, nil
}

func (c *Consumer) runCanonicalSourceMatchBackfill(ctx context.Context, scoreThreshold, competitorMargin int) (canonicalMatchRunSummary, error) {
	runID := uuid.NewString()
	var summary canonicalMatchRunSummary
	err := c.pool.QueryRow(ctx, `
WITH base AS (
    SELECT
        link.target_id,
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_unit_match_key
    FROM public.target_sources link
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = link.source_id
    WHERE link.target_type = 'listing'
        AND link.source_type = 'source_listing'
        AND link.link_status <> 'rejected'
        AND link.link_method <> 'manual'
        AND sl.sale_listing_source_kind = 'ad'
        AND COALESCE(sl.sale_listing_unit_match_key, '') <> ''
),
candidate_pairs AS (
    SELECT
        base.target_id,
        base.sale_listing_id AS matched_sale_listing_id,
        candidate.sale_listing_id,
        candidate.sale_listing_source_provider,
        candidate.sale_listing_source_kind,
        candidate.sale_listing_native_id,
        candidate.sale_listing_first_seen_at,
        candidate.sale_listing_last_seen_at,
        active_link.target_source_id AS active_target_source_id,
        active_link.target_id AS active_target_id,
        row_number() OVER (
            PARTITION BY candidate.sale_listing_id
            ORDER BY base.target_id, base.sale_listing_id
        ) AS candidate_rank
    FROM base
    JOIN public.property_source_offerings candidate ON candidate.sale_listing_unit_match_key = base.sale_listing_unit_match_key
        AND candidate.sale_listing_id <> base.sale_listing_id
        AND candidate.sale_listing_source_kind = 'ad'
        AND candidate.sale_listing_source_provider <> base.sale_listing_source_provider
    LEFT JOIN public.target_sources active_link ON active_link.target_type = 'listing'
        AND active_link.source_type = 'source_listing'
        AND active_link.source_id = candidate.sale_listing_id
        AND active_link.link_status <> 'rejected'
),
linkable AS (
    SELECT *
    FROM candidate_pairs
    WHERE candidate_rank = 1
        AND (
            active_target_source_id IS NULL
            OR active_target_id = target_id
        )
),
inserted AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        'listing',
        target_id,
        'source_listing',
        sale_listing_id,
        'confirmed',
        'source_match_auto',
        100,
        jsonb_build_object('method', 'unit_match_key_exact_backfill', 'matched_source_listing_id', matched_sale_listing_id, 'provider', sale_listing_source_provider, 'native_id', sale_listing_native_id),
        sale_listing_first_seen_at,
        sale_listing_last_seen_at,
        now(),
        now()
    FROM linkable
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_status ELSE EXCLUDED.link_status END,
        link_method = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_method ELSE EXCLUDED.link_method END,
        link_score = GREATEST(public.target_sources.link_score, EXCLUDED.link_score),
        link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_source_id
)
SELECT
    $1::text,
    (SELECT count(*)::int4 FROM candidate_pairs),
    (SELECT count(*)::int4 FROM inserted),
    (SELECT count(*)::int4 FROM candidate_pairs WHERE active_target_source_id IS NOT NULL AND active_target_id <> target_id)`, runID).Scan(&summary.RunID, &summary.Candidates, &summary.AutoLinked, &summary.Ambiguous)
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match backfill: %w", err)
	}
	return summary, nil
}

func (c *Consumer) updateCanonicalSourceMatchState(ctx context.Context, saleListingID, status string, nextAttemptAt *time.Time, runID *string) error {
	_, err := c.pool.Exec(ctx, `
WITH updated_source AS (
    UPDATE public.property_source_offerings
    SET
        sale_listing_source_match_status = $2,
        sale_listing_source_match_next_attempt_at = $3,
        sale_listing_source_match_last_attempted_at = now(),
        sale_listing_source_match_attempt_count = sale_listing_source_match_attempt_count + 1,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = $1::uuid
    RETURNING sale_listing_id, sale_listing_updated_at
)
UPDATE public.source_listings src
SET normalized_at = updated_source.sale_listing_updated_at,
    updated_at = updated_source.sale_listing_updated_at
FROM updated_source
WHERE src.source_listing_id = updated_source.sale_listing_id`, saleListingID, status, nextAttemptAt)
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
