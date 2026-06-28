package postal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	client "koditon/internal/clients/postal"
	"koditon/internal/db"
	"koditon/internal/platform/logging"
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

type PostalCodesUpsertResult struct {
	Upserted int64
	Skipped  int
}

func (s *Service) Sync(ctx context.Context, logger *slog.Logger) (*SyncResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logging.With(logger, logging.Op("postal.sync"))
	records, err := s.FetchPostalData(ctx, logger)
	if err != nil {
		return nil, err
	}
	adAreaIDs, err := s.UpsertAdAreas(ctx, records, logger)
	if err != nil {
		return nil, err
	}
	municipalityIDs, err := s.UpsertMunicipalities(ctx, records, logger)
	if err != nil {
		return nil, err
	}
	postalCodes, err := s.UpsertPostalCodes(ctx, records, adAreaIDs, municipalityIDs, logger)
	if err != nil {
		return nil, err
	}
	return &SyncResult{
		TotalRecords:           len(records),
		AdAreasUpserted:        len(adAreaIDs),
		MunicipalitiesUpserted: len(municipalityIDs),
		PostalCodesUpserted:    postalCodes.Upserted,
		SkippedRecords:         postalCodes.Skipped,
	}, nil
}

func (s *Service) FetchPostalData(ctx context.Context, logger *slog.Logger) ([]*client.PostalCodeRecord, error) {
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "postal code fetch started", "provider", "posti")
	records, err := s.client.FetchPostalCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch postal codes: %w", err)
	}
	logger.InfoContext(ctx, "postal codes fetched", "count", len(records))
	return records, nil
}

func (s *Service) UpsertAdAreas(ctx context.Context, records []*client.PostalCodeRecord, logger *slog.Logger) (map[string]uuid.UUID, error) {
	if logger == nil {
		logger = slog.Default()
	}
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
	return adAreaIDs, nil
}

func (s *Service) UpsertMunicipalities(ctx context.Context, records []*client.PostalCodeRecord, logger *slog.Logger) (map[string]uuid.UUID, error) {
	if logger == nil {
		logger = slog.Default()
	}
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
	return municipalityIDs, nil
}

func (s *Service) UpsertPostalCodes(ctx context.Context, records []*client.PostalCodeRecord, adAreaIDs, municipalityIDs map[string]uuid.UUID, logger *slog.Logger) (PostalCodesUpsertResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	postalCodeParams := mapUpsertPostalCodesBulkParams(records, adAreaIDs, municipalityIDs)
	skipped := len(records) - len(postalCodeParams.Codes)
	upserted, err := s.queries.UpsertPostalPostalCodesBulk(ctx, postalCodeParams)
	if err != nil {
		return PostalCodesUpsertResult{}, fmt.Errorf("upsert postal codes: %w", err)
	}
	logger.InfoContext(ctx, "postal codes synced",
		"total", len(records),
		"upserted", upserted,
		"skipped", skipped,
		"outcome", logging.OutcomeSuccess,
	)
	return PostalCodesUpsertResult{Upserted: upserted, Skipped: skipped}, nil
}
