package server

import (
	"context"
	"koditon-go/internal/db"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type listTransactionsInput struct {
	Neighborhoods []uuid.UUID `query:"neighborhoods"`
}

type listTransactionsOutput struct {
	Body []PricesTransaction
}

func (s *Server) listTransactionsHandler(ctx context.Context, input *listTransactionsInput) (*listTransactionsOutput, error) {
	if len(input.Neighborhoods) == 0 {
		return nil, huma.Error400BadRequest("At least one neighborhood UUID is required")
	}
	neighborhoodIDs := make([]pgtype.UUID, len(input.Neighborhoods))
	for i, neighborhoodID := range input.Neighborhoods {
		neighborhoodIDs[i] = pgtype.UUID{
			Bytes: neighborhoodID,
			Valid: true,
		}
	}
	transactions, err := s.db.ListTransactionsByNeighborhoods(ctx, neighborhoodIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "list transactions by neighborhoods", "err", err)
		return nil, huma.Error500InternalServerError("Internal server error")
	}
	response := make([]PricesTransaction, len(transactions))
	for i, row := range transactions {
		response[i] = mapTransactionResponse(row)
	}
	return &listTransactionsOutput{Body: response}, nil
}

func mapTransactionResponse(row db.ListTransactionsByNeighborhoodsRow) PricesTransaction {
	neighborhoodID, _ := uuidFromPg(row.PricesNeighborhoodsID)
	transactionID, _ := uuidFromPg(row.PricesTransactionsID)
	var neighborhoodName string
	if row.PricesNeighborhoodsName.Valid {
		neighborhoodName = row.PricesNeighborhoodsName.String
	}
	return PricesTransaction{
		Id: transactionID,
		Neighborhood: TransactionNeighborhood{
			Id:         neighborhoodID,
			Name:       neighborhoodName,
			PostalCode: stringPtrFromPg(row.PricesPostalCodesCode),
		},
		Description:         row.PricesTransactionsDescription,
		Type:                row.PricesTransactionsType,
		Area:                row.PricesTransactionsArea,
		Price:               row.PricesTransactionsPrice,
		PricePerSquareMeter: row.PricesTransactionsPricePerSquareMeter,
		BuildYear:           row.PricesTransactionsBuildYear,
		Floor:               row.PricesTransactionsFloor.String,
		Elevator:            row.PricesTransactionsElevator,
		Condition:           row.PricesTransactionsCondition.String,
		Plot:                row.PricesTransactionsPlot.String,
		EnergyClass:         stringPtrFromPg(row.PricesTransactionsEnergyClass),
		FirstSeenAt:         timeFromPg(row.PricesTransactionsCreatedAt),
		LastSeenAt:          timeFromPg(row.PricesTransactionsUpdatedAt),
		Category:            row.PricesTransactionsCategory,
	}
}
