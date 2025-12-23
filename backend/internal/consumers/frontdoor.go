package consumers

import (
	"context"
	"fmt"
	"log/slog"

	taskqueuedb "koditon-go/internal/taskqueue/db"
)

func (c *Consumer) handleFrontdoorSitemapSync(ctx context.Context, logger *slog.Logger) error {
	adIDs, buildingIDs, err := c.frontdoorService.SyncSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor sitemap sync failed", "error", err)
		return fmt.Errorf("frontdoor sitemap sync: %w", err)
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
	entityType, externalID, err := parseEntityID(task.EntityID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse entity ID", "entity_id", task.EntityID, "error", err)
		return err
	}
	switch entityType {
	case "ad":
		if err := c.frontdoorService.SyncAd(ctx, externalID); err != nil {
			logger.ErrorContext(ctx, "frontdoor ad sync failed", "external_id", externalID, "error", err)
			return fmt.Errorf("sync frontdoor ad %s: %w", externalID, err)
		}
		logger.InfoContext(ctx, "frontdoor ad synced", "external_id", externalID)
		return nil
	case "building":
		if err := c.frontdoorService.SyncBuilding(ctx, externalID); err != nil {
			logger.ErrorContext(ctx, "frontdoor building sync failed", "external_id", externalID, "error", err)
			return fmt.Errorf("sync frontdoor building %s: %w", externalID, err)
		}
		logger.InfoContext(ctx, "frontdoor building synced", "external_id", externalID)
		return nil
	default:
		return &EntityParseError{
			EntityID: task.EntityID,
			Reason:   fmt.Sprintf("unknown frontdoor entity type: %s", entityType),
		}
	}
}
