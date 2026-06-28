package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

const (
	TaskTypeFrontdoorSitemapSync          = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorBuildingsSitemapSync = "frontdoor_buildings_sitemap_sync"
	TaskTypeFrontdoorSync                 = "frontdoor_sync"
	TaskTypeFrontdoorAdDataHashBackfill   = "frontdoor_ad_data_hash_backfill"
)

type sourceAdDataHashBackfillPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

func (c *Consumer) handleFrontdoorSitemapSync(ctx context.Context, logger *slog.Logger, _ syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.sitemap_sync"))
	adIDs, buildingIDs, err := c.syncRunner.FrontdoorSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	var enqueueErrors int
	for _, adID := range adIDs {
		if enqErr := c.enqueueFrontdoorTask(ctx, nil, adID, TaskTypeFrontdoorSync); enqErr != nil {
			enqueueErrors++
		}
	}
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueFrontdoorTask(ctx, nil, buildingID, TaskTypeFrontdoorSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "frontdoor sitemap sync completed", "ads", len(adIDs), "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleFrontdoorBuildingsSitemapSync(ctx context.Context, logger *slog.Logger, _ syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.buildings_sitemap_sync"))
	_, buildingIDs, err := c.syncRunner.FrontdoorSitemap(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor buildings sitemap sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	var enqueueErrors int
	for _, buildingID := range buildingIDs {
		if enqErr := c.enqueueFrontdoorTask(ctx, nil, buildingID, TaskTypeFrontdoorSync); enqErr != nil {
			enqueueErrors++
		}
	}
	logger.InfoContext(ctx, "frontdoor buildings sitemap sync completed", "buildings", len(buildingIDs), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleFrontdoorEntitySync(ctx context.Context, logger *slog.Logger, msg syncMessage) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.entity_sync"))
	if err := c.syncRunner.FrontdoorSyncEntity(ctx, msg.Data.EntityID); err != nil {
		logger.ErrorContext(ctx, "frontdoor sync failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	entityType, externalID, err := parseJobEntity(msg.Data.EntityID)
	if err == nil && entityType == "ad" {
		ad, loadErr := c.queries.GetFrontdoorAdByExternalID(ctx, externalID)
		if loadErr != nil {
			return fmt.Errorf("load synced frontdoor ad for canonicalization enqueue: %w", loadErr)
		}
		if ad.FrontdoorAdDataHash != nil {
			if err := c.enqueueCanonicalizeSourceAd(ctx, "frontdoor_ad", ad.FrontdoorAdID.String(), int32(priorityNormal)); err != nil {
				return err
			}
		}
	} else if err == nil && entityType == "building" {
		buildingID, parseErr := uuid.Parse(externalID)
		if parseErr == nil {
			announcements, loadErr := c.queries.ListFrontdoorBuildingAnnouncements(ctx, buildingID)
			if loadErr != nil {
				return fmt.Errorf("load synced frontdoor building announcements for canonicalization enqueue: %w", loadErr)
			}
			for _, announcement := range announcements {
				if err := c.enqueueCanonicalizeSourceAd(ctx, "frontdoor_building_announcement", announcement.FrontdoorBuildingAnnouncementID.String(), int32(priorityNormal)); err != nil {
					return err
				}
			}
		}
	}
	logger.InfoContext(ctx, "frontdoor entity synced", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleFrontdoorAdDataHashBackfill(ctx context.Context, logger *slog.Logger, job syncJobEnvelope) error {
	logger = logging.With(logger, logging.Op("consumer.frontdoor.ad_data_hash_backfill"))
	payload := sourceAdDataHashBackfillPayload{Limit: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return newPermanentError(fmt.Errorf("decode frontdoor ad data hash backfill payload: %w", err), "invalid payload")
		}
	}
	scanned, updated, batches, err := c.syncRunner.FrontdoorBackfillAdDataHashes(ctx, payload.Limit)
	if err != nil {
		logger.ErrorContext(ctx, "frontdoor ad data hash backfill failed", "error", err, "outcome", logging.OutcomeError)
		return err
	}
	logger.InfoContext(ctx, "frontdoor ad data hash backfill completed", "scanned", scanned, "updated", updated, "batches", batches, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueFrontdoorTask(ctx context.Context, _ any, entityID, taskType string) error {
	_, err := workflows.Spawn(ctx, c.frontdoorWorkflowClient, workflows.SpawnRequest{
		Provider: "frontdoor",
		Kind:     taskType,
		EntityID: entityID,
	})
	return err
}
