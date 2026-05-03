package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

const (
	TaskTypeShortcutSitemapSync          = "shortcut_sitemap_sync"
	TaskTypeShortcutBuildingsSitemapSync = "shortcut_buildings_sitemap_sync"
	TaskTypeShortcutScraperSync          = "shortcut_scraper_sync"
	TaskTypeShortcutAPISync              = "shortcut_api_sync"
	TaskTypeShortcutAdDataHashBackfill   = "shortcut_ad_data_hash_backfill"
)

func (c *Consumer) handleShortcutTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "shortcut", logger, msg, c.runShortcutSyncJob)
}

func (c *Consumer) handleShortcutSitemapSync(ctx context.Context, logger *slog.Logger, _ taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.sitemap_sync"))
	buildingIDs, adIDs, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	shortcutQueue := taskqueue.NewQueue(c.pool, "shortcut")
	var enqueueErrors int
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueShortcutTask(ctx, shortcutQueue, buildingID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	for _, adID := range adIDs {
		if enqErr := c.enqueueShortcutTask(ctx, shortcutQueue, adID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "shortcut sitemap sync completed", "buildings", len(buildingIDs), "ads", len(adIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutBuildingsSitemapSync(ctx context.Context, logger *slog.Logger, _ taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.buildings_sitemap_sync"))
	buildingIDs, _, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut buildings sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	shortcutQueue := taskqueue.NewQueue(c.pool, "shortcut")
	var enqueueErrors int
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueShortcutTask(ctx, shortcutQueue, buildingID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "shortcut buildings sitemap sync completed", "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutScraperSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.scraper_sync"))
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut scraper sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	if err := c.enqueueShortcutCanonicalizationForSyncedEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "shortcut scraper entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutAPISync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.api_sync"))
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut api sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	if err := c.enqueueShortcutCanonicalizationForSyncedEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "shortcut api entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueShortcutCanonicalizationForSyncedEntity(ctx context.Context, entityID string) error {
	entityType, sourceID, err := parseJobEntity(entityID)
	if err != nil || entityType != "ad" {
		return nil
	}
	adID, err := strconv.ParseInt(sourceID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse synced shortcut ad id: %w", err)
	}
	ad, err := c.queries.GetShortcutAdByID(ctx, adID)
	if err != nil {
		return fmt.Errorf("load synced shortcut ad for canonicalization enqueue: %w", err)
	}
	if ad.ShortcutAdDataHash == nil {
		return nil
	}
	return c.enqueueCanonicalizeSourceAd(ctx, "shortcut_ad", sourceID, int32(taskqueue.PriorityNormal))
}

func (c *Consumer) handleShortcutAdDataHashBackfill(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.ad_data_hash_backfill"))
	payload := sourceAdDataHashBackfillPayload{Limit: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode shortcut ad data hash backfill payload: %w", err), "invalid payload")
		}
	}
	scanned, updated, batches, err := c.syncRunner.ShortcutBackfillAdDataHashes(ctx, payload.Limit)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut ad data hash backfill failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	logger.InfoContext(ctx, "shortcut ad data hash backfill completed", "scanned", scanned, "updated", updated, "batches", batches, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueShortcutTask(ctx context.Context, _ *taskqueue.Queue, entityID, taskType string) error {
	_, err := c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "shortcut",
		Kind:        taskType,
		EntityID:    entityID,
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
	})
	return err
}

func (c *Consumer) runShortcutSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	msg := taskqueue.Message{Data: taskqueue.MessageData{EntityID: job.SyncJobEntityID, TaskType: job.SyncJobKind}}
	switch job.SyncJobKind {
	case TaskTypeShortcutSitemapSync:
		return c.handleShortcutSitemapSync(ctx, logger, msg)
	case TaskTypeShortcutBuildingsSitemapSync:
		return c.handleShortcutBuildingsSitemapSync(ctx, logger, msg)
	case TaskTypeShortcutScraperSync:
		return c.handleShortcutScraperSync(ctx, logger, msg)
	case TaskTypeShortcutAPISync:
		return c.handleShortcutAPISync(ctx, logger, msg)
	case TaskTypeShortcutAdDataHashBackfill:
		return c.handleShortcutAdDataHashBackfill(ctx, logger, job)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown shortcut sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}
