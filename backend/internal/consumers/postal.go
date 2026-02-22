package consumers

import (
	"context"
	"log/slog"
)

func (c *Consumer) handlePostalSync(ctx context.Context, logger *slog.Logger) error {
	logger.InfoContext(ctx, "processing postal sync task")
	result, err := c.syncRunner.PostalSync(ctx, logger)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "completed postal sync", "total_records", result.TotalRecords, "ad_areas_upserted", result.AdAreasUpserted, "municipalities_upserted", result.MunicipalitiesUpserted, "postal_codes_upserted", result.PostalCodesUpserted, "skipped_records", result.SkippedRecords)
	return nil
}
