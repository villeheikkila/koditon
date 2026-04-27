package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
)

const (
	TaskTypePostalSync = "postal_sync"
)

func (c *Consumer) handlePostalTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "postal", logger, msg, c.runPostalSyncJob)
}

func (c *Consumer) handlePostalSync(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.postal.sync"))
	logger.InfoContext(ctx, "postal sync started")
	result, err := c.syncRunner.PostalSync(ctx, logger)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "postal sync completed", "total_records", result.TotalRecords, "ad_areas_upserted", result.AdAreasUpserted, "municipalities_upserted", result.MunicipalitiesUpserted, "postal_codes_upserted", result.PostalCodesUpserted, "skipped_records", result.SkippedRecords, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) runPostalSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	switch job.SyncJobKind {
	case TaskTypePostalSync:
		return c.handlePostalSync(ctx, logger)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown postal sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}
