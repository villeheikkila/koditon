package oauthapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"koditon-go/internal/auth"
)

type authorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Resource            string
	Scope               []string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type oauthClientCredentials struct {
	ClientID     string
	ClientSecret string
	UsedBasic    bool
}

type oauthClientAuthResult struct {
	Client       oauthClient
	Credentials  oauthClientCredentials
	HasClientID  bool
	SecretNeeded bool
}

func parseAuthorizeRequest(values url.Values) authorizeRequest {
	scopeText := strings.TrimSpace(values.Get("scope"))
	var scopes []string
	if scopeText != "" {
		scopes = strings.Fields(scopeText)
	}
	return authorizeRequest{
		ResponseType:        strings.TrimSpace(values.Get("response_type")),
		ClientID:            strings.TrimSpace(values.Get("client_id")),
		RedirectURI:         strings.TrimSpace(values.Get("redirect_uri")),
		Resource:            strings.TrimSpace(values.Get("resource")),
		Scope:               scopes,
		State:               strings.TrimSpace(values.Get("state")),
		CodeChallenge:       strings.TrimSpace(values.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(values.Get("code_challenge_method")),
	}
}

func parseClientCredentials(r *http.Request) (oauthClientCredentials, error) {
	creds := oauthClientCredentials{
		ClientID:     strings.TrimSpace(r.FormValue("client_id")),
		ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return creds, nil
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return oauthClientCredentials{}, fmt.Errorf("unsupported authorization header")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[1]))
	if err != nil {
		return oauthClientCredentials{}, fmt.Errorf("invalid basic authorization header")
	}
	pair := strings.SplitN(string(decoded), ":", 2)
	if len(pair) != 2 {
		return oauthClientCredentials{}, fmt.Errorf("invalid basic authorization header")
	}
	if creds.ClientID != "" && creds.ClientID != strings.TrimSpace(pair[0]) {
		return oauthClientCredentials{}, fmt.Errorf("conflicting client credentials")
	}
	creds.ClientID = strings.TrimSpace(pair[0])
	creds.ClientSecret = strings.TrimSpace(pair[1])
	creds.UsedBasic = true
	return creds, nil
}

func (h *Handler) authenticateOAuthClient(ctx context.Context, r *http.Request, allowImplicitPublic bool) (*oauthClientAuthResult, int, string, string, error) {
	creds, err := parseClientCredentials(r)
	if err != nil {
		return nil, http.StatusBadRequest, "invalid_request", err.Error(), nil
	}

	var (
		client oauthClient
		ok     bool
	)
	if creds.ClientID != "" {
		client, ok, err = h.resolveClientByID(ctx, creds.ClientID)
	} else if allowImplicitPublic {
		client, ok, err = h.resolveTokenClient(ctx, "")
	} else {
		return nil, http.StatusBadRequest, "invalid_client", "client_id is required", nil
	}
	if err != nil {
		return nil, http.StatusInternalServerError, "server_error", "failed to resolve client", err
	}
	if !ok {
		return nil, http.StatusBadRequest, "invalid_client", "unknown client_id", nil
	}

	secretNeeded := client.ClientType == "confidential"
	if secretNeeded {
		if !h.validateClientSecret(client, creds.ClientSecret) {
			return nil, http.StatusUnauthorized, "invalid_client", "invalid client secret", nil
		}
	} else if creds.ClientSecret != "" {
		return nil, http.StatusBadRequest, "invalid_client", "public clients must not authenticate with a client secret", nil
	}

	return &oauthClientAuthResult{
		Client:       client,
		Credentials:  creds,
		HasClientID:  creds.ClientID != "",
		SecretNeeded: secretNeeded,
	}, 0, "", "", nil
}

func writeOAuthInvalidClient(w http.ResponseWriter, status int, description string, challenge bool) {
	if challenge {
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
	}
	writeOAuthError(w, status, "invalid_client", description)
}

func writeOAuthClientAuthFailure(w http.ResponseWriter, status int, code, description string) {
	if code == "invalid_client" {
		writeOAuthInvalidClient(w, status, description, status == http.StatusUnauthorized)
		return
	}
	writeOAuthError(w, status, code, description)
}

func ensureFirstPartyExtensionGrantClient(w http.ResponseWriter, authResult *oauthClientAuthResult) bool {
	if authResult == nil || !authResult.HasClientID || authResult.Client.ClientID != "koditon-apple" {
		writeOAuthError(w, http.StatusBadRequest, "unauthorized_client", "grant_type is restricted to the first-party mobile client")
		return false
	}
	return true
}

func (h *Handler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	req := parseAuthorizeRequest(r.URL.Query())
	client, requestedScopes, err := h.validateAuthorizeRequest(r.Context(), req)
	if err != nil {
		if h.tryRedirectAuthorizeError(w, r, req, authorizeErrorCode(err), err.Error()) {
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Scope = append([]string(nil), requestedScopes...)
	handoff, token, err := h.createAuthorizationHandoff(r.Context(), req, client)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create oauth authorization handoff failed", "error", err)
		http.Error(w, "failed to prepare oauth authorization handoff", http.StatusInternalServerError)
		return
	}
	h.renderAuthorizeHandoffPage(w, handoff, token)
}

func (h *Handler) getUserAllowedScopes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := h.authService.GetRoleNamesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return auth.ScopesForRoles(roles), nil
}

func (h *Handler) resolveFirstPartyUserGrantScopes(ctx context.Context, client oauthClient, scopeText string, userID uuid.UUID) ([]string, error) {
	requested := strings.Fields(strings.TrimSpace(scopeText))
	requested = h.normalizeRequestedScopesForClient(client, requested)
	if len(requested) == 0 {
		return nil, fmt.Errorf("scope is required")
	}
	userAllowed, err := h.getUserAllowedScopes(ctx, userID)
	if err != nil {
		return nil, err
	}
	granted, err := auth.ClampRequestedScopesToUserGrants(requested, client.Scopes, userAllowed)
	if err != nil {
		return nil, err
	}
	if len(granted) != len(requested) {
		h.logger.WarnContext(
			ctx,
			"clamped first-party oauth scopes to user grants",
			"client_id", client.ClientID,
			"user_id", userID,
			"requested_scopes", requested,
			"granted_scopes", granted,
		)
	}
	return granted, nil
}

func (h *Handler) resolveAuthorizeRequestScopes(ctx context.Context, client oauthClient, requested []string, userID uuid.UUID) ([]string, error) {
	requested = h.normalizeRequestedScopesForClient(client, requested)
	if len(requested) == 0 {
		return nil, fmt.Errorf("scope is required")
	}
	userAllowed, err := h.getUserAllowedScopes(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := auth.ValidateRequestedScopes(requested, client.Scopes, userAllowed); err != nil {
		return nil, err
	}
	return append([]string(nil), requested...), nil
}

func authorizeErrorCode(err error) string {
	if err == nil {
		return "invalid_request"
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "response_type"):
		return "unsupported_response_type"
	case strings.Contains(msg, "scope"):
		return "invalid_scope"
	default:
		return "invalid_request"
	}
}

func (h *Handler) tryRedirectAuthorizeError(w http.ResponseWriter, r *http.Request, req authorizeRequest, code, description string) bool {
	if !h.canRedirectAuthorizeError(r.Context(), req) {
		return false
	}
	redirectWithError(w, r, req.RedirectURI, req.State, code, description)
	return true
}

func (h *Handler) canRedirectAuthorizeError(ctx context.Context, req authorizeRequest) bool {
	if strings.TrimSpace(req.ClientID) == "" || strings.TrimSpace(req.RedirectURI) == "" {
		return false
	}
	client, ok, err := h.resolveClientByID(ctx, req.ClientID)
	if err != nil || !ok {
		return false
	}
	return containsExact(client.RedirectURIs, req.RedirectURI)
}

func (h *Handler) resolveTokenClient(ctx context.Context, clientID string) (oauthClient, bool, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID != "" {
		client, ok, err := h.resolveClientByID(ctx, clientID)
		return client, ok, err
	}
	var selected oauthClient
	for _, client := range h.clients {
		if strings.EqualFold(client.ClientType, "public") {
			if selected.ClientID != "" {
				return oauthClient{}, false, nil
			}
			selected = client
		}
	}
	if selected.ClientID == "" {
		return oauthClient{}, false, nil
	}
	return selected, true, nil
}
