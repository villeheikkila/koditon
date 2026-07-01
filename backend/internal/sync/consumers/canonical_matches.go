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
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return canonicalMatchSaleListingRow{}, fmt.Errorf("parse sale listing id: %w", err)
	}
	result, err := c.queries.LoadCanonicalMatchSaleListing(ctx, &id)
	if err != nil {
		return canonicalMatchSaleListingRow{}, err
	}
	return canonicalMatchSaleListingRow{ID: stringValue(result.ID), LinkMethod: &result.LinkMethod, LinkStatus: &result.LinkStatus, Status: result.SaleListingSourceMatchStatus, AttemptCount: result.SaleListingSourceMatchAttemptCount}, nil
}

func (c *Consumer) runCanonicalSourceMatchForSaleListing(ctx context.Context, saleListingID string) (canonicalMatchRunSummary, error) {
	runID := uuid.NewString()
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("parse sale listing id: %w", err)
	}
	summary, err := c.queries.RunCanonicalSourceMatchForSaleListing(ctx, db.RunCanonicalSourceMatchForSaleListingParams{RunID: runID, SaleListingID: id})
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match: %w", err)
	}
	return canonicalMatchRunSummary{RunID: stringValue(summary.RunID), Candidates: int32Value(summary.Candidates), AutoLinked: int32Value(summary.AutoLinked), Ambiguous: int32Value(summary.Ambiguous)}, nil
}

func (c *Consumer) runCanonicalSourceMatchBackfill(ctx context.Context, scoreThreshold, competitorMargin int) (canonicalMatchRunSummary, error) {
	runID := uuid.NewString()
	summary, err := c.queries.RunCanonicalSourceMatchBackfill(ctx, runID)
	if err != nil {
		return canonicalMatchRunSummary{}, fmt.Errorf("run canonical sale listing source match backfill: %w", err)
	}
	return canonicalMatchRunSummary{RunID: stringValue(summary.RunID), Candidates: int32Value(summary.Candidates), AutoLinked: int32Value(summary.AutoLinked), Ambiguous: int32Value(summary.Ambiguous)}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int32Value(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func (c *Consumer) updateCanonicalSourceMatchState(ctx context.Context, saleListingID, status string, nextAttemptAt *time.Time, runID *string) error {
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return fmt.Errorf("parse sale listing id: %w", err)
	}
	if err := c.queries.UpdateCanonicalSourceMatchState(ctx, db.UpdateCanonicalSourceMatchStateParams{Status: status, NextAttemptAt: nextAttemptAt, SaleListingID: id}); err != nil {
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
