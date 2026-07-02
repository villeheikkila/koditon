package api

import "koditon/internal/domain/auth"

func scopesForOperationID(operationID string) ([]string, bool) {
	scopes, ok := operationIDScopes[operationID]
	return scopes, ok
}

var operationIDScopes = map[string][]string{
	"ping":                                           {auth.ScopeCoreRead},
	"postal-cities":                                  {auth.ScopeCoreRead},
	"prices-transactions":                            {auth.ScopeCoreRead},
	"prices-transactions-filtered":                   {auth.ScopeCoreRead},
	"availability-locations":                         {auth.ScopeCoreRead},
	"availability-categories":                        {auth.ScopeCoreRead},
	"availability-types":                             {auth.ScopeCoreRead},
	"availability-plots":                             {auth.ScopeCoreRead},
	"property-targets-map":                           {},
	"property-targets-detail":                        {},
	"property-targets-resolved-values":               {},
	"property-targets-claims":                        {},
	"property-targets-renovation-events":             {},
	"property-targets-documents":                     {},
	"transaction-match-postals":                      {auth.ScopeCoreRead},
	"transaction-match-candidates":                   {auth.ScopeCoreRead},
	"property-targets-resolve":                       {auth.ScopeProfileWrite},
	"property-documents-manager-certificates-upload": {auth.ScopeProfileWrite},
	"property-documents-detail":                      {auth.ScopeCoreRead},
	"property-documents-download":                    {auth.ScopeCoreRead},
	"property-documents-attachment-set":              {auth.ScopeProfileWrite},
	"source-refresh":                                 {auth.ScopeProfileWrite},
	"entity-detail":                                  {},
	"search":                                         {},
	"auth-apple-web":                                 {},
	"auth-passkey-authenticate-options":              {},
	"auth-passkey-authenticate":                      {},
	"auth-session-refresh":                           {},
	"auth-session-sign-out":                          {},
	"auth-passkey-register-options":                  {auth.ScopeProfileWrite},
	"auth-passkey-register-finish":                   {auth.ScopeProfileWrite},
}

var publicOperationIDs = map[string]struct{}{
	"entity-detail":                      {},
	"search":                             {},
	"property-targets-map":               {},
	"property-targets-detail":            {},
	"property-targets-resolved-values":   {},
	"property-targets-claims":            {},
	"property-targets-renovation-events": {},
	"property-targets-documents":         {},
	"auth-apple-web":                     {},
	"auth-passkey-authenticate-options":  {},
	"auth-passkey-authenticate":          {},
	"auth-session-refresh":               {},
	"auth-session-sign-out":              {},
}
