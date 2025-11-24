package hintatiedot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/db"
	"koditon-go/internal/util"
)

type SyncService struct {
	queries *db.Queries
	api     *Client
	logger  *slog.Logger
	nowFunc func() time.Time
}

func NewSyncService(queries *db.Queries, api *Client, logger *slog.Logger) *SyncService {
	return &SyncService{
		queries: queries,
		api:     api,
		logger:  logger,
		nowFunc: time.Now,
	}
}

func (s *SyncService) SyncCity(ctx context.Context, city string) error {
	s.logger.InfoContext(ctx, "starting city sync", "city", city)

	// Upsert city
	s.logger.DebugContext(ctx, "upserting city", "city", city)
	cityRow, err := s.queries.UpsertHintatiedotCity(ctx, city)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to upsert city", "city", city, "err", err)
		return fmt.Errorf("upsert city %q: %w", city, err)
	}
	cityID := cityRow.HintatiedotCitiesID
	s.logger.InfoContext(ctx, "city upserted", "city", city, "city_id", cityID)

	// Fetch and upsert postal codes
	s.logger.DebugContext(ctx, "fetching postal codes", "city", city)
	postalCodes, err := s.api.FetchPostalCodes(ctx, city)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch postal codes", "city", city, "err", err)
		return fmt.Errorf("fetch postal codes for %q: %w", city, err)
	}
	postalCodes = util.UniqueStrings(postalCodes)
	s.logger.InfoContext(ctx, "fetched postal codes", "city", city, "count", len(postalCodes))

	postalCodeIDs := make(map[string]pgtype.UUID, len(postalCodes))
	if len(postalCodes) > 0 {
		s.logger.DebugContext(ctx, "bulk upserting postal codes", "city", city, "count", len(postalCodes))
		rows, err := s.queries.UpsertHintatiedotPostalCodesBulk(ctx, db.UpsertHintatiedotPostalCodesBulkParams{
			Codes:  postalCodes,
			CityID: cityID,
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to bulk upsert postal codes", "city", city, "count", len(postalCodes), "err", err)
			return fmt.Errorf("bulk upsert postal codes for %q: %w", city, err)
		}
		for _, row := range rows {
			postalCodeIDs[row.HintatiedotPostalCodesCode] = row.HintatiedotPostalCodesID
		}
		s.logger.InfoContext(ctx, "postal codes upserted", "city", city, "count", len(rows))
	}

	// Fetch and upsert neighborhoods
	s.logger.DebugContext(ctx, "fetching neighborhoods", "city", city)
	neighborhoods, err := s.api.FetchNeighborhoods(ctx, city)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch neighborhoods", "city", city, "err", err)
		return fmt.Errorf("fetch neighborhoods for %q: %w", city, err)
	}
	neighborhoods = util.UniqueStrings(neighborhoods)
	s.logger.InfoContext(ctx, "fetched neighborhoods", "city", city, "count", len(neighborhoods))

	neighborhoodIDs := make(map[string]pgtype.UUID, len(neighborhoods))
	if len(neighborhoods) > 0 {
		s.logger.DebugContext(ctx, "bulk upserting neighborhoods", "city", city, "count", len(neighborhoods))
		rows, err := s.queries.UpsertHintatiedotNeighborhoodsBulk(ctx, db.UpsertHintatiedotNeighborhoodsBulkParams{
			Names:  neighborhoods,
			CityID: cityID,
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to bulk upsert neighborhoods", "city", city, "count", len(neighborhoods), "err", err)
			return fmt.Errorf("bulk upsert neighborhoods for %q: %w", city, err)
		}
		for _, row := range rows {
			key := util.NormalizeString(row.HintatiedotNeighborhoodsName)
			neighborhoodIDs[key] = row.HintatiedotNeighborhoodsID
		}
		s.logger.InfoContext(ctx, "neighborhoods upserted", "city", city, "count", len(rows))
	}

	// Fetch and upsert transactions
	s.logger.DebugContext(ctx, "fetching transactions", "city", city)
	transactions, err := s.api.GetAllTransactions(ctx, city)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch transactions", "city", city, "err", err)
		return fmt.Errorf("fetch transactions for %q: %w", city, err)
	}
	s.logger.InfoContext(ctx, "fetched transactions", "city", city, "count", len(transactions))

	if len(transactions) > 0 {
		s.logger.DebugContext(ctx, "building transaction params", "city", city, "count", len(transactions))
		params, err := s.buildTransactionParams(transactions, neighborhoodIDs)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to build transaction params", "city", city, "err", err)
			return err
		}
		s.logger.DebugContext(ctx, "bulk upserting transactions", "city", city, "count", len(transactions))
		rowsAffected, err := s.queries.UpsertHintatiedotTransactionsBulk(ctx, *params)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to bulk upsert transactions", "city", city, "count", len(transactions), "err", err)
			return fmt.Errorf("bulk upsert transactions for %q: %w", city, err)
		}
		s.logger.InfoContext(ctx, "transactions upserted", "city", city, "rows_affected", rowsAffected)
	}

	s.logger.InfoContext(ctx, "city sync complete",
		"city", city,
		"transactions", len(transactions),
		"neighborhoods", len(neighborhoodIDs),
		"postal_codes", len(postalCodeIDs),
	)
	return nil
}

func parseElevator(val string) bool {
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "on":
		return true
	case "ei":
		return false
	default:
		return false
	}
}

func (s *SyncService) buildTransactionParams(transactions []*TransactionEntity, neighborhoodIDs map[string]pgtype.UUID) (*db.UpsertHintatiedotTransactionsBulkParams, error) {
	count := len(transactions)
	params := db.UpsertHintatiedotTransactionsBulkParams{
		Descriptions:         make([]string, count),
		Types:                make([]string, count),
		Areas:                make([]float64, count),
		Prices:               make([]int32, count),
		PricePerSquareMeters: make([]int32, count),
		BuildYears:           make([]int32, count),
		Floors:               make([]string, count),
		Elevators:            make([]bool, count),
		Conditions:           make([]string, count),
		Plots:                make([]string, count),
		EnergyClasses:        make([]string, count),
		Categories:           make([]string, count),
		NeighborhoodIds:      make([]pgtype.UUID, count),
	}
	for i, tx := range transactions {
		key := util.NormalizeString(tx.Neighborhood)
		neighborhoodID, ok := neighborhoodIDs[key]
		if !ok {
			s.logger.Error("neighborhood missing ID during transaction build",
				"neighborhood", tx.Neighborhood,
				"normalized_key", key,
				"transaction_index", i,
				"available_neighborhoods", len(neighborhoodIDs),
			)
			return nil, fmt.Errorf("neighborhood %q missing ID during transaction build", tx.Neighborhood)
		}
		params.Descriptions[i] = tx.Description
		params.Types[i] = strings.TrimSpace(tx.Type)
		params.Areas[i] = tx.Area
		params.Prices[i] = int32(tx.Price)
		params.PricePerSquareMeters[i] = int32(tx.PricePerSquareMeter)
		params.BuildYears[i] = int32(tx.BuildYear)
		params.Floors[i] = strings.TrimSpace(tx.Floor)
		params.Elevators[i] = parseElevator(tx.Elevator)
		params.Conditions[i] = strings.TrimSpace(tx.Condition)
		params.Plots[i] = strings.TrimSpace(tx.Plot)
		params.EnergyClasses[i] = strings.TrimSpace(tx.EnergyClass)
		params.Categories[i] = strings.TrimSpace(tx.Category)
		params.NeighborhoodIds[i] = neighborhoodID
	}

	if err := util.ValidateEqualLengths("transaction bulk params",
		len(params.Descriptions),
		len(params.Types),
		len(params.Areas),
		len(params.Prices),
		len(params.PricePerSquareMeters),
		len(params.BuildYears),
		len(params.Floors),
		len(params.Elevators),
		len(params.Conditions),
		len(params.Plots),
		len(params.EnergyClasses),
		len(params.Categories),
		len(params.NeighborhoodIds),
	); err != nil {
		return nil, err
	}

	return &params, nil
}
