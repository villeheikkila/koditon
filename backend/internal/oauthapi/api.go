package oauthapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"koditon-go/internal/auth"
	db "koditon-go/internal/db"
	"koditon-go/internal/emailauth"
	"koditon-go/internal/runtimecfg"

	"github.com/google/uuid"
)

const (
	grantAuthorizationCode = "authorization_code"
	grantRefreshToken      = "refresh_token"
	grantDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	grantAppleCode         = "urn:koditon:params:oauth:grant-type:apple_authorization_code"
	grantEmailAuthTicket   = "urn:koditon:params:oauth:grant-type:email_auth_ticket"
	grantPasskeyAssertion  = "urn:koditon:params:oauth:grant-type:passkey_assertion"
	koditonCLIClientID        = "koditon-cli"
	dcrRateLimitPerMinute  = 30
	devicePollInterval     = 5 * time.Second
	devicePollSlowDownStep = 5 * time.Second
)

var (
	allowedDynamicRedirectURIs = []string{
		"https://chatgpt.com/connector_platform_oauth_redirect",
		"https://platform.openai.com/apps-manage/oauth",
	}
	allowedDynamicScopes = []string{auth.ScopeMCPCoreRead}
	allowedDynamicCIDRs  []string
)

type Config struct {
	HTTP      runtimecfg.HTTPConfig
	Queries   *db.Queries
	EmailAuth *emailauth.Service
}

// DeviceNotifier is an optional interface for sending push notifications on OAuth device logins.
type DeviceNotifier interface {
	NotifyOAuthDeviceVerificationRequest(ctx context.Context, userID int64, authorizationID uuid.UUID, userCode, continueURL string) error
}

type Handler struct {
	logger           *slog.Logger
	authService      *auth.Service
	notifier         DeviceNotifier
	emailAuthService *emailauth.Service
	publicAPIBaseURL string
	webBaseURL       string
	cookieSigningKey []byte
	clients          map[string]oauthClient
	scopesSupported  []string
	queries          *db.Queries
	dcrConfig        oauthDCRConfig
	dcrLimiter       *registrationRateLimiter
	devicePoller     *devicePollingLimiter
}

type oauthClient struct {
	ClientID           string   `json:"client_id"`
	DisplayName        string   `json:"display_name,omitempty"`
	LogoURL            string   `json:"logo_url,omitempty"`
	ClientType         string   `json:"client_type"`
	ClientSecretSHA256 string   `json:"client_secret_sha256,omitempty"`
	RedirectURIs       []string `json:"redirect_uris"`
	Scopes             []string `json:"scopes"`
	AudienceMode       auth.OAuthAudienceMode
}

func New(logger *slog.Logger, cfg Config, authService *auth.Service, notifier DeviceNotifier) (*Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if authService == nil {
		return nil, fmt.Errorf("auth service is required")
	}
	if strings.TrimSpace(cfg.HTTP.APIPublicBaseURL) == "" {
		return nil, fmt.Errorf("public api base url is required")
	}
	if strings.TrimSpace(cfg.HTTP.OAuthCookieSigningKey) == "" {
		return nil, fmt.Errorf("cookie signing key is required")
	}
	clients, err := staticClients(cfg.HTTP.WebBaseURL)
	if err != nil {
		return nil, err
	}
	dcrCfg, err := parseOAuthDCRConfig()
	if err != nil {
		return nil, err
	}
	return &Handler{
		logger:           logger.With("component", "oauthapi"),
		authService:      authService,
		notifier:         notifier,
		emailAuthService: cfg.EmailAuth,
		publicAPIBaseURL: strings.TrimRight(strings.TrimSpace(cfg.HTTP.APIPublicBaseURL), "/"),
		webBaseURL:       strings.TrimRight(strings.TrimSpace(cfg.HTTP.WebBaseURL), "/"),
		cookieSigningKey: []byte(cfg.HTTP.OAuthCookieSigningKey),
		clients:          clients,
		scopesSupported:  collectSupportedScopes(clients, allowedDynamicScopes),
		queries:          cfg.Queries,
		dcrConfig:        dcrCfg,
		dcrLimiter:       &registrationRateLimiter{windows: map[string]registrationWindow{}},
		devicePoller:     &devicePollingLimiter{nextPolls: map[string]time.Time{}},
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if recoveredPath, ok := recoverMalformedOAuthPath(r); ok {
		http.Redirect(w, r, recoveredPath, http.StatusTemporaryRedirect)
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/.well-known/oauth-protected-resource":
		h.handleProtectedResourceMetadata(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/.well-known/oauth-protected-resource/mcp":
		h.handleProtectedResourceMetadata(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/.well-known/oauth-authorization-server":
		h.handleAuthorizationServerMetadata(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/jwks":
		h.handleJWKS(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/login":
		h.handleLoginPage(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/login/device/poll":
		h.handleLoginDevicePoll(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/authorize/handoff":
		h.handleAuthorizeHandoffPage(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/authorize/handoff/status":
		h.handleAuthorizeHandoffStatus(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/authorize/handoff/resolve":
		h.handleAuthorizeHandoffResolve(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/authorize/handoff/approve":
		h.handleAuthorizeHandoffApprove(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/authorize/handoff/deny":
		h.handleAuthorizeHandoffDeny(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/app/open":
		h.handleAppOpenPage(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/device_authorization":
		h.handleDeviceAuthorization(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/device/verify":
		h.handleDeviceVerifyPage(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/device/verify/request":
		h.handleDeviceVerifyRequest(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/device/verify/approve":
		h.handleDeviceVerifyApprove(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/device/verify/deny":
		h.handleDeviceVerifyDeny(w, r)
	case (r.Method == http.MethodGet || r.Method == http.MethodPost) && r.URL.Path == "/oauth/callback":
		h.handleOAuthCallback(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/oauth/authorize":
		h.handleAuthorize(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/token":
		h.handleToken(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/register":
		h.handleRegisterClient(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/oauth/revoke":
		h.handleRevoke(w, r)
	default:
		http.NotFound(w, r)
	}
}

func recoverMalformedOAuthPath(r *http.Request) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}
	const prefix = "/oauth/"
	path := strings.TrimSpace(r.URL.Path)
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	tail := strings.TrimPrefix(path, prefix)
	if tail == "" {
		return "", false
	}

	candidates := []string{tail}
	if rawPath := strings.TrimSpace(r.URL.RawPath); rawPath != "" && strings.HasPrefix(rawPath, prefix) {
		candidates = append(candidates, strings.TrimPrefix(rawPath, prefix))
	}

	for _, candidate := range candidates {
		decoded := strings.TrimSpace(candidate)
		for i := 0; i < 3; i++ {
			next, err := url.PathUnescape(decoded)
			if err != nil || next == decoded {
				break
			}
			decoded = next
		}
		decoded = strings.Trim(strings.TrimSpace(decoded), `"'`)
		switch {
		case strings.HasPrefix(decoded, "https:/") && !strings.HasPrefix(decoded, "https://"):
			decoded = "https://" + strings.TrimPrefix(decoded, "https:/")
		case strings.HasPrefix(decoded, "http:/") && !strings.HasPrefix(decoded, "http://"):
			decoded = "http://" + strings.TrimPrefix(decoded, "http:/")
		}
		if !strings.HasPrefix(decoded, "https://") && !strings.HasPrefix(decoded, "http://") {
			continue
		}

		embedded, err := url.Parse(decoded)
		if err != nil || embedded == nil {
			continue
		}
		if !sameHost(embedded.Host, r.Host) {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(embedded.Path), prefix) {
			continue
		}

		query := embedded.Query()
		for key, values := range r.URL.Query() {
			if _, exists := query[key]; exists {
				continue
			}
			for _, value := range values {
				query.Add(key, strings.Trim(strings.TrimSpace(value), `"'`))
			}
		}
		embedded.RawQuery = query.Encode()
		embedded.Fragment = ""
		return embedded.RequestURI(), true
	}
	return "", false
}

func sameHost(a, b string) bool {
	aHost := strings.ToLower(strings.TrimSpace(a))
	bHost := strings.ToLower(strings.TrimSpace(b))
	if aHost == "" || bHost == "" {
		return false
	}

	if parsedA, _, err := net.SplitHostPort(aHost); err == nil {
		aHost = parsedA
	}
	if parsedB, _, err := net.SplitHostPort(bHost); err == nil {
		bHost = parsedB
	}
	return aHost == bHost
}

func (h *Handler) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pubSet := h.authService.JWTService().PublicKeySet()
	buf, err := json.Marshal(pubSet)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to serialize jwks"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf)
}

func (h *Handler) handleProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	metadata := map[string]any{
		"resource":                 auth.CanonicalProtectedResource(h.publicAPIBaseURL),
		"authorization_servers":    []string{h.publicAPIBaseURL},
		"scopes_supported":         h.scopesSupported,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Koditon MCP",
	}
	writeJSON(w, http.StatusOK, metadata)
}

func (h *Handler) handleAuthorizationServerMetadata(w http.ResponseWriter, _ *http.Request) {
	metadata := map[string]any{
		"issuer":                                h.publicAPIBaseURL,
		"authorization_endpoint":                h.publicAPIBaseURL + "/oauth/authorize",
		"token_endpoint":                        h.publicAPIBaseURL + "/oauth/token",
		"device_authorization_endpoint":         h.publicAPIBaseURL + "/oauth/device_authorization",
		"jwks_uri":                              h.publicAPIBaseURL + "/oauth/jwks",
		"scopes_supported":                      h.scopesSupported,
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{grantAuthorizationCode, grantRefreshToken, grantDeviceCode, grantAppleCode, grantEmailAuthTicket, grantPasskeyAssertion},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_post", "client_secret_basic"},
		"code_challenge_methods_supported":      []string{"S256"},
		"revocation_endpoint":                   h.publicAPIBaseURL + "/oauth/revoke",
	}
	metadata["registration_endpoint"] = h.publicAPIBaseURL + "/oauth/register"
	writeJSON(w, http.StatusOK, metadata)
}

func (h *Handler) validateAuthorizeRequest(ctx context.Context, req authorizeRequest) (oauthClient, []string, error) {
	if req.ResponseType != "code" {
		return oauthClient{}, nil, fmt.Errorf("response_type must be code")
	}
	client, ok, err := h.resolveClientByID(ctx, req.ClientID)
	if err != nil {
		return oauthClient{}, nil, err
	}
	if !ok {
		return oauthClient{}, nil, fmt.Errorf("unknown client_id")
	}
	if _, err := h.resolveAudienceForClient(client, req.Resource); err != nil {
		return oauthClient{}, nil, err
	}
	if !containsExact(client.RedirectURIs, req.RedirectURI) {
		return oauthClient{}, nil, fmt.Errorf("redirect_uri is not allowed")
	}
	if req.CodeChallenge == "" {
		return oauthClient{}, nil, fmt.Errorf("code_challenge is required")
	}
	if !strings.EqualFold(req.CodeChallengeMethod, "S256") {
		return oauthClient{}, nil, fmt.Errorf("code_challenge_method must be S256")
	}
	scopes := h.normalizeRequestedScopesForClient(client, req.Scope)
	if len(scopes) == 0 {
		return oauthClient{}, nil, fmt.Errorf("scope is required")
	}
	if err := auth.ValidateRequestedScopes(scopes, client.Scopes, nil); err != nil {
		return oauthClient{}, nil, err
	}
	return client, scopes, nil
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, description string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect uri", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("error", code)
	if description != "" {
		q.Set("error_description", description)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	writeOAuthErrorWithCode(w, status, code, description, "")
}

func writeOAuthErrorWithCode(w http.ResponseWriter, status int, code, description, errorCode string) {
	payload := map[string]string{
		"error":             code,
		"error_description": description,
	}
	if strings.TrimSpace(errorCode) != "" {
		payload["error_code"] = strings.TrimSpace(errorCode)
	}
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
