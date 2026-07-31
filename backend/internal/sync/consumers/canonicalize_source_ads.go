package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/domain/listingmodel"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	"koditon/internal/sync/workflows"
)

const (
	TaskTypeCanonicalizeSourceAdsFanout         = "canonicalize_source_ads_fanout"
	TaskTypeCanonicalizeSourceAd                = "canonicalize_source_ad"
	TaskTypeCanonicalLinkFrontdoorAnnouncements = "canonical_link_frontdoor_announcements"
	currentSourceAdCanonicalizationVersion      = int32(3)
)

type canonicalizeSourceAdsFanoutPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type canonicalLinkFrontdoorAnnouncementsPayload struct {
	Limit       int32 `json:"limit,omitempty"`
	MinAgeHours int32 `json:"min_age_hours,omitempty"`
}

type canonicalizeSourceAdPayload struct {
	SourceTable string `json:"source_table,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
}

func (c *Consumer) canonicalizeFrontdoorBuildingAnnouncement(ctx context.Context, logger *slog.Logger, sourceID string) error {
	announcementID, err := uuid.Parse(sourceID)
	if err != nil {
		return newPermanentError(fmt.Errorf("frontdoor announcement source id must be a uuid: %w", err), "invalid source id")
	}
	announcement, err := c.queries.GetFrontdoorBuildingAnnouncementByID(ctx, &announcementID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load frontdoor building announcement for canonicalization: %w", err)
	}
	model := listingmodel.NewService(logger, c.pool)
	if strings.HasPrefix(announcement.FrontdoorBuildingAnnouncementIdentityKey, "legacy:") || announcement.FrontdoorBuildingAnnouncementRentPeriod != nil || announcement.FrontdoorBuildingAnnouncementRentalUniqueNo != nil {
		if err := model.RemoveFrontdoorBuildingAnnouncement(ctx, announcementID); err != nil {
			return fmt.Errorf("delete excluded frontdoor announcement source offering: %w", err)
		}
		version := currentSourceAdCanonicalizationVersion
		return c.queries.MarkFrontdoorBuildingAnnouncementDataNormalized(ctx, db.MarkFrontdoorBuildingAnnouncementDataNormalizedParams{FrontdoorBuildingAnnouncementDataNormalizedVersion: &version, FrontdoorBuildingAnnouncementID: &announcementID})
	}
	result, err := model.ReconcileFrontdoorBuildingAnnouncement(ctx, announcementID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("reconcile frontdoor building announcement listing model: %w", err)
	}
	saleListingID := result.SourceListingID
	if err := c.queries.RefreshSourceListingRenovationsFromFrontdoorBuilding(ctx, &saleListingID); err != nil {
		return fmt.Errorf("refresh frontdoor announcement renovations: %w", err)
	}
	if err := properties.ProjectListingRenovationEvents(ctx, c.pool, saleListingID); err != nil {
		return err
	}
	if _, err := c.queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "source_listing_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty for source offering: %w", err)
	}
	version := currentSourceAdCanonicalizationVersion
	if err := c.queries.MarkFrontdoorBuildingAnnouncementDataNormalized(ctx, db.MarkFrontdoorBuildingAnnouncementDataNormalizedParams{FrontdoorBuildingAnnouncementDataNormalizedVersion: &version, FrontdoorBuildingAnnouncementID: &announcementID}); err != nil {
		return fmt.Errorf("mark frontdoor announcement data normalized: %w", err)
	}
	logger.InfoContext(ctx, "frontdoor building announcement canonicalized", "frontdoor_building_announcement_id", sourceID, "sale_listing_id", saleListingID.String(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) canonicalizeFrontdoorAd(ctx context.Context, logger *slog.Logger, sourceID string) error {
	frontdoorAdID, err := uuid.Parse(sourceID)
	if err != nil {
		return newPermanentError(fmt.Errorf("frontdoor source id must be a uuid: %w", err), "invalid source id")
	}
	ad, err := c.queries.GetFrontdoorAdByID(ctx, &frontdoorAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load frontdoor ad for canonicalization: %w", err)
	}
	if len(ad.FrontdoorAdData) == 0 {
		return nil
	}
	result, err := listingmodel.NewService(logger, c.pool).ReconcileFrontdoorAd(ctx, frontdoorAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("reconcile frontdoor ad listing model: %w", err)
	}
	saleListingID := result.SourceListingID
	version := currentSourceAdCanonicalizationVersion
	if err := c.queries.MarkFrontdoorAdDataNormalized(ctx, db.MarkFrontdoorAdDataNormalizedParams{FrontdoorAdDataNormalizedVersion: &version, FrontdoorAdExternalID: &ad.FrontdoorAdExternalID, FrontdoorAdDataHash: ad.FrontdoorAdDataHash}); err != nil {
		return fmt.Errorf("mark frontdoor ad data normalized: %w", err)
	}
	if _, err := c.queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "source_listing_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty for frontdoor ad source offering: %w", err)
	}
	logger.InfoContext(ctx, "frontdoor ad canonicalized", "frontdoor_ad_id", sourceID, "sale_listing_id", saleListingID.String(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) canonicalizeShortcutAd(ctx context.Context, logger *slog.Logger, sourceID string) error {
	shortcutAdID, err := strconv.ParseInt(sourceID, 10, 64)
	if err != nil {
		return newPermanentError(fmt.Errorf("shortcut source id must be an integer: %w", err), "invalid source id")
	}
	ad, err := c.queries.GetShortcutAdByID(ctx, &shortcutAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load shortcut ad for canonicalization: %w", err)
	}
	if len(ad.ShortcutAdData) == 0 {
		return nil
	}
	if ad.ShortcutAdType != "listing" {
		if err := listingmodel.NewService(logger, c.pool).RemoveShortcutAdListing(ctx, shortcutAdID); err != nil {
			return fmt.Errorf("delete shortcut non-listing sale listing: %w", err)
		}
		version := currentSourceAdCanonicalizationVersion
		return c.queries.MarkShortcutAdDataNormalized(ctx, db.MarkShortcutAdDataNormalizedParams{ShortcutAdDataNormalizedVersion: &version, ShortcutAdID: &shortcutAdID, ShortcutAdDataHash: ad.ShortcutAdDataHash})
	}
	result, err := listingmodel.NewService(logger, c.pool).ReconcileShortcutAd(ctx, shortcutAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("reconcile shortcut ad listing model: %w", err)
	}
	saleListingID := result.SourceListingID
	version := currentSourceAdCanonicalizationVersion
	if err := c.queries.MarkShortcutAdDataNormalized(ctx, db.MarkShortcutAdDataNormalizedParams{ShortcutAdDataNormalizedVersion: &version, ShortcutAdID: &shortcutAdID, ShortcutAdDataHash: ad.ShortcutAdDataHash}); err != nil {
		return fmt.Errorf("mark shortcut ad data normalized: %w", err)
	}
	if _, err := c.queries.MarkListingDimensionTargetsDirty(ctx, db.MarkListingDimensionTargetsDirtyParams{SaleListingID: saleListingID, Reason: "source_listing_changed"}); err != nil {
		return fmt.Errorf("mark dimension targets dirty for shortcut ad source offering: %w", err)
	}
	logger.InfoContext(ctx, "shortcut ad canonicalized", "shortcut_ad_id", shortcutAdID, "sale_listing_id", saleListingID.String(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueCanonicalizeSourceAd(ctx context.Context, sourceTable, sourceID string, priority int32) error {
	payload, err := json.Marshal(canonicalizeSourceAdPayload{SourceTable: sourceTable, SourceID: sourceID})
	if err != nil {
		return fmt.Errorf("marshal canonicalize source ad payload: %w", err)
	}
	req := workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalizeSourceAd,
		Params:   payload,
	}
	if priority > 0 {
		task, ok := absurd.TaskFromContext(ctx)
		if !ok {
			return errors.New("source refresh canonicalization requires a workflow task context")
		}
		req.IdempotencyKey = workflows.SourceRefreshIdempotencyKey(TaskTypeCanonicalizeSourceAd, task.TaskID(), sourceTable+":"+sourceID)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, req)
	return err
}

func (c *Consumer) enqueueCanonicalizeSourceAdsFanout(ctx context.Context, limit int32) error {
	payload, err := json.Marshal(canonicalizeSourceAdsFanoutPayload{Limit: limit})
	if err != nil {
		return fmt.Errorf("marshal canonicalize source ads fanout payload: %w", err)
	}
	_, err = workflows.Spawn(ctx, c.canonicalWorkflowClient, workflows.SpawnTaskRequest{
		TaskName: TaskTypeCanonicalizeSourceAdsFanout,
		Params:   payload,
	})
	return err
}
