package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type SignInWithAppleInput struct {
	Body struct {
		AuthorizationCode string `json:"authorization_code" required:"true" doc:"Authorization code from Apple Sign In"`
		Nonce             string `json:"nonce,omitempty" doc:"Nonce used in the Apple Sign In request"`
	}
	DeviceID  string `header:"X-Device-ID" required:"true" format:"uuid" doc:"Device identifier for session tracking"`
	UserAgent string `header:"User-Agent"`
}

type SignInWithAppleOutput struct {
	Body AuthTokensResponse
}

func (h *Handlers) SignInWithApple(ctx context.Context, input *SignInWithAppleInput) (*SignInWithAppleOutput, error) {
	parsed, err := uuid.Parse(input.DeviceID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid device ID format", err)
	}
	deviceID := &parsed
	resp, err := h.service.SignInWithApple(ctx, SignInWithAppleRequest{
		AuthorizationCode: input.Body.AuthorizationCode,
		Nonce:             input.Body.Nonce,
		DeviceID:          deviceID,
		UserAgent:         input.UserAgent,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("authentication failed", err)
	}
	return &SignInWithAppleOutput{
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
