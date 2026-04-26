package oauthapi

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"koditon-go/internal/auth"
	"koditon-go/internal/logging"
)

func (h *Handler) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.browser.login_page"))
	cont := strings.TrimSpace(r.URL.Query().Get("continue"))
	if cont == "" {
		cont = "/oauth/authorize"
	}
	if oauthClientIDFromContinueURL(cont) != koditonCLIClientID {
		http.NotFound(w, r)
		return
	}
	client, ok := h.clients[koditonCLIClientID]
	if !ok {
		http.Error(w, "oauth client not configured", http.StatusServiceUnavailable)
		return
	}
	resp, err := h.authService.CreateOAuthDeviceAuthorization(r.Context(), auth.OAuthCreateDeviceAuthorizationRequest{
		ClientID: client.ClientID,
		Scopes:   append([]string(nil), client.Scopes...),
		Audience: auth.CanonicalAPIAudience(h.publicAPIBaseURL),
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "create oauth login device authorization failed", "error", err, "outcome", logging.OutcomeError)
		http.Error(w, "failed to prepare login verification", http.StatusInternalServerError)
		return
	}
	appOpenURL := h.publicAPIBaseURL + "/oauth/app/open?kind=device&user_code=" + url.QueryEscape(resp.UserCode)
	qrCodeSVG, err := renderQRCodeSVG(appOpenURL)
	if err != nil {
		logger.ErrorContext(r.Context(), "render oauth login qr code failed", "error", err, "outcome", logging.OutcomeError)
		http.Error(w, "failed to prepare login verification", http.StatusInternalServerError)
		return
	}
	data := deviceLoginPageData{
		UserCode:   resp.UserCode,
		AppOpenURL: appOpenURL,
		QRCodeSVG:  qrCodeSVG,
		Interval:   resp.Interval,
		ExpiresIn:  resp.ExpiresIn,
	}
	_ = deviceLoginPageTemplate.Execute(w, data)
}

func (h *Handler) handleLoginDevicePoll(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.browser.device_poll"))
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "invalid", "reason": "form"})
		return
	}
	userCode := strings.ToUpper(strings.TrimSpace(r.FormValue("user_code")))
	if userCode == "" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "invalid", "reason": "user_code"})
		return
	}
	details, err := h.authService.GetOAuthDeviceAuthorizationDetailsByUserCode(r.Context(), userCode)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeJSON(w, http.StatusOK, map[string]any{"status": "invalid", "reason": "user_code"})
		default:
			logger.ErrorContext(r.Context(), "poll oauth login device authorization failed", "error", err, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to poll login status")
		}
		return
	}
	if details.ClientID != koditonCLIClientID || details.ConsumedAt != nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "invalid", "reason": "device_code"})
		return
	}
	switch {
	case details.ExpiresAt.Before(time.Now()):
		writeJSON(w, http.StatusOK, map[string]any{"status": "expired"})
	case details.DeniedAt != nil:
		writeJSON(w, http.StatusOK, map[string]any{"status": "denied"})
	case details.ApprovedBy != nil:
		writeJSON(w, http.StatusOK, map[string]any{"status": "approved"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
	}
}

func (h *Handler) handleDeviceVerifyPage(w http.ResponseWriter, r *http.Request) {
	userCode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("user_code")))
	result := strings.TrimSpace(r.URL.Query().Get("result"))
	continueURL := strings.TrimSpace(r.URL.Query().Get("continue"))
	authCode := strings.TrimSpace(r.URL.Query().Get("auth_code"))
	if userCode == "" {
		http.Error(w, "missing user_code", http.StatusBadRequest)
		return
	}
	verificationURL := h.publicAPIBaseURL + "/oauth/device/verify?user_code=" + url.QueryEscape(userCode)
	if normalizedContinue, err := normalizeOAuthContinueURL(continueURL); err == nil {
		continueURL = normalizedContinue
		verificationURL += "&continue=" + url.QueryEscape(normalizedContinue)
	}
	deepLink := url.URL{
		Scheme: "koditon",
		Host:   "oauth",
		Path:   "/device/verify",
	}
	q := deepLink.Query()
	q.Set("user_code", userCode)
	q.Set("verification_url", verificationURL)
	if continueURL != "" {
		q.Set("continue", continueURL)
	}
	deepLink.RawQuery = q.Encode()

	data := deviceVerifyPageData{
		UserCode:          userCode,
		VerificationURL:   verificationURL,
		NativeURL:         template.URL(deepLink.String()),
		Result:            result,
		AuthorizationCode: authCode,
	}
	_ = deviceVerifyPageTemplate.Execute(w, data)
}

func (h *Handler) handleDeviceVerifyRequest(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.device.verify_request"))
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	if h.notifier == nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "notification service is unavailable")
		return
	}
	tokenString, ok := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
	if !ok {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	claims, err := h.authService.VerifyAccessToken(r.Context(), tokenString)
	if err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
		return
	}

	userCode := strings.ToUpper(strings.TrimSpace(r.FormValue("user_code")))
	if userCode == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "user_code is required")
		return
	}
	continueURL, normalizeErr := normalizeOAuthContinueURL(r.FormValue("continue"))
	if normalizeErr != nil {
		continueURL = ""
	}

	details, err := h.authService.GetOAuthDeviceAuthorizationDetailsByUserCode(r.Context(), userCode)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid user_code")
		default:
			logger.ErrorContext(r.Context(), "load oauth device authorization failed", "error", err, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to load verification request")
		}
		return
	}
	if details.ClientID != koditonCLIClientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "unsupported client")
		return
	}
	if details.ExpiresAt.Before(time.Now()) {
		writeOAuthError(w, http.StatusBadRequest, "expired_token", "device code is expired")
		return
	}
	if details.DeniedAt != nil || details.ConsumedAt != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code is not approvable")
		return
	}
	if details.ApprovedBy != nil && *details.ApprovedBy != claims.UserID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code is already approved by another user")
		return
	}

	if h.notifier != nil {
		user, err := h.authService.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			logger.ErrorContext(r.Context(), "load user failed for oauth device notification", "error", err, "user_id", claims.UserID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to create verification notification")
			return
		}
		if err := h.notifier.NotifyOAuthDeviceVerificationRequest(
			r.Context(),
			user.UserIDBigint,
			details.ID,
			userCode,
			continueURL,
		); err != nil {
			logger.ErrorContext(r.Context(), "create oauth device verification notification failed", "error", err, "user_id", claims.UserID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to create verification notification")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleDeviceVerifyApprove(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.device.verify_approve"))
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	tokenString, ok := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
	if !ok {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	claims, err := h.authService.VerifyAccessToken(r.Context(), tokenString)
	if err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
		return
	}
	userCode := strings.ToUpper(strings.TrimSpace(r.FormValue("user_code")))
	if userCode == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "user_code is required")
		return
	}
	if err := h.authService.ApproveOAuthDeviceAuthorization(r.Context(), auth.OAuthApproveDeviceAuthorizationRequest{
		UserCode: userCode,
		UserID:   claims.UserID,
	}); err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid user_code")
		case errors.Is(err, auth.ErrOAuthExpiredToken):
			writeOAuthError(w, http.StatusBadRequest, "expired_token", "device code is expired")
		case errors.Is(err, auth.ErrOAuthInvalidGrant):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code is not approvable")
		default:
			logger.ErrorContext(r.Context(), "approve oauth device authorization failed", "error", err, "user_id", claims.UserID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to approve device code")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleDeviceVerifyDeny(w http.ResponseWriter, r *http.Request) {
	logger := logging.With(h.logger, logging.Op("oauth.device.verify_deny"))
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	tokenString, ok := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
	if !ok {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	claims, err := h.authService.VerifyAccessToken(r.Context(), tokenString)
	if err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
		return
	}
	userCode := strings.ToUpper(strings.TrimSpace(r.FormValue("user_code")))
	if userCode == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "user_code is required")
		return
	}
	if err := h.authService.DenyOAuthDeviceAuthorization(r.Context(), auth.OAuthDenyDeviceAuthorizationRequest{
		UserCode: userCode,
		UserID:   claims.UserID,
	}); err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthInvalidRequest):
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid user_code")
		case errors.Is(err, auth.ErrOAuthExpiredToken):
			writeOAuthError(w, http.StatusBadRequest, "expired_token", "device code is expired")
		case errors.Is(err, auth.ErrOAuthInvalidGrant):
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code is not deniable")
		default:
			logger.ErrorContext(r.Context(), "deny oauth device authorization failed", "error", err, "user_id", claims.UserID, "outcome", logging.OutcomeError)
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to deny device code")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "browser-based oauth callback is no longer supported; continue in the Koditon app", http.StatusGone)
}

func bearerTokenFromAuthorizationHeader(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func oauthClientIDFromContinueURL(raw string) string {
	cont := strings.TrimSpace(raw)
	if cont == "" {
		return ""
	}
	u, err := url.Parse(cont)
	if err != nil {
		return ""
	}
	if u.Path != "/oauth/authorize" {
		return ""
	}
	return strings.TrimSpace(u.Query().Get("client_id"))
}

func normalizeOAuthContinueURL(raw string) (string, error) {
	cont := strings.TrimSpace(raw)
	if cont == "" {
		return "/oauth/authorize", nil
	}
	u, err := url.Parse(cont)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(u.Path) != "/oauth/authorize" {
		return "", fmt.Errorf("continue path must be /oauth/authorize")
	}
	normalized := u.RequestURI()
	if normalized == "" {
		return "", fmt.Errorf("invalid continue url")
	}
	return normalized, nil
}

type deviceLoginPageData struct {
	UserCode   string
	AppOpenURL string
	QRCodeSVG  template.HTML
	Interval   int
	ExpiresIn  int
}

type deviceVerifyPageData struct {
	UserCode          string
	VerificationURL   string
	NativeURL         template.URL
	Result            string
	AuthorizationCode string
}
