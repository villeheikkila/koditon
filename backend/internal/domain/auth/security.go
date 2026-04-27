package auth

import (
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterSecurityScheme(config *huma.Config, publicAPIBaseURL string) {
	if config.Components.SecuritySchemes == nil {
		config.Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
	}

	baseURL := strings.TrimRight(publicAPIBaseURL, "/")
	authorizeURL := baseURL + "/oauth/authorize"
	tokenURL := baseURL + "/oauth/token"
	if baseURL == "" {
		authorizeURL = "/oauth/authorize"
		tokenURL = "/oauth/token"
	}

	config.Components.SecuritySchemes["bearer"] = &huma.SecurityScheme{
		Type:        "oauth2",
		Description: "OAuth 2.0 access token",
		Flows: &huma.OAuthFlows{
			AuthorizationCode: &huma.OAuthFlow{
				AuthorizationURL: authorizeURL,
				TokenURL:         tokenURL,
				Scopes:           openAPIScopes(),
			},
		},
	}
}

func openAPIScopes() map[string]string {
	scopes := make(map[string]string, len(userScopes)+len(adminScopes))
	for _, scope := range append(append([]string{}, userScopes...), adminScopes...) {
		if _, ok := scopes[scope]; ok {
			continue
		}
		scopes[scope] = scope
	}
	keys := make([]string, 0, len(scopes))
	for scope := range scopes {
		keys = append(keys, scope)
	}
	slices.Sort(keys)
	ordered := make(map[string]string, len(keys))
	for _, scope := range keys {
		ordered[scope] = scopes[scope]
	}
	return ordered
}
