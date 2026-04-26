package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/db"
	"koditon-go/internal/taskqueue"
)

const (
	TaskTypeShortcutSitemapSync = "shortcut_sitemap_sync"
	TaskTypeShortcutScraperSync = "shortcut_scraper_sync"
	TaskTypeShortcutAPISync     = "shortcut_api_sync"
)

func (c *Consumer) handleShortcutTask(ctx context.Context, msg taskqueue.Message) error {
	logger := c.logger.With("task_type", msg.Data.TaskType, "entity_id", msg.Data.EntityID, "sync_task_id", msg.Data.SyncTaskID)
	var err error
	switch msg.Data.TaskType {
	case TaskTypeShortcutSitemapSync:
		err = c.handleShortcutSitemapSync(ctx, logger, msg)
	case TaskTypeShortcutScraperSync:
		err = c.handleShortcutScraperSync(ctx, logger, msg)
	case TaskTypeShortcutAPISync:
		err = c.handleShortcutAPISync(ctx, logger, msg)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown shortcut task type: %s", msg.Data.TaskType), "unrecognized task type")
	}
	if err != nil {
		return classifyError(err)
	}
	return nil
}

func (c *Consumer) handleShortcutSitemapSync(ctx context.Context, logger *slog.Logger, _ taskqueue.Message) error {
	buildingIDs, adIDs, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut sitemap sync failed", "error", err)
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
	logger.InfoContext(ctx, "shortcut sitemap sync completed", "buildings", len(buildingIDs), "ads", len(adIDs), "enqueue_errors", enqueueErrors)
	return nil
}

func (c *Consumer) handleShortcutScraperSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut scraper sync failed", "entity_id", msg.Data.EntityID, "error", err)
		return err
	}
	logger.InfoContext(ctx, "shortcut scraper entity synced", "entity_id", msg.Data.EntityID)
	return nil
}

func (c *Consumer) handleShortcutAPISync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut api sync failed", "entity_id", msg.Data.EntityID, "error", err)
		return err
	}
	logger.InfoContext(ctx, "shortcut api entity synced", "entity_id", msg.Data.EntityID)
	return nil
}

func (c *Consumer) enqueueShortcutTask(ctx context.Context, queue *taskqueue.Queue, entityID, taskType string) error {
	task, err := c.queries.UpsertShortcutSyncTask(ctx, db.UpsertShortcutSyncTaskParams{
		ShortcutSyncTaskEntityID:    entityID,
		ShortcutSyncTaskType:        taskType,
		ShortcutSyncTaskPriority:    int32(taskqueue.PriorityNormal),
		ShortcutSyncTaskMaxAttempts: int32(3),
	})
	if err != nil {
		return nil // ON CONFLICT DO NOTHING - active task already exists for this entity
	}
	_, err = queue.Send(ctx, taskqueue.MessageData{
		SyncTaskID: task.ShortcutSyncTaskID,
		EntityID:   entityID,
		TaskType:   taskType,
	})
	return err
}
