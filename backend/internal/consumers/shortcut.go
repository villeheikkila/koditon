package consumers

import (
	"context"
	"fmt"
	"log/slog"

	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handleShortcutSitemapSync(ctx context.Context, logger *slog.Logger) error {
	buildingIDs, adIDs, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut sitemap sync failed", "error", err)
		return err
	}
	var regErrors []error
	if len(buildingIDs) > 0 {
		if _, regErr := c.taskQueueClient.RegisterEntities(ctx, buildingIDs, "shortcut_building", "daily"); regErr != nil {
			logger.ErrorContext(ctx, "failed to register building entities", "error", regErr, "count", len(buildingIDs))
			regErrors = append(regErrors, fmt.Errorf("register building entities: %w", regErr))
		}
	}
	if len(adIDs) > 0 {
		if _, regErr := c.taskQueueClient.RegisterEntities(ctx, adIDs, "shortcut_ad", "daily"); regErr != nil {
			logger.ErrorContext(ctx, "failed to register ad entities", "error", regErr, "count", len(adIDs))
			regErrors = append(regErrors, fmt.Errorf("register ad entities: %w", regErr))
		}
	}
	if len(regErrors) > 0 && len(buildingIDs) == 0 && len(adIDs) == 0 {
		return fmt.Errorf("shortcut sitemap sync: all entity registrations failed")
	}
	logger.InfoContext(ctx, "shortcut sitemap sync completed", "buildings", len(buildingIDs), "ads", len(adIDs))
	return nil
}

func (c *Consumer) handleShortcutScraperSync(ctx context.Context, logger *slog.Logger, task taskqueuedb.TaskQueueTask) error {
	if err := c.syncRunner.ShortcutSyncEntity(ctx, task.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut scraper sync failed", "entity_id", task.EntityID, "error", err)
		return err
	}
	logger.InfoContext(ctx, "shortcut scraper entity synced", "entity_id", task.EntityID)
	return nil
}

func (c *Consumer) handleShortcutAPISync(ctx context.Context, logger *slog.Logger, task taskqueuedb.TaskQueueTask) error {
	if err := c.syncRunner.ShortcutSyncEntity(ctx, task.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut api sync failed", "entity_id", task.EntityID, "error", err)
		return err
	}
	logger.InfoContext(ctx, "shortcut api entity synced", "entity_id", task.EntityID)
	return nil
}
