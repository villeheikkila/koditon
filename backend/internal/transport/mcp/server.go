package mcpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"koditon/internal/db"
	"koditon/internal/domain/ads"
	"koditon/internal/domain/auth"
	"koditon/internal/domain/authz"
	"koditon/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxMCPRequestBodyBytes = 1 << 20 // 1 MiB

type Handler struct {
	mcpHandler               http.Handler
	authService              *auth.Service
	resourceMetadataURL      string
	openAIAppsChallengeToken string
	environment              string
	logger                   *slog.Logger
}

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func New(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger, authSvc *auth.Service) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "koditon-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{}})

	impl := &toolImpl{
		adsSvc:  ads.NewService(pool),
		queries: db.New(pool),
		config: toolImplConfig{
			webBaseURL:           cfg.WebBaseURL,
			shortcutSitemapBase:  cfg.Shortcut.SitemapBase,
			frontdoorSitemapBase: cfg.Frontdoor.SitemapBase,
		},
		logger: logger.With("component", "mcpserver"),
	}

	toolSecurityScopes := map[string][]string{
		"koditon_find_listings":                {auth.ScopeMCPCoreRead},
		"koditon_query_properties":             {auth.ScopeMCPCoreRead},
		"koditon_get_property_detail":          {auth.ScopeMCPCoreRead},
		"koditon_search_listings":              {auth.ScopeMCPCoreRead},
		"koditon_get_listing_detail":           {auth.ScopeMCPCoreRead},
		"koditon_search_transactions":          {auth.ScopeMCPCoreRead},
		"koditon_search_transactions_advanced": {auth.ScopeMCPCoreRead},
		"koditon_match_ads_from_transaction":   {auth.ScopeMCPCoreRead},
		"koditon_list_cities":                  {auth.ScopeMCPCoreRead},
		"koditon_list_available_locations":     {auth.ScopeMCPCoreRead},
		"koditon_list_categories":              {auth.ScopeMCPCoreRead},
	}

	mcp.AddTool(server, impl.findListingsTool(), impl.findListings)
	mcp.AddTool(server, impl.queryPropertiesTool(), impl.queryProperties)
	mcp.AddTool(server, impl.getPropertyDetailTool(), impl.getPropertyDetail)
	mcp.AddTool(server, impl.searchListingsTool(), impl.searchListings)
	mcp.AddTool(server, impl.getListingDetailTool(), impl.getListingDetail)
	mcp.AddTool(server, impl.searchTransactionsTool(), impl.searchTransactions)
	mcp.AddTool(server, impl.searchTransactionsAdvancedTool(), impl.searchTransactionsAdvanced)
	mcp.AddTool(server, impl.matchAdsFromTransactionTool(), impl.matchAdsFromTransaction)
	mcp.AddTool(server, impl.listCitiesTool(), impl.listCities)
	mcp.AddTool(server, impl.listAvailableLocationsTool(), impl.listAvailableLocations)
	mcp.AddTool(server, impl.listCategoriesTool(), impl.listCategories)
	registerListingsAppResource(server, cfg.APIPublicBaseURL, cfg.WebBaseURL)

	streamable := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, CrossOriginProtection: mcpCrossOriginProtection(cfg)})
	authConfigured := cfg.APIPublicBaseURL != "" && cfg.Auth.OAuthCookieKey != ""
	var base http.Handler = streamable
	if authConfigured {
		base = wrapToolsListSecuritySchemes(streamable, toolSecurityScopes)
	}

	resourceMetadataURL := strings.TrimRight(strings.TrimSpace(cfg.APIPublicBaseURL), "/") + "/.well-known/oauth-protected-resource/mcp"
	requiredAudience := ""
	if cfg.APIPublicBaseURL != "" {
		requiredAudience = auth.CanonicalProtectedResource(cfg.APIPublicBaseURL)
	}

	mcpHandler := base
	if authSvc != nil && authConfigured {
		mcpHandler = requireAuth(authSvc, []string{auth.ScopeMCPCoreRead}, requiredAudience, resourceMetadataURL, base)
	}

	return &Handler{
		mcpHandler:               mcpHandler,
		authService:              authSvc,
		resourceMetadataURL:      resourceMetadataURL,
		openAIAppsChallengeToken: strings.TrimSpace(cfg.OpenAIAppsChallengeToken),
		environment:              string(cfg.Environment),
		logger:                   logger.With("component", "mcpserver"),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	setCORSHeaders(w, r, h.environment)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/.well-known/openai-apps-challenge":
		h.handleOpenAIAppsChallenge(w)
	default:
		h.mcpHandler.ServeHTTP(w, r)
	}
}

func (h *Handler) handleOpenAIAppsChallenge(w http.ResponseWriter) {
	if h.openAIAppsChallengeToken == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, h.openAIAppsChallengeToken)
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

var allowedCORSOrigins = []string{
	"https://chatgpt.com",
	"https://platform.openai.com",
	"https://claude.ai",
}

var localMCPBrowserOrigins = []string{
	"http://127.0.0.1:6274",
	"http://127.0.0.1:6277",
	"http://localhost:6274",
	"http://localhost:6277",
}

func mcpCrossOriginProtection(cfg config.Config) *http.CrossOriginProtection {
	protection := http.NewCrossOriginProtection()
	for _, origin := range allowedCORSOrigins {
		_ = protection.AddTrustedOrigin(origin)
	}
	for _, origin := range strings.Split(cfg.CORSAllowedOrigins, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			_ = protection.AddTrustedOrigin(trimmed)
		}
	}
	if origin := originFromRawURL(cfg.APIPublicBaseURL); origin != "" {
		_ = protection.AddTrustedOrigin(origin)
	}
	if origin := originFromRawURL(cfg.WebBaseURL); origin != "" {
		_ = protection.AddTrustedOrigin(origin)
	}
	if cfg.Environment.IsDevelopment() {
		for _, origin := range localMCPBrowserOrigins {
			_ = protection.AddTrustedOrigin(origin)
		}
	}
	return protection
}

func originFromRawURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request, environment string) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	allowed := false
	if !strings.EqualFold(environment, "production") {
		allowed = true
	} else {
		for _, o := range allowedCORSOrigins {
			if strings.EqualFold(origin, o) {
				allowed = true
				break
			}
		}
	}
	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Mcp-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
}

func requireAuth(authService *auth.Service, requiredScopes []string, requiredAudience, resourceMetadataURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyCopy []byte
		if r.Body != nil {
			bodyReader := http.MaxBytesReader(w, r.Body, maxMCPRequestBodyBytes)
			var readErr error
			bodyCopy, readErr = io.ReadAll(bodyReader)
			if readErr != nil {
				var maxBytesErr *http.MaxBytesError
				if errors.As(readErr, &maxBytesErr) {
					writeMCPRequestError(w, http.StatusRequestEntityTooLarge, "request body too large")
					return
				}
				writeMCPRequestError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyCopy))
		}

		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		result, verifyErr := authz.VerifyAuthorization(r.Context(), authService, authorization, requiredScopes, requiredAudience)
		if verifyErr != nil {
			if challenge := auth.BuildBearerChallenge(verifyErr.Status, verifyErr.Message, resourceMetadataURL, requiredScopes...); challenge != "" {
				w.Header().Set("WWW-Authenticate", challenge)
			}
			writeAuthError(w, r, bodyCopy, verifyErr.Status, verifyErr.Message, w.Header().Get("WWW-Authenticate"))
			return
		}

		if bodyCopy != nil {
			r.Body = io.NopCloser(bytes.NewReader(bodyCopy))
		}
		next.ServeHTTP(w, r.WithContext(result.Context))
	})
}

func writeMCPRequestError(w http.ResponseWriter, status int, message string) {
	envelope := map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error": map[string]any{
			"code":    -32600,
			"message": message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

func writeAuthError(w http.ResponseWriter, r *http.Request, requestBody []byte, status int, message string, wwwAuthenticate string) {
	if req, ok := parseSingleMCPRequest(requestBody); ok && req.Method == "tools/call" {
		writeToolAuthError(w, status, req.ID, message, wwwAuthenticate)
		return
	}
	errData := map[string]any{}
	if meta := auth.MCPWWWAuthenticateMeta(wwwAuthenticate); meta != nil {
		errData["_meta"] = meta
	}
	envelope := map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error": map[string]any{
			"code":    -32001,
			"message": message,
			"data":    errData,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

func writeToolAuthError(w http.ResponseWriter, status int, id any, message string, wwwAuthenticate string) {
	result := map[string]any{
		"content": []map[string]string{{"type": "text", "text": message}},
		"isError": true,
	}
	if meta := auth.MCPWWWAuthenticateMeta(wwwAuthenticate); meta != nil {
		result["_meta"] = meta
	}
	envelope := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

func parseSingleMCPRequest(body []byte) (*mcpJSONRPCRequest, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] == '[' {
		return nil, false
	}
	var req mcpJSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, false
	}
	if req.Method == "" {
		return nil, false
	}
	return &req, true
}

func oauthToolSecuritySchemes(requiredScopes []string) []map[string]any {
	return []map[string]any{{"type": "oauth2", "scopes": append([]string(nil), requiredScopes...)}}
}

func wrapToolsListSecuritySchemes(next http.Handler, securityScopesByTool map[string][]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &bufferedResponse{header: make(http.Header)}
		next.ServeHTTP(rec, r)

		body := injectToolSecuritySchemes(rec.body.Bytes(), securityScopesByTool)
		for name, values := range rec.header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		if rec.status != 0 {
			w.WriteHeader(rec.status)
		}
		_, _ = w.Write(body)
	})
}

func injectToolSecuritySchemes(body []byte, securityScopesByTool map[string][]string) []byte {
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body
	}
	result, ok := envelope["result"].(map[string]any)
	if !ok {
		return body
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		return body
	}
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		toolName, _ := tool["name"].(string)
		scopes := securityScopesByTool[strings.TrimSpace(toolName)]
		if len(scopes) == 0 {
			scopes = []string{auth.ScopeMCPCoreRead}
		}
		schemes := oauthToolSecuritySchemes(scopes)
		tool["securitySchemes"] = schemes
		meta, _ := tool["_meta"].(map[string]any)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["securitySchemes"] = schemes
		tool["_meta"] = meta
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return body
	}
	return encoded
}

type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (r *bufferedResponse) Header() http.Header        { return r.header }
func (r *bufferedResponse) WriteHeader(statusCode int) { r.status = statusCode }
func (r *bufferedResponse) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

type toolImpl struct {
	adsSvc  *ads.Service
	queries *db.Queries
	config  toolImplConfig
	logger  *slog.Logger
}

func newToolResultError(msg string) *mcp.CallToolResult {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil
}
