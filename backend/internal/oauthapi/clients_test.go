package oauthapi

import (
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestRegistrationClientIPPrefersRemoteAddrWhenProxyUntrusted(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/oauth/register", nil)
	req.RemoteAddr = "203.0.113.9:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.5")

	got := registrationClientIP(req)
	if got != "203.0.113.9" {
		t.Fatalf("registrationClientIP = %q, want %q", got, "203.0.113.9")
	}
}

func TestRegistrationClientIPUsesForwardedForForTrustedProxy(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/oauth/register", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "198.51.100.5, 127.0.0.1")

	got := registrationClientIP(req)
	if got != "198.51.100.5" {
		t.Fatalf("registrationClientIP = %q, want %q", got, "198.51.100.5")
	}
}

func TestNormalizeDynamicRedirectURIAcceptsChatGPTConnectorCallback(t *testing.T) {
	t.Parallel()

	h := &Handler{
		dcrConfig: oauthDCRConfig{
			allowedRedirectURIs: map[string]struct{}{
				"https://chatgpt.com/connector_platform_oauth_redirect": {},
			},
		},
	}
	got, err := h.normalizeDynamicRedirectURI("https://chatgpt.com/connector/oauth/kBP4I-G-Rawl")
	if err != nil {
		t.Fatalf("normalizeDynamicRedirectURI returned error: %v", err)
	}
	if got != "https://chatgpt.com/connector/oauth/kBP4I-G-Rawl" {
		t.Fatalf("normalizeDynamicRedirectURI = %q, want %q", got, "https://chatgpt.com/connector/oauth/kBP4I-G-Rawl")
	}
}

func TestIsChatGPTConnectorOAuthRedirect(t *testing.T) {
	t.Parallel()

	allow := func(raw string) bool {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return isChatGPTConnectorOAuthRedirect(u)
	}

	if !allow("https://chatgpt.com/connector/oauth/kBP4I-G-Rawl") {
		t.Fatal("expected connector callback to be allowed")
	}
	if allow("https://chatgpt.com/connector/oauth/") {
		t.Fatal("expected empty callback suffix to be rejected")
	}
	if allow("http://chatgpt.com/connector/oauth/kBP4I-G-Rawl") {
		t.Fatal("expected non-https callback to be rejected")
	}
	if allow("https://example.com/connector/oauth/kBP4I-G-Rawl") {
		t.Fatal("expected non-chatgpt host callback to be rejected")
	}
}

func TestResolvedDynamicClientLogoURIUsesTrustedChatGPTDefault(t *testing.T) {
	t.Parallel()

	got, err := resolvedDynamicClientLogoURI("", []string{"https://chatgpt.com/connector/oauth/kBP4I-G-Rawl"})
	if err != nil {
		t.Fatalf("resolvedDynamicClientLogoURI returned error: %v", err)
	}
	if got != defaultChatGPTLogoURI {
		t.Fatalf("resolvedDynamicClientLogoURI = %q, want %q", got, defaultChatGPTLogoURI)
	}
}

func TestResolvedDynamicClientLogoURIPrefersValidProvidedLogo(t *testing.T) {
	t.Parallel()

	got, err := resolvedDynamicClientLogoURI(
		"https://openai.com/favicon.ico",
		[]string{"https://chatgpt.com/connector/oauth/kBP4I-G-Rawl"},
	)
	if err != nil {
		t.Fatalf("resolvedDynamicClientLogoURI returned error: %v", err)
	}
	if got != "https://openai.com/favicon.ico" {
		t.Fatalf("resolvedDynamicClientLogoURI = %q, want provided logo_uri", got)
	}
}

func TestResolvedDynamicClientLogoURIRejectsUntrustedHost(t *testing.T) {
	t.Parallel()

	_, err := resolvedDynamicClientLogoURI("https://example.com/logo.png", nil)
	if err == nil {
		t.Fatal("expected resolvedDynamicClientLogoURI to reject untrusted logo host")
	}
	if !strings.Contains(err.Error(), "host is not allowed") {
		t.Fatalf("resolvedDynamicClientLogoURI error = %v, want host validation", err)
	}
}

func TestResolveOAuthClientLogoURLFromDynamicMetadata(t *testing.T) {
	t.Parallel()

	logo := resolveOAuthClientLogoURL("dyn_test", []byte(`{"logo_uri":"https://chatgpt.com/favicon.ico"}`))
	if logo != "https://chatgpt.com/favicon.ico" {
		t.Fatalf("resolveOAuthClientLogoURL = %q, want chatgpt favicon", logo)
	}
}

func TestCollectSupportedScopesIncludesAdditionalScopes(t *testing.T) {
	t.Parallel()

	scopes := collectSupportedScopes(map[string]oauthClient{
		"getmaku-web": {
			Scopes: []string{"core:read", "profile:read"},
		},
	}, []string{"mcp:core:read", "check-ins:write"})

	expected := []string{"check-ins:write", "core:read", "mcp:core:read", "profile:read"}
	if !slices.Equal(scopes, expected) {
		t.Fatalf("collectSupportedScopes = %v, want %v", scopes, expected)
	}
}

func TestDefaultDCRClientScopesUsesAllowedScopes(t *testing.T) {
	t.Parallel()

	got := defaultDCRClientScopes([]string{"mcp:core:read", "check-ins:write"})
	want := []string{"mcp:core:read", "check-ins:write"}
	if !slices.Equal(got, want) {
		t.Fatalf("defaultDCRClientScopes = %v, want %v", got, want)
	}
}

func TestDefaultDCRClientScopesFallsBackToCoreRead(t *testing.T) {
	t.Parallel()

	got := defaultDCRClientScopes(nil)
	want := []string{"mcp:core:read"}
	if !slices.Equal(got, want) {
		t.Fatalf("defaultDCRClientScopes(nil) = %v, want %v", got, want)
	}
}

func TestNormalizeRequestedScopesForDynamicClientUsesAllowedSet(t *testing.T) {
	t.Parallel()

	h := &Handler{
		dcrConfig: oauthDCRConfig{
			allowedScopes: []string{"mcp:core:read", "check-ins:write"},
		},
	}
	client := oauthClient{
		ClientID: "dyn_abc",
		Scopes:   []string{"mcp:core:read"},
	}

	got := h.normalizeRequestedScopesForClient(client, []string{"mcp:core:read"})
	want := []string{"mcp:core:read", "check-ins:write"}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizeRequestedScopesForClient(dynamic) = %v, want %v", got, want)
	}
}

func TestStaticClientsAllowAdminProductReadForFirstPartyApp(t *testing.T) {
	t.Parallel()

	clients, err := staticClients("https://getmaku.com")
	if err != nil {
		t.Fatalf("staticClients returned error: %v", err)
	}

	client, ok := clients["maku-apple"]
	if !ok {
		t.Fatal("maku-apple client missing from static clients")
	}
	if !slices.Contains(client.Scopes, "admin:products:read") {
		t.Fatalf("maku-apple scopes missing admin product read: %s", strings.Join(client.Scopes, " "))
	}
}
