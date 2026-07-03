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
	SourceListingID    uuid.UUID
	SearchDocuments    int32
}

// NewService creates a listing model service.
func NewService(logger *slog.Logger, pool *pgxpool.Pool) *Service {
	return &Service{logger: logger.With("component", "listingmodel"), pool: pool, queries: db.New(pool)}
}

// RemoveShortcutAdListing removes the canonical listing projection for a non-listing Shortcut ad.
func (s *Service) RemoveShortcutAdListing(ctx context.Context, shortcutAdID int64) error {
	shortcutAdIDText := fmt.Sprint(shortcutAdID)
	if err := s.queries.DeleteShortcutAdSourceListing(ctx, &shortcutAdIDText); err != nil {
		return fmt.Errorf("delete shortcut source listing: %w", err)
	}
	return nil
}

// RemoveFrontdoorBuildingAnnouncement removes the canonical listing projection for a rental Frontdoor announcement.
func (s *Service) RemoveFrontdoorBuildingAnnouncement(ctx context.Context, announcementID uuid.UUID) error {
	announcementIDText := announcementID.String()
	if err := s.queries.DeleteFrontdoorBuildingAnnouncementSourceListing(ctx, &announcementIDText); err != nil {
		return fmt.Errorf("delete frontdoor announcement source listing: %w", err)
	}
	return nil
}

// ReconcileShortcutAd publishes one Shortcut ad into the canonical listing graph.
func (s *Service) ReconcileShortcutAd(ctx context.Context, shortcutAdID int64) (Result, error) {
	sourceListingID, err := s.queries.UpsertShortcutAdSourceListing(ctx, &shortcutAdID)
	if err != nil {
		return Result{}, fmt.Errorf("upsert shortcut ad source listing: %w", err)
	}
	return s.reconcileSourceListing(ctx, sourceListingID)
}

// ReconcileFrontdoorAd publishes one Frontdoor ad into the canonical listing graph.
func (s *Service) ReconcileFrontdoorAd(ctx context.Context, frontdoorAdID uuid.UUID) (Result, error) {
	sourceListingID, err := s.queries.UpsertFrontdoorAdSourceListing(ctx, &frontdoorAdID)
	if err != nil {
		return Result{}, fmt.Errorf("upsert frontdoor ad source listing: %w", err)
	}
	return s.reconcileSourceListing(ctx, sourceListingID)
}

// ReconcileFrontdoorBuildingAnnouncement publishes one Frontdoor building announcement into the canonical listing graph.
func (s *Service) ReconcileFrontdoorBuildingAnnouncement(ctx context.Context, announcementID uuid.UUID) (Result, error) {
	sourceListingID, err := s.queries.UpsertFrontdoorBuildingAnnouncementSourceListing(ctx, &announcementID)
	if err != nil {
		return Result{}, fmt.Errorf("upsert frontdoor announcement source listing: %w", err)
	}
	return s.reconcileSourceListing(ctx, sourceListingID)
}

// ReconcileSourceListing publishes one normalized source listing into the canonical listing graph.
func (s *Service) ReconcileSourceListing(ctx context.Context, sourceListingID uuid.UUID) (Result, error) {
	return s.reconcileSourceListing(ctx, sourceListingID)
}

func (s *Service) reconcileSourceListing(ctx context.Context, sourceListingID uuid.UUID) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin listing model reconciliation: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			s.logger.DebugContext(ctx, "rollback listing model reconciliation", "error", rollbackErr)
		}
	}()
	qtx := s.queries.WithTx(tx)
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
