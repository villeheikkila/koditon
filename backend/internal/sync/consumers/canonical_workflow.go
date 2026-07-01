package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

var canonicalWorkflowKinds = []string{
	TaskTypeCanonicalizeSourceAdsFanout,
	TaskTypeCanonicalizeSourceAd,
	TaskTypeCanonicalMatchSaleListingSourcesBackfill,
	TaskTypeCanonicalMatchSaleListingSourcesFanout,
	TaskTypeCanonicalMatchSaleListingSource,
	TaskTypeCanonicalRebuildDimensionLayerBackfill,
	TaskTypeCanonicalRebuildDimensionLayerListing,
	TaskTypeCanonicalResolveDirtyDimensionTargets,
	TaskTypeCanonicalResolveDimensionTarget,
	TaskTypeCanonicalBackfillBuildingCoordinates,
	TaskTypeCanonicalBackfillDetachedHouses,
	TaskTypeCanonicalProjectManagerCertificate,
}

var canonicalLLMWorkflowKinds = []string{
	TaskTypeCanonicalExtractManagerCertificate,
}

type canonicalFanoutResult struct {
	Enqueued int `json:"enqueued"`
}

type canonicalMatchListingWorkflowResult struct {
	SaleListingID string                    `json:"sale_listing_id"`
	Status        string                    `json:"status"`
	Run           *canonicalMatchRunSummary `json:"run,omitempty"`
}

type canonicalMatchLoadedListing struct {
	Found bool                         `json:"found"`
	Row   canonicalMatchSaleListingRow `json:"row"`
}

type dimensionLayerBackfillResult struct {
	LastSaleListingID string `json:"last_sale_listing_id,omitempty"`
	Enqueued          int    `json:"enqueued"`
}

type detachedHouseBackfillResult struct {
	Count int32 `json:"count"`
}

func (c *Consumer) startCanonicalWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.canonicalWorkflowClient == nil {
		return errors.New("canonical absurd workflow client is not configured")
	}
	for _, kind := range canonicalWorkflowKinds {
		def, ok := workflows.FindDefinition(kind)
		if !ok {
			return fmt.Errorf("missing canonical workflow definition: %s", kind)
		}
		task := absurd.Task[json.RawMessage, json.RawMessage](
			kind,
			c.handleCanonicalWorkflow,
			absurd.TaskOptions{QueueName: workflows.QueueCanonicalDB, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation},
		)
		if err := c.canonicalWorkflowClient.Register(task); err != nil {
			return fmt.Errorf("register canonical workflow %s: %w", kind, err)
		}
	}
	logger := logging.With(c.logger, logging.Op("consumer.canonical.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.canonicalWorkflowCancel = cancel
	c.canonicalWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.canonicalWorkflowDone)
		logger.InfoContext(workerCtx, "canonical absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueueCanonicalDB)
		err := c.canonicalWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "canonical-db",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "canonical absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "canonical absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
	return nil
}

func (c *Consumer) startCanonicalLLMWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.canonicalLLMWorkflowClient == nil {
		return errors.New("canonical llm absurd workflow client is not configured")
	}
	if c.propertiesService == nil {
		return errors.New("properties service is not configured")
	}
	for _, kind := range canonicalLLMWorkflowKinds {
		def, ok := workflows.FindDefinition(kind)
		if !ok {
			return fmt.Errorf("missing canonical llm workflow definition: %s", kind)
		}
		task := absurd.Task[json.RawMessage, json.RawMessage](
			kind,
			c.handleCanonicalWorkflow,
			absurd.TaskOptions{QueueName: workflows.QueueCanonicalLLM, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation},
		)
		if err := c.canonicalLLMWorkflowClient.Register(task); err != nil {
			return fmt.Errorf("register canonical llm workflow %s: %w", kind, err)
		}
	}
	logger := logging.With(c.logger, logging.Op("consumer.canonical_llm.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.canonicalLLMWorkflowCancel = cancel
	c.canonicalLLMWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.canonicalLLMWorkflowDone)
		logger.InfoContext(workerCtx, "canonical llm absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueueCanonicalLLM)
		err := c.canonicalLLMWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "canonical-llm",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "canonical llm absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "canonical llm absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
	return nil
}

func (c *Consumer) handleCanonicalWorkflow(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	taskName := absurd.MustTaskContext(ctx).TaskName()
	logger := logging.With(c.logger,
		logging.Op("consumer.canonical.workflow"),
		slog.String("task_type", taskName),
	)
	var result any
	var err error
	switch taskName {
	case TaskTypeCanonicalizeSourceAdsFanout:
		result, err = c.runCanonicalizeSourceAdsFanoutWorkflow(ctx, logger, params)
	case TaskTypeCanonicalizeSourceAd:
		result, err = c.runCanonicalizeSourceAdWorkflow(ctx, logger, params)
	case TaskTypeCanonicalMatchSaleListingSourcesBackfill:
		result, err = c.runCanonicalMatchBackfillWorkflow(ctx, logger, params)
	case TaskTypeCanonicalMatchSaleListingSourcesFanout:
		result, err = c.runCanonicalMatchFanoutWorkflow(ctx, logger, params)
	case TaskTypeCanonicalMatchSaleListingSource:
		result, err = c.runCanonicalMatchSaleListingSourceWorkflow(ctx, logger, params)
	case TaskTypeCanonicalRebuildDimensionLayerBackfill:
		result, err = c.runDimensionLayerBackfillWorkflow(ctx, logger, params)
	case TaskTypeCanonicalRebuildDimensionLayerListing:
		result, err = c.runDimensionLayerListingWorkflow(ctx, logger, params)
	case TaskTypeCanonicalResolveDirtyDimensionTargets:
		result, err = c.runResolveDirtyDimensionTargetsWorkflow(ctx, logger, params)
	case TaskTypeCanonicalResolveDimensionTarget:
		result, err = c.runResolveDimensionTargetWorkflow(ctx, logger, params)
	case TaskTypeCanonicalBackfillBuildingCoordinates:
		result, err = c.runCanonicalBackfillBuildingCoordinatesWorkflow(ctx, logger)
	case TaskTypeCanonicalBackfillDetachedHouses:
		result, err = c.runCanonicalBackfillDetachedHousesWorkflow(ctx, logger, params)
	case TaskTypeCanonicalExtractManagerCertificate:
		result, err = c.runCanonicalExtractManagerCertificateWorkflow(ctx, logger, params)
	case TaskTypeCanonicalProjectManagerCertificate:
		result, err = c.runCanonicalProjectManagerCertificateWorkflow(ctx, logger, params)
	default:
		return nil, fmt.Errorf("unknown canonical workflow kind: %s", taskName)
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical workflow result: %w", err)
	}
	return raw, nil
}

func (c *Consumer) runCanonicalizeSourceAdsFanoutWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalFanoutResult, error) {
	payload, err := decodeCanonicalizeSourceAdsFanoutWorkflowPayload(params)
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	result, err := absurd.Step(ctx, "scan-and-spawn-source-ads", func(ctx context.Context) (canonicalFanoutResult, error) {
		rows, err := c.pool.Query(ctx, `
(SELECT 'frontdoor_ad'::text AS source_table, frontdoor_ad_id::text AS source_id
 FROM public.frontdoor_ads
 WHERE frontdoor_ad_data IS NOT NULL
     AND (frontdoor_ad_data_hash IS NULL
         OR frontdoor_ad_data_normalized_at IS NULL
         OR frontdoor_ad_data_changed_at > frontdoor_ad_data_normalized_at
         OR frontdoor_ad_data_normalized_version < $2)
 ORDER BY frontdoor_ad_updated_at ASC
 LIMIT $1)
UNION ALL
(SELECT 'shortcut_ad'::text AS source_table, shortcut_ad_id::text AS source_id
 FROM public.shortcut_ads
 WHERE shortcut_ad_data IS NOT NULL
     AND (shortcut_ad_data_hash IS NULL
         OR shortcut_ad_data_normalized_at IS NULL
         OR shortcut_ad_data_changed_at > shortcut_ad_data_normalized_at
         OR shortcut_ad_data_normalized_version < $2)
 ORDER BY shortcut_ad_updated_at ASC NULLS FIRST
 LIMIT $1)
UNION ALL
(SELECT 'frontdoor_building_announcement'::text AS source_table, frontdoor_building_announcement_id::text AS source_id
 FROM public.frontdoor_building_announcements
 WHERE frontdoor_building_announcement_rent_period IS NULL
     AND frontdoor_building_announcement_rental_unique_no IS NULL
     AND (frontdoor_building_announcement_data_normalized_at IS NULL
         OR frontdoor_building_announcement_data_normalized_version < $2)
 ORDER BY frontdoor_building_announcement_last_seen_at ASC
 LIMIT $1)`, payload.Limit, currentSourceAdCanonicalizationVersion)
		if err != nil {
			return canonicalFanoutResult{}, fmt.Errorf("list source ads for canonicalization: %w", err)
		}
		defer rows.Close()
		enqueued := 0
		for rows.Next() {
			var sourceTable string
			var sourceID string
			if err := rows.Scan(&sourceTable, &sourceID); err != nil {
				return canonicalFanoutResult{}, fmt.Errorf("scan canonicalize source ad fanout row: %w", err)
			}
			if err := c.enqueueCanonicalizeSourceAd(ctx, sourceTable, sourceID, 0); err != nil {
				return canonicalFanoutResult{}, err
			}
			enqueued++
		}
		if err := rows.Err(); err != nil {
			return canonicalFanoutResult{}, fmt.Errorf("iterate canonicalize source ad fanout rows: %w", err)
		}
		return canonicalFanoutResult{Enqueued: enqueued}, nil
	})
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	if result.Enqueued > 0 {
		if err := absurd.SleepFor(ctx, "pause-before-next-fanout", 30*time.Second); err != nil {
			return canonicalFanoutResult{}, err
		}
		if _, err := absurd.Step(ctx, "spawn-next-fanout", func(ctx context.Context) (canonicalFanoutResult, error) {
			if err := c.spawnCanonicalizeSourceAdsFanout(ctx, payload.Limit); err != nil {
				return canonicalFanoutResult{}, err
			}
			return canonicalFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return canonicalFanoutResult{}, err
		}
	}
	logger.InfoContext(ctx, "canonicalize source ad tasks spawned", "count", result.Enqueued, "next_fanout_scheduled", result.Enqueued > 0, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) runCanonicalizeSourceAdWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalizeSourceAdPayload, error) {
	payload, err := decodeCanonicalizeSourceAdWorkflowPayload(params)
	if err != nil {
		return canonicalizeSourceAdPayload{}, err
	}
	_, err = absurd.Step(ctx, "canonicalize-source-ad", func(ctx context.Context) (struct{}, error) {
		switch payload.SourceTable {
		case "frontdoor_ad":
			return struct{}{}, c.canonicalizeFrontdoorAd(ctx, logger, payload.SourceID)
		case "shortcut_ad":
			return struct{}{}, c.canonicalizeShortcutAd(ctx, logger, payload.SourceID)
		case "frontdoor_building_announcement":
			return struct{}{}, c.canonicalizeFrontdoorBuildingAnnouncement(ctx, logger, payload.SourceID)
		default:
			return struct{}{}, fmt.Errorf("unknown source table %q", payload.SourceTable)
		}
	})
	if err != nil {
		return canonicalizeSourceAdPayload{}, err
	}
	return payload, nil
}

func (c *Consumer) runCanonicalMatchBackfillWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalMatchRunSummary, error) {
	payload, err := decodeCanonicalMatchBackfillWorkflowPayload(params)
	if err != nil {
		return canonicalMatchRunSummary{}, err
	}
	run, err := absurd.Step(ctx, "run-source-match-backfill", func(ctx context.Context) (canonicalMatchRunSummary, error) {
		return c.runCanonicalSourceMatchBackfill(ctx, int(payload.ScoreThreshold), int(payload.CompetitorMargin))
	})
	if err != nil {
		return canonicalMatchRunSummary{}, err
	}
	logger.InfoContext(ctx, "canonical sale listing source backfill matched", "run_id", run.RunID, "candidates", run.Candidates, "auto_linked", run.AutoLinked, "ambiguous", run.Ambiguous, "outcome", logging.OutcomeSuccess)
	return run, nil
}

func (c *Consumer) runCanonicalMatchFanoutWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalFanoutResult, error) {
	payload, err := decodeCanonicalMatchFanoutWorkflowPayload(params)
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	return absurd.Step(ctx, "scan-and-spawn-listings", func(ctx context.Context) (canonicalFanoutResult, error) {
		rows, err := c.pool.Query(ctx, `
SELECT sl.sale_listing_id::text, COALESCE(sl.sale_listing_source_match_attempt_count, 0)
FROM public.property_source_offerings sl
JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
WHERE sl.sale_listing_source_kind = 'ad'
    AND source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.link_status <> 'rejected'
    AND source_link.link_method <> 'manual'
    AND COALESCE(sl.sale_listing_source_match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at) <= now()
ORDER BY COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at), sl.sale_listing_updated_at
LIMIT $1`, payload.Limit)
		if err != nil {
			return canonicalFanoutResult{}, fmt.Errorf("list sale listings for canonical source matching: %w", err)
		}
		defer rows.Close()
		enqueued := 0
		for rows.Next() {
			var saleListingID string
			var attempt int32
			if err := rows.Scan(&saleListingID, &attempt); err != nil {
				return canonicalFanoutResult{}, fmt.Errorf("scan canonical source match fanout row: %w", err)
			}
			if err := c.spawnCanonicalSourceMatchSaleListing(ctx, saleListingID, attempt+1); err != nil {
				return canonicalFanoutResult{}, err
			}
			enqueued++
		}
		if err := rows.Err(); err != nil {
			return canonicalFanoutResult{}, fmt.Errorf("iterate canonical source match fanout rows: %w", err)
		}
		logger.InfoContext(ctx, "canonical sale listing source match tasks spawned", "count", enqueued, "outcome", logging.OutcomeSuccess)
		return canonicalFanoutResult{Enqueued: enqueued}, nil
	})
}

func (c *Consumer) runCanonicalMatchSaleListingSourceWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalMatchListingWorkflowResult, error) {
	payload, err := decodeCanonicalMatchSaleListingWorkflowPayload(params)
	if err != nil {
		return canonicalMatchListingWorkflowResult{}, err
	}
	loaded, err := absurd.Step(ctx, "load-listing", func(ctx context.Context) (canonicalMatchLoadedListing, error) {
		row, err := c.loadCanonicalMatchSaleListing(ctx, payload.SaleListingID)
		if errors.Is(err, pgx.ErrNoRows) {
			return canonicalMatchLoadedListing{}, nil
		}
		return canonicalMatchLoadedListing{Found: true, Row: row}, err
	})
	if err != nil {
		return canonicalMatchListingWorkflowResult{}, err
	}
	if !loaded.Found {
		return canonicalMatchListingWorkflowResult{SaleListingID: payload.SaleListingID, Status: "not_found"}, nil
	}
	row := loaded.Row
	if row.LinkMethod != nil && *row.LinkMethod == "manual" {
		if err := c.updateCanonicalSourceMatchStateStep(ctx, "mark-manual-linked", row.ID, "manual_linked", nil, nil); err != nil {
			return canonicalMatchListingWorkflowResult{}, err
		}
		if err := c.projectTypedHousingCompanyProfileForSaleListingStep(ctx, row.ID); err != nil {
			return canonicalMatchListingWorkflowResult{}, err
		}
		return canonicalMatchListingWorkflowResult{SaleListingID: row.ID, Status: "manual_linked"}, nil
	}
	run, err := absurd.Step(ctx, "run-source-match", func(ctx context.Context) (canonicalMatchRunSummary, error) {
		return c.runCanonicalSourceMatchForSaleListing(ctx, row.ID)
	})
	if err != nil {
		return canonicalMatchListingWorkflowResult{}, err
	}
	if run.AutoLinked > 0 {
		logger.InfoContext(ctx, "canonical sale listing source auto-linked", "sale_listing_id", row.ID, "run_id", run.RunID, "outcome", logging.OutcomeSuccess)
		if err := c.updateCanonicalSourceMatchStateStep(ctx, "mark-auto-linked", row.ID, "auto_linked", nil, &run.RunID); err != nil {
			return canonicalMatchListingWorkflowResult{}, err
		}
		if err := c.projectTypedHousingCompanyProfileForSaleListingStep(ctx, row.ID); err != nil {
			return canonicalMatchListingWorkflowResult{}, err
		}
		return canonicalMatchListingWorkflowResult{SaleListingID: row.ID, Status: "auto_linked", Run: &run}, nil
	}
	if run.Ambiguous > 0 {
		logger.InfoContext(ctx, "canonical sale listing source needs review", "sale_listing_id", row.ID, "run_id", run.RunID, "candidates", run.Ambiguous)
		if err := c.updateCanonicalSourceMatchStateStep(ctx, "mark-needs-review", row.ID, "needs_review", nil, &run.RunID); err != nil {
			return canonicalMatchListingWorkflowResult{}, err
		}
		return canonicalMatchListingWorkflowResult{SaleListingID: row.ID, Status: "needs_review", Run: &run}, nil
	}
	next := time.Now().UTC().Add(7 * 24 * time.Hour)
	status := "deferred"
	if run.Candidates == 0 {
		status = "noop"
	}
	if err := c.updateCanonicalSourceMatchStateStep(ctx, "mark-"+status, row.ID, status, &next, &run.RunID); err != nil {
		return canonicalMatchListingWorkflowResult{}, err
	}
	return canonicalMatchListingWorkflowResult{SaleListingID: row.ID, Status: status, Run: &run}, nil
}

func (c *Consumer) runDimensionLayerBackfillWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (dimensionLayerBackfillResult, error) {
	payload, err := decodeDimensionLayerBackfillWorkflowPayload(params)
	if err != nil {
		return dimensionLayerBackfillResult{}, err
	}
	var cursor *uuid.UUID
	if payload.AfterSaleListingID != "" {
		parsed, err := uuid.Parse(payload.AfterSaleListingID)
		if err != nil {
			return dimensionLayerBackfillResult{}, fmt.Errorf("parse dimension layer cursor: %w", err)
		}
		cursor = &parsed
	}
	listingIDs, err := absurd.Step(ctx, "list-listing-ids", func(ctx context.Context) ([]uuid.UUID, error) {
		return c.listDimensionLayerBackfillListingIDs(ctx, cursor, payload.BatchSize)
	})
	if err != nil {
		return dimensionLayerBackfillResult{}, err
	}
	result, err := absurd.Step(ctx, "spawn-listing-rebuilds", func(ctx context.Context) (dimensionLayerBackfillResult, error) {
		result := dimensionLayerBackfillResult{}
		for _, listingID := range listingIDs {
			if err := c.enqueueDimensionLayerListing(ctx, listingID, "backfill", nil); err != nil {
				return dimensionLayerBackfillResult{}, fmt.Errorf("enqueue dimension layer listing %s: %w", listingID, err)
			}
			result.LastSaleListingID = listingID.String()
			result.Enqueued++
		}
		return result, nil
	})
	if err != nil {
		return dimensionLayerBackfillResult{}, err
	}
	if len(listingIDs) == int(payload.BatchSize) && result.LastSaleListingID != "" {
		if err := absurd.SleepFor(ctx, "pause-before-next-page", time.Minute); err != nil {
			return dimensionLayerBackfillResult{}, err
		}
		if _, err := absurd.Step(ctx, "spawn-next-page", func(ctx context.Context) (canonicalFanoutResult, error) {
			if err := c.enqueueDimensionLayerBackfill(ctx, result.LastSaleListingID, payload.BatchSize); err != nil {
				return canonicalFanoutResult{}, err
			}
			return canonicalFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return dimensionLayerBackfillResult{}, err
		}
	}
	logger.InfoContext(ctx, "dimension layer backfill batch spawned", "enqueued", result.Enqueued, "last_sale_listing_id", result.LastSaleListingID, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) runDimensionLayerListingWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (json.RawMessage, error) {
	payload, err := decodeDimensionLayerListingWorkflowPayload(params)
	if err != nil {
		return nil, err
	}
	saleListingID, err := uuid.Parse(payload.SaleListingID)
	if err != nil {
		return nil, fmt.Errorf("parse sale listing id: %w", err)
	}
	if _, err := absurd.Step(ctx, "ensure-physical-building", func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.queries.EnsurePhysicalBuildingForSaleListing(ctx, saleListingID)
	}); err != nil {
		return nil, fmt.Errorf("ensure physical building: %w", err)
	}
	if _, err := absurd.Step(ctx, "project-renovation-events", func(ctx context.Context) (struct{}, error) {
		return struct{}{}, properties.ProjectListingRenovationEvents(ctx, c.pool, saleListingID)
	}); err != nil {
		return nil, err
	}
	result, err := absurd.Step(ctx, "rebuild-listing-dimension-layer", func(ctx context.Context) (json.RawMessage, error) {
		return c.queries.RebuildListingDimensionLayerAt(ctx, db.RebuildListingDimensionLayerAtParams{SaleListingID: saleListingID, ExpectedDirtyAt: payload.ExpectedDirtyAt})
	})
	if err != nil {
		return nil, fmt.Errorf("rebuild listing dimension layer: %w", err)
	}
	logger.InfoContext(ctx, "dimension layer listing rebuilt", "sale_listing_id", saleListingID.String(), "result", string(result), "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) runResolveDirtyDimensionTargetsWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (canonicalFanoutResult, error) {
	payload, err := decodeDirtyDimensionTargetsWorkflowPayload(params)
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	targets, err := absurd.Step(ctx, "list-dirty-targets", func(ctx context.Context) ([]dirtyDimensionTargetRow, error) {
		return c.listDirtyDimensionTargets(ctx, payload.Limit)
	})
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	return absurd.Step(ctx, "spawn-target-resolvers", func(ctx context.Context) (canonicalFanoutResult, error) {
		enqueued := 0
		for _, target := range targets {
			if target.TargetType == "listing" {
				if err := c.enqueueDimensionLayerListing(ctx, target.TargetID, "dirty_target", &target.DirtyAt); err != nil {
					return canonicalFanoutResult{}, err
				}
			} else {
				if err := c.enqueueDimensionTarget(ctx, target.TargetType, target.TargetID, target.DirtyAt); err != nil {
					return canonicalFanoutResult{}, err
				}
			}
			if err := c.markDimensionTargetQueued(ctx, target.TargetType, target.TargetID); err != nil {
				return canonicalFanoutResult{}, err
			}
			enqueued++
		}
		logger.InfoContext(ctx, "dirty dimension target tasks spawned", "count", enqueued, "outcome", logging.OutcomeSuccess)
		return canonicalFanoutResult{Enqueued: enqueued}, nil
	})
}

func (c *Consumer) runResolveDimensionTargetWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (json.RawMessage, error) {
	payload, err := decodeDirtyDimensionTargetWorkflowPayload(params)
	if err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(payload.TargetID)
	if err != nil {
		return nil, fmt.Errorf("parse target id: %w", err)
	}
	result, err := absurd.Step(ctx, "resolve-values", func(ctx context.Context) (json.RawMessage, error) {
		return c.queries.ResolveDimensionTarget(ctx, db.ResolveDimensionTargetParams{TargetType: payload.TargetType, TargetID: targetID, ExpectedDirtyAt: payload.ExpectedDirtyAt})
	})
	if err != nil {
		return nil, fmt.Errorf("resolve dimension target %s:%s: %w", payload.TargetType, payload.TargetID, err)
	}
	logger.InfoContext(ctx, "dimension target resolved", "target_type", payload.TargetType, "target_id", payload.TargetID, "result", string(result), "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) runCanonicalBackfillBuildingCoordinatesWorkflow(ctx context.Context, logger *slog.Logger) (canonicalFanoutResult, error) {
	_, err := absurd.Step(ctx, "backfill-building-coordinates", func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.handleCanonicalBackfillBuildingCoordinates(ctx, logger)
	})
	if err != nil {
		return canonicalFanoutResult{}, err
	}
	return canonicalFanoutResult{Enqueued: 0}, nil
}

func (c *Consumer) runCanonicalBackfillDetachedHousesWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (detachedHouseBackfillResult, error) {
	payload, err := decodeDetachedHouseBackfillWorkflowPayload(params)
	if err != nil {
		return detachedHouseBackfillResult{}, err
	}
	count, err := absurd.Step(ctx, "backfill-detached-property-houses", func(ctx context.Context) (int32, error) {
		return c.queries.BackfillDetachedPropertyHouses(ctx, payload.BatchSize)
	})
	if err != nil {
		return detachedHouseBackfillResult{}, fmt.Errorf("backfill detached property houses: %w", err)
	}
	if count == payload.BatchSize {
		if err := absurd.SleepFor(ctx, "pause-before-next-detached-house-backfill", 30*time.Second); err != nil {
			return detachedHouseBackfillResult{}, err
		}
		if _, err := absurd.Step(ctx, "spawn-next-detached-house-backfill", func(ctx context.Context) (canonicalFanoutResult, error) {
			if err := c.enqueueDetachedHouseBackfill(ctx, payload.BatchSize); err != nil {
				return canonicalFanoutResult{}, err
			}
			return canonicalFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return detachedHouseBackfillResult{}, err
		}
	}
	logger.InfoContext(ctx, "detached houses backfilled", "count", count, "outcome", logging.OutcomeSuccess)
	return detachedHouseBackfillResult{Count: count}, nil
}

func (c *Consumer) runCanonicalExtractManagerCertificateWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (properties.ManagerCertificateSourceExtractionResult, error) {
	payload, err := decodeManagerCertificateDocumentWorkflowPayload(params)
	if err != nil {
		return properties.ManagerCertificateSourceExtractionResult{}, err
	}
	result, err := absurd.Step(ctx, "extract-manager-certificate-source", func(ctx context.Context) (properties.ManagerCertificateSourceExtractionResult, error) {
		return c.propertiesService.ExtractManagerCertificateSource(ctx, payload.PropertyDocumentID, payload.Model)
	})
	if err != nil {
		return properties.ManagerCertificateSourceExtractionResult{}, fmt.Errorf("extract manager certificate source %s: %w", payload.PropertyDocumentID, err)
	}
	documentID, err := uuid.Parse(result.Document.ID)
	if err != nil {
		return properties.ManagerCertificateSourceExtractionResult{}, fmt.Errorf("parse extracted document id: %w", err)
	}
	if _, err := absurd.Step(ctx, "spawn-manager-certificate-projection", func(ctx context.Context) (canonicalFanoutResult, error) {
		if err := c.enqueueManagerCertificateProjection(ctx, documentID); err != nil {
			return canonicalFanoutResult{}, err
		}
		return canonicalFanoutResult{Enqueued: 1}, nil
	}); err != nil {
		return properties.ManagerCertificateSourceExtractionResult{}, err
	}
	logger.InfoContext(ctx, "manager certificate source extracted", "property_document_id", payload.PropertyDocumentID, "model", result.Model, "schema_version", result.SchemaVersion, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) runCanonicalProjectManagerCertificateWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (properties.ManagerCertificateExtractionResult, error) {
	payload, err := decodeManagerCertificateDocumentWorkflowPayload(params)
	if err != nil {
		return properties.ManagerCertificateExtractionResult{}, err
	}
	result, err := absurd.Step(ctx, "project-manager-certificate-extraction", func(ctx context.Context) (properties.ManagerCertificateExtractionResult, error) {
		return c.propertiesService.ProjectManagerCertificateExtraction(ctx, payload.PropertyDocumentID)
	})
	if err != nil {
		return properties.ManagerCertificateExtractionResult{}, fmt.Errorf("project manager certificate %s: %w", payload.PropertyDocumentID, err)
	}
	logger.InfoContext(ctx, "manager certificate projected", "property_document_id", payload.PropertyDocumentID, "offering_id", result.Document.OfferingID, "claims", result.Claims, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func decodeCanonicalMatchBackfillWorkflowPayload(raw json.RawMessage) (canonicalMatchBackfillPayload, error) {
	payload := canonicalMatchBackfillPayload{ScoreThreshold: 95, CompetitorMargin: 10}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return canonicalMatchBackfillPayload{}, fmt.Errorf("decode canonical source match backfill payload: %w", err)
		}
	}
	if payload.ScoreThreshold <= 0 {
		payload.ScoreThreshold = 95
	}
	if payload.CompetitorMargin < 0 {
		payload.CompetitorMargin = 10
	}
	return payload, nil
}

func decodeCanonicalizeSourceAdsFanoutWorkflowPayload(raw json.RawMessage) (canonicalizeSourceAdsFanoutPayload, error) {
	payload := canonicalizeSourceAdsFanoutPayload{Limit: 1000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return canonicalizeSourceAdsFanoutPayload{}, fmt.Errorf("decode canonicalize source ads fanout payload: %w", err)
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 1000
	}
	return payload, nil
}

func decodeCanonicalizeSourceAdWorkflowPayload(raw json.RawMessage) (canonicalizeSourceAdPayload, error) {
	var payload canonicalizeSourceAdPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return canonicalizeSourceAdPayload{}, fmt.Errorf("decode canonicalize source ad payload: %w", err)
		}
	}
	payload.SourceTable = strings.TrimSpace(payload.SourceTable)
	payload.SourceID = strings.TrimSpace(payload.SourceID)
	if payload.SourceTable == "" || payload.SourceID == "" {
		return canonicalizeSourceAdPayload{}, fmt.Errorf("source_table and source_id are required")
	}
	return payload, nil
}

func decodeCanonicalMatchFanoutWorkflowPayload(raw json.RawMessage) (canonicalMatchFanoutPayload, error) {
	payload := canonicalMatchFanoutPayload{Limit: 5000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return canonicalMatchFanoutPayload{}, fmt.Errorf("decode canonical source match fanout payload: %w", err)
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 5000
	}
	return payload, nil
}

func decodeCanonicalMatchSaleListingWorkflowPayload(raw json.RawMessage) (canonicalMatchSaleListingPayload, error) {
	var payload canonicalMatchSaleListingPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return canonicalMatchSaleListingPayload{}, fmt.Errorf("decode canonical sale listing source match payload: %w", err)
		}
	}
	payload.SaleListingID = strings.TrimSpace(payload.SaleListingID)
	if payload.SaleListingID == "" {
		return canonicalMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id is required")
	}
	if _, err := uuid.Parse(payload.SaleListingID); err != nil {
		return canonicalMatchSaleListingPayload{}, fmt.Errorf("sale_listing_id must be a uuid: %w", err)
	}
	return payload, nil
}

func decodeDimensionLayerBackfillWorkflowPayload(raw json.RawMessage) (dimensionLayerBackfillPayload, error) {
	payload := dimensionLayerBackfillPayload{BatchSize: 500}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return dimensionLayerBackfillPayload{}, fmt.Errorf("decode dimension layer backfill payload: %w", err)
		}
	}
	if payload.BatchSize <= 0 || payload.BatchSize > 5000 {
		payload.BatchSize = 500
	}
	if payload.AfterSaleListingID != "" {
		if _, err := uuid.Parse(payload.AfterSaleListingID); err != nil {
			return dimensionLayerBackfillPayload{}, fmt.Errorf("after_sale_listing_id must be a uuid: %w", err)
		}
	}
	return payload, nil
}

func decodeDimensionLayerListingWorkflowPayload(raw json.RawMessage) (dimensionLayerListingPayload, error) {
	var payload dimensionLayerListingPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return dimensionLayerListingPayload{}, fmt.Errorf("decode dimension layer listing payload: %w", err)
		}
	}
	payload.SaleListingID = strings.TrimSpace(payload.SaleListingID)
	if payload.SaleListingID == "" {
		return dimensionLayerListingPayload{}, fmt.Errorf("sale_listing_id is required")
	}
	if _, err := uuid.Parse(payload.SaleListingID); err != nil {
		return dimensionLayerListingPayload{}, fmt.Errorf("sale_listing_id must be a uuid: %w", err)
	}
	return payload, nil
}

func decodeDirtyDimensionTargetsWorkflowPayload(raw json.RawMessage) (dirtyDimensionTargetsPayload, error) {
	payload := dirtyDimensionTargetsPayload{Limit: 1000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return dirtyDimensionTargetsPayload{}, fmt.Errorf("decode dirty dimension targets payload: %w", err)
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 1000
	}
	return payload, nil
}

func decodeDirtyDimensionTargetWorkflowPayload(raw json.RawMessage) (dirtyDimensionTargetPayload, error) {
	var payload dirtyDimensionTargetPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return dirtyDimensionTargetPayload{}, fmt.Errorf("decode dimension target payload: %w", err)
		}
	}
	payload.TargetType = strings.TrimSpace(payload.TargetType)
	payload.TargetID = strings.TrimSpace(payload.TargetID)
	if payload.TargetType == "" || payload.TargetID == "" {
		return dirtyDimensionTargetPayload{}, fmt.Errorf("target_type and target_id are required")
	}
	if _, err := uuid.Parse(payload.TargetID); err != nil {
		return dirtyDimensionTargetPayload{}, fmt.Errorf("target_id must be a uuid: %w", err)
	}
	return payload, nil
}

func decodeDetachedHouseBackfillWorkflowPayload(raw json.RawMessage) (detachedHouseBackfillPayload, error) {
	payload := detachedHouseBackfillPayload{BatchSize: 1000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return detachedHouseBackfillPayload{}, fmt.Errorf("decode detached house backfill payload: %w", err)
		}
	}
	if payload.BatchSize <= 0 || payload.BatchSize > 5000 {
		payload.BatchSize = 1000
	}
	return payload, nil
}

func decodeManagerCertificateDocumentWorkflowPayload(raw json.RawMessage) (managerCertificateDocumentPayload, error) {
	var payload managerCertificateDocumentPayload
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return managerCertificateDocumentPayload{}, fmt.Errorf("decode manager certificate payload: %w", err)
		}
	}
	if payload.PropertyDocumentID == "" {
		payload.PropertyDocumentID = strings.TrimSpace(payload.DocumentID)
	}
	payload.PropertyDocumentID = strings.TrimSpace(payload.PropertyDocumentID)
	payload.Model = strings.TrimSpace(payload.Model)
	if payload.PropertyDocumentID == "" {
		return managerCertificateDocumentPayload{}, fmt.Errorf("property_document_id is required")
	}
	if _, err := uuid.Parse(payload.PropertyDocumentID); err != nil {
		return managerCertificateDocumentPayload{}, fmt.Errorf("property_document_id must be a uuid: %w", err)
	}
	return payload, nil
}

func (c *Consumer) spawnCanonicalSourceMatchSaleListing(ctx context.Context, saleListingID string, attempt int32) error {
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

func (c *Consumer) spawnCanonicalizeSourceAdsFanout(ctx context.Context, limit int32) error {
	payload, err := json.Marshal(canonicalizeSourceAdsFanoutPayload{Limit: limit})
	if err != nil {
		return fmt.Errorf("marshal canonicalize source ads fanout payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalizeSourceAdsFanout,
		Params:   payload,
	})
	return err
}

func (c *Consumer) updateCanonicalSourceMatchStateStep(ctx context.Context, stepName string, saleListingID, status string, nextAttemptAt *time.Time, runID *string) error {
	_, err := absurd.Step(ctx, stepName, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.updateCanonicalSourceMatchState(ctx, saleListingID, status, nextAttemptAt, runID)
	})
	return err
}

func (c *Consumer) projectTypedHousingCompanyProfileForSaleListingStep(ctx context.Context, saleListingID string) error {
	_, err := absurd.Step(ctx, "project-typed-housing-company-profile", func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.projectTypedHousingCompanyProfileForSaleListing(ctx, saleListingID)
	})
	return err
}
