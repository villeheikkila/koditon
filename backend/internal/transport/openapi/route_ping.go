package api

import (
	"context"

	"koditon-go/internal/platform/logging"
)

type PingRequest struct {
	Message string `json:"message"`
}

type PingResponse struct {
	Echo string `json:"echo"`
}

type pingInput struct {
	Body PingRequest
}

type pingOutput struct {
	Body PingResponse
}

func (a *API) pingHandler(ctx context.Context, input *pingInput) (*pingOutput, error) {
	logging.With(a.logger, logging.Op("api.ping")).InfoContext(ctx, "ping received", "message_length", len(input.Body.Message))
	return &pingOutput{
		Body: PingResponse{Echo: input.Body.Message},
	}, nil
}
