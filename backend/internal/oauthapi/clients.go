package oauthapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"koditon-go/internal/auth"
	db "koditon-go/internal/db"
)

type oauthDCRConfig struct {
	allowedRedirectURIs map[string]struct{}
	allowedScopes       []string
	allowedCIDRs        []netip.Prefix
	rateLimitPerMinute  int
}

type registrationRateLimiter struct {
	mu      sync.Mutex
	windows map[string]registrationWindow
}

type registrationWindow struct {
	minute int64
	count  int
}

type devicePollingLimiter struct {
	mu        sync.Mutex
	nextPolls map[string]time.Time
}

type oauthDynamicClientMetadata struct {
	LogoURI string `json:"logo_uri,omitempty"`
}

const defaultChatGPTLogoURI = "https://chatgpt.com/favicon.ico"

func (h *Handler) handleRegisterClient(w http.ResponseWriter, r *http.Request) {
	if h.queries == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "server_error", "dynamic client registration is unavailable")
		return
	}
	if !h.allowRegistrationRequest(r) {
		writeOAuthError(w, http.StatusTooManyRequests, "slow_down", "registration rate limit exceeded")
		return
	}
	if allowed := h.isRegistrationSourceAllowed(r); !allowed {
		writeOAuthError(w, http.StatusForbidden, "access_denied", "registration source is not allowed")
		return
	}
	var payload struct {
		RedirectURIs            []string `json:"redirect_uris"`
		ClientName              string   `json:"client_name"`
		LogoURI                 string   `json:"logo_uri"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		Scope                   string   `json:"scope"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if len(payload.RedirectURIs) == 0 {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "redirect_uris is required")
		return
	}

	redirectURIs := make([]string, 0, len(payload.RedirectURIs))
	for _, rawURI := range payload.RedirectURIs {
		normalizedURI, err := h.normalizeDynamicRedirectURI(rawURI)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", err.Error())
			return
		}
		redirectURIs = append(redirectURIs, normalizedURI)
	}

	method := strings.TrimSpace(payload.TokenEndpointAuthMethod)
	if method == "" {
		method = "none"
	}
	if method != "none" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "token_endpoint_auth_method must be none")
		return
	}

	clientScopes := defaultDCRClientScopes(h.dcrConfig.allowedScopes)
	if scopeText := strings.TrimSpace(payload.Scope); scopeText != "" {
		clientScopes = strings.Fields(scopeText)
	}
	if err := h.validateDCRRequestedScopes(clientScopes); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error())
		return
	}

	clientIDToken, err := randomToken(24)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to generate client_id")
		return
	}
	clientID := "dyn_" + clientIDToken
	clientType := "public"
	clientName := strings.TrimSpace(payload.ClientName)
	logoURI, err := resolvedDynamicClientLogoURI(payload.LogoURI, redirectURIs)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
		return
	}
	metadataJSON, err := json.Marshal(oauthDynamicClientMetadata{LogoURI: logoURI})
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to encode client metadata")
		return
	}

	row, err := h.queries.CreateOAuthDynamicClient(r.Context(), db.CreateOAuthDynamicClientParams{
		OauthDynamicClientID:                      &clientID,
		OauthDynamicClientType:                    &clientType,
		OauthDynamicClientRedirectUris:            redirectURIs,
		OauthDynamicClientScopes:                  clientScopes,
		OauthDynamicClientTokenEndpointAuthMethod: &method,
		OauthDynamicClientName:                    stringOrNil(clientName),
		OauthDynamicClientMetadata:                metadataJSON,
	})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create oauth dynamic client failed", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to register client")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  row.OauthDynamicClientID,
		"client_id_issued_at":        row.OauthDynamicClientIssuedAt.Unix(),
		"client_name":                clientName,
		"logo_uri":                   logoURI,
		"redirect_uris":              row.OauthDynamicClientRedirectUris,
		"token_endpoint_auth_method": row.OauthDynamicClientTokenEndpointAuthMethod,
		"grant_types":                []string{grantAuthorizationCode, grantRefreshToken},
		"response_types":             []string{"code"},
		"scope":                      strings.Join(row.OauthDynamicClientScopes, " "),
	})
}

func (h *Handler) resolveClientByID(ctx context.Context, clientID string) (oauthClient, bool, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return oauthClient{}, false, nil
	}
	if client, ok := h.clients[clientID]; ok {
		return client, true, nil
	}
	if h.queries == nil {
		return oauthClient{}, false, nil
	}
	row, err := h.queries.GetOAuthDynamicClientByID(ctx, &clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return oauthClient{}, false, nil
		}
		return oauthClient{}, false, fmt.Errorf("get dynamic oauth client: %w", err)
	}
	effectiveScopes := effectiveDynamicClientScopes(row.OauthDynamicClientScopes, h.dcrConfig.allowedScopes)
	return oauthClient{
		ClientID:     row.OauthDynamicClientID,
		DisplayName:  resolveOAuthClientDisplayName(row.OauthDynamicClientID, row.OauthDynamicClientName),
		LogoURL:      resolveOAuthClientLogoURL(row.OauthDynamicClientID, row.OauthDynamicClientMetadata),
		ClientType:   strings.ToLower(strings.TrimSpace(row.OauthDynamicClientType)),
		RedirectURIs: append([]string(nil), row.OauthDynamicClientRedirectUris...),
		Scopes:       effectiveScopes,
		AudienceMode: auth.OAuthAudienceModeProtectedResource,
	}, true, nil
}

func (h *Handler) resolveAudienceForClient(client oauthClient, resource string) (string, error) {
	return auth.ResolveOAuthAudience(h.publicAPIBaseURL, client.AudienceMode, resource)
}

func (h *Handler) validateClientSecret(client oauthClient, providedSecret string) bool {
	if client.ClientType != "confidential" {
		return true
	}
	if providedSecret == "" || client.ClientSecretSHA256 == "" {
		return false
	}
	providedHash := hashText(providedSecret)
	return subtle.ConstantTimeCompare([]byte(providedHash), []byte(client.ClientSecretSHA256)) == 1
}

func staticClients(webBaseURL string) (map[string]oauthClient, error) {
	webCallback, err := joinURLPath(webBaseURL, "/oauth/callback")
	if err != nil {
		return nil, err
	}

	appClientScopes := []string{
		auth.ScopeCoreRead,
		auth.ScopeMCPCoreRead,
		auth.ScopeProfileRead,
		auth.ScopeProfileWrite,
	}

	clients := []oauthClient{
		{
			ClientID:     "koditon-apple",
			DisplayName:  resolveOAuthClientDisplayName("koditon-apple", nil),
			LogoURL:      resolveOAuthClientLogoURL("koditon-apple", nil),
			ClientType:   "public",
			RedirectURIs: []string{"koditon://oauth/callback"},
			Scopes:       appClientScopes,
			AudienceMode: auth.OAuthAudienceModeAPI,
		},
		{
			ClientID:     "koditon-cli",
			DisplayName:  resolveOAuthClientDisplayName("koditon-cli", nil),
			LogoURL:      resolveOAuthClientLogoURL("koditon-cli", nil),
			ClientType:   "public",
			RedirectURIs: []string{"http://localhost:8484"},
			Scopes:       appClientScopes,
			AudienceMode: auth.OAuthAudienceModeAPI,
		},
		{
			ClientID:     "koditon-web",
			DisplayName:  resolveOAuthClientDisplayName("koditon-web", nil),
			LogoURL:      resolveOAuthClientLogoURL("koditon-web", nil),
			ClientType:   "public",
			RedirectURIs: []string{webCallback},
			Scopes:       appClientScopes,
			AudienceMode: auth.OAuthAudienceModeAPI,
		},
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("at least one oauth client is required")
	}
	byID := make(map[string]oauthClient, len(clients))
	for _, client := range clients {
		client.ClientID = strings.TrimSpace(client.ClientID)
		if client.ClientID == "" {
			return nil, fmt.Errorf("oauth client id is required")
		}
		if len(client.RedirectURIs) == 0 {
			return nil, fmt.Errorf("oauth client %s must define redirect_uris", client.ClientID)
		}
		if len(client.Scopes) == 0 {
			return nil, fmt.Errorf("oauth client %s must define scopes", client.ClientID)
		}
		if client.ClientType == "" {
			client.ClientType = "public"
		}
		client.ClientType = strings.ToLower(strings.TrimSpace(client.ClientType))
		if client.ClientType != "public" && client.ClientType != "confidential" {
			return nil, fmt.Errorf("oauth client %s has invalid client_type", client.ClientID)
		}
		if client.ClientType == "confidential" && strings.TrimSpace(client.ClientSecretSHA256) == "" {
			return nil, fmt.Errorf("oauth confidential client %s requires client_secret_sha256", client.ClientID)
		}
		if _, exists := byID[client.ClientID]; exists {
			return nil, fmt.Errorf("duplicate oauth client id %s", client.ClientID)
		}
		byID[client.ClientID] = client
	}
	return byID, nil
}

func joinURLPath(base, path string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", base, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base url %q", base)
	}
	u.Path = path
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func collectSupportedScopes(clients map[string]oauthClient, additionalScopes []string) []string {
	if len(clients) == 0 && len(additionalScopes) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	scopes := make([]string, 0)
	for _, client := range clients {
		for _, scope := range client.Scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			scopes = append(scopes, scope)
		}
	}
	for _, scope := range additionalScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	return scopes
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func (h *Handler) normalizeDynamicRedirectURI(raw string) (string, error) {
	redirectURI := strings.TrimSpace(raw)
	if redirectURI == "" {
		return "", fmt.Errorf("redirect uri is required")
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect uri")
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid redirect uri")
	}
	if strings.ToLower(strings.TrimSpace(u.Scheme)) != "https" {
		return "", fmt.Errorf("redirect uri scheme must be https")
	}
	if !h.isAllowedDynamicRedirectURI(redirectURI, u) {
		return "", fmt.Errorf("redirect uri is not allowed")
	}
	u.RawFragment = ""
	if normalized := u.String(); normalized != "" {
		return normalized, nil
	}
	return "", fmt.Errorf("invalid redirect uri")
}

func (h *Handler) isAllowedDynamicRedirectURI(raw string, parsed *url.URL) bool {
	if _, ok := h.dcrConfig.allowedRedirectURIs[raw]; ok {
		return true
	}
	if parsed == nil {
		return false
	}
	return isChatGPTConnectorOAuthRedirect(parsed)
}

func isChatGPTConnectorOAuthRedirect(u *url.URL) bool {
	if u == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(u.Scheme), "https") {
		return false
	}
	if strings.TrimSpace(u.Port()) != "" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(u.Hostname()), "chatgpt.com") {
		return false
	}
	path := strings.TrimSpace(u.Path)
	return strings.HasPrefix(path, "/connector/oauth/") && len(path) > len("/connector/oauth/")
}

func (h *Handler) validateDCRRequestedScopes(requested []string) error {
	if len(requested) == 0 {
		return fmt.Errorf("scope is required")
	}
	return auth.ValidateRequestedScopes(requested, h.dcrConfig.allowedScopes, nil)
}

func defaultDCRClientScopes(allowed []string) []string {
	scopes := make([]string, 0, len(allowed))
	for _, scope := range allowed {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		return []string{auth.ScopeMCPCoreRead}
	}
	return scopes
}

func resolveOAuthClientDisplayName(clientID string, dynamicName *string) string {
	if dynamicName != nil {
		if trimmed := strings.TrimSpace(*dynamicName); trimmed != "" {
			return trimmed
		}
	}
	if metadata, ok := auth.OAuthClientMetadataForID(clientID); ok {
		if trimmed := strings.TrimSpace(metadata.DisplayName); trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(clientID)
}

func resolveOAuthClientLogoURL(clientID string, dynamicMetadata []byte) string {
	if metadata, ok := auth.OAuthClientMetadataForID(clientID); ok {
		if trimmed := strings.TrimSpace(metadata.LogoURL); trimmed != "" {
			return trimmed
		}
	}
	if len(dynamicMetadata) == 0 {
		return ""
	}
	var metadata oauthDynamicClientMetadata
	if err := json.Unmarshal(dynamicMetadata, &metadata); err != nil {
		return ""
	}
	return strings.TrimSpace(metadata.LogoURI)
}

func resolvedDynamicClientLogoURI(raw string, redirectURIs []string) (string, error) {
	logoURI := strings.TrimSpace(raw)
	if logoURI != "" {
		normalized, err := normalizeDynamicClientLogoURI(logoURI)
		if err != nil {
			return "", err
		}
		return normalized, nil
	}
	if logoURI := defaultDynamicClientLogoURI(redirectURIs); logoURI != "" {
		return logoURI, nil
	}
	return "", nil
}

func defaultDynamicClientLogoURI(redirectURIs []string) string {
	for _, raw := range redirectURIs {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		host := strings.ToLower(strings.TrimSpace(u.Hostname()))
		switch host {
		case "chatgpt.com", "chat.openai.com", "platform.openai.com":
			return defaultChatGPTLogoURI
		}
	}
	return ""
}

func normalizeDynamicClientLogoURI(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil {
		return "", fmt.Errorf("logo_uri must be a valid https URL")
	}
	if !strings.EqualFold(strings.TrimSpace(u.Scheme), "https") || strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("logo_uri must be a valid https URL")
	}
	if !isTrustedDynamicClientLogoHost(u) {
		return "", fmt.Errorf("logo_uri host is not allowed")
	}
	u.RawFragment = ""
	return u.String(), nil
}

func isTrustedDynamicClientLogoHost(u *url.URL) bool {
	if u == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(u.Hostname())) {
	case "chatgpt.com", "chat.openai.com", "platform.openai.com", "openai.com":
		return true
	default:
		return false
	}
}

func effectiveDynamicClientScopes(stored []string, allowed []string) []string {
	merged := make([]string, 0, len(stored)+len(allowed))
	seen := make(map[string]struct{}, len(stored)+len(allowed))
	appendScopes := func(values []string) {
		for _, scope := range values {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			if _, ok := seen[scope]; ok {
				continue
			}
			seen[scope] = struct{}{}
			merged = append(merged, scope)
		}
	}
	appendScopes(stored)
	appendScopes(defaultDCRClientScopes(allowed))
	return merged
}

func isDynamicOAuthClientID(clientID string) bool {
	return strings.HasPrefix(strings.TrimSpace(clientID), "dyn_")
}

func (h *Handler) normalizeRequestedScopesForClient(client oauthClient, requested []string) []string {
	scopes := append([]string(nil), requested...)
	if len(scopes) == 0 {
		scopes = append(scopes, client.Scopes...)
	}
	if isDynamicOAuthClientID(client.ClientID) {
		return effectiveDynamicClientScopes(scopes, h.dcrConfig.allowedScopes)
	}
	return scopes
}

func (h *Handler) allowRegistrationRequest(r *http.Request) bool {
	return h.dcrLimiter.allow(registrationClientIP(r), h.dcrConfig.rateLimitPerMinute)
}

func (h *Handler) isRegistrationSourceAllowed(r *http.Request) bool {
	if len(h.dcrConfig.allowedCIDRs) == 0 {
		return true
	}
	ipText := registrationClientIP(r)
	addr, err := netip.ParseAddr(ipText)
	if err != nil {
		return false
	}
	for _, prefix := range h.dcrConfig.allowedCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func registrationClientIP(r *http.Request) string {
	remoteIP := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteIP); err == nil && host != "" {
		remoteIP = strings.TrimSpace(host)
	}
	if remoteIP == "" {
		return ""
	}
	// Trust forwarding headers only when request comes from a local trusted proxy.
	if isTrustedRegistrationProxy(remoteIP) {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 {
				candidate := strings.TrimSpace(parts[0])
				if host, _, err := net.SplitHostPort(candidate); err == nil && strings.TrimSpace(host) != "" {
					candidate = strings.TrimSpace(host)
				}
				if addr, err := netip.ParseAddr(candidate); err == nil {
					return addr.String()
				}
			}
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			if host, _, err := net.SplitHostPort(realIP); err == nil && strings.TrimSpace(host) != "" {
				realIP = strings.TrimSpace(host)
			}
			if addr, err := netip.ParseAddr(realIP); err == nil {
				return addr.String()
			}
		}
	}
	return remoteIP
}

func isTrustedRegistrationProxy(remoteIP string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(remoteIP))
	if err != nil {
		return false
	}
	return addr.IsLoopback()
}

func (l *registrationRateLimiter) allow(key string, limit int) bool {
	if limit <= 0 {
		return true
	}
	nowMinute := time.Now().Unix() / 60
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windows == nil {
		l.windows = map[string]registrationWindow{}
	}
	entry := l.windows[key]
	if entry.minute != nowMinute {
		entry.minute = nowMinute
		entry.count = 0
	}
	if entry.count >= limit {
		l.windows[key] = entry
		return false
	}
	entry.count++
	l.windows[key] = entry

	for k, v := range l.windows {
		if v.minute < nowMinute-2 {
			delete(l.windows, k)
		}
	}
	return true
}

func (l *devicePollingLimiter) Allow(clientID, deviceCode string, interval, slowDown time.Duration) bool {
	if l == nil {
		return true
	}
	key := strings.TrimSpace(clientID) + ":" + strings.TrimSpace(deviceCode)
	if key == ":" {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.nextPolls == nil {
		l.nextPolls = map[string]time.Time{}
	}
	nextAllowed := l.nextPolls[key]
	if !nextAllowed.IsZero() && now.Before(nextAllowed) {
		l.nextPolls[key] = nextAllowed.Add(slowDown)
		return false
	}
	l.nextPolls[key] = now.Add(interval)
	for k, allowedAt := range l.nextPolls {
		if allowedAt.Before(now.Add(-1 * time.Hour)) {
			delete(l.nextPolls, k)
		}
	}
	return true
}

func parseOAuthDCRConfig() (oauthDCRConfig, error) {
	redirectSet := make(map[string]struct{})
	for _, value := range allowedDynamicRedirectURIs {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		redirectSet[trimmed] = struct{}{}
	}
	if len(redirectSet) == 0 {
		return oauthDCRConfig{}, fmt.Errorf("dynamic registration redirect allowlist is empty")
	}

	allowedScopes := slices.Clone(allowedDynamicScopes)
	if err := auth.ValidateRequestedScopes(allowedScopes, auth.ScopesForRoles(nil), nil); err != nil {
		return oauthDCRConfig{}, fmt.Errorf("invalid dcr allowed scopes: %w", err)
	}

	var cidrs []netip.Prefix
	for _, value := range allowedDynamicCIDRs {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(trimmed)
		if err != nil {
			return oauthDCRConfig{}, fmt.Errorf("invalid dcr cidr %q: %w", trimmed, err)
		}
		cidrs = append(cidrs, prefix)
	}

	return oauthDCRConfig{
		allowedRedirectURIs: redirectSet,
		allowedScopes:       allowedScopes,
		allowedCIDRs:        cidrs,
		rateLimitPerMinute:  dcrRateLimitPerMinute,
	}, nil
}

func stringOrNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func randomToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("size must be positive")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
