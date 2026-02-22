package consumers

import (
	"context"
	"fmt"
	"log/slog"

	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handleFrontdoorSitemapSync(ctx context.Context, logger *slog.Logger) error {
	adIDs, buildingIDs, err := c.syncRunner.FrontdoorSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor sitemap sync failed", "error", err)
		return err
	}
	var regErrors []error
	if len(adIDs) > 0 {
		if _, regErr := c.taskQueueClient.RegisterEntities(ctx, adIDs, "frontdoor_ad", "daily"); regErr != nil {
			logger.ErrorContext(ctx, "failed to register ad entities", "error", regErr, "count", len(adIDs))
			regErrors = append(regErrors, fmt.Errorf("register ad entities: %w", regErr))
		}
	}
	if len(buildingIDs) > 0 {
		if _, regErr := c.taskQueueClient.RegisterEntities(ctx, buildingIDs, "frontdoor_building", "daily"); regErr != nil {
			logger.ErrorContext(ctx, "failed to register building entities", "error", regErr, "count", len(buildingIDs))
			regErrors = append(regErrors, fmt.Errorf("register building entities: %w", regErr))
		}
	}
	if len(regErrors) > 0 && len(adIDs) == 0 && len(buildingIDs) == 0 {
		return fmt.Errorf("frontdoor sitemap sync: all entity registrations failed")
	}
	logger.InfoContext(ctx, "frontdoor sitemap sync completed", "ads", len(adIDs), "buildings", len(buildingIDs))
	return nil
}

func (c *Consumer) handleFrontdoorSync(ctx context.Context, logger *slog.Logger, task taskqueuedb.TaskQueueTask) error {
	if err := c.syncRunner.FrontdoorSyncEntity(ctx, task.EntityID); err != nil {
		logger.ErrorContext(ctx, "frontdoor sync failed", "entity_id", task.EntityID, "error", err)
		return err
	}
	logger.InfoContext(ctx, "frontdoor entity synced", "entity_id", task.EntityID)
	return nil
}
