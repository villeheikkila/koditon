package mcpserver

import (
	domainauth "koditon/internal/domain/auth"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func mcpProtectedResourceMetadata(publicBaseURL string, scopes []string) oauthex.ProtectedResourceMetadata {
	return oauthex.ProtectedResourceMetadata{
		Resource:               domainauth.CanonicalProtectedResource(publicBaseURL),
		AuthorizationServers:   []string{publicBaseURL},
		ScopesSupported:        append([]string(nil), scopes...),
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Koditon MCP",
	}
}

func mcpBearerTokenOptions(resourceMetadataURL string, scopes []string) *mcpauth.RequireBearerTokenOptions {
	return &mcpauth.RequireBearerTokenOptions{ResourceMetadataURL: resourceMetadataURL, Scopes: append([]string(nil), scopes...)}
}
