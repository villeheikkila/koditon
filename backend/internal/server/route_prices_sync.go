package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type syncPricesCityInput struct {
	Body SyncPricesRequest
}

type syncPricesCityOutput struct {
	Body SyncPricesResponse
}

func (s *Server) syncPricesCityHandler(ctx context.Context, input *syncPricesCityInput) (*syncPricesCityOutput, error) {
	city := strings.TrimSpace(input.Body.City)
	if city == "" {
		s.logger.WarnContext(ctx, "sync city request missing city name")
		return nil, huma.Error400BadRequest("City is required")
	}
	s.logger.InfoContext(ctx, "initiating city sync", "city", city)
	if err := s.pricesSync.SyncCity(ctx, city); err != nil {
		s.logger.ErrorContext(ctx, "city sync failed", "city", city, "err", err)
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Failed to sync city: %v", err))
	}
	s.logger.InfoContext(ctx, "city sync succeeded", "city", city)
	return &syncPricesCityOutput{
		Body: SyncPricesResponse{
			Status: "ok",
			City:   city,
		},
	}, nil
}
