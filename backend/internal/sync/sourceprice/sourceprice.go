package sourceprice

import (
	"context"
	"math"

	"github.com/google/uuid"

	"koditon/internal/db"
)

// ParserVersion identifies the current source price normalization contract.
const ParserVersion = int32(1)

// Observation contains one source listing's normalized price state.
type Observation struct {
	AskingPrice       *int64
	DebtFreePrice     *int64
	DebtShareAmount   *int64
	PricePerM2        *float64
	SourcePayloadHash *string
}

// Observe refreshes the current price period or opens a new one when the state changes.
func Observe(ctx context.Context, queries *db.Queries, sourceListingID uuid.UUID, observation Observation) (bool, error) {
	currency := "EUR"
	parserVersion := ParserVersion
	observationMethod := "sync"
	row, err := queries.ObserveSourceListingPrice(ctx, db.ObserveSourceListingPriceParams{
		SourceListingID:   &sourceListingID,
		AskingPrice:       observation.AskingPrice,
		DebtFreePrice:     observation.DebtFreePrice,
		DebtShareAmount:   observation.DebtShareAmount,
		PricePerM2:        observation.PricePerM2,
		Currency:          &currency,
		SourcePayloadHash: observation.SourcePayloadHash,
		ParserVersion:     &parserVersion,
		ObservationMethod: &observationMethod,
	})
	if err != nil {
		return false, err
	}
	return row.PriceChanged, nil
}

// RoundedAmount normalizes a non-negative currency amount to whole units.
func RoundedAmount(value *float64) *int64 {
	value = NonNegative(value)
	if value == nil || *value >= float64(math.MaxInt64) {
		return nil
	}
	amount := int64(math.Round(*value))
	return &amount
}

// NonNegative normalizes a finite non-negative floating-point source value.
func NonNegative(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return nil
	}
	return value
}
