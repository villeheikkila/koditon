package prices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/prices/client"
	"koditon-go/internal/prices/db"
	"koditon-go/internal/util"
)

type Service struct {
	client  *client.Client
	queries *db.Queries
	nowFunc func() time.Time
}

func NewService(
	dbtx db.DBTX,
	baseURL string,
) (*Service, error) {
	pricesClient, err := client.NewClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("create prices client: %w", err)
	}
	return &Service{
		client:  pricesClient,
		queries: db.New(dbtx),
		nowFunc: time.Now,
	}, nil
}

func (s *Service) FetchCities(ctx context.Context) ([]string, error) {
	cities, err := s.client.FetchCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cities: %w", err)
	}
	if len(cities) == 0 {
		return []string{}, nil
	}
	return cities, nil
}

func (s *Service) SyncCity(ctx context.Context, cityName string) error {
	cityRow, err := s.queries.UpsertPricesCity(ctx, mapUpsertCityParams(cityName))
	if err != nil {
		return fmt.Errorf("upsert city %q: %w", cityName, err)
	}
	cityID := cityRow.PricesCitiesID
	postalCodes, err := s.client.FetchPostalCodes(ctx, cityName)
	if err != nil {
		return fmt.Errorf("fetch postal codes for %q: %w", cityName, err)
	}
	postalCodes = util.UniqueStrings(postalCodes)
	postalCodeIDs := make(map[string]pgtype.UUID, len(postalCodes))
	if len(postalCodes) > 0 {
		rows, err := s.queries.UpsertPricesPostalCodesBulk(ctx, mapUpsertPostalCodesBulkParams(postalCodes, cityID))
		if err != nil {
			return fmt.Errorf("bulk upsert postal codes for %q: %w", cityName, err)
		}
		for _, row := range rows {
			postalCodeIDs[row.PricesPostalCodesCode] = row.PricesPostalCodesID
		}
	}
	neighborhoods, err := s.client.FetchNeighborhoods(ctx, cityName)
	if err != nil {
		return fmt.Errorf("fetch neighborhoods for %q: %w", cityName, err)
	}
	neighborhoods = util.UniqueStrings(neighborhoods)
	transactions, err := s.client.GetAllTransactions(ctx, cityName)
	if err != nil {
		return fmt.Errorf("fetch transactions for %q: %w", cityName, err)
	}
	transactionNeighborhoods := make(map[string]bool)
	for _, tx := range transactions {
		normalized := strings.TrimSpace(tx.Neighborhood)
		if normalized != "" {
			transactionNeighborhoods[normalized] = true
		}
	}
	for name := range transactionNeighborhoods {
		neighborhoods = append(neighborhoods, name)
	}
	neighborhoods = util.UniqueStrings(neighborhoods)
	neighborhoodIDs := make(map[string]pgtype.UUID, len(neighborhoods))
	if len(neighborhoods) > 0 {
		rows, err := s.queries.UpsertPricesNeighborhoodsBulk(ctx, mapUpsertNeighborhoodsBulkParams(neighborhoods, cityID))
		if err != nil {
			return fmt.Errorf("bulk upsert neighborhoods for %q: %w", cityName, err)
		}
		for _, row := range rows {
			key := util.NormalizeString(row.PricesNeighborhoodsName)
			neighborhoodIDs[key] = row.PricesNeighborhoodsID
		}
	}
	if len(transactions) > 0 {
		periodIdentifier := s.nowFunc().Format("2006-01")
		params, err := mapUpsertTransactionsBulkParams(transactions, neighborhoodIDs, periodIdentifier)
		if err != nil {
			return fmt.Errorf("build transaction params for %q: %w", cityName, err)
		}
		if _, err := s.queries.UpsertPricesTransactionsBulk(ctx, params); err != nil {
			return fmt.Errorf("bulk upsert transactions for %q: %w", cityName, err)
		}
	}
	return nil
}

func parseElevator(val string) (bool, error) {
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "on":
		return true, nil
	case "ei":
		return false, nil
	default:
		return false, fmt.Errorf("invalid elevator value: %q", val)
	}
}
