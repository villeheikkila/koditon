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

	"koditon/internal/platform/logging"
	"koditon/internal/sync/sourcejson"
	"koditon/internal/sync/workflows"
)

var frontdoorWorkflowKinds = []string{
	TaskTypeFrontdoorSitemapSync,
	TaskTypeFrontdoorBuildingsSitemapSync,
	TaskTypeFrontdoorSync,
	TaskTypeFrontdoorAdDataHashBackfill,
}

type frontdoorSitemapResult struct {
	AdEntityIDs       []string `json:"ad_entity_ids,omitempty"`
	BuildingEntityIDs []string `json:"building_entity_ids,omitempty"`
}

type frontdoorFanoutResult struct {
	Enqueued int `json:"enqueued"`
}

type frontdoorEntityResult struct {
	EntityID string `json:"entity_id"`
	Type     string `json:"type"`
}

type sourceEntityParams struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

func (c *Consumer) startFrontdoorWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.frontdoorWorkflowClient == nil {
		return errors.New("frontdoor absurd workflow client is not configured")
	}
	if c.frontdoorService == nil {
		return errors.New("frontdoor service is not configured")
	}
	for _, kind := range frontdoorWorkflowKinds {
		def, ok := workflows.FindDefinition(kind)
		if !ok {
			return fmt.Errorf("missing frontdoor workflow definition: %s", kind)
		}
		task := absurd.Task[json.RawMessage, json.RawMessage](
			kind,
			c.handleFrontdoorWorkflow,
			absurd.TaskOptions{QueueName: workflows.QueueFrontdoor, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation},
		)
		if err := c.frontdoorWorkflowClient.Register(task); err != nil {
			return fmt.Errorf("register frontdoor workflow %s: %w", kind, err)
		}
	}
	logger := logging.With(c.logger, logging.Op("consumer.frontdoor.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.frontdoorWorkflowCancel = cancel
	c.frontdoorWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.frontdoorWorkflowDone)
		logger.InfoContext(workerCtx, "frontdoor absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueueFrontdoor)
		err := c.frontdoorWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "frontdoor",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "frontdoor absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "frontdoor absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
	return nil
}

func (c *Consumer) handleFrontdoorWorkflow(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	taskName := absurd.MustTaskContext(ctx).TaskName()
	logger := logging.With(c.logger,
		logging.Op("consumer.frontdoor.workflow"),
		slog.String("task_type", taskName),
	)
	var result any
	var err error
	switch taskName {
	case TaskTypeFrontdoorSitemapSync:
		result, err = c.runFrontdoorSitemapWorkflow(ctx, logger, false)
	case TaskTypeFrontdoorBuildingsSitemapSync:
		result, err = c.runFrontdoorSitemapWorkflow(ctx, logger, true)
	case TaskTypeFrontdoorSync:
		result, err = c.runFrontdoorEntityWorkflow(ctx, logger, params)
	case TaskTypeFrontdoorAdDataHashBackfill:
		result, err = c.runFrontdoorAdDataHashBackfillWorkflow(ctx, logger, params)
	default:
		return nil, fmt.Errorf("unknown frontdoor workflow kind: %s", taskName)
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal frontdoor workflow result: %w", err)
	}
	return raw, nil
}

func (c *Consumer) runFrontdoorSitemapWorkflow(ctx context.Context, logger *slog.Logger, buildingsOnly bool) (frontdoorSitemapResult, error) {
	sitemap, err := absurd.Step(ctx, "fetch-sitemap", func(ctx context.Context) (frontdoorSitemapResult, error) {
		adIDs, buildingIDs, err := c.frontdoorService.SyncSitemap(ctx)
		if err != nil {
			return frontdoorSitemapResult{}, err
		}
		return frontdoorSitemapResult{AdEntityIDs: adIDs, BuildingEntityIDs: buildingIDs}, nil
	})
	if err != nil {
		return frontdoorSitemapResult{}, err
	}
	_, err = absurd.Step(ctx, "spawn-frontdoor-syncs", func(ctx context.Context) (frontdoorFanoutResult, error) {
		enqueued := 0
		if !buildingsOnly {
			for _, entityID := range sitemap.AdEntityIDs {
				if err := c.spawnFrontdoorSync(ctx, entityID); err != nil {
					return frontdoorFanoutResult{}, err
				}
				enqueued++
			}
		}
		for _, entityID := range sitemap.BuildingEntityIDs {
			if err := c.spawnFrontdoorSync(ctx, entityID); err != nil {
				return frontdoorFanoutResult{}, err
			}
			enqueued++
		}
		logger.InfoContext(ctx, "frontdoor sync tasks spawned", "count", enqueued, "buildings_only", buildingsOnly, "outcome", logging.OutcomeSuccess)
		return frontdoorFanoutResult{Enqueued: enqueued}, nil
	})
	if err != nil {
		return frontdoorSitemapResult{}, err
	}
	if buildingsOnly {
		sitemap.AdEntityIDs = nil
	}
	return sitemap, nil
}

func (c *Consumer) runFrontdoorEntityWorkflow(ctx context.Context, logger *slog.Logger, raw json.RawMessage) (frontdoorEntityResult, error) {
	params, err := decodeSourceEntityParams(raw)
	if err != nil {
		return frontdoorEntityResult{}, err
	}
	switch params.SourceType {
	case "ad":
		if _, err := absurd.Step(ctx, "fetch-source", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.frontdoorService.SyncAd(ctx, params.SourceID)
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
		if _, err := absurd.Step(ctx, "canonicalize-source-ad", func(ctx context.Context) (frontdoorFanoutResult, error) {
			ad, err := c.queries.GetFrontdoorAdByExternalID(ctx, &params.SourceID)
			if err != nil {
				return frontdoorFanoutResult{}, fmt.Errorf("load synced frontdoor ad for canonicalization enqueue: %w", err)
			}
			if ad.FrontdoorAdDataHash == nil {
				return frontdoorFanoutResult{}, nil
			}
			if err := c.enqueueCanonicalizeSourceAd(ctx, "frontdoor_ad", ad.FrontdoorAdID.String(), int32(priorityNormal)); err != nil {
				return frontdoorFanoutResult{}, err
			}
			return frontdoorFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
	case "building":
		if _, err := absurd.Step(ctx, "fetch-source", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.frontdoorService.SyncBuilding(ctx, params.SourceID)
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
		if _, err := absurd.Step(ctx, "canonicalize-source-announcements", func(ctx context.Context) (frontdoorFanoutResult, error) {
			buildingID, err := uuid.Parse(params.SourceID)
			if err != nil {
				return frontdoorFanoutResult{}, nil
			}
			announcements, err := c.queries.ListFrontdoorBuildingAnnouncements(ctx, &buildingID)
			if err != nil {
				return frontdoorFanoutResult{}, fmt.Errorf("load synced frontdoor building announcements for canonicalization enqueue: %w", err)
			}
			enqueued := 0
			for _, announcement := range announcements {
				if err := c.enqueueCanonicalizeSourceAd(ctx, "frontdoor_building_announcement", announcement.FrontdoorBuildingAnnouncementID.String(), int32(priorityNormal)); err != nil {
					return frontdoorFanoutResult{}, err
				}
				enqueued++
			}
			return frontdoorFanoutResult{Enqueued: enqueued}, nil
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
	default:
		return frontdoorEntityResult{}, fmt.Errorf("unknown frontdoor source type: %s", params.SourceType)
	}
	entityID := params.SourceType + ":" + params.SourceID
	logger.InfoContext(ctx, "frontdoor entity workflow completed", "source_type", params.SourceType, "source_id", params.SourceID, "outcome", logging.OutcomeSuccess)
	return frontdoorEntityResult{EntityID: entityID, Type: params.SourceType}, nil
}

func (c *Consumer) spawnFrontdoorSync(ctx context.Context, entityID string) error {
	entityType, sourceID, err := parseJobEntity(entityID)
	if err != nil {
		return err
	}
	params, err := json.Marshal(sourceEntityParams{SourceType: entityType, SourceID: sourceID})
	if err != nil {
		return err
	}
	_, err = workflows.Spawn(ctx, c.frontdoorWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeFrontdoorSync,
		Params:   params,
	})
	return err
}

func (c *Consumer) runFrontdoorAdDataHashBackfillWorkflow(ctx context.Context, logger *slog.Logger, params json.RawMessage) (sourcejson.BackfillResult, error) {
	payload, err := decodeSourceAdDataHashBackfillPayload(params, "frontdoor")
	if err != nil {
		return sourcejson.BackfillResult{}, err
	}
	result, err := absurd.Step(ctx, "backfill-data-hashes", func(ctx context.Context) (sourcejson.BackfillResult, error) {
		return c.frontdoorService.BackfillAdDataHashes(ctx, payload.Limit)
	})
	if err != nil {
		return sourcejson.BackfillResult{}, err
	}
	if result.Scanned >= int(payload.Limit) {
		if err := absurd.SleepFor(ctx, "pause-before-next-batch", 30*time.Second); err != nil {
			return sourcejson.BackfillResult{}, err
		}
		if _, err := absurd.Step(ctx, "spawn-next-batch", func(ctx context.Context) (frontdoorFanoutResult, error) {
			if err := c.spawnFrontdoorHashBackfill(ctx, payload.Limit); err != nil {
				return frontdoorFanoutResult{}, err
			}
			return frontdoorFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return sourcejson.BackfillResult{}, err
		}
	}
	logger.InfoContext(ctx, "frontdoor ad data hash backfill batch completed", "scanned", result.Scanned, "updated", result.Updated, "limit", payload.Limit, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) spawnFrontdoorHashBackfill(ctx context.Context, limit int32) error {
	payload, err := json.Marshal(sourceAdDataHashBackfillPayload{Limit: limit})
	if err != nil {
		return fmt.Errorf("marshal frontdoor hash backfill payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.frontdoorWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeFrontdoorAdDataHashBackfill,
		Params:   payload,
	})
	return err
}

func decodeSourceEntityParams(raw json.RawMessage) (sourceEntityParams, error) {
	var params sourceEntityParams
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return sourceEntityParams{}, fmt.Errorf("decode source entity params: %w", err)
	}
	params.SourceType = strings.TrimSpace(params.SourceType)
	params.SourceID = strings.TrimSpace(params.SourceID)
	if params.SourceType == "" || params.SourceID == "" {
		return sourceEntityParams{}, fmt.Errorf("source_type and source_id are required")
	}
	return params, nil
}
