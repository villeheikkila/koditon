package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SignInAnonymousInput struct {
	Body      struct{}
	DeviceID  string `header:"X-Device-ID" required:"true" format:"uuid" doc:"Device identifier for session tracking"`
	UserAgent string `header:"User-Agent"`
}

type SignInAnonymousOutput struct {
	Body AuthTokensResponse
}

func (h *Handlers) SignInAnonymous(ctx context.Context, input *SignInAnonymousInput) (*SignInAnonymousOutput, error) {
	parsed, err := uuid.Parse(input.DeviceID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid device ID format")
	}
	deviceID := &pgtype.UUID{Bytes: parsed, Valid: true}
	resp, err := h.service.SignInAnonymous(ctx, SignInAnonymousRequest{
		DeviceID:  deviceID,
		UserAgent: input.UserAgent,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("authentication failed", err)
	}
	return &SignInAnonymousOutput{
		Body: AuthTokensResponse{
			AccessToken:           resp.Tokens.AccessToken,
			AccessTokenExpiresAt:  resp.Tokens.AccessTokenExpiresAt,
			RefreshToken:          resp.Tokens.RefreshToken,
			RefreshTokenExpiresAt: resp.Tokens.RefreshTokenExpiresAt,
			UserID:                resp.UserID.String(),
			IsNewUser:             resp.IsNewUser,
		},
	}, nil
}
