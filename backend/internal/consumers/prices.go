package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/db"
	"koditon-go/internal/logging"
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
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
		slog.Int64("sync_task_id", msg.Data.SyncTaskID),
	)
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
	logger = logging.With(logger, logging.Op("consumer.prices.cities_init"))
	logger.InfoContext(ctx, "prices cities initialization started")
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
		logger.InfoContext(ctx, "prices city entities enqueued", "count", len(cities), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	}
	return nil
}

func (c *Consumer) handlePricesSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.prices.city_sync"))
	logger.InfoContext(ctx, "prices city sync started")
	if err := c.syncRunner.PricesSyncCityEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices city sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesNeighborhoodPostalCodeSync(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.neighborhood_postal_code_sync"))
	logger.InfoContext(ctx, "prices neighborhood postal code sync started")
	err := c.syncRunner.PricesSyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
		if p.Page > 0 {
			logger.DebugContext(ctx, "postal code transactions fetch started", "city", p.City, "postal_code", p.PostalCode, "page", p.Page)
		} else if p.Updated > 0 {
			logger.InfoContext(ctx, "neighborhood postal code mappings updated", "city", p.City, "postal_code", p.PostalCode, "updated", p.Updated)
		}
	})
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices neighborhood postal code sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesSyncAll(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.sync_all"))
	logger.InfoContext(ctx, "prices sync all started")
	cfg := prices.DefaultSyncAllConfig()
	cfg.Logger = logger
	cfg.Concurrency = 5
	result, err := c.syncRunner.PricesSyncAll(ctx, cfg)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices sync all completed", "cities", result.CitiesProcessed, "postal_codes", result.PostalCodesProcessed, "neighborhoods", result.NeighborhoodsUpdated, "transactions", result.TransactionsProcessed, "errors", len(result.Errors), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueuePricesTask(ctx context.Context, queue *taskqueue.Queue, entityID, taskType string) error {
	task, err := c.queries.UpsertPricesSyncTask(ctx, db.UpsertPricesSyncTaskParams{
		PricesSyncTaskEntityID:    entityID,
		PricesSyncTaskType:        taskType,
		PricesSyncTaskPriority:    int32(taskqueue.PriorityNormal),
		PricesSyncTaskMaxAttempts: int32(3),
	})
	if err != nil {
		return nil // ON CONFLICT DO NOTHING - active task already exists for this entity
	}
	_, err = queue.Send(ctx, taskqueue.MessageData{
		SyncTaskID: task.PricesSyncTaskID,
		EntityID:   entityID,
		TaskType:   taskType,
	})
	return err
}
