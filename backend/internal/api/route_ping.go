package api

import (
	"context"
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
	a.logger.InfoContext(ctx, "ping received", "message", input.Body.Message)
	return &pingOutput{
		Body: PingResponse{Echo: input.Body.Message},
	}, nil
}
