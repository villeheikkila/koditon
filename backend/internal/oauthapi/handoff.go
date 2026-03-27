package oauthapi

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"koditon-go/internal/auth"
	db "koditon-go/internal/db"
	"koditon-go/internal/util"
)

const (
	oauthAuthorizationHandoffTTL = 5 * time.Minute
)

type oauthAuthorizationHandoff struct {
	ID                  string
	TokenHash           string
	UserCode            string
	ClientID            string
	ClientDisplayName   string
	ClientLogoURL       string
	RedirectURI         string
	RedirectHost        string
	Scopes              []string
	Audience            string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
	Status              string
	ApprovedBy          uuid.UUID
	AuthorizationCode   string
	RedirectURL         string
}

type authorizeHandoffPageData struct {
	HandoffID     string
	ClientName    string
	ClientLogoURL string
	RedirectHost  string
	ScopeText     string
	UserCode      string
	ExpiresIn     int
	AppOpenURL    string
	QRCodeSVG     template.HTML
	StatusURL     string
}

type appOpenPageData struct {
	OpenURL template.URL
}

func (h *Handler) createAuthorizationHandoff(
	ctx context.Context,
	req authorizeRequest,
	client oauthClient,
) (*oauthAuthorizationHandoff, string, error) {
	if h.queries == nil {
		return nil, "", errors.New("oauth authorization handoffs unavailable")
	}
	token, err := randomToken(32)
	if err != nil {
		return nil, "", err
	}
	userCode, err := randomToken(4)
	if err != nil {
		return nil, "", err
	}
	clientName := strings.TrimSpace(client.DisplayName)
	if clientName == "" {
		clientName = resolveOAuthClientDisplayName(client.ClientID, nil)
	}
	if clientName == "" {
		clientName = strings.TrimSpace(req.ClientID)
	}
	row, err := h.queries.CreateOAuthAuthorizationHandoff(ctx, db.CreateOAuthAuthorizationHandoffParams{
		OauthAuthorizationHandoffTokenHash:           stringPtr(hashText(token)),
		OauthAuthorizationHandoffUserCode:            stringPtr(strings.ToUpper(userCode[:4])),
		OauthClientID:                                stringPtr(strings.TrimSpace(req.ClientID)),
		OauthAuthorizationHandoffRedirectUri:         stringPtr(strings.TrimSpace(req.RedirectURI)),
		OauthAuthorizationHandoffScopes:              append([]string(nil), req.Scope...),
		OauthAuthorizationHandoffAudience:            stringPtr(strings.TrimSpace(req.Resource)),
		OauthAuthorizationHandoffState:               stringPtr(strings.TrimSpace(req.State)),
		OauthAuthorizationHandoffCodeChallenge:       stringPtr(strings.TrimSpace(req.CodeChallenge)),
		OauthAuthorizationHandoffCodeChallengeMethod: stringPtr(strings.TrimSpace(req.CodeChallengeMethod)),
		OauthAuthorizationHandoffExpiresAt:           util.TimeToPg(time.Now().Add(oauthAuthorizationHandoffTTL)),
	})
	if err != nil {
		return nil, "", err
	}
	handoff := handoffFromStoredRow(
		row.OauthAuthorizationHandoffID,
		row.OauthAuthorizationHandoffTokenHash,
		row.OauthAuthorizationHandoffUserCode,
		row.OauthClientID,
		row.OauthAuthorizationHandoffRedirectUri,
		row.OauthAuthorizationHandoffScopes,
		row.OauthAuthorizationHandoffAudience,
		row.OauthAuthorizationHandoffState,
		row.OauthAuthorizationHandoffCodeChallenge,
		row.OauthAuthorizationHandoffCodeChallengeMethod,
		row.UserUuid,
		row.OauthAuthorizationHandoffAuthorizationCode,
		row.OauthAuthorizationHandoffRedirectUrl,
		row.OauthAuthorizationHandoffDeniedAt.Valid,
		row.OauthAuthorizationHandoffCompletedAt.Valid,
		row.OauthAuthorizationHandoffExpiresAt,
	)
	handoff.ClientDisplayName = clientName
	handoff.ClientLogoURL = strings.TrimSpace(client.LogoURL)
	return handoff, token, nil
}

func (h *Handler) getAuthorizationHandoffByID(ctx context.Context, id string) (*oauthAuthorizationHandoff, bool) {
	if h.queries == nil {
		return nil, false
	}
	parsedID, err := uuid.Parse(strings.Trim(strings.TrimSpace(id), `"'`))
	if err != nil {
		return nil, false
	}
	row, err := h.queries.GetOAuthAuthorizationHandoffByID(ctx, util.UUIDToPg(parsedID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	return handoffFromStoredRow(
		row.OauthAuthorizationHandoffID,
		row.OauthAuthorizationHandoffTokenHash,
		row.OauthAuthorizationHandoffUserCode,
		row.OauthClientID,
		row.OauthAuthorizationHandoffRedirectUri,
		row.OauthAuthorizationHandoffScopes,
		row.OauthAuthorizationHandoffAudience,
		row.OauthAuthorizationHandoffState,
		row.OauthAuthorizationHandoffCodeChallenge,
		row.OauthAuthorizationHandoffCodeChallengeMethod,
		row.UserUuid,
		row.OauthAuthorizationHandoffAuthorizationCode,
		row.OauthAuthorizationHandoffRedirectUrl,
		row.OauthAuthorizationHandoffDeniedAt.Valid,
		row.OauthAuthorizationHandoffCompletedAt.Valid,
		row.OauthAuthorizationHandoffExpiresAt,
	), true
}

func (h *Handler) getAuthorizationHandoffByToken(ctx context.Context, token string) (*oauthAuthorizationHandoff, bool) {
	if h.queries == nil {
		return nil, false
	}
	row, err := h.queries.GetOAuthAuthorizationHandoffByTokenHash(ctx, stringPtr(hashText(strings.TrimSpace(token))))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	return handoffFromStoredRow(
		row.OauthAuthorizationHandoffID,
		row.OauthAuthorizationHandoffTokenHash,
		row.OauthAuthorizationHandoffUserCode,
		row.OauthClientID,
		row.OauthAuthorizationHandoffRedirectUri,
		row.OauthAuthorizationHandoffScopes,
		row.OauthAuthorizationHandoffAudience,
		row.OauthAuthorizationHandoffState,
		row.OauthAuthorizationHandoffCodeChallenge,
		row.OauthAuthorizationHandoffCodeChallengeMethod,
		row.UserUuid,
		row.OauthAuthorizationHandoffAuthorizationCode,
		row.OauthAuthorizationHandoffRedirectUrl,
		row.OauthAuthorizationHandoffDeniedAt.Valid,
		row.OauthAuthorizationHandoffCompletedAt.Valid,
		row.OauthAuthorizationHandoffExpiresAt,
	), true
}

func (h *Handler) getAuthorizationHandoffByUserCode(ctx context.Context, userCode string) (*oauthAuthorizationHandoff, bool) {
	if h.queries == nil {
		return nil, false
	}
	row, err := h.queries.GetOAuthAuthorizationHandoffByUserCode(
		ctx,
		stringPtr(strings.ToUpper(strings.TrimSpace(userCode))),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	return handoffFromStoredRow(
		row.OauthAuthorizationHandoffID,
		row.OauthAuthorizationHandoffTokenHash,
		row.OauthAuthorizationHandoffUserCode,
		row.OauthClientID,
		row.OauthAuthorizationHandoffRedirectUri,
		row.OauthAuthorizationHandoffScopes,
		row.OauthAuthorizationHandoffAudience,
		row.OauthAuthorizationHandoffState,
		row.OauthAuthorizationHandoffCodeChallenge,
		row.OauthAuthorizationHandoffCodeChallengeMethod,
		row.UserUuid,
		row.OauthAuthorizationHandoffAuthorizationCode,
		row.OauthAuthorizationHandoffRedirectUrl,
		row.OauthAuthorizationHandoffDeniedAt.Valid,
		row.OauthAuthorizationHandoffCompletedAt.Valid,
		row.OauthAuthorizationHandoffExpiresAt,
	), true
}

func handoffFromStoredRow(
	id uuid.UUID,
	tokenHash string,
	userCode string,
	clientID string,
	redirectURI string,
	scopes []string,
	audience string,
	state string,
	codeChallenge string,
	codeChallengeMethod string,
	userUUID pgtype.UUID,
	authorizationCode *string,
	redirectURL *string,
	denied bool,
	completed bool,
	expiresAt time.Time,
) *oauthAuthorizationHandoff {
	status := "pending"
	switch {
	case time.Now().After(expiresAt):
		status = "expired"
	case completed:
		status = "completed"
	case denied:
		status = "denied"
	case authorizationCode != nil && redirectURL != nil && userUUID.Valid:
		status = "approved"
	}
	redirectHost := redirectURI
	if parsed, err := url.Parse(redirectURI); err == nil && parsed.Host != "" {
		redirectHost = parsed.Host
	}
	handoff := &oauthAuthorizationHandoff{
		ID:                  id.String(),
		TokenHash:           tokenHash,
		UserCode:            userCode,
		ClientID:            clientID,
		ClientDisplayName:   clientID,
		RedirectURI:         redirectURI,
		RedirectHost:        redirectHost,
		Scopes:              append([]string(nil), scopes...),
		Audience:            audience,
		State:               state,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           expiresAt,
		Status:              status,
		ApprovedBy:          util.PgUUIDToUUID(userUUID),
	}
	if authorizationCode != nil {
		handoff.AuthorizationCode = *authorizationCode
	}
	if redirectURL != nil {
		handoff.RedirectURL = *redirectURL
	}
	return handoff
}

func (h *Handler) handoffStatusPayload(handoff *oauthAuthorizationHandoff) map[string]any {
	payload := map[string]any{
		"status": handoff.Status,
	}
	if handoff.RedirectURL != "" {
		payload["redirect_url"] = handoff.RedirectURL
	}
	return payload
}

func (h *Handler) renderAuthorizeHandoffPage(w http.ResponseWriter, handoff *oauthAuthorizationHandoff, token string) {
	if handoff == nil {
		http.Error(w, "handoff not found", http.StatusNotFound)
		return
	}
	appOpenURL := h.publicAPIBaseURL + "/oauth/app/open?kind=authorize&handoff_token=" + url.QueryEscape(token)
	qrCodeSVG, err := renderQRCodeSVG(appOpenURL)
	if err != nil {
		http.Error(w, "failed to render qr code", http.StatusInternalServerError)
		return
	}
	clientName, clientLogoURL := h.resolveHandoffClientBranding(context.Background(), handoff)
	data := authorizeHandoffPageData{
		HandoffID:     handoff.ID,
		ClientName:    clientName,
		ClientLogoURL: clientLogoURL,
		RedirectHost:  handoff.RedirectHost,
		ScopeText:     strings.Join(handoff.Scopes, " "),
		UserCode:      handoff.UserCode,
		ExpiresIn:     secondsRemaining(handoff.ExpiresAt),
		AppOpenURL:    appOpenURL,
		QRCodeSVG:     qrCodeSVG,
		StatusURL:     h.publicAPIBaseURL + "/oauth/authorize/handoff/status?id=" + url.QueryEscape(handoff.ID),
	}
	_ = authorizeHandoffTemplate.Execute(w, data)
}

func (h *Handler) handleAuthorizeHandoffPage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	token := strings.TrimSpace(r.URL.Query().Get("handoff_token"))
	switch {
	case token != "":
		handoff, ok := h.getAuthorizationHandoffByToken(r.Context(), token)
		if !ok {
			http.Error(w, "handoff not found", http.StatusNotFound)
			return
		}
		h.renderAuthorizeHandoffPage(w, handoff, token)
	case id != "":
		handoff, ok := h.getAuthorizationHandoffByID(r.Context(), id)
		if !ok {
			http.Error(w, "handoff not found", http.StatusNotFound)
			return
		}
		clientName, clientLogoURL := h.resolveHandoffClientBranding(r.Context(), handoff)
		_ = authorizeHandoffTemplate.Execute(w, authorizeHandoffPageData{
			HandoffID:     handoff.ID,
			ClientName:    clientName,
			ClientLogoURL: clientLogoURL,
			RedirectHost:  handoff.RedirectHost,
			ScopeText:     strings.Join(handoff.Scopes, " "),
			UserCode:      handoff.UserCode,
			ExpiresIn:     secondsRemaining(handoff.ExpiresAt),
			StatusURL:     h.publicAPIBaseURL + "/oauth/authorize/handoff/status?id=" + url.QueryEscape(handoff.ID),
		})
	default:
		http.Error(w, "missing handoff", http.StatusBadRequest)
	}
}

func (h *Handler) resolveHandoffClientBranding(ctx context.Context, handoff *oauthAuthorizationHandoff) (string, string) {
	if handoff == nil {
		return "", ""
	}
	name := strings.TrimSpace(handoff.ClientDisplayName)
	logoURL := strings.TrimSpace(handoff.ClientLogoURL)
	if name != "" && logoURL != "" {
		return name, logoURL
	}
	client, ok, err := h.resolveClientByID(ctx, handoff.ClientID)
	if err == nil && ok {
		if name == "" {
			name = strings.TrimSpace(client.DisplayName)
		}
		if logoURL == "" {
			logoURL = strings.TrimSpace(client.LogoURL)
		}
	}
	if name == "" {
		name = strings.TrimSpace(handoff.ClientID)
	}
	return name, logoURL
}

func (h *Handler) handleAuthorizeHandoffStatus(w http.ResponseWriter, r *http.Request) {
	handoff, ok := h.getAuthorizationHandoffByID(r.Context(), r.URL.Query().Get("id"))
	if !ok {
		writeOAuthError(w, http.StatusNotFound, "invalid_request", "handoff not found")
		return
	}
	writeJSON(w, http.StatusOK, h.handoffStatusPayload(handoff))
}

func (h *Handler) handleAuthorizeHandoffResolve(w http.ResponseWriter, r *http.Request) {
	tokenString, hasBearer := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
	if !hasBearer {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	if _, err := h.authService.VerifyAccessToken(r.Context(), tokenString); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
		return
	}
	var payload struct {
		HandoffToken string `json:"handoff_token"`
		UserCode     string `json:"user_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	var (
		handoff *oauthAuthorizationHandoff
		ok      bool
	)
	switch {
	case strings.TrimSpace(payload.HandoffToken) != "":
		handoff, ok = h.getAuthorizationHandoffByToken(r.Context(), payload.HandoffToken)
	case strings.TrimSpace(payload.UserCode) != "":
		handoff, ok = h.getAuthorizationHandoffByUserCode(r.Context(), payload.UserCode)
	default:
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "handoff_token or user_code is required")
		return
	}
	if !ok {
		writeOAuthError(w, http.StatusNotFound, "invalid_request", "handoff not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"handoff_id":          handoff.ID,
		"client_id":           handoff.ClientID,
		"client_display_name": handoff.ClientDisplayName,
		"redirect_host":       handoff.RedirectHost,
		"scopes":              handoff.Scopes,
		"expires_at_unix":     handoff.ExpiresAt.Unix(),
	})
}

func (h *Handler) handleAuthorizeHandoffApprove(w http.ResponseWriter, r *http.Request) {
	if h.queries == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "server_error", "handoff storage is unavailable")
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
	var payload struct {
		HandoffID string `json:"handoff_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(payload.HandoffID))
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid handoff id")
		return
	}
	handoff, ok := h.getAuthorizationHandoffByID(r.Context(), id.String())
	if !ok {
		writeOAuthError(w, http.StatusNotFound, "invalid_request", "handoff not found")
		return
	}
	if handoff.Status == "approved" || handoff.Status == "completed" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "redirect_url": handoff.RedirectURL})
		return
	}
	if handoff.Status != "pending" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "handoff is not approvable")
		return
	}

	client, ok, err := h.resolveClientByID(r.Context(), handoff.ClientID)
	if err != nil || !ok {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "unknown client_id")
		return
	}
	scopes, err := h.resolveAuthorizeRequestScopes(r.Context(), client, handoff.Scopes, claims.UserID)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error())
		return
	}
	audience, err := h.resolveAudienceForClient(client, handoff.Audience)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	code, err := h.authService.CreateOAuthAuthorizationCode(r.Context(), auth.OAuthCreateAuthorizationCodeRequest{
		ClientID:            handoff.ClientID,
		UserID:              claims.UserID,
		RedirectURI:         handoff.RedirectURI,
		Scopes:              scopes,
		Audience:            audience,
		CodeChallenge:       handoff.CodeChallenge,
		CodeChallengeMethod: handoff.CodeChallengeMethod,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create oauth authorization code from handoff failed", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to approve handoff")
		return
	}
	redirectURL, err := buildRedirectURL(handoff.RedirectURI, code, handoff.State)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid redirect uri")
		return
	}
	row, err := h.queries.ApproveOAuthAuthorizationHandoffByID(r.Context(), db.ApproveOAuthAuthorizationHandoffByIDParams{
		UserUuid: util.UUIDToPg(claims.UserID),
		OauthAuthorizationHandoffAuthorizationCode: stringPtr(code),
		OauthAuthorizationHandoffRedirectUrl:       stringPtr(redirectURL),
		OauthAuthorizationHandoffID:                util.UUIDToPg(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "handoff is not approvable")
			return
		}
		h.logger.ErrorContext(r.Context(), "persist approved oauth authorization handoff failed", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to approve handoff")
		return
	}
	updated := handoffFromStoredRow(
		row.OauthAuthorizationHandoffID,
		row.OauthAuthorizationHandoffTokenHash,
		row.OauthAuthorizationHandoffUserCode,
		row.OauthClientID,
		row.OauthAuthorizationHandoffRedirectUri,
		row.OauthAuthorizationHandoffScopes,
		row.OauthAuthorizationHandoffAudience,
		row.OauthAuthorizationHandoffState,
		row.OauthAuthorizationHandoffCodeChallenge,
		row.OauthAuthorizationHandoffCodeChallengeMethod,
		row.UserUuid,
		row.OauthAuthorizationHandoffAuthorizationCode,
		row.OauthAuthorizationHandoffRedirectUrl,
		row.OauthAuthorizationHandoffDeniedAt.Valid,
		row.OauthAuthorizationHandoffCompletedAt.Valid,
		row.OauthAuthorizationHandoffExpiresAt,
	)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "redirect_url": updated.RedirectURL})
}

func (h *Handler) handleAuthorizeHandoffDeny(w http.ResponseWriter, r *http.Request) {
	if h.queries == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "server_error", "handoff storage is unavailable")
		return
	}
	tokenString, ok := bearerTokenFromAuthorizationHeader(r.Header.Get("Authorization"))
	if !ok {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing bearer token")
		return
	}
	if _, err := h.authService.VerifyAccessToken(r.Context(), tokenString); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
		return
	}
	var payload struct {
		HandoffID string `json:"handoff_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(payload.HandoffID))
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid handoff id")
		return
	}
	if _, err := h.queries.DenyOAuthAuthorizationHandoffByID(r.Context(), util.UUIDToPg(id)); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.logger.ErrorContext(r.Context(), "deny oauth authorization handoff failed", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to deny handoff")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleAppOpenPage(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	var target url.URL
	target.Scheme = "koditon"
	target.Host = "oauth"

	query := target.Query()
	switch kind {
	case "authorize":
		token := strings.TrimSpace(r.URL.Query().Get("handoff_token"))
		if token == "" {
			http.Error(w, "missing handoff_token", http.StatusBadRequest)
			return
		}
		target.Path = "/authorize"
		query.Set("handoff_token", token)
	case "device":
		userCode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("user_code")))
		if userCode == "" {
			http.Error(w, "missing user_code", http.StatusBadRequest)
			return
		}
		target.Path = "/device/verify"
		query.Set("user_code", userCode)
		if continueURL := strings.TrimSpace(r.URL.Query().Get("continue")); continueURL != "" {
			query.Set("continue", continueURL)
		}
	default:
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}
	target.RawQuery = query.Encode()
	_ = appOpenPageTemplate.Execute(w, appOpenPageData{
		OpenURL: template.URL(target.String()),
	})
}

func buildRedirectURL(redirectURI, code, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("code", code)
	if strings.TrimSpace(state) != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func stringPtr(s string) *string {
	return &s
}

func secondsRemaining(expiresAt time.Time) int {
	remaining := int(time.Until(expiresAt).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}
