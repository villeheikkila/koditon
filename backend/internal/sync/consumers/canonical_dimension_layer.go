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
	tag, err := c.pool.Exec(ctx, `
UPDATE public.physical_buildings pb
SET physical_building_latitude = coordinates.lat,
    physical_building_longitude = coordinates.lng,
    physical_building_updated_at = now()
FROM (
    SELECT DISTINCT ON (pu.physical_building_id)
        pu.physical_building_id,
        COALESCE(fb.frontdoor_building_latitude, sb.shortcut_building_latitude, sl.sale_listing_latitude, postgis.ST_Y(hc.housing_company_geom)::double precision) AS lat,
        COALESCE(fb.frontdoor_building_longitude, sb.shortcut_building_longitude, sl.sale_listing_longitude, postgis.ST_X(hc.housing_company_geom)::double precision) AS lng
    FROM public.property_units pu
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    LEFT JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    LEFT JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE pu.physical_building_id IS NOT NULL
    ORDER BY pu.physical_building_id,
        (fb.frontdoor_building_latitude IS NOT NULL AND fb.frontdoor_building_longitude IS NOT NULL) DESC,
        (sb.shortcut_building_latitude IS NOT NULL AND sb.shortcut_building_longitude IS NOT NULL) DESC,
        (sl.sale_listing_latitude IS NOT NULL AND sl.sale_listing_longitude IS NOT NULL) DESC,
        sl.sale_listing_last_seen_at DESC NULLS LAST
) coordinates
WHERE pb.physical_building_id = coordinates.physical_building_id
  AND coordinates.lat IS NOT NULL
  AND coordinates.lng IS NOT NULL
  AND (pb.physical_building_latitude IS NULL OR pb.physical_building_longitude IS NULL)`)
	if err != nil {
		return fmt.Errorf("backfill building coordinates: %w", err)
	}
	logger.InfoContext(ctx, "building coordinates backfilled", "rows", tag.RowsAffected(), "outcome", logging.OutcomeSuccess)
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
	rows, err := c.pool.Query(ctx, `
SELECT source_listing_id
FROM public.source_listings
WHERE ($1::uuid IS NULL OR source_listing_id > $1::uuid)
ORDER BY source_listing_id
LIMIT $2`, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list dimension layer backfill listings: %w", err)
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan dimension layer backfill listing: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dimension layer backfill listings: %w", err)
	}
	return out, nil
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
	rows, err := c.pool.Query(ctx, `
SELECT target_type, target_id, dirty_at
FROM public.property_dimension_dirty_targets
WHERE (resolved_at IS NULL OR resolved_at < dirty_at)
    AND (queued_at IS NULL OR queued_at < dirty_at OR queued_at < now() - interval '30 minutes')
ORDER BY dirty_at
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list dirty dimension targets: %w", err)
	}
	defer rows.Close()
	targets := make([]dirtyDimensionTargetRow, 0, limit)
	for rows.Next() {
		var target dirtyDimensionTargetRow
		if err := rows.Scan(&target.TargetType, &target.TargetID, &target.DirtyAt); err != nil {
			return nil, fmt.Errorf("scan dirty dimension target: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dirty dimension targets: %w", err)
	}
	return targets, nil
}

func (c *Consumer) markDimensionTargetQueued(ctx context.Context, targetType string, targetID uuid.UUID) error {
	if _, err := c.pool.Exec(ctx, `SELECT public.fnc__mark_dimension_target_queued($1, $2)`, targetType, targetID); err != nil {
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
