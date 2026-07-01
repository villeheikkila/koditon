package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

const (
	TaskTypeCanonicalRebuildDimensionLayerBackfill = "canonical_rebuild_dimension_layer_backfill"
	TaskTypeCanonicalRebuildDimensionLayerListing  = "canonical_rebuild_dimension_layer_listing"
	TaskTypeCanonicalResolveDirtyDimensionTargets  = "canonical_resolve_dirty_dimension_targets"
	TaskTypeCanonicalResolveDimensionTarget        = "canonical_resolve_dimension_target"
	TaskTypeCanonicalExtractManagerCertificate     = "canonical_extract_manager_certificate"
	TaskTypeCanonicalProjectManagerCertificate     = "canonical_project_manager_certificate"
	TaskTypeCanonicalBackfillBuildingCoordinates   = "canonical_backfill_building_coordinates"
	TaskTypeCanonicalRebuildSpatialReadModel       = "canonical_rebuild_spatial_read_model"
	TaskTypeCanonicalBackfillDetachedHouses        = "canonical_backfill_detached_houses"
)

type dimensionLayerBackfillPayload struct {
	BatchSize          int32  `json:"batch_size,omitempty"`
	AfterSaleListingID string `json:"after_sale_listing_id,omitempty"`
}

type dimensionLayerBackfillCheckpoint struct {
	LastSaleListingID string    `json:"last_sale_listing_id,omitempty"`
	Enqueued          int64     `json:"enqueued"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type dimensionLayerListingPayload struct {
	SaleListingID   string     `json:"sale_listing_id"`
	Reason          string     `json:"reason,omitempty"`
	ExpectedDirtyAt *time.Time `json:"expected_dirty_at,omitempty"`
}

type dirtyDimensionTargetsPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type detachedHouseBackfillPayload struct {
	BatchSize int32 `json:"batch_size,omitempty"`
}

type dirtyDimensionTargetPayload struct {
	TargetType      string     `json:"target_type"`
	TargetID        string     `json:"target_id"`
	ExpectedDirtyAt *time.Time `json:"expected_dirty_at,omitempty"`
}

type managerCertificateDocumentPayload struct {
	DocumentID         string `json:"document_id,omitempty"`
	PropertyDocumentID string `json:"property_document_id"`
	Model              string `json:"model,omitempty"`
}

type dirtyDimensionTargetRow struct {
	TargetType string
	TargetID   uuid.UUID
	DirtyAt    time.Time
}

func (c *Consumer) handleCanonicalExtractManagerCertificate(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.extract_manager_certificate"))
	if c.propertiesService == nil {
		return newPermanentError(fmt.Errorf("properties service is not configured"), "properties service missing")
	}
	payload, err := decodeManagerCertificateDocumentWorkflowPayload(params)
	if err != nil {
		return newPermanentError(err, "invalid payload")
	}
	result, err := c.propertiesService.ExtractManagerCertificateSource(ctx, payload.PropertyDocumentID, payload.Model)
	if err != nil {
		return fmt.Errorf("extract manager certificate source %s: %w", payload.PropertyDocumentID, err)
	}
	documentID, err := uuid.Parse(result.Document.ID)
	if err != nil {
		return newPermanentError(fmt.Errorf("parse extracted document id: %w", err), "invalid result")
	}
	if err := c.enqueueManagerCertificateProjection(ctx, documentID); err != nil {
		return fmt.Errorf("enqueue manager certificate projection %s: %w", payload.PropertyDocumentID, err)
	}
	logger.InfoContext(ctx, "manager certificate source extracted", "property_document_id", payload.PropertyDocumentID, "model", result.Model, "schema_version", result.SchemaVersion, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalProjectManagerCertificate(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.project_manager_certificate"))
	if c.propertiesService == nil {
		return newPermanentError(fmt.Errorf("properties service is not configured"), "properties service missing")
	}
	payload, err := decodeManagerCertificateDocumentWorkflowPayload(params)
	if err != nil {
		return newPermanentError(err, "invalid payload")
	}
	result, err := c.propertiesService.ProjectManagerCertificateExtraction(ctx, payload.PropertyDocumentID)
	if err != nil {
		return fmt.Errorf("project manager certificate %s: %w", payload.PropertyDocumentID, err)
	}
	logger.InfoContext(ctx, "manager certificate projected", "property_document_id", payload.PropertyDocumentID, "offering_id", result.Document.OfferingID, "claims", result.Claims, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalBackfillBuildingCoordinates(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.backfill_building_coordinates"))
	rows, err := c.queries.BackfillBuildingCoordinates(ctx)
	if err != nil {
		return fmt.Errorf("backfill building coordinates: %w", err)
	}
	logger.InfoContext(ctx, "building coordinates backfilled", "rows", rows, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalRebuildSpatialReadModel(ctx context.Context, logger *slog.Logger, _ json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_spatial_read_model"))
	logger.InfoContext(ctx, "spatial read model is served by direct SQL", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalBackfillDetachedHouses(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.backfill_detached_houses"))
	payload := detachedHouseBackfillPayload{BatchSize: 1000}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &payload); err != nil {
			return newPermanentError(fmt.Errorf("decode detached house backfill payload: %w", err), "invalid payload")
		}
	}
	if payload.BatchSize <= 0 || payload.BatchSize > 5000 {
		payload.BatchSize = 1000
	}
	count, err := c.queries.BackfillDetachedPropertyHouses(ctx, payload.BatchSize)
	if err != nil {
		return fmt.Errorf("backfill detached property houses: %w", err)
	}
	if count == payload.BatchSize {
		if err := c.enqueueDetachedHouseBackfill(ctx, payload.BatchSize); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "detached houses backfilled", "count", count, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalRebuildDimensionLayerBackfill(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_dimension_layer_backfill"))
	payload := dimensionLayerBackfillPayload{BatchSize: 500}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &payload); err != nil {
			return newPermanentError(fmt.Errorf("decode dimension layer backfill payload: %w", err), "invalid payload")
		}
	}
	if payload.BatchSize <= 0 || payload.BatchSize > 5000 {
		payload.BatchSize = 500
	}
	var cursor *uuid.UUID
	if payload.AfterSaleListingID != "" {
		parsed, err := uuid.Parse(payload.AfterSaleListingID)
		if err != nil {
			return newPermanentError(fmt.Errorf("parse dimension layer checkpoint cursor: %w", err), "invalid checkpoint")
		}
		cursor = &parsed
	}
	listingIDs, err := c.listDimensionLayerBackfillListingIDs(ctx, cursor, payload.BatchSize)
	if err != nil {
		return err
	}
	enqueued := int64(0)
	lastSaleListingID := payload.AfterSaleListingID
	if len(listingIDs) == 0 {
		logger.InfoContext(ctx, "dimension layer backfill completed", "enqueued", enqueued, "outcome", logging.OutcomeSuccess)
		return nil
	}
	for _, listingID := range listingIDs {
		if err := c.enqueueDimensionLayerListing(ctx, listingID, "backfill", nil); err != nil {
			return fmt.Errorf("enqueue dimension layer listing %s: %w", listingID, err)
		}
		cursor = &listingID
		lastSaleListingID = listingID.String()
		enqueued++
	}
	if len(listingIDs) == int(payload.BatchSize) {
		if err := c.enqueueDimensionLayerBackfill(ctx, lastSaleListingID, payload.BatchSize); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "dimension layer backfill batch enqueued", "enqueued", enqueued, "last_sale_listing_id", lastSaleListingID, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalRebuildDimensionLayerListing(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_dimension_layer_listing"))
	payload, err := decodeDimensionLayerListingWorkflowPayload(params)
	if err != nil {
		return newPermanentError(err, "invalid payload")
	}
	saleListingID, err := uuid.Parse(payload.SaleListingID)
	if err != nil {
		return newPermanentError(fmt.Errorf("parse sale listing id: %w", err), "invalid payload")
	}
	result, err := c.rebuildDimensionLayerForListing(ctx, saleListingID, payload.ExpectedDirtyAt)
	if err != nil {
		return fmt.Errorf("rebuild dimension layer for listing %s: %w", saleListingID, err)
	}
	logger.InfoContext(ctx, "dimension layer listing rebuilt", "sale_listing_id", saleListingID.String(), "result", string(result), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalResolveDirtyDimensionTargets(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.resolve_dirty_dimension_targets"))
	payload := dirtyDimensionTargetsPayload{Limit: 1000}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &payload); err != nil {
			return newPermanentError(fmt.Errorf("decode dirty dimension targets payload: %w", err), "invalid payload")
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 1000
	}
	targets, err := c.listDirtyDimensionTargets(ctx, payload.Limit)
	if err != nil {
		return err
	}
	enqueued := 0
	for _, target := range targets {
		if target.TargetType == "listing" {
			if err := c.enqueueDimensionLayerListing(ctx, target.TargetID, "dirty_target", &target.DirtyAt); err != nil {
				return err
			}
		} else {
			if err := c.enqueueDimensionTarget(ctx, target.TargetType, target.TargetID, target.DirtyAt); err != nil {
				return err
			}
		}
		if err := c.markDimensionTargetQueued(ctx, target.TargetType, target.TargetID); err != nil {
			return err
		}
		enqueued++
	}
	logger.InfoContext(ctx, "dirty dimension target jobs enqueued", "count", enqueued, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalResolveDimensionTarget(ctx context.Context, logger *slog.Logger, params json.RawMessage) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.resolve_dimension_target"))
	payload, err := decodeDirtyDimensionTargetWorkflowPayload(params)
	if err != nil {
		return newPermanentError(err, "invalid payload")
	}
	targetID, err := uuid.Parse(payload.TargetID)
	if err != nil {
		return newPermanentError(fmt.Errorf("parse target id: %w", err), "invalid payload")
	}
	result, err := c.queries.ResolveDimensionTarget(ctx, db.ResolveDimensionTargetParams{TargetType: payload.TargetType, TargetID: targetID, ExpectedDirtyAt: payload.ExpectedDirtyAt})
	if err != nil {
		return fmt.Errorf("resolve dimension target %s:%s: %w", payload.TargetType, payload.TargetID, err)
	}
	logger.InfoContext(ctx, "dimension target resolved", "target_type", payload.TargetType, "target_id", payload.TargetID, "result", string(result), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) listDimensionLayerBackfillListingIDs(ctx context.Context, cursor *uuid.UUID, limit int32) ([]uuid.UUID, error) {
	ids, err := c.queries.ListDimensionLayerBackfillListingIDs(ctx, db.ListDimensionLayerBackfillListingIDsParams{Cursor: cursor, LimitCount: limit})
	if err != nil {
		return nil, fmt.Errorf("list dimension layer backfill listings: %w", err)
	}
	return ids, nil
}

func (c *Consumer) rebuildDimensionLayerForListing(ctx context.Context, saleListingID uuid.UUID, expectedDirtyAt *time.Time) (json.RawMessage, error) {
	if err := c.queries.EnsurePhysicalBuildingForSaleListing(ctx, saleListingID); err != nil {
		return nil, fmt.Errorf("ensure physical building: %w", err)
	}
	if err := properties.ProjectListingRenovationEvents(ctx, c.pool, saleListingID); err != nil {
		return nil, err
	}
	result, err := c.queries.RebuildListingDimensionLayerAt(ctx, db.RebuildListingDimensionLayerAtParams{SaleListingID: saleListingID, ExpectedDirtyAt: expectedDirtyAt})
	if err != nil {
		return nil, fmt.Errorf("rebuild listing dimension layer: %w", err)
	}
	return result, nil
}

func (c *Consumer) enqueueDimensionLayerListing(ctx context.Context, saleListingID uuid.UUID, reason string, expectedDirtyAt *time.Time) error {
	payload, err := json.Marshal(dimensionLayerListingPayload{SaleListingID: saleListingID.String(), Reason: reason, ExpectedDirtyAt: expectedDirtyAt})
	if err != nil {
		return fmt.Errorf("marshal dimension layer listing payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalRebuildDimensionLayerListing,
		Params:   payload,
	})
	return err
}

func (c *Consumer) enqueueDimensionTarget(ctx context.Context, targetType string, targetID uuid.UUID, expectedDirtyAt time.Time) error {
	payload, err := json.Marshal(dirtyDimensionTargetPayload{TargetType: targetType, TargetID: targetID.String(), ExpectedDirtyAt: &expectedDirtyAt})
	if err != nil {
		return fmt.Errorf("marshal dimension target payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalResolveDimensionTarget,
		Params:   payload,
	})
	return err
}

func (c *Consumer) enqueueDirtyDimensionTargetFanout(ctx context.Context) error {
	payload, err := json.Marshal(dirtyDimensionTargetsPayload{Limit: 1000})
	if err != nil {
		return fmt.Errorf("marshal dirty dimension targets payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalResolveDirtyDimensionTargets,
		Params:   payload,
	})
	return err
}

func (c *Consumer) enqueueDimensionLayerBackfill(ctx context.Context, afterSaleListingID string, batchSize int32) error {
	payload, err := json.Marshal(dimensionLayerBackfillPayload{BatchSize: batchSize, AfterSaleListingID: afterSaleListingID})
	if err != nil {
		return fmt.Errorf("marshal dimension layer backfill payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalRebuildDimensionLayerBackfill,
		Params:   payload,
	})
	return err
}

func (c *Consumer) enqueueDetachedHouseBackfill(ctx context.Context, batchSize int32) error {
	payload, err := json.Marshal(detachedHouseBackfillPayload{BatchSize: batchSize})
	if err != nil {
		return fmt.Errorf("marshal detached house backfill payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalBackfillDetachedHouses,
		Params:   payload,
	})
	return err
}

func (c *Consumer) enqueueManagerCertificateProjection(ctx context.Context, documentID uuid.UUID) error {
	payload, err := json.Marshal(managerCertificateDocumentPayload{DocumentID: documentID.String(), PropertyDocumentID: documentID.String()})
	if err != nil {
		return fmt.Errorf("marshal manager certificate projection payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalProjectManagerCertificate,
		Params:   payload,
	})
	return err
}

func (c *Consumer) listDirtyDimensionTargets(ctx context.Context, limit int32) ([]dirtyDimensionTargetRow, error) {
	rows, err := c.queries.ListDirtyDimensionTargets(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list dirty dimension targets: %w", err)
	}
	targets := make([]dirtyDimensionTargetRow, 0, len(rows))
	for _, row := range rows {
		targets = append(targets, dirtyDimensionTargetRow{TargetType: row.TargetType, TargetID: row.TargetID, DirtyAt: row.DirtyAt})
	}
	return targets, nil
}

func (c *Consumer) markDimensionTargetQueued(ctx context.Context, targetType string, targetID uuid.UUID) error {
	if _, err := c.queries.MarkDimensionTargetQueued(ctx, db.MarkDimensionTargetQueuedParams{TargetType: targetType, TargetID: targetID}); err != nil {
		return fmt.Errorf("mark dimension target queued: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
