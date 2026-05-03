package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

const (
	TaskTypeCanonicalizeSourceAdsFanout    = "canonicalize_source_ads_fanout"
	TaskTypeCanonicalizeSourceAd           = "canonicalize_source_ad"
	currentSourceAdCanonicalizationVersion = int32(1)
)

type canonicalizeSourceAdsFanoutPayload struct {
	Limit int32 `json:"limit,omitempty"`
}

type canonicalizeSourceAdPayload struct {
	SourceTable string `json:"source_table,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
}

func (c *Consumer) handleCanonicalizeSourceAdsFanout(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonicalize.source_ads_fanout"))
	payload := canonicalizeSourceAdsFanoutPayload{Limit: 1000}
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return taskqueue.NewPermanentError(fmt.Errorf("decode canonicalize source ads fanout payload: %w", err), "invalid payload")
		}
	}
	if payload.Limit <= 0 || payload.Limit > 5000 {
		payload.Limit = 1000
	}
	rows, err := c.pool.Query(ctx, `
(SELECT 'frontdoor_ad'::text AS source_table, frontdoor_ad_id::text AS source_id
 FROM public.frontdoor_ads
 WHERE frontdoor_ad_data_hash IS NOT NULL
     AND (frontdoor_ad_data_normalized_at IS NULL
         OR frontdoor_ad_data_changed_at > frontdoor_ad_data_normalized_at
         OR frontdoor_ad_data_normalized_version < $2)
 ORDER BY frontdoor_ad_updated_at ASC
 LIMIT $1)
UNION ALL
(SELECT 'shortcut_ad'::text AS source_table, shortcut_ad_id::text AS source_id
 FROM public.shortcut_ads
 WHERE shortcut_ad_data_hash IS NOT NULL
     AND (shortcut_ad_data_normalized_at IS NULL
         OR shortcut_ad_data_changed_at > shortcut_ad_data_normalized_at
         OR shortcut_ad_data_normalized_version < $2)
 ORDER BY shortcut_ad_updated_at ASC NULLS FIRST
 LIMIT $1)`, payload.Limit, currentSourceAdCanonicalizationVersion)
	if err != nil {
		return fmt.Errorf("list source ads for canonicalization: %w", err)
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var sourceTable string
		var sourceID string
		if err := rows.Scan(&sourceTable, &sourceID); err != nil {
			return fmt.Errorf("scan canonicalize source ad fanout row: %w", err)
		}
		if err := c.enqueueCanonicalizeSourceAd(ctx, sourceTable, sourceID, int32(taskqueue.PriorityLow)); err != nil {
			return err
		}
		enqueued++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate canonicalize source ad fanout rows: %w", err)
	}
	logger.InfoContext(ctx, "canonicalize source ad jobs enqueued", "count", enqueued, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleCanonicalizeSourceAd(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.canonicalize.source_ad"))
	payload, err := decodeCanonicalizeSourceAdPayload(job)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid payload")
	}
	switch payload.SourceTable {
	case "frontdoor_ad":
		return c.canonicalizeFrontdoorAd(ctx, logger, payload.SourceID)
	case "shortcut_ad":
		return c.canonicalizeShortcutAd(ctx, logger, payload.SourceID)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown source table %q", payload.SourceTable), "invalid source table")
	}
}

func decodeCanonicalizeSourceAdPayload(job db.SyncJob) (canonicalizeSourceAdPayload, error) {
	var payload canonicalizeSourceAdPayload
	if len(job.SyncJobPayload) > 0 {
		if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
			return canonicalizeSourceAdPayload{}, fmt.Errorf("decode canonicalize source ad payload: %w", err)
		}
	}
	if payload.SourceTable == "" || payload.SourceID == "" {
		sourceTable, sourceID, err := parseJobEntity(job.SyncJobEntityID)
		if err != nil {
			return canonicalizeSourceAdPayload{}, err
		}
		payload.SourceTable = sourceTable
		payload.SourceID = sourceID
	}
	payload.SourceTable = strings.TrimSpace(payload.SourceTable)
	payload.SourceID = strings.TrimSpace(payload.SourceID)
	if payload.SourceTable == "" || payload.SourceID == "" {
		return canonicalizeSourceAdPayload{}, fmt.Errorf("source_table and source_id are required")
	}
	return payload, nil
}

func (c *Consumer) canonicalizeFrontdoorAd(ctx context.Context, logger *slog.Logger, sourceID string) error {
	frontdoorAdID, err := uuid.Parse(sourceID)
	if err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("frontdoor source id must be a uuid: %w", err), "invalid source id")
	}
	ad, err := c.queries.GetFrontdoorAdByID(ctx, frontdoorAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load frontdoor ad for canonicalization: %w", err)
	}
	if ad.FrontdoorAdDataHash == nil {
		return nil
	}
	saleListingID, err := c.queries.CanonicalizeFrontdoorAdSaleListing(ctx, frontdoorAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("canonicalize frontdoor ad sale listing: %w", err)
	}
	if err := c.queries.MarkFrontdoorAdDataNormalized(ctx, db.MarkFrontdoorAdDataNormalizedParams{FrontdoorAdDataNormalizedVersion: currentSourceAdCanonicalizationVersion, FrontdoorAdExternalID: ad.FrontdoorAdExternalID, FrontdoorAdDataHash: ad.FrontdoorAdDataHash}); err != nil {
		return fmt.Errorf("mark frontdoor ad data normalized: %w", err)
	}
	if err := c.enqueueCanonicalSourceMatchSaleListing(ctx, saleListingID.String(), 1, time.Now()); err != nil {
		return err
	}
	logger.InfoContext(ctx, "frontdoor ad canonicalized", "frontdoor_ad_id", sourceID, "sale_listing_id", saleListingID.String(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) canonicalizeShortcutAd(ctx context.Context, logger *slog.Logger, sourceID string) error {
	shortcutAdID, err := strconv.ParseInt(sourceID, 10, 64)
	if err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("shortcut source id must be an integer: %w", err), "invalid source id")
	}
	ad, err := c.queries.GetShortcutAdByID(ctx, shortcutAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("load shortcut ad for canonicalization: %w", err)
	}
	if ad.ShortcutAdDataHash == nil {
		return nil
	}
	if ad.ShortcutAdType != "listing" {
		if err := c.queries.DeleteSaleListingForShortcutAd(ctx, &shortcutAdID); err != nil {
			return fmt.Errorf("delete shortcut non-listing sale listing: %w", err)
		}
		return c.queries.MarkShortcutAdDataNormalized(ctx, db.MarkShortcutAdDataNormalizedParams{ShortcutAdDataNormalizedVersion: currentSourceAdCanonicalizationVersion, ShortcutAdID: shortcutAdID, ShortcutAdDataHash: ad.ShortcutAdDataHash})
	}
	saleListingID, err := c.queries.CanonicalizeShortcutAdSaleListing(ctx, shortcutAdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("canonicalize shortcut ad sale listing: %w", err)
	}
	if err := c.queries.MarkShortcutAdDataNormalized(ctx, db.MarkShortcutAdDataNormalizedParams{ShortcutAdDataNormalizedVersion: currentSourceAdCanonicalizationVersion, ShortcutAdID: shortcutAdID, ShortcutAdDataHash: ad.ShortcutAdDataHash}); err != nil {
		return fmt.Errorf("mark shortcut ad data normalized: %w", err)
	}
	if err := c.enqueueCanonicalSourceMatchSaleListing(ctx, saleListingID.String(), 1, time.Now()); err != nil {
		return err
	}
	logger.InfoContext(ctx, "shortcut ad canonicalized", "shortcut_ad_id", shortcutAdID, "sale_listing_id", saleListingID.String(), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueueCanonicalizeSourceAd(ctx context.Context, sourceTable, sourceID string, priority int32) error {
	payload, err := json.Marshal(canonicalizeSourceAdPayload{SourceTable: sourceTable, SourceID: sourceID})
	if err != nil {
		return fmt.Errorf("marshal canonicalize source ad payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "canonical",
		Kind:        TaskTypeCanonicalizeSourceAd,
		EntityID:    fmt.Sprintf("%s:%s", sourceTable, sourceID),
		Priority:    priority,
		MaxAttempts: 3,
		Payload:     payload,
	})
	return err
}
