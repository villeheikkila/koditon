package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/taskqueue"
)

const (
	TaskTypePostalSync = "postal_sync"
)

func (c *Consumer) handlePostalTask(ctx context.Context, msg taskqueue.Message) error {
	logger := c.logger.With("task_type", msg.Data.TaskType, "entity_id", msg.Data.EntityID, "pending_task_id", msg.Data.PendingTaskID)
	var err error
	switch msg.Data.TaskType {
	case TaskTypePostalSync:
		err = c.handlePostalSync(ctx, logger, msg)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown postal task type: %s", msg.Data.TaskType), "unrecognized task type")
	}
	if err != nil {
		return classifyError(err)
	}
	return nil
}

func (c *Consumer) handlePostalSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger.InfoContext(ctx, "processing postal sync task")
	result, err := c.syncRunner.PostalSync(ctx, logger)
	if err != nil {
		return err
	}
	c.deletePendingTask(ctx, logger, msg, c.queries.DeletePostalPendingTask)
	logger.InfoContext(ctx, "completed postal sync", "total_records", result.TotalRecords, "ad_areas_upserted", result.AdAreasUpserted, "municipalities_upserted", result.MunicipalitiesUpserted, "postal_codes_upserted", result.PostalCodesUpserted, "skipped_records", result.SkippedRecords)
	return nil
}
