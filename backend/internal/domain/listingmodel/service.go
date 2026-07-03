package listingmodel

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
)

// Service owns the evidence-backed canonical listing lifecycle.
type Service struct {
	logger  *slog.Logger
	pool    *pgxpool.Pool
	queries *db.Queries
}

// Result describes the canonical entities produced from one evidence source.
type Result struct {
	EvidenceSourceID   uuid.UUID
	PropertyOfferingID uuid.UUID
	ListingID          uuid.UUID
	SearchDocuments    int32
}

// NewService creates a listing model service.
func NewService(logger *slog.Logger, pool *pgxpool.Pool) *Service {
	return &Service{logger: logger.With("component", "listingmodel"), pool: pool, queries: db.New(pool)}
}

// ReconcileSourceOffering publishes one normalized source offering into the canonical listing graph.
func (s *Service) ReconcileSourceOffering(ctx context.Context, sourceOfferingID uuid.UUID) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin listing model reconciliation: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			s.logger.DebugContext(ctx, "rollback listing model reconciliation", "error", rollbackErr)
		}
	}()
	row, err := s.queries.WithTx(tx).ReconcileSourceOfferingListingModel(ctx, sourceOfferingID)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile source offering listing model: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit listing model reconciliation: %w", err)
	}
	result := Result{PropertyOfferingID: row.PropertyOfferingID, ListingID: row.ListingID}
	if row.EvidenceSourceID != nil {
		result.EvidenceSourceID = *row.EvidenceSourceID
	}
	if row.SearchDocuments != nil {
		result.SearchDocuments = *row.SearchDocuments
	}
	s.logger.InfoContext(ctx, "source offering reconciled into listing model", "source_offering_id", sourceOfferingID.String(), "evidence_source_id", result.EvidenceSourceID.String(), "listing_id", result.ListingID.String(), "outcome", logging.OutcomeSuccess)
	return result, nil
}
