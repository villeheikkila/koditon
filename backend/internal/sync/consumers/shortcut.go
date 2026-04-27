package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

const (
	TaskTypeShortcutSitemapSync = "shortcut_sitemap_sync"
	TaskTypeShortcutScraperSync = "shortcut_scraper_sync"
	TaskTypeShortcutAPISync     = "shortcut_api_sync"
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
		if enqErr := c.enqueueShortcutTask(ctx, shortcutQueue, taskqueue.EntityPrefixBuilding+buildingID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	for _, adID := range adIDs {
		if enqErr := c.enqueueShortcutTask(ctx, shortcutQueue, taskqueue.EntityPrefixAd+adID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "shortcut sitemap sync completed", "buildings", len(buildingIDs), "ads", len(adIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutScraperSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.scraper_sync"))
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut scraper sync failed", "error", err, "outcome", logging.OutcomeError)
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
	logger.InfoContext(ctx, "shortcut api entity synced", "outcome", logging.OutcomeSuccess)
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
	case TaskTypeShortcutScraperSync:
		return c.handleShortcutScraperSync(ctx, logger, msg)
	case TaskTypeShortcutAPISync:
		return c.handleShortcutAPISync(ctx, logger, msg)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown shortcut sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}
