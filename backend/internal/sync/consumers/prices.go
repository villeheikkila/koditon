package consumers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/domain/properties"
)

const (
	TaskTypePricesMatchSaleListingsBackfill = "prices_match_sale_listings_backfill"
	TaskTypePricesMatchSaleListingsFanout   = "prices_match_sale_listings_fanout"
	TaskTypePricesMatchSaleListing          = "prices_match_sale_listing"

	pricesMatchInitialDelay = 7 * 24 * time.Hour
	pricesMatchRetryDelay   = 7 * 24 * time.Hour
	pricesMatchMaxAge       = 4 * 30 * 24 * time.Hour
)

type pricesMatchFanoutPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type pricesMatchBackfillPayload struct {
	ScoreThreshold   int32 `json:"score_threshold,omitempty"`
	CompetitorMargin int32 `json:"competitor_margin,omitempty"`
}

type pricesMatchSaleListingPayload struct {
	SaleListingID string `json:"sale_listing_id"`
	Attempt       int32  `json:"attempt,omitempty"`
}

type pricesMatchSaleListingRow struct {
	ID            string
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

func (c *Consumer) loadPricesMatchSaleListing(ctx context.Context, saleListingID string) (pricesMatchSaleListingRow, error) {
	var row pricesMatchSaleListingRow
	var transactionID *string
	err := c.pool.QueryRow(ctx, `
SELECT
    sale_listing_id::text,
    sale_listing_last_seen_at,
    prices_transaction_id::text,
    sale_listing_prices_match_status,
    sale_listing_prices_match_attempt_count,
    sale_listing_prices_match_expires_at
FROM public.property_source_offerings
WHERE sale_listing_id = $1::uuid`, saleListingID).Scan(&row.ID, &row.LastSeenAt, &transactionID, &row.Status, &row.AttemptCount, &row.ExpiresAt)
	row.TransactionID = transactionID
	return row, err
}

func (c *Consumer) runPricesMatchForSaleListing(ctx context.Context, saleListingID string) (pricesMatchRunSummary, error) {
	listingID, err := uuid.Parse(saleListingID)
	if err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("parse sale listing id: %w", err)
	}
	summary, err := c.propertiesService.RunSaleListingTransactionMatch(ctx, properties.TransactionMatchRunOptions{AutoLinkSafe: true, ScoreThreshold: 90, CompetitorMargin: 15, TargetListingID: &listingID})
	if err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("run prices sale listing match: %w", err)
	}
	return pricesMatchRunSummary{RunID: summary.RunID, Candidates: summary.Candidates, AutoLinked: summary.AutoLinked, Ambiguous: summary.Ambiguous}, nil
}

func (c *Consumer) runPricesMatchBackfill(ctx context.Context, scoreThreshold, competitorMargin int) (pricesMatchRunSummary, error) {
	summary, err := c.propertiesService.RunSaleListingTransactionMatch(ctx, properties.TransactionMatchRunOptions{AutoLinkSafe: true, ScoreThreshold: int32(scoreThreshold), CompetitorMargin: int32(competitorMargin)})
	if err != nil {
		return pricesMatchRunSummary{}, fmt.Errorf("run prices sale listing match backfill: %w", err)
	}
	return pricesMatchRunSummary{RunID: summary.RunID, Candidates: summary.Candidates, AutoLinked: summary.AutoLinked, Ambiguous: summary.Ambiguous}, nil
}

func (c *Consumer) updatePricesMatchState(ctx context.Context, saleListingID, status string, nextAttemptAt *time.Time, runID *string, expiresAt *time.Time) error {
	_, err := c.pool.Exec(ctx, `
UPDATE public.property_source_offerings
SET
    sale_listing_prices_match_status = $2,
    sale_listing_prices_match_next_attempt_at = $3,
    sale_listing_prices_match_last_attempted_at = now(),
    sale_listing_prices_match_attempt_count = sale_listing_prices_match_attempt_count + 1,
    sale_listing_prices_match_run_id = COALESCE($4::uuid, sale_listing_prices_match_run_id),
    sale_listing_prices_match_expires_at = COALESCE($5, sale_listing_prices_match_expires_at),
    sale_listing_updated_at = now()
WHERE sale_listing_id = $1::uuid`, saleListingID, status, nextAttemptAt, runID, expiresAt)
	if err != nil {
		return fmt.Errorf("update prices match state: %w", err)
	}
	return nil
}

func parseJobEntity(entityID string) (string, string, error) {
	entityType, value, ok := strings.Cut(entityID, ":")
	if !ok || entityType == "" || value == "" {
		return "", "", fmt.Errorf("expected type:value entity id")
	}
	return entityType, value, nil
}
