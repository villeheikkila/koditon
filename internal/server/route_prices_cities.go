package server

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type PricesCitiesResponse struct {
	Cities []string `json:"cities"`
}

type fetchPricesCitiesOutput struct {
	Body PricesCitiesResponse
}

func (s *Server) fetchPricesCitiesHandler(ctx context.Context, _ *struct{}) (*fetchPricesCitiesOutput, error) {
	cities, err := s.pricesAPI.FetchCities(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "fetch cities from prices", "err", err)
		return nil, huma.Error500InternalServerError("Failed to fetch cities")
	}
	return &fetchPricesCitiesOutput{
		Body: PricesCitiesResponse{
			Cities: cities,
		},
	}, nil
}
