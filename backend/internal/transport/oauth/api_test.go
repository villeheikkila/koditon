package oauthapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"koditon/internal/domain/auth"
)

func TestValidateAuthorizeRequestAcceptsCanonicalResource(t *testing.T) {
	t.Parallel()

	h := &Handler{
		publicAPIBaseURL: "https://api.example.com",
		clients: map[string]oauthClient{
			"koditon-apple": {
				ClientID:     "koditon-apple",
				ClientType:   "public",
				RedirectURIs: []string{"koditon://oauth/callback"},
				Scopes:       []string{auth.ScopeMCPCoreRead},
				AudienceMode: auth.OAuthAudienceModeProtectedResource,
			},
		},
	}

	client, scopes, err := h.validateAuthorizeRequest(context.Background(), authorizeRequest{
		ResponseType:        "code",
		ClientID:            "koditon-apple",
		RedirectURI:         "koditon://oauth/callback",
		Resource:            "https://api.example.com/mcp",
		Scope:               []string{auth.ScopeMCPCoreRead},
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("validateAuthorizeRequest returned error: %v", err)
	}
	if client.ClientID != "koditon-apple" {
		t.Fatalf("client id = %q, want %q", client.ClientID, "koditon-apple")
	}
	if len(scopes) != 1 || scopes[0] != auth.ScopeMCPCoreRead {
		t.Fatalf("scopes = %v, want [%q]", scopes, auth.ScopeMCPCoreRead)
	}
}

func TestResolveAudienceForClientDefaultsToAPIForNonMCPClient(t *testing.T) {
	t.Parallel()

	h := &Handler{publicAPIBaseURL: "https://api.example.com"}

	audience, err := h.resolveAudienceForClient(oauthClient{
		ClientID:     "koditon-apple",
		Scopes:       []string{auth.ScopeCoreRead},
		AudienceMode: auth.OAuthAudienceModeAPI,
	}, "")
	if err != nil {
		t.Fatalf("resolveAudienceForClient returned error: %v", err)
	}
	if audience != "https://api.example.com" {
		t.Fatalf("audience = %q, want %q", audience, "https://api.example.com")
	}
}

func TestResolveAudienceForClientDefaultsToMCPForMCPOnlyClient(t *testing.T) {
	t.Parallel()

	h := &Handler{publicAPIBaseURL: "https://api.example.com"}

	audience, err := h.resolveAudienceForClient(oauthClient{
		ClientID:     "dyn_test",
		Scopes:       []string{auth.ScopeMCPCoreRead},
		AudienceMode: auth.OAuthAudienceModeProtectedResource,
	}, "")
	if err != nil {
		t.Fatalf("resolveAudienceForClient returned error: %v", err)
	}
	if audience != "https://api.example.com/mcp" {
		t.Fatalf("audience = %q, want %q", audience, "https://api.example.com/mcp")
	}
}

func TestResolveAudienceForClientAcceptsAPIAudienceForNonMCPClient(t *testing.T) {
	t.Parallel()

	h := &Handler{publicAPIBaseURL: "https://api.example.com"}

	audience, err := h.resolveAudienceForClient(oauthClient{
		ClientID:     "koditon-apple",
		Scopes:       []string{auth.ScopeCoreRead},
		AudienceMode: auth.OAuthAudienceModeAPI,
	}, "https://api.example.com")
	if err != nil {
		t.Fatalf("resolveAudienceForClient returned error: %v", err)
	}
	if audience != "https://api.example.com" {
		t.Fatalf("audience = %q, want %q", audience, "https://api.example.com")
	}
}

func TestHandleDeviceAuthorizationRejectsUnexpectedResource(t *testing.T) {
	t.Parallel()

	h := &Handler{
		publicAPIBaseURL: "https://api.example.com",
		clients: map[string]oauthClient{
			"koditon-apple": {
				ClientID:     "koditon-apple",
				ClientType:   "public",
				Scopes:       []string{auth.ScopeMCPCoreRead},
				AudienceMode: auth.OAuthAudienceModeProtectedResource,
			},
		},
	}

	form := url.Values{
		"client_id": {"koditon-apple"},
		"resource":  {"https://other.example.com/mcp"},
	}
	req := httptest.NewRequest(http.MethodPost, "/oauth/device_authorization", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.handleDeviceAuthorization(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "invalid_target" {
		t.Fatalf("error = %q, want %q", body["error"], "invalid_target")
	}
}

func TestHandleProtectedResourceMetadataUsesCanonicalResource(t *testing.T) {
	t.Parallel()

	h := &Handler{
		publicAPIBaseURL: "https://api.example.com/",
		scopesSupported:  []string{auth.ScopeMCPCoreRead},
	}

	rec := httptest.NewRecorder()
	h.handleProtectedResourceMetadata(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["resource"]; got != "https://api.example.com/mcp" {
		t.Fatalf("resource = %v, want %q", got, "https://api.example.com/mcp")
	}
}

func TestHandleProtectedResourceMetadataAllowsPreflight(t *testing.T) {
	t.Parallel()
	h := &Handler{
		publicAPIBaseURL: "https://api.example.com/",
		scopesSupported:  []string{auth.ScopeMCPCoreRead},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/.well-known/oauth-protected-resource/mcp", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestRecoverMalformedOAuthPath(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(
		http.MethodGet,
		`https://api.koditon.com/oauth/%2522https:/api.koditon.com/oauth/authorize/handoff/status?id=b437656a-aa18-400b-ba25-e5cae8d89248%22`,
		nil,
	)
	path, ok := recoverMalformedOAuthPath(req)
	if !ok {
		t.Fatal("recoverMalformedOAuthPath should recover malformed oauth URL")
	}
	want := "/oauth/authorize/handoff/status?id=b437656a-aa18-400b-ba25-e5cae8d89248"
	if path != want {
		t.Fatalf("recoverMalformedOAuthPath = %q, want %q", path, want)
	}
}

func TestRecoverMalformedOAuthPathSkipsForeignHost(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(
		http.MethodGet,
		`https://api.koditon.com/oauth/%2522https:/evil.example/oauth/authorize/handoff/status?id=abc%22`,
		nil,
	)
	if _, ok := recoverMalformedOAuthPath(req); ok {
		t.Fatal("recoverMalformedOAuthPath should reject embedded URL from foreign host")
	}
}
