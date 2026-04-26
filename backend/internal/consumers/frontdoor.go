package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/db"
	"koditon-go/internal/logging"
	"koditon-go/internal/taskqueue"
)

const (
	TaskTypeFrontdoorSitemapSync = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorSync        = "frontdoor_sync"
)

func (c *Consumer) handleFrontdoorTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
		slog.Int64("sync_task_id", msg.Data.SyncTaskID),
	)
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

func (c *Consumer) handleFrontdoorSitemapSync(ctx context.Context, logger *slog.Logger, _ taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.sitemap_sync"))
	adIDs, buildingIDs, err := c.syncRunner.FrontdoorSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
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
	logger.InfoContext(ctx, "frontdoor sitemap sync completed", "ads", len(adIDs), "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleFrontdoorEntitySync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.entity_sync"))
	if err := c.syncRunner.FrontdoorSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "frontdoor sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	logger.InfoContext(ctx, "frontdoor entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueFrontdoorTask(ctx context.Context, queue *taskqueue.Queue, entityID, taskType string) error {
	task, err := c.queries.UpsertFrontdoorSyncTask(ctx, db.UpsertFrontdoorSyncTaskParams{
		FrontdoorSyncTaskEntityID:    entityID,
		FrontdoorSyncTaskType:        taskType,
		FrontdoorSyncTaskPriority:    int32(taskqueue.PriorityNormal),
		FrontdoorSyncTaskMaxAttempts: int32(3),
	})
	if err != nil {
		return nil // ON CONFLICT DO NOTHING - active task already exists for this entity
	}
	_, err = queue.Send(ctx, taskqueue.MessageData{
		SyncTaskID: task.FrontdoorSyncTaskID,
		EntityID:   entityID,
		TaskType:   taskType,
	})
	return err
}
