package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"koditon-go/internal/db"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// RFC3339Time wraps time.Time to marshal without fractional seconds for iOS compatibility.
type RFC3339Time struct {
	time.Time
}

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Format("2006-01-02T15:04:05Z"))
}

type pricesTransactionsInput struct {
	MunicipalityID string `query:"municipality_id" required:"true"`
	PostalCodeID   string `query:"postal_code_id" required:"true"`
}

type pricesTransaction struct {
	ID                  string      `json:"id"`
	Description         string      `json:"description"`
	Type                string      `json:"type"`
	Area                float64     `json:"area"`
	Price               int32       `json:"price"`
	PricePerSquareMeter int32       `json:"price_per_square_meter"`
	BuildYear           int32       `json:"build_year"`
	Floor               *string     `json:"floor,omitempty"`
	Elevator            bool        `json:"elevator"`
	Condition           *string     `json:"condition,omitempty"`
	Plot                *string     `json:"plot,omitempty"`
	EnergyClass         *string     `json:"energy_class,omitempty"`
	PeriodIdentifier    string      `json:"period_identifier"`
	CreatedAt           RFC3339Time `json:"created_at"`
	UpdatedAt           RFC3339Time `json:"updated_at"`
	Category            string      `json:"category"`
	NeighborhoodID      string      `json:"neighborhood_id"`
	NeighborhoodName    *string     `json:"neighborhood_name,omitempty"`
	PostalCodeID        string      `json:"postal_code_id"`
	PostalCodeCode      string      `json:"postal_code_code"`
	PostalCodeNameFi    string      `json:"postal_code_name_fi"`
	MunicipalityID      string      `json:"municipality_id"`
	MunicipalityNameFi  string      `json:"municipality_name_fi"`
}

type pricesTransactionsOutput struct {
	Body struct {
		Transactions []pricesTransaction `json:"transactions"`
	}
}

func (a *API) pricesTransactionsHandler(ctx context.Context, input *pricesTransactionsInput) (*pricesTransactionsOutput, error) {
	municipalityID, err := parseUUIDParam(input.MunicipalityID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid municipality_id")
	}
	postalCodeID, err := parseUUIDParam(input.PostalCodeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid postal_code_id")
	}
	rows, err := a.pricesQueries.ListTransactionsByPostalSelection(ctx, db.ListTransactionsByPostalSelectionParams{
		MunicipalityID: municipalityID,
		PostalCodeID:   postalCodeID,
	})
	if err != nil {
		return nil, err
	}
	transactions := make([]pricesTransaction, 0, len(rows))
	for _, row := range rows {
		neighborhoodName := row.PricesNeighborhoodName
		transactions = append(transactions, pricesTransaction{
			ID:                  row.PricesTransactionID.String(),
			Description:         row.PricesTransactionDescription,
			Type:                row.PricesTransactionType,
			Area:                row.PricesTransactionArea,
			Price:               row.PricesTransactionPrice,
			PricePerSquareMeter: row.PricesTransactionPricePerSquareMeter,
			BuildYear:           row.PricesTransactionBuildYear,
			Floor:               row.PricesTransactionFloor,
			Elevator:            row.PricesTransactionElevator,
			Condition:           row.PricesTransactionCondition,
			Plot:                row.PricesTransactionPlot,
			EnergyClass:         row.PricesTransactionEnergyClass,
			PeriodIdentifier:    row.PricesTransactionPeriodIdentifier,
			CreatedAt:           RFC3339Time{row.PricesTransactionCreatedAt},
			UpdatedAt:           RFC3339Time{row.PricesTransactionUpdatedAt},
			Category:            row.PricesTransactionCategory,
			NeighborhoodID:      row.PricesNeighborhoodID.String(),
			NeighborhoodName:    &neighborhoodName,
			PostalCodeID:        row.PostalPostalCodeID.String(),
			PostalCodeCode:      row.PostalPostalCodeCode,
			PostalCodeNameFi:    row.PostalPostalCodeNameFi,
			MunicipalityID:      row.PostalMunicipalityID.String(),
			MunicipalityNameFi:  row.PostalMunicipalityNameFi,
		})
	}
	output := &pricesTransactionsOutput{}
	output.Body.Transactions = transactions
	return output, nil
}

func parseUUIDParam(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

type pricesTransactionsFilteredInput struct {
	MunicipalityIDs string  `query:"municipality_ids" doc:"Comma-separated list of municipality UUIDs"`
	PostalCodeIDs   string  `query:"postal_code_ids" doc:"Comma-separated list of postal code UUIDs"`
	Categories      string  `query:"categories" doc:"Comma-separated list of categories (e.g., Kerrostalo,Rivitalo)"`
	Types           string  `query:"types" doc:"Comma-separated list of types (e.g., Yksiö,Kaksio)"`
	MinArea         float64 `query:"min_area" doc:"Minimum area in square meters (0 = no minimum)"`
	MaxArea         float64 `query:"max_area" doc:"Maximum area in square meters (0 = no maximum)"`
	Limit           int32   `query:"limit" doc:"Maximum number of results (default 100)"`
}

func (a *API) pricesTransactionsFilteredHandler(ctx context.Context, input *pricesTransactionsFilteredInput) (*pricesTransactionsOutput, error) {
	params := db.ListTransactionsFilteredParams{}
	if input.MunicipalityIDs != "" {
		ids := strings.Split(input.MunicipalityIDs, ",")
		uuids := make([]uuid.UUID, 0, len(ids))
		for _, id := range ids {
			parsed, err := parseUUIDParam(strings.TrimSpace(id))
			if err != nil {
				return nil, huma.Error400BadRequest("invalid municipality_id: " + id)
			}
			uuids = append(uuids, parsed)
		}
		params.MunicipalityIds = uuids
	}
	if input.PostalCodeIDs != "" {
		ids := strings.Split(input.PostalCodeIDs, ",")
		uuids := make([]uuid.UUID, 0, len(ids))
		for _, id := range ids {
			parsed, err := parseUUIDParam(strings.TrimSpace(id))
			if err != nil {
				return nil, huma.Error400BadRequest("invalid postal_code_id: " + id)
			}
			uuids = append(uuids, parsed)
		}
		params.PostalCodeIds = uuids
	}
	if input.Categories != "" {
		cats := strings.Split(input.Categories, ",")
		trimmed := make([]string, 0, len(cats))
		for _, c := range cats {
			trimmed = append(trimmed, strings.TrimSpace(c))
		}
		params.Categories = trimmed
	}
	if input.Types != "" {
		types := strings.Split(input.Types, ",")
		trimmed := make([]string, 0, len(types))
		for _, t := range types {
			trimmed = append(trimmed, strings.TrimSpace(t))
		}
		params.Types = trimmed
	}
	if input.MinArea > 0 {
		params.MinArea = &input.MinArea
	}
	if input.MaxArea > 0 {
		params.MaxArea = &input.MaxArea
	}
	if input.Limit > 0 {
		params.LimitCount = &input.Limit
	}
	rows, err := a.pricesQueries.ListTransactionsFiltered(ctx, params)
	if err != nil {
		return nil, err
	}
	transactions := make([]pricesTransaction, 0, len(rows))
	for _, row := range rows {
		neighborhoodName := row.PricesNeighborhoodName
		transactions = append(transactions, pricesTransaction{
			ID:                  row.PricesTransactionID.String(),
			Description:         row.PricesTransactionDescription,
			Type:                row.PricesTransactionType,
			Area:                row.PricesTransactionArea,
			Price:               row.PricesTransactionPrice,
			PricePerSquareMeter: row.PricesTransactionPricePerSquareMeter,
			BuildYear:           row.PricesTransactionBuildYear,
			Floor:               row.PricesTransactionFloor,
			Elevator:            row.PricesTransactionElevator,
			Condition:           row.PricesTransactionCondition,
			Plot:                row.PricesTransactionPlot,
			EnergyClass:         row.PricesTransactionEnergyClass,
			PeriodIdentifier:    row.PricesTransactionPeriodIdentifier,
			CreatedAt:           RFC3339Time{row.PricesTransactionCreatedAt},
			UpdatedAt:           RFC3339Time{row.PricesTransactionUpdatedAt},
			Category:            row.PricesTransactionCategory,
			NeighborhoodID:      row.PricesNeighborhoodID.String(),
			NeighborhoodName:    &neighborhoodName,
			PostalCodeID:        row.PostalPostalCodeID.String(),
			PostalCodeCode:      row.PostalPostalCodeCode,
			PostalCodeNameFi:    row.PostalPostalCodeNameFi,
			MunicipalityID:      row.PostalMunicipalityID.String(),
			MunicipalityNameFi:  row.PostalMunicipalityNameFi,
		})
	}
	output := &pricesTransactionsOutput{}
	output.Body.Transactions = transactions
	return output, nil
}
