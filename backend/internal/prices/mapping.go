package prices

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/prices/client"
	"koditon-go/internal/prices/db"
	"koditon-go/internal/util"
)

func mapUpsertCityParams(name string) string {
	return util.TrimUnicodeSpace(name)
}

func mapUpsertPostalCodesBulkParams(codes []string, cityID pgtype.UUID) *db.UpsertPricesPostalCodesBulkParams {
	trimmed := make([]string, len(codes))
	for i, code := range codes {
		trimmed[i] = util.TrimUnicodeSpace(code)
	}
	return &db.UpsertPricesPostalCodesBulkParams{
		Codes:  trimmed,
		CityID: cityID,
	}
}

func mapUpsertNeighborhoodsBulkParams(names []string, cityID pgtype.UUID) *db.UpsertPricesNeighborhoodsBulkParams {
	trimmed := make([]string, len(names))
	for i, name := range names {
		trimmed[i] = util.TrimUnicodeSpace(name)
	}
	return &db.UpsertPricesNeighborhoodsBulkParams{
		Names:  trimmed,
		CityID: cityID,
	}
}

type transactionKey struct {
	neighborhoodID      string
	description         string
	txType              string
	area                float64
	price               int32
	pricePerSquareMeter int32
	buildYear           int32
	floor               string
	elevator            bool
	condition           string
	plot                string
	energyClass         string
	category            string
	periodIdentifier    string
}

func emptyToNull(s string) string {
	if util.TrimUnicodeSpace(s) == "" {
		return ""
	}
	return util.TrimUnicodeSpace(s)
}

func mapUpsertTransactionsBulkParams(transactions []*client.TransactionEntity, neighborhoodIDs map[string]pgtype.UUID, periodIdentifier string) (*db.UpsertPricesTransactionsBulkParams, error) {
	seen := make(map[transactionKey]struct{})
	params := &db.UpsertPricesTransactionsBulkParams{}
	for _, tx := range transactions {
		key := util.NormalizeString(tx.Neighborhood)
		neighborhoodID, ok := neighborhoodIDs[key]
		if !ok {
			return nil, &neighborhoodNotFoundError{neighborhood: tx.Neighborhood}
		}
		elevator, err := parseElevator(tx.Elevator)
		if err != nil {
			return nil, err
		}
		description := util.TrimUnicodeSpace(tx.Description)
		txType := util.TrimUnicodeSpace(tx.Type)
		area := tx.Area
		price := int32(tx.Price)
		pricePerSqm := int32(tx.PricePerSquareMeter)
		buildYear := int32(tx.BuildYear)
		floor := emptyToNull(tx.Floor)
		condition := emptyToNull(tx.Condition)
		plot := emptyToNull(tx.Plot)
		energyClass := emptyToNull(tx.EnergyClass)
		category := util.TrimUnicodeSpace(tx.Category)
		dedupKey := transactionKey{
			neighborhoodID:      fmt.Sprintf("%v", neighborhoodID),
			description:         description,
			txType:              txType,
			area:                area,
			price:               price,
			pricePerSquareMeter: pricePerSqm,
			buildYear:           buildYear,
			floor:               floor,
			elevator:            elevator,
			condition:           condition,
			plot:                plot,
			energyClass:         energyClass,
			category:            category,
			periodIdentifier:    periodIdentifier,
		}
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}
		params.Descriptions = append(params.Descriptions, description)
		params.Types = append(params.Types, txType)
		params.Areas = append(params.Areas, area)
		params.Prices = append(params.Prices, price)
		params.PricePerSquareMeters = append(params.PricePerSquareMeters, pricePerSqm)
		params.BuildYears = append(params.BuildYears, buildYear)
		params.Floors = append(params.Floors, floor)
		params.Elevators = append(params.Elevators, elevator)
		params.Conditions = append(params.Conditions, condition)
		params.Plots = append(params.Plots, plot)
		params.EnergyClasses = append(params.EnergyClasses, energyClass)
		params.Categories = append(params.Categories, category)
		params.PeriodIdentifiers = append(params.PeriodIdentifiers, periodIdentifier)
		params.NeighborhoodIds = append(params.NeighborhoodIds, neighborhoodID)
	}
	return params, nil
}

type neighborhoodNotFoundError struct {
	neighborhood string
}

func (e *neighborhoodNotFoundError) Error() string {
	return "neighborhood not found: " + e.neighborhood
}
