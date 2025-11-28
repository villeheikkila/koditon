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
	Body []HintatiedotTransaction
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
	response := make([]HintatiedotTransaction, len(transactions))
	for i, row := range transactions {
		response[i] = mapTransactionResponse(row)
	}
	return &listTransactionsOutput{Body: response}, nil
}

func mapTransactionResponse(row db.ListTransactionsByNeighborhoodsRow) HintatiedotTransaction {
	neighborhoodID, _ := uuidFromPg(row.HintatiedotNeighborhoodsID)
	transactionID, _ := uuidFromPg(row.HintatiedotTransactionsID)
	var neighborhoodName string
	if row.HintatiedotNeighborhoodsName.Valid {
		neighborhoodName = row.HintatiedotNeighborhoodsName.String
	}
	return HintatiedotTransaction{
		Id: transactionID,
		Neighborhood: TransactionNeighborhood{
			Id:         neighborhoodID,
			Name:       neighborhoodName,
			PostalCode: stringPtrFromPg(row.HintatiedotPostalCodesCode),
		},
		Description:         row.HintatiedotTransactionsDescription,
		Type:                row.HintatiedotTransactionsType,
		Area:                row.HintatiedotTransactionsArea,
		Price:               row.HintatiedotTransactionsPrice,
		PricePerSquareMeter: row.HintatiedotTransactionsPricePerSquareMeter,
		BuildYear:           row.HintatiedotTransactionsBuildYear,
		Floor:               row.HintatiedotTransactionsFloor.String,
		Elevator:            row.HintatiedotTransactionsElevator,
		Condition:           row.HintatiedotTransactionsCondition.String,
		Plot:                row.HintatiedotTransactionsPlot.String,
		EnergyClass:         stringPtrFromPg(row.HintatiedotTransactionsEnergyClass),
		FirstSeenAt:         timeFromPg(row.HintatiedotTransactionsCreatedAt),
		LastSeenAt:          timeFromPg(row.HintatiedotTransactionsUpdatedAt),
		Category:            row.HintatiedotTransactionsCategory,
	}
}
