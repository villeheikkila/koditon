package auth

import "strings"

func BuildBearerChallenge(status int, message, resourceMetadataURL string, requiredScopes ...string) string {
	if status != 401 && status != 403 {
		return ""
	}
	challenge := "Bearer"
	if status == 401 {
		challenge += ` error="invalid_token"`
	} else {
		challenge += ` error="insufficient_scope"`
	}
	if strings.TrimSpace(message) != "" {
		challenge += ` error_description="` + strings.ReplaceAll(strings.TrimSpace(message), `"`, `'`) + `"`
	}
	if strings.TrimSpace(resourceMetadataURL) != "" {
		challenge += ` resource_metadata="` + strings.TrimSpace(resourceMetadataURL) + `"`
	}
	if status == 403 {
		scopeText := strings.Join(normalizeScopes(requiredScopes), " ")
		if scopeText != "" {
			challenge += ` scope="` + strings.ReplaceAll(scopeText, `"`, `'`) + `"`
		}
	}
	return challenge
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	return normalized
}

func MCPWWWAuthenticateMeta(challenge string) map[string][]string {
	if strings.TrimSpace(challenge) == "" {
		return nil
	}
	return map[string][]string{
		"mcp/www_authenticate": {challenge},
	}
}
