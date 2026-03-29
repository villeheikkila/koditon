package api

import (
	"context"
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"koditon-go/internal/auth"
)

// --- passkey authenticate options ---

type passkeyAuthOptionsOutput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id"`
		Options     json.RawMessage `json:"options"`
	}
}

func (a *API) passkeyAuthOptionsHandler(ctx context.Context, _ *struct{}) (*passkeyAuthOptionsOutput, error) {
	resp, err := a.authService.BeginPasskeyAuthentication(ctx)
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("passkey authentication unavailable")
	}
	out := &passkeyAuthOptionsOutput{}
	out.Body.ChallengeID = resp.ChallengeID.String()
	out.Body.Options = resp.Options
	return out, nil
}

// --- passkey authenticate ---

type passkeyAuthInput struct {
	Body struct {
		ChallengeID    string `json:"challenge_id" required:"true"`
		CredentialJSON string `json:"credential_json" required:"true"`
	}
	RawDeviceID string `header:"X-Device-ID"`
}

type passkeyAuthOutput struct {
	Body struct {
		AccessToken           string `json:"access_token"`
		AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
		UserID                string `json:"user_id"`
	}
}

func (a *API) passkeyAuthHandler(ctx context.Context, input *passkeyAuthInput) (*passkeyAuthOutput, error) {
	challengeID, err := uuid.Parse(input.Body.ChallengeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid challenge_id")
	}
	var deviceID uuid.UUID
	if input.RawDeviceID != "" {
		deviceID, _ = uuid.Parse(input.RawDeviceID)
	}
	finishResp, err := a.authService.FinishPasskeyAuthentication(ctx, auth.FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  json.RawMessage(input.Body.CredentialJSON),
		DeviceID:    deviceID,
	})
	if err != nil {
		switch err {
		case auth.ErrPasskeyNotFound:
			return nil, huma.Error400BadRequest("passkey not found")
		case auth.ErrPasskeyChallenge:
			return nil, huma.Error400BadRequest("passkey challenge is invalid or expired")
		default:
			a.logger.ErrorContext(ctx, "passkey authentication failed", "error", err)
			return nil, huma.Error400BadRequest("passkey authentication failed")
		}
	}
	tokens, err := a.authService.IssueOAuthTokensForUser(ctx, auth.OAuthIssueTokensForUserRequest{
		ClientID:  "koditon-web",
		UserID:    finishResp.UserID,
		Scopes:    []string{auth.ScopeCoreRead},
		SessionID: finishResp.SessionID,
		Audience:  auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "token issuance failed after passkey auth", "error", err)
		return nil, huma.Error500InternalServerError("failed to issue tokens")
	}
	out := &passkeyAuthOutput{}
	out.Body.AccessToken = tokens.AccessToken
	out.Body.AccessTokenExpiresAt = tokens.AccessExpiry.Unix()
	out.Body.RefreshToken = tokens.RefreshToken
	out.Body.RefreshTokenExpiresAt = tokens.RefreshExpiry.Unix()
	out.Body.UserID = finishResp.UserID.String()
	return out, nil
}

// --- passkey register options ---

type passkeyRegisterOptionsOutput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id"`
		Options     json.RawMessage `json:"options"`
	}
}

func (a *API) passkeyRegisterOptionsHandler(ctx context.Context, _ *struct{}) (*passkeyRegisterOptionsOutput, error) {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	deviceID := uuid.Nil
	resp, err := a.authService.BeginPasskeyRegistration(ctx, auth.BeginPasskeyRegistrationRequest{
		UserID:   claims.UserID,
		DeviceID: deviceID,
	})
	if err != nil {
		return nil, huma.Error503ServiceUnavailable("passkey registration unavailable")
	}
	out := &passkeyRegisterOptionsOutput{}
	out.Body.ChallengeID = resp.ChallengeID.String()
	out.Body.Options = resp.Options
	return out, nil
}

// --- passkey register finish ---

type passkeyRegisterFinishInput struct {
	Body struct {
		ChallengeID    string `json:"challenge_id" required:"true"`
		CredentialJSON string `json:"credential_json" required:"true"`
	}
}

type passkeyRegisterFinishOutput struct {
	Body struct {
		CredentialID string `json:"credential_id"`
	}
}

func (a *API) passkeyRegisterFinishHandler(ctx context.Context, input *passkeyRegisterFinishInput) (*passkeyRegisterFinishOutput, error) {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	challengeID, err := uuid.Parse(input.Body.ChallengeID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid challenge_id")
	}
	resp, err := a.authService.FinishPasskeyRegistration(ctx, auth.FinishPasskeyRegistrationRequest{
		UserID:      claims.UserID,
		ChallengeID: challengeID,
		Credential:  json.RawMessage(input.Body.CredentialJSON),
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "passkey registration failed", "error", err)
		return nil, huma.Error400BadRequest("passkey registration failed")
	}
	out := &passkeyRegisterFinishOutput{}
	out.Body.CredentialID = resp.CredentialID
	return out, nil
}

// --- apple web sign in ---

type appleWebAuthInput struct {
	Body struct {
		Code string `json:"code" required:"true"`
	}
	RawDeviceID string `header:"X-Device-ID"`
}

type appleWebAuthOutput struct {
	Body struct {
		AccessToken           string `json:"access_token"`
		AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
		UserID                string `json:"user_id"`
	}
}

func (a *API) appleWebAuthHandler(ctx context.Context, input *appleWebAuthInput) (*appleWebAuthOutput, error) {
	var deviceID uuid.UUID
	if input.RawDeviceID != "" {
		deviceID, _ = uuid.Parse(input.RawDeviceID)
	}
	siwaResp, err := a.authService.SignInWithAppleWeb(ctx, auth.SignInWithAppleWebRequest{
		AuthorizationCode: input.Body.Code,
		DeviceID:          deviceID,
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "apple web sign in failed", "error", err)
		return nil, huma.Error400BadRequest("sign in failed")
	}
	tokens, err := a.authService.IssueOAuthTokensForUser(ctx, auth.OAuthIssueTokensForUserRequest{
		ClientID:  "koditon-web",
		UserID:    siwaResp.UserID,
		Scopes:    []string{auth.ScopeCoreRead},
		SessionID: siwaResp.SessionID,
		Audience:  auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "token issuance failed after apple web auth", "error", err)
		return nil, huma.Error500InternalServerError("failed to issue tokens")
	}
	out := &appleWebAuthOutput{}
	out.Body.AccessToken = tokens.AccessToken
	out.Body.AccessTokenExpiresAt = tokens.AccessExpiry.Unix()
	out.Body.RefreshToken = tokens.RefreshToken
	out.Body.RefreshTokenExpiresAt = tokens.RefreshExpiry.Unix()
	out.Body.UserID = siwaResp.UserID.String()
	return out, nil
}
