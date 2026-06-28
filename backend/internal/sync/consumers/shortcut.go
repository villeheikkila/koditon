package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

const (
	TaskTypeShortcutSitemapSync          = "shortcut_sitemap_sync"
	TaskTypeShortcutBuildingsSitemapSync = "shortcut_buildings_sitemap_sync"
	TaskTypeShortcutScraperSync          = "shortcut_scraper_sync"
	TaskTypeShortcutAPISync              = "shortcut_api_sync"
	TaskTypeShortcutAdDataHashBackfill   = "shortcut_ad_data_hash_backfill"
)

func (c *Consumer) handleShortcutSitemapSync(ctx context.Context, logger *slog.Logger, _ syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.sitemap_sync"))
	buildingIDs, adIDs, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	var enqueueErrors int
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueShortcutTask(ctx, nil, buildingID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	for _, adID := range adIDs {
		if enqErr := c.enqueueShortcutTask(ctx, nil, adID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "shortcut sitemap sync completed", "buildings", len(buildingIDs), "ads", len(adIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutBuildingsSitemapSync(ctx context.Context, logger *slog.Logger, _ syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.buildings_sitemap_sync"))
	buildingIDs, _, err := c.syncRunner.ShortcutSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut buildings sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	var enqueueErrors int
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueShortcutTask(ctx, nil, buildingID, TaskTypeShortcutScraperSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "shortcut buildings sitemap sync completed", "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutScraperSync(ctx context.Context, logger *slog.Logger, msg syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.scraper_sync"))
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut scraper sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	if err := c.enqueueShortcutCanonicalizationForSyncedEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "shortcut scraper entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleShortcutAPISync(ctx context.Context, logger *slog.Logger, msg syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.api_sync"))
	if err := c.syncRunner.ShortcutSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "shortcut api sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	if err := c.enqueueShortcutCanonicalizationForSyncedEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "shortcut api entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueShortcutCanonicalizationForSyncedEntity(ctx context.Context, entityID string) error {
	entityType, sourceID, err := parseJobEntity(entityID)
	if err != nil || entityType != "ad" {
		return nil
	}
	adID, err := strconv.ParseInt(sourceID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse synced shortcut ad id: %w", err)
	}
	ad, err := c.queries.GetShortcutAdByID(ctx, adID)
	if err != nil {
		return fmt.Errorf("load synced shortcut ad for canonicalization enqueue: %w", err)
	}
	if ad.ShortcutAdDataHash == nil {
		return nil
	}
	return c.enqueueCanonicalizeSourceAd(ctx, "shortcut_ad", sourceID, int32(priorityNormal))
}

func (c *Consumer) handleShortcutAdDataHashBackfill(ctx context.Context, logger *slog.Logger, job syncJobEnvelope) error {
	logger = logging.With(logger, logging.Op("consumer.shortcut.ad_data_hash_backfill"))
	payload := sourceAdDataHashBackfillPayload{Limit: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return newPermanentError(fmt.Errorf("decode shortcut ad data hash backfill payload: %w", err), "invalid payload")
		}
	}
	scanned, updated, batches, err := c.syncRunner.ShortcutBackfillAdDataHashes(ctx, payload.Limit)
	if err != nil {
		logger.ErrorContext(ctx, "shortcut ad data hash backfill failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	logger.InfoContext(ctx, "shortcut ad data hash backfill completed", "scanned", scanned, "updated", updated, "batches", batches, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueShortcutTask(ctx context.Context, _ any, entityID, taskType string) error {
	app := c.shortcutAPIWorkflowClient
	if taskType == TaskTypeShortcutScraperSync {
		app = c.shortcutScraperWorkflowClient
	}
	_, err := workflows.Spawn(ctx, app, workflows.SpawnRequest{
		Provider: "shortcut",
		Kind:     taskType,
		EntityID: entityID,
	})
	return err
}
