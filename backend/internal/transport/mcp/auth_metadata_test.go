package mcpserver

import (
	"testing"

	"koditon/internal/domain/auth"
)

func TestMCPProtectedResourceMetadataUsesSDKShape(t *testing.T) {
	t.Parallel()
	metadata := mcpProtectedResourceMetadata("https://api.example", []string{auth.ScopeMCPCoreRead})
	if metadata.Resource != "https://api.example/mcp" {
		t.Fatalf("resource = %q, want https://api.example/mcp", metadata.Resource)
	}
	if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != "https://api.example" {
		t.Fatalf("authorization servers = %#v", metadata.AuthorizationServers)
	}
	if len(metadata.ScopesSupported) != 1 || metadata.ScopesSupported[0] != auth.ScopeMCPCoreRead {
		t.Fatalf("scopes = %#v", metadata.ScopesSupported)
	}
}
