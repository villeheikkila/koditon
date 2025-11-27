package server

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type HintatiedotCitiesResponse struct {
	Cities []string `json:"cities"`
}

type fetchHintatiedotCitiesOutput struct {
	Body HintatiedotCitiesResponse
}

func (s *Server) fetchHintatiedotCitiesHandler(ctx context.Context, _ *struct{}) (*fetchHintatiedotCitiesOutput, error) {
	cities, err := s.hintatiedotAPI.FetchCities(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "fetch cities from hintatiedot", "err", err)
		return nil, huma.Error500InternalServerError("Failed to fetch cities")
	}
	return &fetchHintatiedotCitiesOutput{
		Body: HintatiedotCitiesResponse{
			Cities: cities,
		},
	}, nil
}
