package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/logging"
	"koditon-go/internal/taskqueue"
)

const (
	TaskTypePostalSync = "postal_sync"
)

func (c *Consumer) handlePostalTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
		slog.Int64("sync_task_id", msg.Data.SyncTaskID),
	)
	var err error
	switch msg.Data.TaskType {
	case TaskTypePostalSync:
		err = c.handlePostalSync(ctx, logger)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown postal task type: %s", msg.Data.TaskType), "unrecognized task type")
	}
	if err != nil {
		return classifyError(err)
	}
	return nil
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
