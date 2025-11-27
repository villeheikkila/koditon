package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type syncHintatiedotCityInput struct {
	Body SyncHintatiedotRequest
}

type syncHintatiedotCityOutput struct {
	Body SyncHintatiedotResponse
}

func (s *Server) syncHintatiedotCityHandler(ctx context.Context, input *syncHintatiedotCityInput) (*syncHintatiedotCityOutput, error) {
	city := strings.TrimSpace(input.Body.City)
	if city == "" {
		s.logger.WarnContext(ctx, "sync city request missing city name")
		return nil, huma.Error400BadRequest("City is required")
	}
	s.logger.InfoContext(ctx, "initiating city sync", "city", city)
	if err := s.hintatiedotSync.SyncCity(ctx, city); err != nil {
		s.logger.ErrorContext(ctx, "city sync failed", "city", city, "err", err)
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Failed to sync city: %v", err))
	}
	s.logger.InfoContext(ctx, "city sync succeeded", "city", city)
	return &syncHintatiedotCityOutput{
		Body: SyncHintatiedotResponse{
			Status: "ok",
			City:   city,
		},
	}, nil
}
