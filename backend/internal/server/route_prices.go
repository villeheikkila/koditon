package server

import (
	"context"
	"encoding/json"
	"time"

	pricesdb "koditon-go/internal/prices/db"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// RFC3339Time wraps time.Time to marshal without fractional seconds for iOS compatibility
type RFC3339Time struct {
	time.Time
}

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	// Format as RFC3339 without fractional seconds
	formatted := t.UTC().Format("2006-01-02T15:04:05Z")
	return json.Marshal(formatted)
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

func (s *Server) pricesTransactionsHandler(ctx context.Context, input *pricesTransactionsInput) (*pricesTransactionsOutput, error) {
	municipalityID, err := parseUUIDParam(input.MunicipalityID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid municipality_id")
	}
	postalCodeID, err := parseUUIDParam(input.PostalCodeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid postal_code_id")
	}
	rows, err := s.pricesQueries.ListTransactionsByPostalSelection(ctx, &pricesdb.ListTransactionsByPostalSelectionParams{
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
			ID:                  formatUUID(row.PricesTransactionID),
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
			NeighborhoodID:      formatUUID(row.PricesNeighborhoodID),
			NeighborhoodName:    &neighborhoodName,
			PostalCodeID:        formatUUID(row.PostalPostalCodeID),
			PostalCodeCode:      row.PostalPostalCodeCode,
			PostalCodeNameFi:    row.PostalPostalCodeNameFi,
			MunicipalityID:      formatUUID(row.PostalMunicipalityID),
			MunicipalityNameFi:  row.PostalMunicipalityNameFi,
		})
	}
	output := &pricesTransactionsOutput{}
	output.Body.Transactions = transactions
	return output, nil
}

func parseUUIDParam(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
