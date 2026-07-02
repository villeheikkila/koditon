package prices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"koditon/internal/db"
)

type Service struct {
	queries *db.Queries
}

func NewService(dbtx db.DBTX) *Service {
	return &Service{queries: db.New(dbtx)}
}

type SearchTransactionsRow struct {
	City             string
	Municipality     string
	PostalCode       string
	PostalArea       string
	Neighborhood     string
	Description      string
	Type             string
	Category         string
	Area             float64
	Price            int32
	PricePerSqm      int32
	BuildYear        int32
	Floor            string
	Elevator         bool
	Condition        string
	Plot             string
	EnergyClass      string
	PeriodIdentifier string
	CreatedAt        time.Time
}

func (s *Service) ListCities(ctx context.Context) ([]string, error) {
	rows, err := s.queries.ListPricesCities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list prices cities: %w", err)
	}
	cities := make([]string, 0, len(rows))
	for _, row := range rows {
		cities = append(cities, row.PricesCityName)
	}
	return cities, nil
}

func (s *Service) SearchTransactionsByCityAndAddress(ctx context.Context, cityName, searchTerm string, limit int32) ([]SearchTransactionsRow, error) {
	cityName = strings.TrimSpace(cityName)
	searchTerm = strings.TrimSpace(searchTerm)
	if cityName == "" {
		return nil, fmt.Errorf("city name is required")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.queries.SearchTransactionsByCityAndAddress(ctx, db.SearchTransactionsByCityAndAddressParams{
		CityName:   &cityName,
		SearchTerm: &searchTerm,
		LimitCount: &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search transactions by city and address: %w", err)
	}
	result := make([]SearchTransactionsRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, SearchTransactionsRow{
			City:             row.PricesCityName,
			Municipality:     strings.TrimSpace(row.MunicipalityNameFi),
			PostalCode:       ptrString(row.PostalCode),
			PostalArea:       strings.TrimSpace(row.PostalAreaNameFi),
			Neighborhood:     row.PricesNeighborhoodName,
			Description:      row.PricesTransactionDescription,
			Type:             row.PricesTransactionType,
			Category:         row.PricesTransactionCategory,
			Area:             row.PricesTransactionArea,
			Price:            row.PricesTransactionPrice,
			PricePerSqm:      row.PricesTransactionPricePerSquareMeter,
			BuildYear:        row.PricesTransactionBuildYear,
			Floor:            ptrString(row.PricesTransactionFloor),
			Elevator:         row.PricesTransactionElevator,
			Condition:        ptrString(row.PricesTransactionCondition),
			Plot:             ptrString(row.PricesTransactionPlot),
			EnergyClass:      ptrString(row.PricesTransactionEnergyClass),
			PeriodIdentifier: row.PricesTransactionPeriodIdentifier,
			CreatedAt:        row.PricesTransactionCreatedAt,
		})
	}
	return result, nil
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
