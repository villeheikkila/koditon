package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"koditon/internal/domain/auth"
	"koditon/internal/domain/emailauth"
	"koditon/internal/platform/logging"
)

const secureWebRefreshCookieName = "__Host-koditon_refresh"
const devWebRefreshCookieName = "koditon_refresh"
const webOAuthClientID = "koditon-web"

type webAuthHeaders struct {
	Origin  string `header:"Origin"`
	Referer string `header:"Referer"`
}

type authTokenBody struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
	RefreshToken          string `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at,omitempty"`
	UserID                string `json:"user_id"`
}

type authTokenOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      authTokenBody
}

// --- passkey authenticate options ---

type passkeyAuthOptionsOutput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id"`
		Options     json.RawMessage `json:"options"`
	}
}

func (a *API) passkeyAuthOptionsHandler(ctx context.Context, _ *struct{}) (*passkeyAuthOptionsOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.passkey.options"))
	resp, err := a.authService.BeginPasskeyAuthentication(ctx)
	if err != nil {
		if errors.Is(err, auth.ErrPasskeyConfig) {
			return nil, huma.Error503ServiceUnavailable("passkey authentication unavailable")
		}
		logger.ErrorContext(ctx, "begin passkey authentication failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("internal server error")
	}
	out := &passkeyAuthOptionsOutput{}
	out.Body.ChallengeID = resp.ChallengeID.String()
	out.Body.Options = resp.Options
	return out, nil
}

// --- email auth request ---

type emailAuthRequestInput struct {
	Body struct {
		Email string `json:"email" required:"true" format:"email"`
	}
}

type emailAuthRequestOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func (a *API) emailAuthRequestHandler(ctx context.Context, input *emailAuthRequestInput) (*emailAuthRequestOutput, error) {
	if a.emailAuthService == nil {
		return nil, huma.Error503ServiceUnavailable("email authentication unavailable")
	}
	if err := a.emailAuthService.RequestAuthentication(ctx, input.Body.Email); err != nil {
		switch {
		case errors.Is(err, emailauth.ErrInvalidEmail):
			return nil, huma.Error422UnprocessableEntity("email must be valid")
		default:
			logging.With(a.logger, logging.Op("api.auth.email.request")).ErrorContext(ctx, "email auth request failed", "error", err, "outcome", logging.OutcomeError)
			return nil, huma.Error503ServiceUnavailable("email authentication unavailable")
		}
	}
	out := &emailAuthRequestOutput{}
	out.Body.OK = true
	return out, nil
}

// --- email auth confirm ---

type emailAuthConfirmInput struct {
	Body struct {
		Token string `json:"token" required:"true"`
	}
	RawDeviceID string `header:"X-Device-ID"`
	webAuthHeaders
}

type emailAuthConfirmOutput = authTokenOutput

func (a *API) emailAuthConfirmHandler(ctx context.Context, input *emailAuthConfirmInput) (*emailAuthConfirmOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.email.confirm"))
	if a.emailAuthService == nil {
		return nil, huma.Error503ServiceUnavailable("email authentication unavailable")
	}
	if err := a.validateWebAuthOrigin(input.webAuthHeaders); err != nil {
		return nil, err
	}
	var deviceID uuid.UUID
	if input.RawDeviceID != "" {
		deviceID, _ = uuid.Parse(input.RawDeviceID)
	}
	authTicket, err := a.emailAuthService.ConfirmAuthentication(ctx, input.Body.Token)
	if err != nil {
		switch {
		case errors.Is(err, emailauth.ErrInvalidToken), errors.Is(err, emailauth.ErrTokenExpired), errors.Is(err, emailauth.ErrTokenConsumed):
			return nil, huma.Error422UnprocessableEntity("email sign-in link is invalid or expired")
		default:
			logger.ErrorContext(ctx, "email auth confirmation failed", "error", err, "outcome", logging.OutcomeError)
			return nil, huma.Error500InternalServerError("internal server error")
		}
	}
	confirmedEmail, err := a.emailAuthService.ConsumeAuthenticationTicket(ctx, authTicket)
	if err != nil {
		logger.ErrorContext(ctx, "email auth ticket consume failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("internal server error")
	}
	signResp, err := a.authService.SignInWithEmail(ctx, auth.SignInWithEmailRequest{
		ConfirmedEmail: confirmedEmail,
		DeviceID:       deviceID,
	})
	if err != nil {
		logger.ErrorContext(ctx, "email sign in failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error401Unauthorized("sign in failed")
	}
	tokens, err := a.authService.IssueOAuthTokensForUser(ctx, auth.OAuthIssueTokensForUserRequest{
		ClientID:  webOAuthClientID,
		UserID:    signResp.UserID,
		Scopes:    []string{auth.ScopeCoreRead},
		SessionID: signResp.SessionID,
		Audience:  auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		logger.ErrorContext(ctx, "token issuance failed after email auth", "error", err, "user_id", signResp.UserID, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("failed to issue tokens")
	}
	return a.webAuthTokenOutput(tokens), nil
}

// --- passkey authenticate ---

type passkeyAuthInput struct {
	Body struct {
		ChallengeID    string `json:"challenge_id" required:"true"`
		CredentialJSON string `json:"credential_json" required:"true"`
	}
	RawDeviceID string `header:"X-Device-ID"`
	webAuthHeaders
}

type passkeyAuthOutput = authTokenOutput

func (a *API) passkeyAuthHandler(ctx context.Context, input *passkeyAuthInput) (*passkeyAuthOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.passkey.authenticate"))
	if err := a.validateWebAuthOrigin(input.webAuthHeaders); err != nil {
		return nil, err
	}
	challengeID, err := uuid.Parse(input.Body.ChallengeID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("challenge_id must be a valid UUID")
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
		switch {
		case errors.Is(err, auth.ErrPasskeyNotFound):
			return nil, huma.Error404NotFound("passkey not found")
		case errors.Is(err, auth.ErrPasskeyChallenge):
			return nil, huma.Error422UnprocessableEntity("passkey challenge is invalid or expired")
		default:
			logger.ErrorContext(ctx, "passkey authentication failed", "error", err, "challenge_id", challengeID, "outcome", logging.OutcomeError)
			return nil, huma.Error401Unauthorized("authentication failed")
		}
	}
	tokens, err := a.authService.IssueOAuthTokensForUser(ctx, auth.OAuthIssueTokensForUserRequest{
		ClientID:  webOAuthClientID,
		UserID:    finishResp.UserID,
		Scopes:    []string{auth.ScopeCoreRead},
		SessionID: finishResp.SessionID,
		Audience:  auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		logger.ErrorContext(ctx, "token issuance failed after passkey auth", "error", err, "user_id", finishResp.UserID, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("failed to issue tokens")
	}
	return a.webAuthTokenOutput(tokens), nil
}

// --- passkey register options ---

type passkeyRegisterOptionsOutput struct {
	Body struct {
		ChallengeID string          `json:"challenge_id"`
		Options     json.RawMessage `json:"options"`
	}
}

func (a *API) passkeyRegisterOptionsHandler(ctx context.Context, _ *struct{}) (*passkeyRegisterOptionsOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.passkey.register_options"))
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
		if errors.Is(err, auth.ErrPasskeyConfig) {
			return nil, huma.Error503ServiceUnavailable("passkey registration unavailable")
		}
		logger.ErrorContext(ctx, "begin passkey registration failed", "error", err, "user_id", claims.UserID, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("internal server error")
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
	logger := logging.With(a.logger, logging.Op("api.auth.passkey.register_finish"))
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, huma.Error401Unauthorized("authentication required")
	}
	challengeID, err := uuid.Parse(input.Body.ChallengeID)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("challenge_id must be a valid UUID")
	}
	resp, err := a.authService.FinishPasskeyRegistration(ctx, auth.FinishPasskeyRegistrationRequest{
		UserID:      claims.UserID,
		ChallengeID: challengeID,
		Credential:  json.RawMessage(input.Body.CredentialJSON),
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrPasskeyChallenge):
			return nil, huma.Error422UnprocessableEntity("passkey challenge is invalid or expired")
		default:
			logger.ErrorContext(ctx, "passkey registration failed", "error", err, "user_id", claims.UserID, "challenge_id", challengeID, "outcome", logging.OutcomeError)
			return nil, huma.Error422UnprocessableEntity("passkey registration failed")
		}
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
	webAuthHeaders
}

type appleWebAuthOutput = authTokenOutput

func (a *API) appleWebAuthHandler(ctx context.Context, input *appleWebAuthInput) (*appleWebAuthOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.apple_web"))
	if err := a.validateWebAuthOrigin(input.webAuthHeaders); err != nil {
		return nil, err
	}
	var deviceID uuid.UUID
	if input.RawDeviceID != "" {
		deviceID, _ = uuid.Parse(input.RawDeviceID)
	}
	siwaResp, err := a.authService.SignInWithAppleWeb(ctx, auth.SignInWithAppleWebRequest{
		AuthorizationCode: input.Body.Code,
		DeviceID:          deviceID,
	})
	if err != nil {
		logger.ErrorContext(ctx, "apple web sign in failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error401Unauthorized("sign in failed")
	}
	tokens, err := a.authService.IssueOAuthTokensForUser(ctx, auth.OAuthIssueTokensForUserRequest{
		ClientID:  webOAuthClientID,
		UserID:    siwaResp.UserID,
		Scopes:    []string{auth.ScopeCoreRead},
		SessionID: siwaResp.SessionID,
		Audience:  auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		logger.ErrorContext(ctx, "token issuance failed after apple web auth", "error", err, "user_id", siwaResp.UserID, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("failed to issue tokens")
	}
	return a.webAuthTokenOutput(tokens), nil
}

type webSessionRefreshInput struct {
	SecureRefreshToken string `cookie:"__Host-koditon_refresh"`
	DevRefreshToken    string `cookie:"koditon_refresh"`
	webAuthHeaders
}

type webSessionRefreshOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      struct {
		AccessToken          string `json:"access_token"`
		AccessTokenExpiresAt int64  `json:"access_token_expires_at"`
		UserID               string `json:"user_id"`
	}
}

func (a *API) webSessionRefreshHandler(ctx context.Context, input *webSessionRefreshInput) (*webSessionRefreshOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.auth.session.refresh"))
	if err := a.validateWebAuthOrigin(input.webAuthHeaders); err != nil {
		return nil, err
	}
	refreshToken := input.refreshToken()
	if refreshToken == "" {
		return nil, huma.Error401Unauthorized("session refresh token is missing")
	}
	tokens, err := a.authService.RefreshOAuthTokens(ctx, auth.OAuthRefreshTokensRequest{
		ClientID:     webOAuthClientID,
		RefreshToken: refreshToken,
		Audience:     auth.CanonicalAPIAudience(a.cfg.APIPublicBaseURL),
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrTokenReuse):
			return nil, huma.Error401Unauthorized("session refresh token was reused")
		case errors.Is(err, auth.ErrOAuthInvalidGrant), errors.Is(err, auth.ErrOAuthInvalidRequest):
			return nil, huma.Error401Unauthorized("session refresh token is invalid or expired")
		default:
			logger.ErrorContext(ctx, "web session refresh failed", "error", err, "outcome", logging.OutcomeError)
			return nil, huma.Error500InternalServerError("failed to refresh session")
		}
	}
	out := &webSessionRefreshOutput{}
	out.SetCookie = []http.Cookie{a.webRefreshCookie(tokens.RefreshToken, tokens.RefreshExpiry)}
	out.Body.AccessToken = tokens.AccessToken
	out.Body.AccessTokenExpiresAt = tokens.AccessExpiry.Unix()
	out.Body.UserID = tokens.UserID.String()
	return out, nil
}

type webSessionSignOutInput struct {
	SecureRefreshToken string `cookie:"__Host-koditon_refresh"`
	DevRefreshToken    string `cookie:"koditon_refresh"`
	Authorization      string `header:"Authorization"`
	webAuthHeaders
}

type webSessionSignOutOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      struct {
		OK bool `json:"ok"`
	}
}

func (a *API) webSessionSignOutHandler(ctx context.Context, input *webSessionSignOutInput) (*webSessionSignOutOutput, error) {
	if err := a.validateWebAuthOrigin(input.webAuthHeaders); err != nil {
		return nil, err
	}
	if token := extractBearer(input.Authorization); token != "" {
		if claims, err := a.authService.VerifyAccessToken(ctx, token); err == nil && claims.SessionID != uuid.Nil {
			_ = a.authService.SignOutWithOwnershipCheck(ctx, claims.UserID, claims.SessionID)
		}
	}
	if refreshToken := input.refreshToken(); refreshToken != "" {
		if err := a.authService.RevokeOAuthRefreshTokenForClient(ctx, webOAuthClientID, refreshToken); err != nil {
			logging.With(a.logger, logging.Op("api.auth.session.sign_out")).ErrorContext(ctx, "web session sign out failed", "error", err, "outcome", logging.OutcomeError)
			return nil, huma.Error500InternalServerError("failed to sign out")
		}
	}
	out := &webSessionSignOutOutput{}
	out.SetCookie = a.clearWebRefreshCookies()
	out.Body.OK = true
	return out, nil
}

func (a *API) webAuthTokenOutput(tokens *auth.OAuthTokenResponse) *authTokenOutput {
	return &authTokenOutput{
		SetCookie: []http.Cookie{a.webRefreshCookie(tokens.RefreshToken, tokens.RefreshExpiry)},
		Body: authTokenBody{
			AccessToken:           tokens.AccessToken,
			AccessTokenExpiresAt:  tokens.AccessExpiry.Unix(),
			RefreshToken:          tokens.RefreshToken,
			RefreshTokenExpiresAt: tokens.RefreshExpiry.Unix(),
			UserID:                tokens.UserID.String(),
		},
	}
}

func (a *API) webRefreshCookie(token string, expiresAt time.Time) http.Cookie {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	return http.Cookie{
		Name:     a.webRefreshCookieName(),
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.webRefreshCookieSecure(),
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *API) clearWebRefreshCookies() []http.Cookie {
	return []http.Cookie{
		a.clearWebRefreshCookie(secureWebRefreshCookieName, true),
		a.clearWebRefreshCookie(devWebRefreshCookieName, false),
	}
}

func (a *API) clearWebRefreshCookie(name string, secure bool) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *API) webRefreshCookieName() string {
	if a.webRefreshCookieSecure() {
		return secureWebRefreshCookieName
	}
	return devWebRefreshCookieName
}

func (a *API) webRefreshCookieSecure() bool {
	return !a.cfg.Environment.IsDevelopment()
}

func (a *API) validateWebAuthOrigin(headers webAuthHeaders) error {
	origin := strings.TrimSpace(headers.Origin)
	if origin == "" {
		origin = originFromReferer(headers.Referer)
	}
	if origin == "" {
		return nil
	}
	if a.isAllowedWebAuthOrigin(origin) {
		return nil
	}
	return huma.Error403Forbidden("origin is not allowed")
}

func (a *API) isAllowedWebAuthOrigin(origin string) bool {
	for _, candidate := range []string{a.cfg.WebBaseURL, a.cfg.APIPublicBaseURL} {
		if sameOrigin(origin, candidate) {
			return true
		}
	}
	for _, candidate := range strings.Split(a.cfg.CORSAllowedOrigins, ",") {
		if sameOrigin(origin, candidate) {
			return true
		}
	}
	if a.cfg.Environment.IsDevelopment() {
		u, err := url.Parse(origin)
		if err == nil {
			host := u.Hostname()
			return host == "localhost" || host == "127.0.0.1" || host == "::1"
		}
	}
	return false
}

func sameOrigin(origin, candidate string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	candidate = strings.TrimRight(strings.TrimSpace(candidate), "/")
	if origin == "" || candidate == "" {
		return false
	}
	originURL, originErr := url.Parse(origin)
	candidateURL, candidateErr := url.Parse(candidate)
	if originErr == nil && candidateErr == nil && originURL.Scheme != "" && candidateURL.Scheme != "" {
		return strings.EqualFold(originURL.Scheme, candidateURL.Scheme) && strings.EqualFold(originURL.Host, candidateURL.Host)
	}
	return origin == candidate
}

func originFromReferer(referer string) string {
	u, err := url.Parse(strings.TrimSpace(referer))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func extractBearer(header string) string {
	header = strings.TrimSpace(header)
	if len(header) < len("Bearer ") || !strings.EqualFold(header[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (input webSessionRefreshInput) refreshToken() string {
	if token := strings.TrimSpace(input.SecureRefreshToken); token != "" {
		return token
	}
	return strings.TrimSpace(input.DevRefreshToken)
}

func (input webSessionSignOutInput) refreshToken() string {
	if token := strings.TrimSpace(input.SecureRefreshToken); token != "" {
		return token
	}
	return strings.TrimSpace(input.DevRefreshToken)
}
