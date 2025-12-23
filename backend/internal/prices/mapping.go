package prices

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/prices/client"
	"koditon-go/internal/prices/db"
	"koditon-go/internal/util"
)

func mapUpsertCityParams(name string) string {
	return name
}

func mapUpsertPostalCodesBulkParams(codes []string, cityID pgtype.UUID) *db.UpsertPricesPostalCodesBulkParams {
	return &db.UpsertPricesPostalCodesBulkParams{
		Codes:  codes,
		CityID: cityID,
	}
}

func mapUpsertNeighborhoodsBulkParams(names []string, cityID pgtype.UUID) *db.UpsertPricesNeighborhoodsBulkParams {
	return &db.UpsertPricesNeighborhoodsBulkParams{
		Names:  names,
		CityID: cityID,
	}
}

func mapUpsertTransactionsBulkParams(transactions []*client.TransactionEntity, neighborhoodIDs map[string]pgtype.UUID, periodIdentifier string) (*db.UpsertPricesTransactionsBulkParams, error) {
	count := len(transactions)
	params := &db.UpsertPricesTransactionsBulkParams{
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
		PeriodIdentifiers:    make([]string, count),
		NeighborhoodIds:      make([]pgtype.UUID, count),
	}
	for i, tx := range transactions {
		key := util.NormalizeString(tx.Neighborhood)
		neighborhoodID, ok := neighborhoodIDs[key]
		if !ok {
			return nil, &neighborhoodNotFoundError{neighborhood: tx.Neighborhood}
		}
		elevator, err := parseElevator(tx.Elevator)
		if err != nil {
			return nil, err
		}
		params.Descriptions[i] = tx.Description
		params.Types[i] = strings.TrimSpace(tx.Type)
		params.Areas[i] = tx.Area
		params.Prices[i] = int32(tx.Price)
		params.PricePerSquareMeters[i] = int32(tx.PricePerSquareMeter)
		params.BuildYears[i] = int32(tx.BuildYear)
		params.Floors[i] = strings.TrimSpace(tx.Floor)
		params.Elevators[i] = elevator
		params.Conditions[i] = strings.TrimSpace(tx.Condition)
		params.Plots[i] = strings.TrimSpace(tx.Plot)
		params.EnergyClasses[i] = strings.TrimSpace(tx.EnergyClass)
		params.Categories[i] = strings.TrimSpace(tx.Category)
		params.PeriodIdentifiers[i] = periodIdentifier
		params.NeighborhoodIds[i] = neighborhoodID
	}
	return params, nil
}

type neighborhoodNotFoundError struct {
	neighborhood string
}

func (e *neighborhoodNotFoundError) Error() string {
	return "neighborhood not found: " + e.neighborhood
}
