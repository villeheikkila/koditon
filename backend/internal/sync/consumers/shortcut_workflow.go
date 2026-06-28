package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/google/uuid"

	"koditon/internal/platform/logging"
	"koditon/internal/sync/sourcejson"
	"koditon/internal/sync/workflows"
)

var shortcutAPIWorkflowKinds = []string{
	TaskTypeShortcutSitemapSync,
	TaskTypeShortcutBuildingsSitemapSync,
	TaskTypeShortcutAPISync,
	TaskTypeShortcutAdDataHashBackfill,
}

var shortcutScraperWorkflowKinds = []string{
	TaskTypeShortcutScraperSync,
}

type shortcutSitemapResult struct {
	BuildingEntityIDs []string `json:"building_entity_ids,omitempty"`
	AdEntityIDs       []string `json:"ad_entity_ids,omitempty"`
}

type shortcutFanoutResult struct {
	Enqueued int `json:"enqueued"`
}

type shortcutEntityResult struct {
	EntityID string `json:"entity_id"`
	Type     string `json:"type"`
	Mode     string `json:"mode"`
}

func (c *Consumer) startShortcutWorkflowWorkers(ctx context.Context, cfg Config) error {
	if c.shortcutAPIWorkflowClient == nil {
		return errors.New("shortcut api absurd workflow client is not configured")
	}
	if c.shortcutScraperWorkflowClient == nil {
		return errors.New("shortcut scraper absurd workflow client is not configured")
	}
	if c.shortcutService == nil {
		return errors.New("shortcut service is not configured")
	}
	if err := c.registerShortcutWorkflows(c.shortcutAPIWorkflowClient, shortcutAPIWorkflowKinds); err != nil {
		return err
	}
	if err := c.registerShortcutWorkflows(c.shortcutScraperWorkflowClient, shortcutScraperWorkflowKinds); err != nil {
		return err
	}
	c.startShortcutAPIWorker(ctx, cfg)
	c.startShortcutScraperWorker(ctx, cfg)
	return nil
}

func (c *Consumer) registerShortcutWorkflows(app *absurd.Client, kinds []string) error {
	for _, kind := range kinds {
		def, ok := workflows.FindDefinition("shortcut", kind)
		if !ok {
			return fmt.Errorf("missing shortcut workflow definition: %s", kind)
		}
		task := absurd.Task[workflows.Params, workflows.Result](
			kind,
			c.handleShortcutWorkflow,
			absurd.TaskOptions{QueueName: def.Queue, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation},
		)
		if err := app.Register(task); err != nil {
			return fmt.Errorf("register shortcut workflow %s: %w", kind, err)
		}
	}
	return nil
}

func (c *Consumer) startShortcutAPIWorker(ctx context.Context, cfg Config) {
	logger := logging.With(c.logger, logging.Op("consumer.shortcut.api.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.shortcutAPIWorkflowCancel = cancel
	c.shortcutAPIWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.shortcutAPIWorkflowDone)
		logger.InfoContext(workerCtx, "shortcut api absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueueShortcutAPI)
		err := c.shortcutAPIWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "shortcut-api",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "shortcut api absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "shortcut api absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
}

func (c *Consumer) startShortcutScraperWorker(ctx context.Context, cfg Config) {
	logger := logging.With(c.logger, logging.Op("consumer.shortcut.scraper.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.shortcutScraperWorkflowCancel = cancel
	c.shortcutScraperWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.shortcutScraperWorkflowDone)
		logger.InfoContext(workerCtx, "shortcut scraper absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueueShortcutScraper)
		err := c.shortcutScraperWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "shortcut-scraper",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "shortcut scraper absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "shortcut scraper absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
}

func (c *Consumer) handleShortcutWorkflow(ctx context.Context, params workflows.Params) (workflows.Result, error) {
	if err := workflows.ValidateParams(params); err != nil {
		return workflows.Result{}, err
	}
	if params.Provider != "shortcut" {
		return workflows.Result{}, fmt.Errorf("invalid shortcut workflow provider: %s", params.Provider)
	}
	logger := logging.With(c.logger,
		logging.Op("consumer.shortcut.workflow"),
		slog.String("task_type", params.Kind),
		slog.String("entity_id", params.EntityID),
	)
	var result any
	var err error
	switch params.Kind {
	case TaskTypeShortcutSitemapSync:
		result, err = c.runShortcutSitemapWorkflow(ctx, logger, false)
	case TaskTypeShortcutBuildingsSitemapSync:
		result, err = c.runShortcutSitemapWorkflow(ctx, logger, true)
	case TaskTypeShortcutScraperSync:
		result, err = c.runShortcutEntityWorkflow(ctx, logger, params, "scraper")
	case TaskTypeShortcutAPISync:
		result, err = c.runShortcutEntityWorkflow(ctx, logger, params, "api")
	case TaskTypeShortcutAdDataHashBackfill:
		result, err = c.runShortcutAdDataHashBackfillWorkflow(ctx, logger, params)
	default:
		return workflows.Result{}, fmt.Errorf("unknown shortcut workflow kind: %s", params.Kind)
	}
	if err != nil {
		return workflows.Result{}, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return workflows.Result{}, fmt.Errorf("marshal shortcut workflow result: %w", err)
	}
	return workflows.Result{Status: "succeeded", Result: raw}, nil
}

func (c *Consumer) runShortcutSitemapWorkflow(ctx context.Context, logger *slog.Logger, buildingsOnly bool) (shortcutSitemapResult, error) {
	sitemap, err := absurd.Step(ctx, "fetch-sitemap", func(ctx context.Context) (shortcutSitemapResult, error) {
		buildingIDs, adIDs, err := c.shortcutService.SyncSitemap(ctx)
		if err != nil {
			return shortcutSitemapResult{}, err
		}
		return shortcutSitemapResult{BuildingEntityIDs: buildingIDs, AdEntityIDs: adIDs}, nil
	})
	if err != nil {
		return shortcutSitemapResult{}, err
	}
	_, err = absurd.Step(ctx, "spawn-shortcut-scraper-syncs", func(ctx context.Context) (shortcutFanoutResult, error) {
		enqueued := 0
		for _, entityID := range sitemap.BuildingEntityIDs {
			if err := c.spawnShortcutSync(ctx, TaskTypeShortcutScraperSync, entityID); err != nil {
				return shortcutFanoutResult{}, err
			}
			enqueued++
		}
		if !buildingsOnly {
			for _, entityID := range sitemap.AdEntityIDs {
				if err := c.spawnShortcutSync(ctx, TaskTypeShortcutScraperSync, entityID); err != nil {
					return shortcutFanoutResult{}, err
				}
				enqueued++
			}
		}
		logger.InfoContext(ctx, "shortcut scraper sync tasks spawned", "count", enqueued, "buildings_only", buildingsOnly, "outcome", logging.OutcomeSuccess)
		return shortcutFanoutResult{Enqueued: enqueued}, nil
	})
	if err != nil {
		return shortcutSitemapResult{}, err
	}
	if buildingsOnly {
		sitemap.AdEntityIDs = nil
	}
	return sitemap, nil
}

func (c *Consumer) runShortcutEntityWorkflow(ctx context.Context, logger *slog.Logger, params workflows.Params, mode string) (shortcutEntityResult, error) {
	entityType, sourceID, err := parseJobEntity(params.EntityID)
	if err != nil {
		return shortcutEntityResult{}, err
	}
	switch entityType {
	case "ad":
		adID, err := strconv.ParseInt(sourceID, 10, 64)
		if err != nil {
			return shortcutEntityResult{}, fmt.Errorf("invalid shortcut ad id: %w", err)
		}
		if _, err := absurd.Step(ctx, "fetch-source", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.shortcutService.SyncAd(ctx, adID)
		}); err != nil {
			return shortcutEntityResult{}, err
		}
		if _, err := absurd.Step(ctx, "canonicalize-source-ad", func(ctx context.Context) (shortcutFanoutResult, error) {
			ad, err := c.queries.GetShortcutAdByID(ctx, adID)
			if err != nil {
				return shortcutFanoutResult{}, fmt.Errorf("load synced shortcut ad for canonicalization enqueue: %w", err)
			}
			if ad.ShortcutAdDataHash == nil {
				return shortcutFanoutResult{}, nil
			}
			if err := c.enqueueCanonicalizeSourceAd(ctx, "shortcut_ad", sourceID, int32(priorityNormal)); err != nil {
				return shortcutFanoutResult{}, err
			}
			return shortcutFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return shortcutEntityResult{}, err
		}
	case "building":
		buildingID, err := uuid.Parse(sourceID)
		if err != nil {
			return shortcutEntityResult{}, fmt.Errorf("invalid shortcut building id: %w", err)
		}
		if _, err := absurd.Step(ctx, "fetch-source", func(ctx context.Context) (struct{}, error) {
			return struct{}{}, c.shortcutService.SyncBuilding(ctx, buildingID)
		}); err != nil {
			return shortcutEntityResult{}, err
		}
	default:
		return shortcutEntityResult{}, fmt.Errorf("unknown shortcut entity type: %s", entityType)
	}
	logger.InfoContext(ctx, "shortcut entity workflow completed", "entity_id", params.EntityID, "mode", mode, "outcome", logging.OutcomeSuccess)
	return shortcutEntityResult{EntityID: params.EntityID, Type: entityType, Mode: mode}, nil
}

func (c *Consumer) spawnShortcutSync(ctx context.Context, kind string, entityID string) error {
	app := c.shortcutAPIWorkflowClient
	if kind == TaskTypeShortcutScraperSync {
		app = c.shortcutScraperWorkflowClient
	}
	_, err := workflows.Spawn(ctx, app, workflows.SpawnRequest{
		Provider: "shortcut",
		Kind:     kind,
		EntityID: entityID,
	})
	return err
}

func (c *Consumer) runShortcutAdDataHashBackfillWorkflow(ctx context.Context, logger *slog.Logger, params workflows.Params) (sourcejson.BackfillResult, error) {
	payload, err := decodeSourceAdDataHashBackfillPayload(params.Payload, "shortcut")
	if err != nil {
		return sourcejson.BackfillResult{}, err
	}
	result, err := absurd.Step(ctx, "backfill-data-hashes", func(ctx context.Context) (sourcejson.BackfillResult, error) {
		return c.shortcutService.BackfillAdDataHashes(ctx, payload.Limit)
	})
	if err != nil {
		return sourcejson.BackfillResult{}, err
	}
	if result.Scanned >= int(payload.Limit) {
		if err := absurd.SleepFor(ctx, "pause-before-next-batch", 30*time.Second); err != nil {
			return sourcejson.BackfillResult{}, err
		}
		if _, err := absurd.Step(ctx, "spawn-next-batch", func(ctx context.Context) (shortcutFanoutResult, error) {
			if err := c.spawnShortcutHashBackfill(ctx, payload.Limit); err != nil {
				return shortcutFanoutResult{}, err
			}
			return shortcutFanoutResult{Enqueued: 1}, nil
		}); err != nil {
			return sourcejson.BackfillResult{}, err
		}
	}
	logger.InfoContext(ctx, "shortcut ad data hash backfill batch completed", "scanned", result.Scanned, "updated", result.Updated, "limit", payload.Limit, "outcome", logging.OutcomeSuccess)
	return result, nil
}

func (c *Consumer) spawnShortcutHashBackfill(ctx context.Context, limit int32) error {
	payload, err := json.Marshal(sourceAdDataHashBackfillPayload{Limit: limit})
	if err != nil {
		return fmt.Errorf("marshal shortcut hash backfill payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.shortcutAPIWorkflowClient, workflows.SpawnRequest{
		Provider: "shortcut",
		Kind:     TaskTypeShortcutAdDataHashBackfill,
		EntityID: fmt.Sprintf("shortcut:ad_data_hash_backfill:%d", time.Now().UTC().UnixNano()),
		Payload:  payload,
	})
	return err
}
