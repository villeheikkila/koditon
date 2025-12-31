package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"koditon-go/internal/prices"
	"koditon-go/internal/taskqueue"
	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handlePricesCitiesInit(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices cities initialization task")
	cities, err := c.pricesService.FetchCities(ctx)
	if err != nil {
		return fmt.Errorf("fetch cities: %w", err)
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
	entityType, cityName, err := parseEntityID(task.EntityID)
	if err != nil {
		return err
	}
	if entityType != "city" {
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   fmt.Sprintf("expected city entity type for prices sync, got: %s", entityType),
		}
	}
	logger.InfoContext(ctx, "syncing prices for city", "city", cityName)
	return c.pricesService.SyncCity(ctx, cityName)
}

func (c *Consumer) handlePricesNeighborhoodPostalCodeSync(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing prices neighborhood postal code sync task")
	err := c.pricesService.SyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
		if p.Page > 0 {
			logger.DebugContext(ctx, "fetching postal code transactions",
				"city", p.City,
				"postal_code", p.PostalCode,
				"page", p.Page,
			)
		} else if p.Updated > 0 {
			logger.InfoContext(ctx, "updated neighborhood postal code mappings",
				"city", p.City,
				"postal_code", p.PostalCode,
				"updated", p.Updated,
			)
		}
	})
	if err != nil {
		return fmt.Errorf("sync neighborhood postal codes: %w", err)
	}
	logger.InfoContext(ctx, "completed prices neighborhood postal code sync")
	return nil
}
