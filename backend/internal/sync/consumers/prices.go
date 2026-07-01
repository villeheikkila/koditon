package consumers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
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
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return pricesMatchSaleListingRow{}, fmt.Errorf("parse sale listing id: %w", err)
	}
	result, err := c.queries.LoadPricesMatchSaleListing(ctx, &id)
	if err != nil {
		return pricesMatchSaleListingRow{}, err
	}
	row := pricesMatchSaleListingRow{ID: stringValue(result.ID), LastSeenAt: result.SaleListingLastSeenAt, Status: result.SaleListingPricesMatchStatus, AttemptCount: result.SaleListingPricesMatchAttemptCount, ExpiresAt: result.SaleListingPricesMatchExpiresAt}
	if stringValue(result.TransactionID) != "" {
		row.TransactionID = result.TransactionID
	}
	return row, nil
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
	id, err := uuid.Parse(saleListingID)
	if err != nil {
		return fmt.Errorf("parse sale listing id: %w", err)
	}
	var parsedRunID *uuid.UUID
	if runID != nil && strings.TrimSpace(*runID) != "" {
		id, err := uuid.Parse(*runID)
		if err != nil {
			return fmt.Errorf("parse prices match run id: %w", err)
		}
		parsedRunID = &id
	}
	if err := c.queries.UpdatePricesMatchState(ctx, db.UpdatePricesMatchStateParams{Status: status, NextAttemptAt: nextAttemptAt, RunID: parsedRunID, ExpiresAt: expiresAt, SaleListingID: id}); err != nil {
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
