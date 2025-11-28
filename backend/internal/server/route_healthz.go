package server

import (
	"context"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type healthOutput struct {
	Body HealthResponse
}

func (s *Server) healthHandler(ctx context.Context, _ *struct{}) (*healthOutput, error) {
	return &healthOutput{
		Body: HealthResponse{Status: "ok"},
	}, nil
}
