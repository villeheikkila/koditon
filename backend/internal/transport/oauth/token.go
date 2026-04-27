package oauthapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/domain/auth"
	"koditon/internal/domain/auth/apple"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/util"
)

func (h *Handler) handleToken(w http.ResponseWriter, r *http.Request) {
	setOAuthTokenResponseHeaders(w)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	switch strings.TrimSpace(r.FormValue("grant_type")) {
	case grantAuthorizationCode:
		h.handleAuthorizationCodeToken(w, r)
	case grantRefreshToken:
		h.handleRefreshToken(w, r)
	case grantDeviceCode:
		h.handleDeviceCodeToken(w, r)
	case grantAppleCode:
		h.handleAppleAuthorizationCodeToken(w, r)
	case grantEmailAuthTicket:
		h.handleEmailAuthTicketToken(w, r)
	case grantPasskeyAssertion:
		h.handlePasskeyToken(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type")
	}
}

func setOAuthTokenResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func writeOAuthTokenSuccess(w http.ResponseWriter, tokenResp *auth.OAuthTokenResponse) {
	response := map[string]any{
		"token_type":               "Bearer",
		"access_token":             tokenResp.AccessToken,
		"expires_in":               int(time.Until(tokenResp.AccessExpiry).Seconds()),
		"refresh_token":            tokenResp.RefreshToken,
		"refresh_token_expires_in": int(time.Until(tokenResp.RefreshExpiry).Seconds()),
		"scope":                    strings.Join(tokenResp.Scopes, " "),
	}
	if tokenResp.SessionID != uuid.Nil {
		response["session_id"] = tokenResp.SessionID.String()
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleAuthorizationCodeToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.authorization_code"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, false)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	client := authResult.Client
	code := strings.TrimSpace(r.FormValue("code"))
	redirectURI := strings.TrimSpace(r.FormValue("redirect_uri"))
	codeVerifier := strings.TrimSpace(r.FormValue("code_verifier"))
	resource := strings.TrimSpace(r.FormValue("resource"))
	if code == "" || redirectURI == "" || codeVerifier == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code, redirect_uri and code_verifier are required")
		return
	}
	audience, err := h.resolveAudienceForClient(client, resource)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	tokenResp, err := h.authService.ExchangeOAuthAuthorizationCode(r.Context(), auth.OAuthExchangeAuthorizationCodeRequest{
		ClientID:     client.ClientID,
		Code:         code,
		RedirectURI:  redirectURI,
		CodeVerifier: codeVerifier,
		Audience:     audience,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, auth.ErrOAuthInvalidGrant):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
		default:
			logger.ErrorContext(r.Context(), "oauth authorization code exchange failed", "error", err, "client_id", client.ClientID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		}
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

func (h *Handler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.refresh"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth token client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	client := authResult.Client
	refreshToken := strings.TrimSpace(r.FormValue("refresh_token"))
	resource := strings.TrimSpace(r.FormValue("resource"))
	if refreshToken == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}
	audience, err := h.resolveAudienceForClient(client, resource)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	tokenResp, err := h.authService.RefreshOAuthTokens(r.Context(), auth.OAuthRefreshTokensRequest{
		ClientID:     client.ClientID,
		RefreshToken: refreshToken,
		Audience:     audience,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, auth.ErrTokenReuse):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token reuse detected")
		case errors.Is(err, auth.ErrOAuthInvalidGrant):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
		default:
			logger.ErrorContext(r.Context(), "oauth refresh token exchange failed", "error", err, "client_id", client.ClientID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		}
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

func (h *Handler) handleRevoke(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.revoke"))
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth revoke client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}
	if err := h.authService.RevokeOAuthRefreshTokenForClient(r.Context(), authResult.Client.ClientID, token); err != nil {
		logger.ErrorContext(r.Context(), "oauth token revocation failed", "error", err, "client_id", authResult.Client.ClientID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to revoke token")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleDeviceAuthorization(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.device.authorize"))
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, false)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth device authorization client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	client := authResult.Client
	resource := strings.TrimSpace(r.FormValue("resource"))
	audience, err := h.resolveAudienceForClient(client, resource)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}

	scopeText := strings.TrimSpace(r.FormValue("scope"))
	requestedScopes := strings.Fields(scopeText)
	scopes := h.normalizeRequestedScopesForClient(client, requestedScopes)
	if len(scopes) == 0 {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "scope is required")
		return
	}
	if err := auth.ValidateRequestedScopes(scopes, client.Scopes, nil); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error())
		return
	}

	resp, err := h.authService.CreateOAuthDeviceAuthorization(r.Context(), auth.OAuthCreateDeviceAuthorizationRequest{
		ClientID: client.ClientID,
		Scopes:   scopes,
		Audience: audience,
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "create oauth device authorization failed", "error", err, "client_id", client.ClientID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to create device authorization")
		return
	}

	verificationURI := h.publicAPIBaseURL + "/oauth/device/verify"
	verificationComplete := verificationURI + "?user_code=" + url.QueryEscape(resp.UserCode)
	writeJSON(w, http.StatusOK, map[string]any{
		"device_code":               resp.DeviceCode,
		"user_code":                 resp.UserCode,
		"verification_uri":          verificationURI,
		"verification_uri_complete": verificationComplete,
		"expires_in":                resp.ExpiresIn,
		"interval":                  resp.Interval,
	})
}

func (h *Handler) handleDeviceCodeToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.device_code"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth token client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	client := authResult.Client
	deviceCode := strings.TrimSpace(r.FormValue("device_code"))
	resource := strings.TrimSpace(r.FormValue("resource"))
	if deviceCode == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "device_code is required")
		return
	}
	audience, err := h.resolveAudienceForClient(client, resource)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	if h.devicePoller != nil && !h.devicePoller.Allow(client.ClientID, deviceCode, devicePollInterval, devicePollSlowDownStep) {
		writeOAuthError(w, http.StatusBadRequest, "slow_down", "polling too quickly; increase the polling interval")
		return
	}
	tokenResp, err := h.authService.ExchangeOAuthDeviceCode(r.Context(), auth.OAuthExchangeDeviceCodeRequest{
		ClientID:   client.ClientID,
		DeviceCode: deviceCode,
		Audience:   audience,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthPending):
			writeOAuthError(w, http.StatusBadRequest, "authorization_pending", "authorization is pending")
		case errors.Is(err, auth.ErrOAuthAccessDenied):
			writeOAuthError(w, http.StatusBadRequest, "access_denied", "authorization was denied")
		case errors.Is(err, auth.ErrOAuthExpiredToken):
			writeOAuthError(w, http.StatusBadRequest, "expired_token", "device code is expired")
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, auth.ErrOAuthInvalidGrant):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code is invalid")
		default:
			logger.ErrorContext(r.Context(), "oauth device code exchange failed", "error", err, "client_id", client.ClientID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		}
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

func (h *Handler) handleAppleAuthorizationCodeToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.apple_authorization_code"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth token client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	if !ensureFirstPartyExtensionGrantClient(w, authResult) {
		return
	}
	client := authResult.Client
	authorizationCode := strings.TrimSpace(r.FormValue("apple_authorization_code"))
	nonce := strings.TrimSpace(r.FormValue("nonce"))
	if authorizationCode == "" || nonce == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "apple_authorization_code and nonce are required")
		return
	}
	deviceReq, deviceErr := buildDeviceContextRequest(r)
	if deviceErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", deviceErr.Error())
		return
	}
	resp, err := h.authService.SignInWithApple(r.Context(), auth.SignInWithAppleRequest{
		AuthorizationCode: authorizationCode,
		Nonce:             nonce,
		DeviceID:          deviceReq.deviceID,
		UserAgent:         r.UserAgent(),
		IP:                deviceReq.ip,
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "oauth apple grant failed", "error", err, "client_id", client.ClientID, "outcome", logging.OutcomeError)
		var exchangeErr *apple.TokenExchangeError
		switch {
		case errors.As(err, &exchangeErr) && exchangeErr.ErrorCode == "invalid_grant":
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "apple authorization code is invalid")
		case errors.As(err, &exchangeErr) && exchangeErr.ErrorCode == "invalid_request":
			writeOAuthError(w, http.StatusBadGateway, "server_error", "apple sign-in token exchange was rejected")
		default:
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "apple authorization code is invalid")
		}
		return
	}
	scopeText := strings.TrimSpace(r.FormValue("scope"))
	scopes, scopeErr := h.resolveFirstPartyUserGrantScopes(r.Context(), client, scopeText, resp.UserID)
	if scopeErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", scopeErr.Error())
		return
	}
	tokenResp, issueErr := h.authService.IssueOAuthTokensForUser(r.Context(), auth.OAuthIssueTokensForUserRequest{
		ClientID:  client.ClientID,
		UserID:    resp.UserID,
		Scopes:    scopes,
		SessionID: resp.SessionID,
		Audience:  auth.CanonicalAPIAudience(h.publicAPIBaseURL),
	})
	if issueErr != nil {
		logger.ErrorContext(r.Context(), "issue oauth tokens failed", "error", issueErr, "client_id", client.ClientID, "user_id", resp.UserID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

func (h *Handler) handleEmailAuthTicketToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.email_auth_ticket"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth token client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	if !ensureFirstPartyExtensionGrantClient(w, authResult) {
		return
	}
	client := authResult.Client
	authTicket := strings.TrimSpace(r.FormValue("auth_ticket"))
	if authTicket == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "auth_ticket is required")
		return
	}
	deviceReq, deviceErr := buildDeviceContextRequest(r)
	if deviceErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", deviceErr.Error())
		return
	}
	if h.emailAuthService == nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "email auth service not configured")
		return
	}
	confirmedEmail, consumeErr := h.emailAuthService.ConsumeAuthenticationTicket(r.Context(), authTicket)
	if consumeErr != nil {
		logger.ErrorContext(r.Context(), "oauth email auth ticket consume failed", "error", consumeErr, "client_id", client.ClientID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "email auth ticket is invalid")
		return
	}
	resp, signErr := h.authService.SignInWithEmail(r.Context(), auth.SignInWithEmailRequest{
		ConfirmedEmail: confirmedEmail,
		DeviceID:       deviceReq.deviceID,
		UserAgent:      r.UserAgent(),
		IP:             deviceReq.ip,
	})
	if signErr != nil {
		logger.ErrorContext(r.Context(), "oauth email auth grant failed", "error", signErr, "client_id", client.ClientID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "email sign-in is invalid")
		return
	}
	scopeText := strings.TrimSpace(r.FormValue("scope"))
	scopes, scopeErr := h.resolveFirstPartyUserGrantScopes(r.Context(), client, scopeText, resp.UserID)
	if scopeErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", scopeErr.Error())
		return
	}
	tokenResp, issueErr := h.authService.IssueOAuthTokensForUser(r.Context(), auth.OAuthIssueTokensForUserRequest{
		ClientID:  client.ClientID,
		UserID:    resp.UserID,
		Scopes:    scopes,
		SessionID: resp.SessionID,
		Audience:  auth.CanonicalAPIAudience(h.publicAPIBaseURL),
	})
	if issueErr != nil {
		logger.ErrorContext(r.Context(), "issue oauth tokens failed", "error", issueErr, "client_id", client.ClientID, "user_id", resp.UserID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

func (h *Handler) handlePasskeyToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.token.passkey_assertion"))
	authResult, status, oauthErrCode, description, err := h.authenticateOAuthClient(r.Context(), r, true)
	if err != nil {
		logger.ErrorContext(r.Context(), "resolve oauth token client failed", "error", err, "outcome", logging.OutcomeError)
		writeOAuthError(w, status, oauthErrCode, description)
		return
	}
	if status != 0 {
		writeOAuthClientAuthFailure(w, status, oauthErrCode, description)
		return
	}
	if !ensureFirstPartyExtensionGrantClient(w, authResult) {
		return
	}
	client := authResult.Client
	challengeIDText := strings.TrimSpace(r.FormValue("challenge_id"))
	credentialText := strings.TrimSpace(r.FormValue("passkey_credential_json"))
	if challengeIDText == "" || credentialText == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "challenge_id and passkey_credential_json are required")
		return
	}
	challengeID, err := uuid.Parse(challengeIDText)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "challenge_id must be a valid uuid")
		return
	}
	credential := json.RawMessage(credentialText)
	deviceReq, deviceErr := buildDeviceContextRequest(r)
	if deviceErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", deviceErr.Error())
		return
	}
	resp, signErr := h.authService.FinishPasskeyAuthentication(r.Context(), auth.FinishPasskeyAuthenticateRequest{
		ChallengeID: challengeID,
		Credential:  credential,
		DeviceID:    deviceReq.deviceID,
		UserAgent:   r.UserAgent(),
		IP:          deviceReq.ip,
	})
	if signErr != nil {
		logger.ErrorContext(r.Context(), "oauth passkey assertion grant failed", "error", signErr, "client_id", client.ClientID, "outcome", logging.OutcomeError)
		switch {
		case errors.Is(signErr, auth.ErrPasskeyNotFound):
			writeOAuthErrorWithCode(w, http.StatusBadRequest, "invalid_grant", "passkey account not found", "passkey_not_found")
		case errors.Is(signErr, auth.ErrPasskeyChallenge):
			writeOAuthErrorWithCode(w, http.StatusBadRequest, "invalid_grant", "passkey challenge is invalid or expired", "passkey_challenge_invalid")
		default:
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "passkey assertion is invalid")
		}
		return
	}
	scopeText := strings.TrimSpace(r.FormValue("scope"))
	scopes, scopeErr := h.resolveFirstPartyUserGrantScopes(r.Context(), client, scopeText, resp.UserID)
	if scopeErr != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", scopeErr.Error())
		return
	}
	tokenResp, issueErr := h.authService.IssueOAuthTokensForUser(r.Context(), auth.OAuthIssueTokensForUserRequest{
		ClientID:  client.ClientID,
		UserID:    resp.UserID,
		Scopes:    scopes,
		SessionID: resp.SessionID,
		Audience:  auth.CanonicalAPIAudience(h.publicAPIBaseURL),
	})
	if issueErr != nil {
		logger.ErrorContext(r.Context(), "issue oauth tokens failed", "error", issueErr, "client_id", client.ClientID, "user_id", resp.UserID, "outcome", logging.OutcomeError)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	writeOAuthTokenSuccess(w, tokenResp)
}

type deviceContextRequest struct {
	deviceID uuid.UUID
	ip       string
}

func buildDeviceContextRequest(r *http.Request) (deviceContextRequest, error) {
	rawDeviceID := strings.TrimSpace(r.Header.Get("Device-Id"))
	if rawDeviceID == "" {
		return deviceContextRequest{}, fmt.Errorf("device-id header is required")
	}
	out := deviceContextRequest{}
	if deviceID, err := parseDeviceID(rawDeviceID); err == nil {
		out.deviceID = deviceID
	} else {
		return deviceContextRequest{}, fmt.Errorf("device-id must be a valid public id")
	}
	out.ip = clientIPFromRequest(r)
	return out, nil
}

func clientIPFromRequest(r *http.Request) string {
	remoteIP := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteIP); err == nil && host != "" {
		remoteIP = strings.TrimSpace(host)
	}
	if remoteIP == "" {
		return ""
	}

	if isTrustedForwardedProxy(remoteIP) {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 {
				candidate := strings.TrimSpace(parts[0])
				if host, _, err := net.SplitHostPort(candidate); err == nil && strings.TrimSpace(host) != "" {
					candidate = strings.TrimSpace(host)
				}
				if parsed := net.ParseIP(candidate); parsed != nil {
					return parsed.String()
				}
			}
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			if host, _, err := net.SplitHostPort(realIP); err == nil && strings.TrimSpace(host) != "" {
				realIP = strings.TrimSpace(host)
			}
			if parsed := net.ParseIP(realIP); parsed != nil {
				return parsed.String()
			}
		}
	}
	return remoteIP
}

func isTrustedForwardedProxy(remoteIP string) bool {
	parsed := net.ParseIP(strings.TrimSpace(remoteIP))
	return parsed != nil && parsed.IsLoopback()
}

func parseDeviceID(raw string) (uuid.UUID, error) {
	if decoded, err := util.DecodeUUIDBase62(raw); err == nil {
		return decoded, nil
	}
	return uuid.Parse(raw)
}
