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
	TaskTypeFrontdoorSitemapSync = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorSync        = "frontdoor_sync"
)

func (c *Consumer) handleFrontdoorTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "frontdoor", logger, msg, c.runFrontdoorSyncJob)
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

func (c *Consumer) enqueueFrontdoorTask(ctx context.Context, _ *taskqueue.Queue, entityID, taskType string) error {
	_, err := c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "frontdoor",
		Kind:        taskType,
		EntityID:    entityID,
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
	})
	return err
}

func (c *Consumer) runFrontdoorSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	msg := taskqueue.Message{Data: taskqueue.MessageData{EntityID: job.SyncJobEntityID, TaskType: job.SyncJobKind}}
	switch job.SyncJobKind {
	case TaskTypeFrontdoorSitemapSync:
		return c.handleFrontdoorSitemapSync(ctx, logger, msg)
	case TaskTypeFrontdoorSync:
		return c.handleFrontdoorEntitySync(ctx, logger, msg)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown frontdoor sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}
