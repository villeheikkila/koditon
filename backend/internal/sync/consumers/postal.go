package consumers

import (
	"context"
	"log/slog"

	"koditon/internal/platform/logging"
)

const (
	TaskTypePostalSync = "postal_sync"
)

func (c *Consumer) handlePostalSync(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.postal.sync"))
	logger.InfoContext(ctx, "postal sync started")
	result, err := c.syncRunner.PostalSync(ctx, logger)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "postal sync completed", "total_records", result.TotalRecords, "ad_areas_upserted", result.AdAreasUpserted, "municipalities_upserted", result.MunicipalitiesUpserted, "postal_codes_upserted", result.PostalCodesUpserted, "skipped_records", result.SkippedRecords, "outcome", logging.OutcomeSuccess)
	return nil
}
