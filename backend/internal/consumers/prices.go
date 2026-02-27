package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/db"
	"koditon-go/internal/prices"
	"koditon-go/internal/taskqueue"
)

const (
	TaskTypePricesCitiesInit                 = "prices_cities_init"
	TaskTypePricesSync                       = "prices_sync"
	TaskTypePricesNeighborhoodPostalCodeSync = "prices_neighborhood_postal_code_sync"
	TaskTypePricesSyncAll                    = "prices_sync_all"
)

func (c *Consumer) handlePricesTask(ctx context.Context, msg taskqueue.Message) error {
	logger := c.logger.With("task_type", msg.Data.TaskType, "entity_id", msg.Data.EntityID, "pending_task_id", msg.Data.PendingTaskID)
	var err error
	switch msg.Data.TaskType {
	case TaskTypePricesCitiesInit:
		err = c.handlePricesCitiesInit(ctx, logger)
	case TaskTypePricesSync:
		err = c.handlePricesSync(ctx, logger, msg)
	case TaskTypePricesNeighborhoodPostalCodeSync:
		err = c.handlePricesNeighborhoodPostalCodeSync(ctx, logger)
	case TaskTypePricesSyncAll:
		err = c.handlePricesSyncAll(ctx, logger)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown prices task type: %s", msg.Data.TaskType), "unrecognized task type")
	}
	if err != nil {
		return classifyError(err)
	}
	return nil
}

func (c *Consumer) handlePricesCitiesInit(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices cities initialization task")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	if len(cities) > 0 {
		pricesQueue := taskqueue.NewQueue(c.pool, "prices")
		var enqueueErrors int
		for _, city := range cities {
			if enqErr := c.enqueuePricesTask(ctx, pricesQueue, taskqueue.EntityPrefixCity+city, TaskTypePricesSync); enqErr != nil {
				enqueueErrors++
			}
		}
		logger.InfoContext(ctx, "city entities enqueued", "count", len(cities), "enqueue_errors", enqueueErrors)
	}
	return nil
}

func (c *Consumer) handlePricesSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger.InfoContext(ctx, "syncing prices city", "entity_id", msg.Data.EntityID)
	if err := c.syncRunner.PricesSyncCityEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) handlePricesNeighborhoodPostalCodeSync(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices neighborhood postal code sync task")
	err := c.syncRunner.PricesSyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
		if p.Page > 0 {
			logger.DebugContext(ctx, "fetching postal code transactions", "city", p.City, "postal_code", p.PostalCode, "page", p.Page)
		} else if p.Updated > 0 {
			logger.InfoContext(ctx, "updated neighborhood postal code mappings", "city", p.City, "postal_code", p.PostalCode, "updated", p.Updated)
		}
	})
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "completed prices neighborhood postal code sync")
	return nil
}

func (c *Consumer) handlePricesSyncAll(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices sync all task")
	cfg := prices.DefaultSyncAllConfig()
	cfg.Logger = logger
	cfg.Concurrency = 5
	result, err := c.syncRunner.PricesSyncAll(ctx, cfg)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "completed prices sync all", "cities", result.CitiesProcessed, "postal_codes", result.PostalCodesProcessed, "neighborhoods", result.NeighborhoodsUpdated, "transactions", result.TransactionsProcessed, "errors", len(result.Errors))
	return nil
}

func (c *Consumer) enqueuePricesTask(ctx context.Context, queue *taskqueue.Queue, entityID, taskType string) error {
	task, err := c.queries.UpsertPricesPendingTask(ctx, db.UpsertPricesPendingTaskParams{
		PricesPendingTaskEntityID:    entityID,
		PricesPendingTaskType:        taskType,
		PricesPendingTaskPriority:    int32(taskqueue.PriorityNormal),
		PricesPendingTaskMaxAttempts: int32(3),
	})
	if err != nil {
		return nil // ON CONFLICT DO NOTHING - active task already exists for this entity
	}
	_, err = queue.Send(ctx, taskqueue.MessageData{
		PendingTaskID: task.PricesPendingTaskID,
		EntityID:      entityID,
		TaskType:      taskType,
	})
	return err
}
