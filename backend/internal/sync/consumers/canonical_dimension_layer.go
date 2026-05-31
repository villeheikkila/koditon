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
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

const (
	TaskTypeCanonicalRebuildDimensionLayerBackfill = "canonical_rebuild_dimension_layer_backfill"
	TaskTypeCanonicalRebuildDimensionLayerListing  = "canonical_rebuild_dimension_layer_listing"
	TaskTypeCanonicalResolveDirtyDimensionTargets  = "canonical_resolve_dirty_dimension_targets"
	TaskTypeCanonicalResolveDimensionTarget        = "canonical_resolve_dimension_target"
	TaskTypeCanonicalExtractManagerCertificate     = "canonical_extract_manager_certificate"
	TaskTypeCanonicalProjectManagerCertificate     = "canonical_project_manager_certificate"
	TaskTypeCanonicalBackfillTargetSources         = "canonical_backfill_target_sources"
	TaskTypeCanonicalBackfillBuildingCoordinates   = "canonical_backfill_building_coordinates"
	TaskTypeCanonicalRebuildSpatialReadModel       = "canonical_rebuild_spatial_read_model"
	TaskTypeCanonicalBackfillDetachedHouses        = "canonical_backfill_detached_houses"
)

func (c *Consumer) handleCanonicalTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "canonical", logger, msg, c.runCanonicalSyncJob)
}

func (c *Consumer) runCanonicalSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	switch job.SyncJobKind {
	case TaskTypeCanonicalizeSourceAdsFanout:
		return c.handleCanonicalizeSourceAdsFanout(ctx, logger, job)
	case TaskTypeCanonicalizeSourceAd:
		return c.handleCanonicalizeSourceAd(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSourcesBackfill:
		return c.handleCanonicalMatchSaleListingSourcesBackfill(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSourcesFanout:
		return c.handleCanonicalMatchSaleListingSourcesFanout(ctx, logger, job)
	case TaskTypeCanonicalMatchSaleListingSource:
		return c.handleCanonicalMatchSaleListingSource(ctx, logger, job)
	case TaskTypeCanonicalRebuildDimensionLayerBackfill:
		return c.handleCanonicalRebuildDimensionLayerBackfill(ctx, logger, job)
	case TaskTypeCanonicalRebuildDimensionLayerListing:
		return c.handleCanonicalRebuildDimensionLayerListing(ctx, logger, job)
	case TaskTypeCanonicalResolveDirtyDimensionTargets:
		return c.handleCanonicalResolveDirtyDimensionTargets(ctx, logger, job)
	case TaskTypeCanonicalResolveDimensionTarget:
		return c.handleCanonicalResolveDimensionTarget(ctx, logger, job)
	case TaskTypeCanonicalExtractManagerCertificate:
		return c.handleCanonicalExtractManagerCertificate(ctx, logger, job)
	case TaskTypeCanonicalProjectManagerCertificate:
		return c.handleCanonicalProjectManagerCertificate(ctx, logger, job)
	case TaskTypeCanonicalBackfillTargetSources:
		return c.handleCanonicalBackfillTargetSources(ctx, logger, job)
	case TaskTypeCanonicalBackfillBuildingCoordinates:
		return c.handleCanonicalBackfillBuildingCoordinates(ctx, logger, job)
	case TaskTypeCanonicalRebuildSpatialReadModel:
		return c.handleCanonicalRebuildSpatialReadModel(ctx, logger, job)
	case TaskTypeCanonicalBackfillDetachedHouses:
		return c.handleCanonicalBackfillDetachedHouses(ctx, logger, job)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown canonical sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}

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
	PropertyDocumentID string `json:"property_document_id"`
	Model              string `json:"model,omitempty"`
}

type dirtyDimensionTargetRow struct {
	TargetType string
	TargetID   uuid.UUID
	DirtyAt    time.Time
}

func (c *Consumer) handleCanonicalExtractManagerCertificate(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.extract_manager_certificate"))
	if c.propertiesService == nil {
		return taskqueue.NewPermanentError(fmt.Errorf("properties service is not configured"), "properties service missing")
	}
	payload, err := decodeManagerCertificateDocumentPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	result, err := c.propertiesService.ExtractManagerCertificateSource(ctx, payload.PropertyDocumentID, payload.Model)
	if err != nil {
		return fmt.Errorf("extract manager certificate source %s: %w", payload.PropertyDocumentID, err)
	}
	documentID, err := uuid.Parse(result.Document.ID)
	if err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("parse extracted document id: %w", err), "invalid result")
	}
	if err := c.enqueueManagerCertificateProjection(ctx, documentID, time.Now()); err != nil {
		return fmt.Errorf("enqueue manager certificate projection %s: %w", payload.PropertyDocumentID, err)
	}
	logger.InfoContext(ctx, "manager certificate source extracted", "property_document_id", payload.PropertyDocumentID, "model", result.Model, "schema_version", result.SchemaVersion, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalProjectManagerCertificate(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.project_manager_certificate"))
	if c.propertiesService == nil {
		return taskqueue.NewPermanentError(fmt.Errorf("properties service is not configured"), "properties service missing")
	}
	payload, err := decodeManagerCertificateDocumentPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	result, err := c.propertiesService.ProjectManagerCertificateExtraction(ctx, payload.PropertyDocumentID)
	if err != nil {
		return fmt.Errorf("project manager certificate %s: %w", payload.PropertyDocumentID, err)
	}
	logger.InfoContext(ctx, "manager certificate projected", "property_document_id", payload.PropertyDocumentID, "offering_id", result.Document.OfferingID, "claims", result.Claims, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalBackfillTargetSources(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.backfill_target_sources"))
	tag, err := c.pool.Exec(ctx, `
INSERT INTO public.property_target_sources (
    target_type, target_id, source_provider, source_kind, source_table, source_id, source_id_value, source_external_id, source_url, link_status, link_method, link_score, link_reasons, first_seen_at, last_seen_at, created_at, updated_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT 'offering'::text AS target_type, pos.property_offering_id AS target_id, sl.sale_listing_source_provider AS source_provider, sl.sale_listing_source_kind AS source_kind, 'property_source_offerings'::text AS source_table, sl.sale_listing_id AS source_id, sl.sale_listing_id::text AS source_id_value, sl.sale_listing_native_id AS source_external_id, sl.sale_listing_url AS source_url, pos.property_offering_source_link_status AS link_status, pos.property_offering_source_link_method AS link_method, pos.property_offering_source_link_score AS link_score, pos.property_offering_source_link_reasons AS link_reasons, sl.sale_listing_first_seen_at AS first_seen_at, sl.sale_listing_last_seen_at AS last_seen_at, pos.property_offering_source_created_at AS created_at, pos.property_offering_source_updated_at AS updated_at
    FROM public.property_offering_sources pos
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, link_status, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_id = COALESCE(EXCLUDED.source_id, property_target_sources.source_id),
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(property_target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, property_target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now();
INSERT INTO public.property_target_sources (
    target_type, target_id, source_provider, source_kind, source_table, source_id, source_id_value, source_external_id, source_url, link_status, link_method, link_score, link_reasons, first_seen_at, last_seen_at, created_at, updated_at
)
SELECT DISTINCT ON (target_type, target_id, source_provider, source_kind, source_table, source_id_value) *
FROM (
    SELECT 'housing_company'::text AS target_type, housing_company_id AS target_id, housing_company_source_provider AS source_provider, housing_company_source_kind AS source_kind, housing_company_source_table AS source_table, housing_company_source_id AS source_id, housing_company_source_id_value AS source_id_value, housing_company_source_external_id AS source_external_id, housing_company_source_url AS source_url, housing_company_source_link_status AS link_status, housing_company_source_link_method AS link_method, housing_company_source_link_score AS link_score, housing_company_source_link_reasons AS link_reasons, housing_company_source_first_seen_at AS first_seen_at, housing_company_source_last_seen_at AS last_seen_at, housing_company_source_created_at AS created_at, housing_company_source_updated_at AS updated_at
    FROM public.housing_company_sources
) rows
ORDER BY target_type, target_id, source_provider, source_kind, source_table, source_id_value, link_status, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
    source_id = COALESCE(EXCLUDED.source_id, property_target_sources.source_id),
    source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
    source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(property_target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, property_target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
    updated_at = now()`)
	if err != nil {
		return fmt.Errorf("backfill target sources: %w", err)
	}
	logger.InfoContext(ctx, "target sources backfilled", "rows", tag.RowsAffected(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalBackfillBuildingCoordinates(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
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
    LEFT JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
    LEFT JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
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

func (c *Consumer) handleCanonicalRebuildSpatialReadModel(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_spatial_read_model"))
	logger.InfoContext(ctx, "spatial read model is served by direct SQL", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalBackfillDetachedHouses(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.backfill_detached_houses"))
	payload := detachedHouseBackfillPayload{BatchSize: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode detached house backfill payload: %w", err), "invalid payload")
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
		if err := c.enqueueDetachedHouseBackfill(ctx, payload.BatchSize, time.Now().Add(30*time.Second)); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "detached houses backfilled", "count", count, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalRebuildDimensionLayerBackfill(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_dimension_layer_backfill"))
	payload := dimensionLayerBackfillPayload{BatchSize: 500}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode dimension layer backfill payload: %w", err), "invalid payload")
		}
	}
	if payload.BatchSize <= 0 || payload.BatchSize > 5000 {
		payload.BatchSize = 500
	}
	checkpoint := dimensionLayerBackfillCheckpoint{}
	if len(job.SyncJobCheckpoint) > 0 {
		if err := json.Unmarshal(job.SyncJobCheckpoint, &checkpoint); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode dimension layer backfill checkpoint: %w", err), "invalid checkpoint")
		}
	}
	var cursor *uuid.UUID
	cursorValue := firstNonEmpty(payload.AfterSaleListingID, checkpoint.LastSaleListingID)
	if cursorValue != "" {
		parsed, err := uuid.Parse(cursorValue)
		if err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("parse dimension layer checkpoint cursor: %w", err), "invalid checkpoint")
		}
		cursor = &parsed
	}
	listingIDs, err := c.listDimensionLayerBackfillListingIDs(ctx, cursor, payload.BatchSize)
	if err != nil {
		return err
	}
	if len(listingIDs) == 0 {
		c.updateSyncJobCheckpoint(ctx, job, checkpoint)
		logger.InfoContext(ctx, "dimension layer backfill completed", "enqueued", checkpoint.Enqueued, "outcome", logging.OutcomeSuccess)
		return nil
	}
	for _, listingID := range listingIDs {
		if err := c.enqueueDimensionLayerListing(ctx, listingID, "backfill", nil, time.Now()); err != nil {
			return fmt.Errorf("enqueue dimension layer listing %s: %w", listingID, err)
		}
		cursor = &listingID
		checkpoint.LastSaleListingID = listingID.String()
		checkpoint.Enqueued++
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	c.updateSyncJobCheckpoint(ctx, job, checkpoint)
	if len(listingIDs) == int(payload.BatchSize) {
		if err := c.enqueueDimensionLayerBackfill(ctx, checkpoint.LastSaleListingID, payload.BatchSize, time.Now().Add(time.Minute)); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "dimension layer backfill batch enqueued", "enqueued", checkpoint.Enqueued, "last_sale_listing_id", checkpoint.LastSaleListingID, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalRebuildDimensionLayerListing(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.rebuild_dimension_layer_listing"))
	payload, err := decodeDimensionLayerListingPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	saleListingID, err := uuid.Parse(payload.SaleListingID)
	if err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("parse sale listing id: %w", err), "invalid payload")
	}
	result, err := c.rebuildDimensionLayerForListing(ctx, saleListingID, payload.ExpectedDirtyAt)
	if err != nil {
		return fmt.Errorf("rebuild dimension layer for listing %s: %w", saleListingID, err)
	}
	logger.InfoContext(ctx, "dimension layer listing rebuilt", "sale_listing_id", saleListingID.String(), "result", string(result), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalResolveDirtyDimensionTargets(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.resolve_dirty_dimension_targets"))
	payload := dirtyDimensionTargetsPayload{Limit: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode dirty dimension targets payload: %w", err), "invalid payload")
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
			if err := c.enqueueDimensionLayerListing(ctx, target.TargetID, "dirty_target", &target.DirtyAt, time.Now()); err != nil {
				return err
			}
		} else {
			if err := c.enqueueDimensionTarget(ctx, target.TargetType, target.TargetID, target.DirtyAt, time.Now()); err != nil {
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

func (c *Consumer) handleCanonicalResolveDimensionTarget(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonical.resolve_dimension_target"))
	payload, err := decodeDirtyDimensionTargetPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	targetID, err := uuid.Parse(payload.TargetID)
	if err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("parse target id: %w", err), "invalid payload")
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
SELECT sale_listing_id
FROM public.property_source_offerings
WHERE ($1::uuid IS NULL OR sale_listing_id > $1::uuid)
ORDER BY sale_listing_id
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
	var result json.RawMessage
	var err error
	if expectedDirtyAt != nil {
		result, err = c.queries.RebuildListingDimensionLayerAt(ctx, db.RebuildListingDimensionLayerAtParams{SaleListingID: saleListingID, ExpectedDirtyAt: expectedDirtyAt})
	} else {
		result, err = c.queries.RebuildListingDimensionLayer(ctx, saleListingID)
	}
	if err != nil {
		return nil, fmt.Errorf("rebuild listing dimension layer: %w", err)
	}
	return result, nil
}

func (c *Consumer) enqueueDimensionLayerListing(ctx context.Context, saleListingID uuid.UUID, reason string, expectedDirtyAt *time.Time, runAfter time.Time) error {
	payload, err := json.Marshal(dimensionLayerListingPayload{SaleListingID: saleListingID.String(), Reason: reason, ExpectedDirtyAt: expectedDirtyAt})
	if err != nil {
		return fmt.Errorf("marshal dimension layer listing payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalRebuildDimensionLayerListing,
		EntityID:    fmt.Sprintf("sale_listing:%s", saleListingID),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}

func (c *Consumer) enqueueDimensionTarget(ctx context.Context, targetType string, targetID uuid.UUID, expectedDirtyAt time.Time, runAfter time.Time) error {
	payload, err := json.Marshal(dirtyDimensionTargetPayload{TargetType: targetType, TargetID: targetID.String(), ExpectedDirtyAt: &expectedDirtyAt})
	if err != nil {
		return fmt.Errorf("marshal dimension target payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalResolveDimensionTarget,
		EntityID:    fmt.Sprintf("dimension_target:%s:%s", targetType, targetID),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}

func (c *Consumer) enqueueDirtyDimensionTargetFanout(ctx context.Context) error {
	payload, err := json.Marshal(dirtyDimensionTargetsPayload{Limit: 1000})
	if err != nil {
		return fmt.Errorf("marshal dirty dimension targets payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalResolveDirtyDimensionTargets,
		EntityID:    "dimension_targets:dirty",
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		Payload:     payload,
	})
	return err
}

func (c *Consumer) enqueueDimensionLayerBackfill(ctx context.Context, afterSaleListingID string, batchSize int32, runAfter time.Time) error {
	payload, err := json.Marshal(dimensionLayerBackfillPayload{BatchSize: batchSize, AfterSaleListingID: afterSaleListingID})
	if err != nil {
		return fmt.Errorf("marshal dimension layer backfill payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalRebuildDimensionLayerBackfill,
		EntityID:    fmt.Sprintf("dimension_layer_backfill:%s", afterSaleListingID),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}

func (c *Consumer) enqueueDetachedHouseBackfill(ctx context.Context, batchSize int32, runAfter time.Time) error {
	payload, err := json.Marshal(detachedHouseBackfillPayload{BatchSize: batchSize})
	if err != nil {
		return fmt.Errorf("marshal detached house backfill payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalBackfillDetachedHouses,
		EntityID:    fmt.Sprintf("detached_house_backfill:%d", runAfter.UnixNano()),
		Priority:    int32(taskqueue.PriorityLow),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
	})
	return err
}

func (c *Consumer) enqueueManagerCertificateProjection(ctx context.Context, documentID uuid.UUID, runAfter time.Time) error {
	payload, err := json.Marshal(managerCertificateDocumentPayload{PropertyDocumentID: documentID.String()})
	if err != nil {
		return fmt.Errorf("marshal manager certificate projection payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalProjectManagerCertificate,
		EntityID:    fmt.Sprintf("property_document:%s", documentID),
		Priority:    int32(taskqueue.PriorityHigh),
		MaxAttempts: 3,
		RunAfter:    runAfter,
		Payload:     payload,
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

func decodeManagerCertificateDocumentPayload(job db.SyncJob) (managerCertificateDocumentPayload, error) {
	var payload managerCertificateDocumentPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return managerCertificateDocumentPayload{}, fmt.Errorf("decode manager certificate payload: %w", err)
		}
	}
	if payload.PropertyDocumentID == "" {
		_, value, err := parseJobEntity(job.SyncJobEntityID)
		if err != nil {
			return managerCertificateDocumentPayload{}, fmt.Errorf("parse property document entity: %w", err)
		}
		payload.PropertyDocumentID = value
	}
	if payload.PropertyDocumentID == "" {
		return managerCertificateDocumentPayload{}, fmt.Errorf("property_document_id is required")
	}
	if _, err := uuid.Parse(payload.PropertyDocumentID); err != nil {
		return managerCertificateDocumentPayload{}, fmt.Errorf("property_document_id must be a uuid: %w", err)
	}
	return payload, nil
}

func decodeDimensionLayerListingPayload(job db.SyncJob) (dimensionLayerListingPayload, error) {
	var payload dimensionLayerListingPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return dimensionLayerListingPayload{}, fmt.Errorf("decode dimension layer listing payload: %w", err)
		}
	}
	if payload.SaleListingID == "" {
		_, value, err := parseJobEntity(job.SyncJobEntityID)
		if err != nil {
			return dimensionLayerListingPayload{}, fmt.Errorf("parse sale listing entity: %w", err)
		}
		payload.SaleListingID = value
	}
	if payload.SaleListingID == "" {
		return dimensionLayerListingPayload{}, fmt.Errorf("sale_listing_id is required")
	}
	if _, err := uuid.Parse(payload.SaleListingID); err != nil {
		return dimensionLayerListingPayload{}, fmt.Errorf("sale_listing_id must be a uuid: %w", err)
	}
	return payload, nil
}

func decodeDirtyDimensionTargetPayload(job db.SyncJob) (dirtyDimensionTargetPayload, error) {
	var payload dirtyDimensionTargetPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return dirtyDimensionTargetPayload{}, fmt.Errorf("decode dimension target payload: %w", err)
		}
	}
	if payload.TargetType == "" || payload.TargetID == "" {
		return dirtyDimensionTargetPayload{}, fmt.Errorf("target_type and target_id are required")
	}
	if _, err := uuid.Parse(payload.TargetID); err != nil {
		return dirtyDimensionTargetPayload{}, fmt.Errorf("target_id must be a uuid: %w", err)
	}
	return payload, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (c *Consumer) updateSyncJobCheckpoint(ctx context.Context, job db.SyncJob, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.syncJobs.UpdateCheckpoint(ctx, job, raw)
}
