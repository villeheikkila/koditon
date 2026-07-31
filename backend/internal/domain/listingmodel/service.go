package listingmodel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	SourceListingID    uuid.UUID
	SearchDocuments    int32
}

// NewService creates a listing model service.
func NewService(logger *slog.Logger, pool *pgxpool.Pool) *Service {
	return &Service{logger: logger.With("component", "listingmodel"), pool: pool, queries: db.New(pool)}
}

// RemoveShortcutAdListing removes the canonical listing projection for a non-listing Shortcut ad.
func (s *Service) RemoveShortcutAdListing(ctx context.Context, shortcutAdID int64) error {
	return s.removeSourceListing(ctx, "shortcut source listing", func(queries *db.Queries) (*uuid.UUID, error) {
		return queries.DeleteShortcutAdSourceListing(ctx, &shortcutAdID)
	})
}

// RemoveFrontdoorBuildingAnnouncement removes the canonical listing projection for a rental Frontdoor announcement.
func (s *Service) RemoveFrontdoorBuildingAnnouncement(ctx context.Context, announcementID uuid.UUID) error {
	return s.removeSourceListing(ctx, "frontdoor announcement source listing", func(queries *db.Queries) (*uuid.UUID, error) {
		return queries.DeleteFrontdoorBuildingAnnouncementSourceListing(ctx, &announcementID)
	})
}

// ReconcileShortcutAd publishes one Shortcut ad into the canonical listing graph.
func (s *Service) ReconcileShortcutAd(ctx context.Context, shortcutAdID int64) (Result, error) {
	return s.reconcile(ctx, "shortcut ad source listing", func(queries *db.Queries) (uuid.UUID, error) {
		return requiredSourceListingID(queries.UpsertShortcutAdSourceListing(ctx, &shortcutAdID))
	})
}

// ReconcileFrontdoorAd publishes one Frontdoor ad into the canonical listing graph.
func (s *Service) ReconcileFrontdoorAd(ctx context.Context, frontdoorAdID uuid.UUID) (Result, error) {
	return s.reconcile(ctx, "frontdoor ad source listing", func(queries *db.Queries) (uuid.UUID, error) {
		return requiredSourceListingID(queries.UpsertFrontdoorAdSourceListing(ctx, &frontdoorAdID))
	})
}

// ReconcileFrontdoorBuildingAnnouncement publishes one Frontdoor building announcement into the canonical listing graph.
func (s *Service) ReconcileFrontdoorBuildingAnnouncement(ctx context.Context, announcementID uuid.UUID) (Result, error) {
	return s.reconcile(ctx, "frontdoor announcement source listing", func(queries *db.Queries) (uuid.UUID, error) {
		return requiredSourceListingID(queries.UpsertFrontdoorBuildingAnnouncementSourceListing(ctx, &announcementID))
	})
}

// ReconcileSourceListing publishes one normalized source listing into the canonical listing graph.
func (s *Service) ReconcileSourceListing(ctx context.Context, sourceListingID uuid.UUID) (Result, error) {
	return s.reconcile(ctx, "source listing", func(_ *db.Queries) (uuid.UUID, error) {
		return sourceListingID, nil
	})
}

func requiredSourceListingID(id *uuid.UUID, err error) (uuid.UUID, error) {
	if err != nil {
		return uuid.Nil, err
	}
	if id == nil {
		return uuid.Nil, pgx.ErrNoRows
	}
	return *id, nil
}

func (s *Service) reconcile(ctx context.Context, sourceName string, resolveSource func(*db.Queries) (uuid.UUID, error)) (Result, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Result{}, fmt.Errorf("begin listing model reconciliation: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			s.logger.DebugContext(ctx, "rollback listing model reconciliation", "error", rollbackErr)
		}
	}()
	qtx := s.queries.WithTx(tx)
	sourceListingID, err := resolveSource(qtx)
	if err != nil {
		return Result{}, fmt.Errorf("upsert %s: %w", sourceName, err)
	}
	row, err := qtx.ReconcileSourceListingModel(ctx, sourceListingID)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile source listing model: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit listing model reconciliation: %w", err)
	}
	result := Result{PropertyOfferingID: row.PropertyOfferingID, ListingID: row.ListingID, SourceListingID: sourceListingID}
	if row.EvidenceSourceID != nil {
		result.EvidenceSourceID = *row.EvidenceSourceID
	}
	if row.SearchDocuments != nil {
		result.SearchDocuments = *row.SearchDocuments
	}
	s.logger.InfoContext(ctx, "source listing reconciled into listing model", "source_listing_id", sourceListingID.String(), "evidence_source_id", result.EvidenceSourceID.String(), "listing_id", result.ListingID.String(), "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (s *Service) removeSourceListing(ctx context.Context, sourceName string, deleteSource func(*db.Queries) (*uuid.UUID, error)) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin removing %s: %w", sourceName, err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			s.logger.DebugContext(ctx, "rollback source listing removal", "source", sourceName, "error", rollbackErr)
		}
	}()
	qtx := s.queries.WithTx(tx)
	replacementSourceListingID, err := deleteSource(qtx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("delete %s: %w", sourceName, err)
	}
	if replacementSourceListingID != nil {
		if _, err := qtx.ReconcileSourceListingModel(ctx, *replacementSourceListingID); err != nil {
			return fmt.Errorf("reproject listing after deleting %s: %w", sourceName, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit removing %s: %w", sourceName, err)
	}
	return nil
}
