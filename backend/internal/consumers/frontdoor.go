package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/db"
	"koditon-go/internal/taskqueue"
)

const (
	TaskTypeFrontdoorSitemapSync = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorSync        = "frontdoor_sync"
)

func (c *Consumer) handleFrontdoorTask(ctx context.Context, msg taskqueue.Message) error {
	logger := c.logger.With("task_type", msg.Data.TaskType, "entity_id", msg.Data.EntityID, "pending_task_id", msg.Data.PendingTaskID)
	var err error
	switch msg.Data.TaskType {
	case TaskTypeFrontdoorSitemapSync:
		err = c.handleFrontdoorSitemapSync(ctx, logger, msg)
	case TaskTypeFrontdoorSync:
		err = c.handleFrontdoorEntitySync(ctx, logger, msg)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown frontdoor task type: %s", msg.Data.TaskType), "unrecognized task type")
	}
	if err != nil {
		return classifyError(err)
	}
	return nil
}

func (c *Consumer) handleFrontdoorSitemapSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	adIDs, buildingIDs, err := c.syncRunner.FrontdoorSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor sitemap sync failed", "error", err)
		return err
	}
	frontdoorQueue := taskqueue.NewQueue(c.pool, "frontdoor")
	var enqueueErrors int
	for _, adID := range adIDs {
		if enqErr := c.enqueueFrontdoorTask(ctx, frontdoorQueue, taskqueue.EntityPrefixAd+adID, TaskTypeFrontdoorSync); enqErr != nil {
			enqueueErrors++
		}
	}
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueFrontdoorTask(ctx, frontdoorQueue, taskqueue.EntityPrefixBuilding+buildingID, TaskTypeFrontdoorSync); enqErr != nil {
			enqueueErrors++
		}
	}
	c.deletePendingTask(ctx, logger, msg, c.queries.DeleteFrontdoorPendingTask)
	logger.InfoContext(ctx, "frontdoor sitemap sync completed", "ads", len(adIDs), "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors)
	return nil
}

func (c *Consumer) handleFrontdoorEntitySync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	if err := c.syncRunner.FrontdoorSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "frontdoor sync failed", "entity_id", msg.Data.EntityID, "error", err)
		return err
	}
	c.deletePendingTask(ctx, logger, msg, c.queries.DeleteFrontdoorPendingTask)
	logger.InfoContext(ctx, "frontdoor entity synced", "entity_id", msg.Data.EntityID)
	return nil
}

func (c *Consumer) enqueueFrontdoorTask(ctx context.Context, queue *taskqueue.Queue, entityID, taskType string) error {
	task, err := c.queries.InsertFrontdoorPendingTask(ctx, db.InsertFrontdoorPendingTaskParams{
		FrontdoorPendingTaskEntityID: entityID,
		FrontdoorPendingTaskType:     taskType,
		FrontdoorPendingTaskPriority: int32(taskqueue.PriorityNormal),
		FrontdoorPendingTaskMaxAttempts: int32(3),
	})
	if err != nil {
		return nil // ON CONFLICT DO NOTHING - task already exists
	}
	_, err = queue.Send(ctx, taskqueue.MessageData{
		PendingTaskID: task.FrontdoorPendingTaskID,
		EntityID:      entityID,
		TaskType:      taskType,
	})
	return err
}
