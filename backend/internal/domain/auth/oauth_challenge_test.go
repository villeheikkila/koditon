package auth

import (
	"strings"
	"testing"
)

func TestBuildBearerChallengeIncludesScopeForInsufficientScope(t *testing.T) {
	t.Parallel()

	challenge := BuildBearerChallenge(
		403,
		"insufficient scope",
		"https://api.example.com/.well-known/oauth-protected-resource/mcp",
		"mcp:core:read",
		"check-ins:write",
	)

	if !strings.Contains(challenge, `error="insufficient_scope"`) {
		t.Fatalf("challenge missing insufficient_scope: %q", challenge)
	}
	if !strings.Contains(challenge, `scope="mcp:core:read check-ins:write"`) {
		t.Fatalf("challenge missing scope list: %q", challenge)
	}
}

func TestBuildBearerChallengeOmitsScopeForUnauthorized(t *testing.T) {
	t.Parallel()

	challenge := BuildBearerChallenge(401, "missing token", "https://api.example.com/.well-known/oauth-protected-resource/mcp", "mcp:core:read")
	if strings.Contains(challenge, `scope="`) {
		t.Fatalf("challenge unexpectedly includes scope for 401: %q", challenge)
	}
}
