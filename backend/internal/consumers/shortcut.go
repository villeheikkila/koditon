package consumers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"

	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handleShortcutSitemapSync(ctx context.Context, logger *slog.Logger) error {
	buildingIDs, adIDs, err := c.shortcutService.SyncSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut sitemap sync failed", "error", err)
		return fmt.Errorf("shortcut sitemap sync: %w", err)
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
	entityType, externalID, err := parseEntityID(task.EntityID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse entity ID", "entity_id", task.EntityID, "error", err)
		return err
	}
	if entityType != "building" {
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   fmt.Sprintf("expected building entity type for scraper, got: %s", entityType),
		}
	}
	buildingID, err := uuid.Parse(externalID)
	if err != nil {
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   "invalid building UUID",
			Err:      err,
		}
	}
	if err := c.shortcutService.SyncBuilding(ctx, buildingID); err != nil {
		logger.ErrorContext(ctx, "shortcut building sync failed", "building_id", buildingID, "error", err)
		return fmt.Errorf("sync shortcut building %s: %w", buildingID, err)
	}
	logger.InfoContext(ctx, "shortcut building synced", "building_id", buildingID)
	return nil
}

func (c *Consumer) handleShortcutAPISync(ctx context.Context, logger *slog.Logger, task taskqueuedb.TaskQueueTask) error {
	entityType, externalID, err := parseEntityID(task.EntityID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse entity ID", "entity_id", task.EntityID, "error", err)
		return err
	}
	if entityType != "ad" {
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   fmt.Sprintf("expected ad entity type for API sync, got: %s", entityType),
		}
	}
	adID, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   "invalid ad ID",
			Err:      err,
		}
	}
	if err := c.shortcutService.SyncAd(ctx, adID); err != nil {
		logger.ErrorContext(ctx, "shortcut ad sync failed", "ad_id", adID, "error", err)
		return fmt.Errorf("sync shortcut ad %d: %w", adID, err)
	}
	logger.InfoContext(ctx, "shortcut ad synced", "ad_id", adID)
	return nil
}
