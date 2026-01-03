package auth

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

type RefreshTokensInput struct {
	Body struct {
		RefreshToken string `json:"refresh_token" required:"true" doc:"Refresh token obtained from sign-in"`
	}
}

type RefreshTokensOutput struct {
	Body struct {
		AccessToken           string `json:"access_token" doc:"New JWT access token"`
		AccessTokenExpiresAt  int64  `json:"access_token_expires_at" doc:"Unix timestamp when access token expires"`
		RefreshToken          string `json:"refresh_token" doc:"New JWT refresh token"`
		RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at" doc:"Unix timestamp when refresh token expires"`
		UserID                string `json:"user_id" format:"uuid" doc:"User ID"`
	}
}

func (h *Handlers) RefreshTokens(ctx context.Context, input *RefreshTokensInput) (*RefreshTokensOutput, error) {
	resp, err := h.service.RefreshTokens(ctx, RefreshTokensRequest{
		RefreshToken: input.Body.RefreshToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenExpired):
			return nil, huma.Error401Unauthorized("refresh token expired")
		case errors.Is(err, ErrTokenReuse):
			return nil, huma.Error401Unauthorized("token reuse detected")
		case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrSessionRevoked):
			return nil, huma.Error401Unauthorized("session invalid")
		case errors.Is(err, ErrInvalidToken):
			return nil, huma.Error401Unauthorized("invalid refresh token")
		default:
			return nil, huma.Error500InternalServerError("refresh failed", err)
		}
	}
	return &RefreshTokensOutput{
		Body: struct {
			AccessToken           string `json:"access_token" doc:"New JWT access token"`
			AccessTokenExpiresAt  int64  `json:"access_token_expires_at" doc:"Unix timestamp when access token expires"`
			RefreshToken          string `json:"refresh_token" doc:"New JWT refresh token"`
			RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at" doc:"Unix timestamp when refresh token expires"`
			UserID                string `json:"user_id" format:"uuid" doc:"User ID"`
		}{
			AccessToken:           resp.Tokens.AccessToken,
			AccessTokenExpiresAt:  resp.Tokens.AccessTokenExpiresAt,
			RefreshToken:          resp.Tokens.RefreshToken,
			RefreshTokenExpiresAt: resp.Tokens.RefreshTokenExpiresAt,
			UserID:                resp.UserID.String(),
		},
	}, nil
}
