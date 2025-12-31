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

type SyncNeighborhoodPostalCodesProgress struct {
	City       string
	PostalCode string
	Page       int
	Updated    int
}

func (s *Service) SyncNeighborhoodPostalCodes(ctx context.Context, progressFn func(SyncNeighborhoodPostalCodesProgress)) error {
	cities, err := s.queries.ListPricesCities(ctx)
	if err != nil {
		return fmt.Errorf("list cities: %w", err)
	}
	for _, city := range cities {
		postalCodes, err := s.queries.ListPricesPostalCodesByCity(ctx, city.PricesCitiesID)
		if err != nil {
			return fmt.Errorf("list postal codes for city %q: %w", city.PricesCitiesName, err)
		}
		for _, pc := range postalCodes {
			updated, err := s.syncNeighborhoodsForPostalCode(ctx, city, pc, progressFn)
			if err != nil {
				return fmt.Errorf("sync neighborhoods for postal code %q in %q: %w", pc.PricesPostalCodesCode, city.PricesCitiesName, err)
			}
			if progressFn != nil {
				progressFn(SyncNeighborhoodPostalCodesProgress{
					City:       city.PricesCitiesName,
					PostalCode: pc.PricesPostalCodesCode,
					Updated:    updated,
				})
			}
		}
	}
	return nil
}

func (s *Service) syncNeighborhoodsForPostalCode(
	ctx context.Context,
	city db.PricesCity,
	pc db.PricesPostalCode,
	progressFn func(SyncNeighborhoodPostalCodesProgress),
) (int, error) {
	neighborhoodNames := make(map[string]struct{})
	nextPage := new(int)
	*nextPage = 0
	page := 0
	for nextPage != nil {
		page = *nextPage
		if progressFn != nil {
			progressFn(SyncNeighborhoodPostalCodesProgress{
				City:       city.PricesCitiesName,
				PostalCode: pc.PricesPostalCodesCode,
				Page:       page,
			})
		}
		response, err := s.client.GetTransactionsForPage(ctx, &client.ApartmentSearchParams{
			City:        city.PricesCitiesName,
			PostalCodes: []string{pc.PricesPostalCodesCode},
			RenderType:  "renderTypeTable",
		}, page)
		if err != nil {
			return 0, fmt.Errorf("fetch page %d: %w", page, err)
		}
		for _, tx := range response.Apartments {
			name := util.TrimUnicodeSpace(tx.Neighborhood)
			if name != "" {
				neighborhoodNames[name] = struct{}{}
			}
		}
		nextPage = response.NextPage
	}
	updated := 0
	for name := range neighborhoodNames {
		err := s.queries.UpdateNeighborhoodPostalCode(ctx, &db.UpdateNeighborhoodPostalCodeParams{
			PostalCodeID: pc.PricesPostalCodesID,
			Name:         name,
			CityID:       city.PricesCitiesID,
		})
		if err != nil {
			return updated, fmt.Errorf("update neighborhood %q: %w", name, err)
		}
		updated++
	}
	return updated, nil
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
		normalized := util.TrimUnicodeSpace(tx.Neighborhood)
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
	val = util.TrimUnicodeSpace(val)
	val = strings.ToLower(val)
	switch val {
	case "on":
		return true, nil
	case "ei":
		return false, nil
	default:
		return false, fmt.Errorf("invalid elevator value: %q", val)
	}
}
