package server

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type fetchTransactionsOutput struct {
	Body struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
}

func (s *Server) fetchTransactionsHandler(ctx context.Context, _ *struct{}) (*fetchTransactionsOutput, error) {
	transactions, err := s.hintatiedotAPI.GetAllTransactions(ctx, "Helsinki")
	if err != nil {
		s.logger.ErrorContext(ctx, "fetch all transactions from hintatiedot", "err", err)
		return nil, huma.Error500InternalServerError("Failed to fetch transactions")
	}

	s.logger.InfoContext(ctx, "list of transactions fetched", "count", len(transactions))

	return &fetchTransactionsOutput{
		Body: struct {
			Status string `json:"status"`
			Count  int    `json:"count"`
		}{
			Status: "ok",
			Count:  len(transactions),
		},
	}, nil
}
