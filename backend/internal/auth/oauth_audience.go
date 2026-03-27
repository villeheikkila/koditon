package auth

import (
	"fmt"
	"strings"
)

type OAuthAudienceMode string

const (
	OAuthAudienceModeAPI               OAuthAudienceMode = "api"
	OAuthAudienceModeProtectedResource OAuthAudienceMode = "protected_resource"
)

func CanonicalProtectedResource(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/mcp"
}

func CanonicalAPIAudience(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func normalizeOAuthAudienceMode(mode OAuthAudienceMode) OAuthAudienceMode {
	switch strings.TrimSpace(string(mode)) {
	case string(OAuthAudienceModeProtectedResource):
		return OAuthAudienceModeProtectedResource
	default:
		return OAuthAudienceModeAPI
	}
}

func ResolveOAuthAudience(baseURL string, mode OAuthAudienceMode, resource string) (string, error) {
	mode = normalizeOAuthAudienceMode(mode)
	resource = strings.TrimSpace(resource)
	if resource == "" {
		if mode == OAuthAudienceModeProtectedResource {
			return CanonicalProtectedResource(baseURL), nil
		}
		return CanonicalAPIAudience(baseURL), nil
	}
	if resource == CanonicalProtectedResource(baseURL) {
		return resource, nil
	}
	if resource == CanonicalAPIAudience(baseURL) && mode != OAuthAudienceModeProtectedResource {
		return resource, nil
	}
	return "", fmt.Errorf("resource must match %s", CanonicalProtectedResource(baseURL))
}
