package api

import "koditon-go/internal/auth"

func scopesForOperationID(operationID string) ([]string, bool) {
	scopes, ok := operationIDScopes[operationID]
	return scopes, ok
}

var operationIDScopes = map[string][]string{
	"ping":                              {auth.ScopeCoreRead},
	"postal-cities":                     {auth.ScopeCoreRead},
	"prices-transactions":               {auth.ScopeCoreRead},
	"prices-transactions-filtered":      {auth.ScopeCoreRead},
	"availability-locations":            {auth.ScopeCoreRead},
	"availability-categories":           {auth.ScopeCoreRead},
	"availability-types":                {auth.ScopeCoreRead},
	"availability-plots":                {auth.ScopeCoreRead},
	"entity-detail":                     {},
	"search":                            {},
	"auth-apple-web":                    {},
	"auth-passkey-authenticate-options": {},
	"auth-passkey-authenticate":         {},
	"auth-passkey-register-options":     {auth.ScopeProfileWrite},
	"auth-passkey-register-finish":      {auth.ScopeProfileWrite},
}

var publicOperationIDs = map[string]struct{}{
	"healthz":                           {},
	"entity-detail":                     {},
	"search":                            {},
	"auth-apple-web":                    {},
	"auth-passkey-authenticate-options": {},
	"auth-passkey-authenticate":         {},
}
