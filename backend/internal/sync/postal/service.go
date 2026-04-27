package postal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"koditon-go/internal/clients/postal"
	"koditon-go/internal/db"
	"koditon-go/internal/platform/logging"
)

type Service struct {
	client  *client.Client
	queries *db.Queries
}

func NewService(dbtx db.DBTX) *Service {
	return &Service{
		client:  client.NewClient(),
		queries: db.New(dbtx),
	}
}

type SyncResult struct {
	TotalRecords           int
	AdAreasUpserted        int
	MunicipalitiesUpserted int
	PostalCodesUpserted    int64
	SkippedRecords         int
}

func (s *Service) Sync(ctx context.Context, logger *slog.Logger) (*SyncResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logging.With(logger, logging.Op("postal.sync"))
	logger.InfoContext(ctx, "postal code fetch started", "provider", "posti")
	records, err := s.client.FetchPostalCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch postal codes: %w", err)
	}
	logger.InfoContext(ctx, "postal codes fetched", "count", len(records))
	adAreaParams := extractAdAreas(records)
	adAreaRows, err := s.queries.UpsertPostalAdAreasBulk(ctx, *adAreaParams)
	if err != nil {
		return nil, fmt.Errorf("upsert ad areas: %w", err)
	}
	adAreaIDs := make(map[string]uuid.UUID, len(adAreaRows))
	for _, row := range adAreaRows {
		adAreaIDs[row.PostalAdAreaCode] = row.PostalAdAreaID
	}
	logger.InfoContext(ctx, "postal ad areas upserted", "count", len(adAreaRows))
	municipalityParams := extractMunicipalities(records)
	municipalityRows, err := s.queries.UpsertPostalMunicipalitiesBulk(ctx, *municipalityParams)
	if err != nil {
		return nil, fmt.Errorf("upsert municipalities: %w", err)
	}
	municipalityIDs := make(map[string]uuid.UUID, len(municipalityRows))
	for _, row := range municipalityRows {
		municipalityIDs[row.PostalMunicipalityCode] = row.PostalMunicipalityID
	}
	logger.InfoContext(ctx, "postal municipalities upserted", "count", len(municipalityRows))
	postalCodeParams := mapUpsertPostalCodesBulkParams(records, adAreaIDs, municipalityIDs)
	skipped := len(records) - len(postalCodeParams.Codes)
	upserted, err := s.queries.UpsertPostalPostalCodesBulk(ctx, postalCodeParams)
	if err != nil {
		return nil, fmt.Errorf("upsert postal codes: %w", err)
	}
	logger.InfoContext(ctx, "postal codes synced",
		"total", len(records),
		"upserted", upserted,
		"skipped", skipped,
		"outcome", logging.OutcomeSuccess,
	)
	return &SyncResult{
		TotalRecords:           len(records),
		AdAreasUpserted:        len(adAreaRows),
		MunicipalitiesUpserted: len(municipalityRows),
		PostalCodesUpserted:    upserted,
		SkippedRecords:         skipped,
	}, nil
}
