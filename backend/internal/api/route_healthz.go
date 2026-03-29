package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type healthOutput struct {
	Body HealthResponse
}

func (a *API) healthHandler(ctx context.Context, _ *struct{}) (*healthOutput, error) {
	if err := a.pool.Ping(ctx); err != nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}
	return &healthOutput{
		Body: HealthResponse{Status: "ok"},
	}, nil
}
