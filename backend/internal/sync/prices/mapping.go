package prices

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	client "koditon/internal/clients/prices"
	"koditon/internal/db"
	"koditon/internal/platform/util"
)

func mapUpsertCityParams(name string) string {
	return util.TrimUnicodeSpace(name)
}

func mapUpsertPostalCodesBulkParams(codes []string, cityID uuid.UUID) db.UpsertPricesPostalCodesBulkParams {
	trimmed := make([]string, len(codes))
	for i, code := range codes {
		trimmed[i] = util.TrimUnicodeSpace(code)
	}
	return db.UpsertPricesPostalCodesBulkParams{
		Codes:  trimmed,
		CityID: cityID,
	}
}

func mapUpsertNeighborhoodsBulkParams(names []string, cityID uuid.UUID) db.UpsertPricesNeighborhoodsBulkParams {
	trimmed := make([]string, len(names))
	for i, name := range names {
		trimmed[i] = util.TrimUnicodeSpace(name)
	}
	return db.UpsertPricesNeighborhoodsBulkParams{
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

func mapUpsertTransactionsBulkParams(transactions []*client.TransactionEntity, neighborhoodIDs map[string]uuid.UUID, periodIdentifier string) (*db.UpsertPricesTransactionsBulkParams, error) {
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
		if _, err := parseConditionMatchCode(condition); err != nil {
			return nil, err
		}
		plot := emptyToNull(tx.Plot)
		plotOwned, err := parsePlotOwned(plot)
		if err != nil {
			return nil, err
		}
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
		params.PlotOwneds = append(params.PlotOwneds, plotOwned != nil && *plotOwned)
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

func parseConditionMatchCode(value string) (string, error) {
	key := plotAliasKey(value)
	switch key {
	case "":
		return "", nil
	case "good", "hyvä", "hyva":
		return "good", nil
	case "satisfactory", "tyyd", "tyydyttävä", "tyydyttava":
		return "satisfactory", nil
	case "tolerable", "poor", "bad", "huono", "välttävä", "valttava":
		return "poor", nil
	case "unclassified", "not_known", "not_shown":
		return "unknown", nil
	default:
		return "", fmt.Errorf("invalid condition value: %q", value)
	}
}

func parsePlotOwned(value string) (*bool, error) {
	key := plotAliasKey(value)
	switch key {
	case "":
		return nil, nil
	case "1", "oma", "own", "owned", "omistus", "omistettu":
		return boolPtr(true), nil
	case "2", "3", "vuokra", "rent", "rented", "rental", "lease", "leased", "vuokralla", "vuokratontti", "optional_rental", "valinnainen_vuokratontti":
		return boolPtr(false), nil
	default:
		return nil, fmt.Errorf("invalid plot ownership value: %q", value)
	}
}

func plotAliasKey(value string) string {
	value = strings.ToLower(util.TrimUnicodeSpace(value))
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == 'å' || r == 'ä' || r == 'ö':
			return r
		default:
			return '_'
		}
	}, value)
	return strings.Trim(value, "_")
}

func boolPtr(value bool) *bool {
	return &value
}
