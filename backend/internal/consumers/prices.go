package consumers

import (
	"context"
	"log/slog"

	"koditon-go/internal/prices"
	"koditon-go/internal/taskqueue"
	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handlePricesCitiesInit(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices cities initialization task")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	if len(cities) > 0 {
		cityEntityIDs := make([]string, 0, len(cities))
		for _, city := range cities {
			cityEntityIDs = append(cityEntityIDs, taskqueue.EntityPrefixCity+city)
		}
		count, regErr := c.taskQueueClient.RegisterEntities(ctx, cityEntityIDs, "prices_city", "daily")
		if regErr != nil {
			logger.WarnContext(ctx, "failed to register city entities", "error", regErr)
		} else {
			logger.InfoContext(ctx, "city entities registered", "count", count)
		}
	}
	return nil
}

func (c *Consumer) handlePricesSync(ctx context.Context, logger *slog.Logger, task taskqueuedb.TaskQueueTask) error {
	logger.InfoContext(ctx, "syncing prices city", "entity_id", task.EntityID)
	return c.syncRunner.PricesSyncCityEntity(ctx, task.EntityID)
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
