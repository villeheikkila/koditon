package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

func (c *Consumer) startFrontdoorWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.frontdoorWorkflowClient == nil {
		return errors.New("frontdoor absurd workflow client is not configured")
	}
	if c.frontdoorService == nil {
		return errors.New("frontdoor service is not configured")
	}
	for _, kind := range frontdoorWorkflowKinds {
		def, ok := workflows.FindDefinition("frontdoor", kind)
		if !ok {
			return fmt.Errorf("missing frontdoor workflow definition: %s", kind)
		}
		task := absurd.Task[workflows.Params, workflows.Result](
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

func (c *Consumer) handleFrontdoorWorkflow(ctx context.Context, params workflows.Params) (workflows.Result, error) {
	if err := workflows.ValidateParams(params); err != nil {
		return workflows.Result{}, err
	}
	if params.Provider != "frontdoor" {
		return workflows.Result{}, fmt.Errorf("invalid frontdoor workflow provider: %s", params.Provider)
	}
	logger := logging.With(c.logger,
		logging.Op("consumer.frontdoor.workflow"),
		slog.String("task_type", params.Kind),
		slog.String("entity_id", params.EntityID),
	)
	var result any
	var err error
	switch params.Kind {
	case TaskTypeFrontdoorSitemapSync:
		result, err = c.runFrontdoorSitemapWorkflow(ctx, logger, false)
	case TaskTypeFrontdoorBuildingsSitemapSync:
		result, err = c.runFrontdoorSitemapWorkflow(ctx, logger, true)
	case TaskTypeFrontdoorSync:
		result, err = c.runFrontdoorEntityWorkflow(ctx, logger, params)
	case TaskTypeFrontdoorAdDataHashBackfill:
		result, err = c.runFrontdoorAdDataHashBackfillWorkflow(ctx, logger, params)
	default:
		return workflows.Result{}, fmt.Errorf("unknown frontdoor workflow kind: %s", params.Kind)
	}
	if err != nil {
		return workflows.Result{}, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return workflows.Result{}, fmt.Errorf("marshal frontdoor workflow result: %w", err)
	}
	return workflows.Result{Status: "succeeded", Result: raw}, nil
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

func (c *Consumer) runFrontdoorEntityWorkflow(ctx context.Context, logger *slog.Logger, params workflows.Params) (frontdoorEntityResult, error) {
	entityType, externalID, err := parseJobEntity(params.EntityID)
	if err != nil {
		return frontdoorEntityResult{}, err
	}
	switch entityType {
	case "ad":
		if _, err := absurd.Step(ctx, "fetch-source", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.frontdoorService.SyncAd(ctx, externalID)
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
		if _, err := absurd.Step(ctx, "canonicalize-source-ad", func(ctx context.Context) (frontdoorFanoutResult, error) {
			ad, err := c.queries.GetFrontdoorAdByExternalID(ctx, externalID)
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
			return struct{}{}, c.frontdoorService.SyncBuilding(ctx, externalID)
		}); err != nil {
			return frontdoorEntityResult{}, err
		}
		if _, err := absurd.Step(ctx, "canonicalize-source-announcements", func(ctx context.Context) (frontdoorFanoutResult, error) {
			buildingID, err := uuid.Parse(externalID)
			if err != nil {
				return frontdoorFanoutResult{}, nil
			}
			announcements, err := c.queries.ListFrontdoorBuildingAnnouncements(ctx, buildingID)
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
		return frontdoorEntityResult{}, fmt.Errorf("unknown frontdoor entity type: %s", entityType)
	}
	logger.InfoContext(ctx, "frontdoor entity workflow completed", "entity_id", params.EntityID, "outcome", logging.OutcomeSuccess)
	return frontdoorEntityResult{EntityID: params.EntityID, Type: entityType}, nil
}

func (c *Consumer) spawnFrontdoorSync(ctx context.Context, entityID string) error {
	_, err := workflows.Spawn(ctx, c.frontdoorWorkflowClient, workflows.SpawnRequest{
		Provider: "frontdoor",
		Kind:     TaskTypeFrontdoorSync,
		EntityID: entityID,
	})
	return err
}

func (c *Consumer) runFrontdoorAdDataHashBackfillWorkflow(ctx context.Context, logger *slog.Logger, params workflows.Params) (sourcejson.BackfillResult, error) {
	payload, err := decodeSourceAdDataHashBackfillPayload(params.Payload, "frontdoor")
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
	_, err = workflows.Spawn(ctx, c.frontdoorWorkflowClient, workflows.SpawnRequest{
		Provider: "frontdoor",
		Kind:     TaskTypeFrontdoorAdDataHashBackfill,
		EntityID: fmt.Sprintf("frontdoor:ad_data_hash_backfill:%d", time.Now().UTC().UnixNano()),
		Payload:  payload,
	})
	return err
}
