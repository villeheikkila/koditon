package oauthapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/danielgtaylor/huma/v2"

	"koditon-go/internal/auth"
	"koditon-go/internal/openapiutil"
)

const (
	tagOAuth = "OAuth"
	tagAuth  = "Auth"
)

const (
	responseOAuthError              = "OAuthError"
	responseOAuthTokenSuccess       = "OAuthTokenSuccess"
	responseOAuthDeviceAuthSuccess  = "OAuthDeviceAuthorizationSuccess"
	responseOAuthJWKSSuccess        = "OAuthJWKS"
	responseOAuthResourceMeta       = "OAuthProtectedResourceMetadata"
	responseOAuthServerMeta         = "OAuthAuthorizationServerMetadata"
	responseOAuthClientRegistration = "OAuthClientRegistrationSuccess"
	requestBodyOAuthToken           = "OAuthTokenRequest"
	requestBodyOAuthDeviceAuth      = "OAuthDeviceAuthorizationRequest"
	requestBodyOAuthClientRegister  = "OAuthClientRegistrationRequest"
	requestBodyOAuthRevoke          = "OAuthRevokeRequest"
)

// RegisterRoutes registers OAuth machine endpoints for unified OpenAPI output.
// Browser-facing OAuth pages remain mounted directly on ServeMux.
func (h *Handler) RegisterRoutes(api huma.API) {
	if h == nil || api == nil {
		return
	}

	h.ensureOpenAPIComponents(api.OpenAPI())

	operations := []struct {
		op          *huma.Operation
		readRawBody bool
	}{
		{
			op: &huma.Operation{
				Method:      http.MethodGet,
				Path:        "/.well-known/oauth-protected-resource",
				OperationID: "oauth-protected-resource-metadata-root-get",
				Summary:     "Get OAuth protected resource metadata (root)",
				Tags:        []string{tagOAuth, tagAuth},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthResourceMeta},
				},
			},
		},
		{
			op: &huma.Operation{
				Method:      http.MethodGet,
				Path:        "/.well-known/oauth-protected-resource/mcp",
				OperationID: "oauth-protected-resource-metadata-get",
				Summary:     "Get OAuth protected resource metadata",
				Tags:        []string{tagOAuth, tagAuth},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthResourceMeta},
				},
			},
		},
		{
			op: &huma.Operation{
				Method:      http.MethodGet,
				Path:        "/.well-known/oauth-authorization-server",
				OperationID: "oauth-authorization-server-metadata-get",
				Summary:     "Get OAuth authorization server metadata",
				Tags:        []string{tagOAuth, tagAuth},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthServerMeta},
				},
			},
		},
		{
			op: &huma.Operation{
				Method:      http.MethodGet,
				Path:        "/oauth/jwks",
				OperationID: "oauth-jwks-get",
				Summary:     "Get JSON Web Key Set",
				Tags:        []string{tagOAuth, tagAuth},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthJWKSSuccess},
				},
			},
		},
		{
			op: &huma.Operation{
				Method:      http.MethodPost,
				Path:        "/oauth/device_authorization",
				OperationID: "oauth-device-authorization-create",
				Summary:     "Create OAuth device authorization request",
				Tags:        []string{tagOAuth, tagAuth},
				RequestBody: &huma.RequestBody{Ref: "#/components/requestBodies/" + requestBodyOAuthDeviceAuth},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthDeviceAuthSuccess},
					"400": {Ref: "#/components/responses/" + responseOAuthError},
					"401": {Ref: "#/components/responses/" + responseOAuthError},
					"500": {Ref: "#/components/responses/" + responseOAuthError},
				},
			},
			readRawBody: true,
		},
		{
			op: &huma.Operation{
				Method:      http.MethodPost,
				Path:        "/oauth/token",
				OperationID: "oauth-token-create",
				Summary:     "Exchange OAuth token grants",
				Tags:        []string{tagOAuth, tagAuth},
				RequestBody: &huma.RequestBody{Ref: "#/components/requestBodies/" + requestBodyOAuthToken},
				Responses: map[string]*huma.Response{
					"200": {Ref: "#/components/responses/" + responseOAuthTokenSuccess},
					"400": {Ref: "#/components/responses/" + responseOAuthError},
					"401": {Ref: "#/components/responses/" + responseOAuthError},
					"500": {Ref: "#/components/responses/" + responseOAuthError},
				},
			},
			readRawBody: true,
		},
		{
			op: &huma.Operation{
				Method:      http.MethodPost,
				Path:        "/oauth/register",
				OperationID: "oauth-client-register-create",
				Summary:     "Register OAuth client dynamically",
				Description: "Registers an external OAuth client for authorization code + PKCE flows.",
				Tags:        []string{tagOAuth, tagAuth},
				RequestBody: &huma.RequestBody{Ref: "#/components/requestBodies/" + requestBodyOAuthClientRegister},
				Responses: map[string]*huma.Response{
					"201": {Ref: "#/components/responses/" + responseOAuthClientRegistration},
					"400": {Ref: "#/components/responses/" + responseOAuthError},
					"403": {Ref: "#/components/responses/" + responseOAuthError},
					"429": {Ref: "#/components/responses/" + responseOAuthError},
					"500": {Ref: "#/components/responses/" + responseOAuthError},
				},
			},
			readRawBody: true,
		},
		{
			op: &huma.Operation{
				Method:      http.MethodPost,
				Path:        "/oauth/revoke",
				OperationID: "oauth-revoke-create",
				Summary:     "Revoke an OAuth token (RFC 7009)",
				Description: "Revokes a refresh token. Access tokens (JWTs) cannot be revoked server-side and expire naturally. Returns 200 OK even if the token is already invalid.",
				Tags:        []string{tagOAuth, tagAuth},
				RequestBody: &huma.RequestBody{Ref: "#/components/requestBodies/" + requestBodyOAuthRevoke},
				Responses: map[string]*huma.Response{
					"200": {Description: "Token revoked successfully (or was already invalid)."},
					"400": {Ref: "#/components/responses/" + responseOAuthError},
					"500": {Ref: "#/components/responses/" + responseOAuthError},
				},
			},
			readRawBody: true,
		},
	}

	for _, item := range operations {
		openapiutil.ApplyOperationPolicyDocumentation(item.op)
		api.OpenAPI().AddOperation(item.op)
		api.Adapter().Handle(item.op, h.passthroughOperation(item.readRawBody))
	}
}

func (h *Handler) passthroughOperation(readRawBody bool) func(huma.Context) {
	return func(ctx huma.Context) {
		var rawBody []byte
		if readRawBody {
			bodyBytes, err := io.ReadAll(ctx.BodyReader())
			if err != nil {
				writeOAuthErrorToHuma(ctx, http.StatusBadRequest, "invalid_request", "invalid form body")
				return
			}
			rawBody = bodyBytes
		}
		h.serveViaHTTP(ctx, rawBody)
	}
}

func (h *Handler) serveViaHTTP(ctx huma.Context, rawBody []byte) {
	requestURL := ctx.URL()
	body := io.Reader(http.NoBody)
	if len(rawBody) > 0 {
		body = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequestWithContext(ctx.Context(), ctx.Method(), requestURL.String(), body)
	if err != nil {
		writeOAuthErrorToHuma(ctx, http.StatusInternalServerError, "server_error", "failed to process request")
		return
	}
	req.URL = &requestURL
	req.RemoteAddr = ctx.RemoteAddr()
	ctx.EachHeader(func(name, value string) {
		req.Header.Add(name, value)
	})
	if len(rawBody) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer func() { _ = res.Body.Close() }()
	for name, values := range res.Header {
		for _, value := range values {
			ctx.AppendHeader(name, value)
		}
	}
	ctx.SetStatus(res.StatusCode)
	_, _ = io.Copy(ctx.BodyWriter(), res.Body)
}

func writeOAuthErrorToHuma(ctx huma.Context, status int, code, description string) {
	payload := map[string]string{
		"error":             code,
		"error_description": description,
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(status)
	_ = json.NewEncoder(ctx.BodyWriter()).Encode(payload)
}

func stringProperty(description string) *huma.Schema {
	return &huma.Schema{Type: "string", Description: description}
}

func tokenGrantSchema(grantType string, requiredFields ...string) *huma.Schema {
	properties := map[string]*huma.Schema{
		"grant_type": {
			Type: "string",
			Enum: []any{grantType},
		},
	}
	for _, field := range requiredFields {
		properties[field] = &huma.Schema{Type: "string"}
	}
	for _, optionalField := range []string{"client_id", "client_secret", "scope", "resource", "redirect_uri", "code_verifier", "nonce"} {
		if _, exists := properties[optionalField]; !exists {
			properties[optionalField] = &huma.Schema{Type: "string"}
		}
	}
	return &huma.Schema{
		Type:       "object",
		Properties: properties,
		Required:   append([]string{"grant_type"}, requiredFields...),
	}
}

func tokenRequestSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"grant_type": {
				Type: "string",
				Enum: []any{
					grantAuthorizationCode,
					grantRefreshToken,
					grantDeviceCode,
				},
			},
			"client_id":     stringProperty("OAuth client ID."),
			"client_secret": stringProperty("OAuth client secret for confidential clients."),
			"code":          stringProperty("Authorization code."),
			"redirect_uri":  stringProperty("Redirect URI matching authorization request."),
			"resource":      stringProperty("Protected resource identifier from discovery metadata."),
			"code_verifier": stringProperty("PKCE verifier."),
			"refresh_token": stringProperty("Refresh token."),
			"device_code":   stringProperty("Device code."),
			"scope":         stringProperty("Requested scopes separated by spaces."),
		},
		Required: []string{"grant_type"},
		Discriminator: &huma.Discriminator{
			PropertyName: "grant_type",
		},
		OneOf: []*huma.Schema{
			tokenGrantSchema(grantAuthorizationCode, "client_id", "code", "redirect_uri", "code_verifier"),
			tokenGrantSchema(grantRefreshToken, "refresh_token"),
			tokenGrantSchema(grantDeviceCode, "device_code"),
		},
	}
}

func deviceAuthorizationRequestSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"client_id":     stringProperty("OAuth client ID."),
			"client_secret": stringProperty("OAuth client secret for confidential clients."),
			"scope":         stringProperty("Requested scopes separated by spaces."),
			"resource":      stringProperty("Protected resource identifier from discovery metadata."),
		},
		Required: []string{"client_id"},
	}
}

func tokenResponseSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"token_type":               {Type: "string", Description: "Token type."},
			"access_token":             {Type: "string", Description: "Bearer access token."},
			"expires_in":               {Type: "integer", Format: "int64", Description: "Access token expiration in seconds."},
			"refresh_token":            {Type: "string", Description: "Refresh token."},
			"refresh_token_expires_in": {Type: "integer", Format: "int64", Description: "Refresh token expiration in seconds."},
			"scope":                    {Type: "string", Description: "Granted scopes separated by spaces."},
			"session_id":               {Type: "string", Format: "uuid", Description: "Associated device session identifier when available."},
		},
		Required: []string{"token_type", "access_token", "expires_in", "refresh_token", "refresh_token_expires_in", "scope"},
	}
}

func oauthErrorSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"error":             {Type: "string", Description: "OAuth error code."},
			"error_description": {Type: "string", Description: "Human-readable OAuth error description."},
			"error_code":        {Type: "string", Description: "Optional app-specific error code."},
		},
		Required: []string{"error", "error_description"},
	}
}

func deviceAuthorizationResponseSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"device_code":               {Type: "string", Description: "Device code for token polling."},
			"user_code":                 {Type: "string", Description: "User-entered verification code."},
			"verification_uri":          {Type: "string", Description: "Verification URL for user interaction."},
			"verification_uri_complete": {Type: "string", Description: "Verification URL with user code prefilled."},
			"expires_in":                {Type: "integer", Format: "int64", Description: "Expiration in seconds."},
			"interval":                  {Type: "integer", Format: "int64", Description: "Recommended polling interval in seconds."},
		},
		Required: []string{"device_code", "user_code", "verification_uri", "verification_uri_complete", "expires_in", "interval"},
	}
}

func jwksSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"keys": {
				Type:  "array",
				Items: &huma.Schema{Type: "object", AdditionalProperties: true},
			},
		},
		Required: []string{"keys"},
	}
}

func protectedResourceMetadataSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"resource": {Type: "string", Description: "Protected resource identifier."},
			"authorization_servers": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"scopes_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"bearer_methods_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"resource_name": {Type: "string", Description: "Display name for the protected resource."},
		},
		Required: []string{"resource", "authorization_servers", "scopes_supported", "bearer_methods_supported", "resource_name"},
	}
}

func authorizationServerMetadataSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"issuer":                 {Type: "string", Description: "Authorization server issuer."},
			"authorization_endpoint": {Type: "string", Description: "Authorization endpoint URL."},
			"token_endpoint":         {Type: "string", Description: "Token endpoint URL."},
			"registration_endpoint":  {Type: "string", Description: "Dynamic client registration endpoint URL."},
			"device_authorization_endpoint": {
				Type: "string", Description: "Device authorization endpoint URL.",
			},
			"jwks_uri":            {Type: "string", Description: "JWKS endpoint URL."},
			"revocation_endpoint": {Type: "string", Description: "Token revocation endpoint URL (RFC 7009)."},
			"scopes_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"response_types_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"grant_types_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"token_endpoint_auth_methods_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"code_challenge_methods_supported": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
		},
		Required: []string{
			"issuer",
			"authorization_endpoint",
			"token_endpoint",
			"device_authorization_endpoint",
			"jwks_uri",
			"scopes_supported",
			"response_types_supported",
			"grant_types_supported",
			"token_endpoint_auth_methods_supported",
			"code_challenge_methods_supported",
		},
	}
}

func clientRegistrationRequestSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"redirect_uris": {
				Type:  "array",
				Items: &huma.Schema{Type: "string"},
			},
			"client_name": {Type: "string"},
			"logo_uri":    {Type: "string", Format: "uri"},
			"token_endpoint_auth_method": {
				Type: "string",
				Enum: []any{"none"},
			},
			"scope": {Type: "string"},
		},
		Required: []string{"redirect_uris"},
	}
}

func clientRegistrationResponseSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"client_id":                  {Type: "string"},
			"client_id_issued_at":        {Type: "integer", Format: "int64"},
			"client_name":                {Type: "string"},
			"logo_uri":                   {Type: "string", Format: "uri"},
			"redirect_uris":              {Type: "array", Items: &huma.Schema{Type: "string"}},
			"token_endpoint_auth_method": {Type: "string"},
			"grant_types":                {Type: "array", Items: &huma.Schema{Type: "string"}},
			"response_types":             {Type: "array", Items: &huma.Schema{Type: "string"}},
			"scope":                      {Type: "string"},
		},
		Required: []string{
			"client_id",
			"client_id_issued_at",
			"redirect_uris",
			"token_endpoint_auth_method",
			"grant_types",
			"response_types",
			"scope",
		},
	}
}

func revokeRequestSchema() *huma.Schema {
	return &huma.Schema{
		Type: "object",
		Properties: map[string]*huma.Schema{
			"token":           stringProperty("The token to revoke."),
			"token_type_hint": stringProperty("Optional hint: 'refresh_token' or 'access_token'."),
		},
		Required: []string{"token"},
	}
}

func (h *Handler) ensureOpenAPIComponents(oapi *huma.OpenAPI) {
	if oapi == nil {
		return
	}
	if oapi.Components.Responses == nil {
		oapi.Components.Responses = map[string]*huma.Response{}
	}
	if oapi.Components.RequestBodies == nil {
		oapi.Components.RequestBodies = map[string]*huma.RequestBody{}
	}
	if oapi.Components.Examples == nil {
		oapi.Components.Examples = map[string]*huma.Example{}
	}

	oapi.Components.Responses[responseOAuthError] = openapiutil.JSONResponse("OAuth error response.", oauthErrorSchema())
	oapi.Components.Responses[responseOAuthTokenSuccess] = openapiutil.JSONResponse("OAuth token response.", tokenResponseSchema())
	oapi.Components.Responses[responseOAuthDeviceAuthSuccess] = openapiutil.JSONResponse("OAuth device authorization response.", deviceAuthorizationResponseSchema())
	oapi.Components.Responses[responseOAuthJWKSSuccess] = openapiutil.JSONResponse("OAuth JWKS response.", jwksSchema())
	oapi.Components.Responses[responseOAuthResourceMeta] = openapiutil.JSONResponse("OAuth protected resource metadata.", protectedResourceMetadataSchema())
	oapi.Components.Responses[responseOAuthServerMeta] = openapiutil.JSONResponse("OAuth authorization server metadata.", authorizationServerMetadataSchema())
	oapi.Components.Responses[responseOAuthClientRegistration] = openapiutil.JSONResponse("OAuth dynamic client registration response.", clientRegistrationResponseSchema())

	oapi.Components.Examples["OAuthDeviceAuthorizationRequestDefault"] = &huma.Example{Value: map[string]any{
		"client_id": koditonCLIClientID,
		"scope":     "mcp:core:read",
	}}
	oapi.Components.Examples["OAuthTokenRequestAuthorizationCode"] = &huma.Example{Value: map[string]any{
		"grant_type":    grantAuthorizationCode,
		"client_id":     "koditon-apple",
		"code":          "auth-code-value",
		"redirect_uri":  "koditon://oauth/callback",
		"code_verifier": "pkce-verifier",
	}}
	oapi.Components.Examples["OAuthTokenRequestRefreshToken"] = &huma.Example{Value: map[string]any{
		"grant_type":    grantRefreshToken,
		"client_id":     "koditon-apple",
		"refresh_token": "refresh-token-value",
	}}
	oapi.Components.Examples["OAuthTokenRequestDeviceCode"] = &huma.Example{Value: map[string]any{
		"grant_type":  grantDeviceCode,
		"client_id":   koditonCLIClientID,
		"device_code": "device-code-value",
	}}
	oapi.Components.Examples["OAuthClientRegistrationRequestDefault"] = &huma.Example{Value: map[string]any{
		"redirect_uris":              []string{"https://chat.openai.com/aip/callback"},
		"client_name":                "ChatGPT MCP Client",
		"logo_uri":                   defaultChatGPTLogoURI,
		"token_endpoint_auth_method": "none",
		"scope":                      auth.ScopeMCPCoreRead,
	}}
	oapi.Components.Examples["OAuthRevokeRequestDefault"] = &huma.Example{Value: map[string]any{
		"token":           "refresh-token-value",
		"token_type_hint": "refresh_token",
	}}
	oapi.Components.RequestBodies[requestBodyOAuthDeviceAuth] = &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"application/x-www-form-urlencoded": {
				Schema: deviceAuthorizationRequestSchema(),
				Examples: map[string]*huma.Example{
					"device_authorization_default": {Ref: "#/components/examples/OAuthDeviceAuthorizationRequestDefault"},
				},
			},
		},
	}
	oapi.Components.RequestBodies[requestBodyOAuthToken] = &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"application/x-www-form-urlencoded": {
				Schema: tokenRequestSchema(),
				Examples: map[string]*huma.Example{
					"authorization_code": {Ref: "#/components/examples/OAuthTokenRequestAuthorizationCode"},
					"refresh_token":      {Ref: "#/components/examples/OAuthTokenRequestRefreshToken"},
					"device_code":        {Ref: "#/components/examples/OAuthTokenRequestDeviceCode"},
				},
			},
		},
	}
	oapi.Components.RequestBodies[requestBodyOAuthClientRegister] = &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"application/json": {
				Schema: clientRegistrationRequestSchema(),
				Examples: map[string]*huma.Example{
					"default": {Ref: "#/components/examples/OAuthClientRegistrationRequestDefault"},
				},
			},
		},
	}
	oapi.Components.RequestBodies[requestBodyOAuthRevoke] = &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"application/x-www-form-urlencoded": {
				Schema: revokeRequestSchema(),
				Examples: map[string]*huma.Example{
					"default": {Ref: "#/components/examples/OAuthRevokeRequestDefault"},
				},
			},
		},
	}
}
