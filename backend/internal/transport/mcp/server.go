package mcpserver

import (
	"bytes"
	"context"
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
	"koditon/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxMCPRequestBodyBytes = 1 << 20 // 1 MiB
const mcpCodeUnauthorized = -32001

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

	runtime := newMCPRuntime(pool, cfg, logger)

	streamable := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return runtime.server
	}, &mcp.StreamableHTTPOptions{Stateless: true, CrossOriginProtection: mcpCrossOriginProtection(cfg)})
	authConfigured := cfg.APIPublicBaseURL != "" && cfg.Auth.OAuthCookieKey != ""
	var base http.Handler = streamable
	if authConfigured {
		base = wrapToolsListSecuritySchemes(streamable, runtime.toolSecurityScopes)
	}

	resourceMetadataURL := strings.TrimRight(strings.TrimSpace(cfg.APIPublicBaseURL), "/") + "/.well-known/oauth-protected-resource/mcp"
	requiredAudience := ""
	if cfg.APIPublicBaseURL != "" {
		protectedResource := mcpProtectedResourceMetadata(cfg.APIPublicBaseURL, []string{auth.ScopeMCPCoreRead})
		requiredAudience = protectedResource.Resource
	}

	mcpHandler := base
	if authSvc != nil && authConfigured {
		bearerOptions := mcpBearerTokenOptions(resourceMetadataURL, []string{auth.ScopeMCPCoreRead})
		mcpHandler = requireAuth(authSvc, bearerOptions.Scopes, requiredAudience, bearerOptions.ResourceMetadataURL, base)
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
	verifier := mcpTokenVerifier(authService, requiredAudience)
	options := mcpBearerTokenOptions(resourceMetadataURL, requiredScopes)
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
		nextCalled := false
		authWriter := &mcpAuthResponseWriter{ResponseWriter: w, header: make(http.Header), nextCalled: func() bool { return nextCalled }}
		protected := mcpauth.RequireBearerToken(verifier, options)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			if bodyCopy != nil {
				r.Body = io.NopCloser(bytes.NewReader(bodyCopy))
			}
			next.ServeHTTP(w, r.WithContext(mcpAuthContext(r.Context())))
		}))
		protected.ServeHTTP(authWriter, r)
		if !nextCalled && authWriter.status != 0 {
			if challenge := authWriter.header.Get("WWW-Authenticate"); challenge != "" {
				w.Header().Set("WWW-Authenticate", challenge)
			}
			writeAuthError(w, r, bodyCopy, authWriter.status, strings.TrimSpace(authWriter.body.String()), authWriter.header.Get("WWW-Authenticate"))
			return
		}
	})
}

const mcpTokenInfoClaimsKey = "koditon.auth.claims"

func mcpTokenVerifier(authService *auth.Service, requiredAudience string) mcpauth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		claims, err := authService.VerifyAccessToken(ctx, token)
		if err != nil {
			return nil, mcpInvalidTokenError(mapAccessTokenVerifierMessage(err))
		}
		if strings.TrimSpace(requiredAudience) != "" && strings.TrimSpace(claims.Audience) != strings.TrimSpace(requiredAudience) {
			return nil, mcpInvalidTokenError("invalid token audience")
		}
		return &mcpauth.TokenInfo{
			Scopes:     auth.LimitScopes(auth.ScopesForRoles(claims.Roles), claims.Scopes),
			Expiration: claims.ExpiresAt,
			UserID:     claims.UserID.String(),
			Extra: map[string]any{
				mcpTokenInfoClaimsKey: claims,
				"session_id":          claims.SessionID.String(),
				"audience":            claims.Audience,
			},
		}, nil
	}
}

func mcpAuthContext(ctx context.Context) context.Context {
	tokenInfo := mcpauth.TokenInfoFromContext(ctx)
	if tokenInfo == nil || tokenInfo.Extra == nil {
		return ctx
	}
	claims, ok := tokenInfo.Extra[mcpTokenInfoClaimsKey].(*auth.AccessTokenClaims)
	if !ok {
		return ctx
	}
	return auth.WithClaims(ctx, claims)
}

func mcpInvalidTokenError(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return mcpauth.ErrInvalidToken
	}
	return fmt.Errorf("%s: %w", message, mcpauth.ErrInvalidToken)
}

func mapAccessTokenVerifierMessage(err error) string {
	switch {
	case errors.Is(err, auth.ErrTokenExpired):
		return "token expired"
	case errors.Is(err, auth.ErrSessionRevoked):
		return "session revoked"
	case errors.Is(err, auth.ErrTokenRevoked):
		return "token revoked"
	case errors.Is(err, auth.ErrInvalidToken):
		return "invalid token"
	default:
		return "invalid token"
	}
}

func writeMCPRequestError(w http.ResponseWriter, status int, message string) {
	envelope := map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error": map[string]any{
			"code":    jsonrpc.CodeInvalidRequest,
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
			"code":    mcpCodeUnauthorized,
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

type mcpAuthResponseWriter struct {
	http.ResponseWriter
	header     http.Header
	body       bytes.Buffer
	status     int
	nextCalled func() bool
}

func (w *mcpAuthResponseWriter) Header() http.Header {
	if w.nextCalled != nil && w.nextCalled() {
		return w.ResponseWriter.Header()
	}
	return w.header
}

func (w *mcpAuthResponseWriter) WriteHeader(statusCode int) {
	if w.nextCalled != nil && w.nextCalled() {
		copyHeader(w.ResponseWriter.Header(), w.header)
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}
	w.status = statusCode
}

func (w *mcpAuthResponseWriter) Write(data []byte) (int, error) {
	if w.nextCalled != nil && w.nextCalled() {
		copyHeader(w.ResponseWriter.Header(), w.header)
		return w.ResponseWriter.Write(data)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

func copyHeader(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
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
