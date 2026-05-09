package api

import "koditon/internal/domain/auth"

func scopesForOperationID(operationID string) ([]string, bool) {
	scopes, ok := operationIDScopes[operationID]
	return scopes, ok
}

var operationIDScopes = map[string][]string{
	"ping":                                       {auth.ScopeCoreRead},
	"postal-cities":                              {auth.ScopeCoreRead},
	"prices-transactions":                        {auth.ScopeCoreRead},
	"prices-transactions-filtered":               {auth.ScopeCoreRead},
	"availability-locations":                     {auth.ScopeCoreRead},
	"availability-categories":                    {auth.ScopeCoreRead},
	"availability-types":                         {auth.ScopeCoreRead},
	"availability-plots":                         {auth.ScopeCoreRead},
	"sale-listings-transaction-match-postals":    {auth.ScopeCoreRead},
	"sale-listings-transaction-match-candidates": {auth.ScopeCoreRead},
	"sale-listings-renovations-extract":          {auth.ScopeCoreRead},
	"sale-listings-description-extract":          {auth.ScopeCoreRead},
	"sale-listings-valuation-inputs-extract":     {auth.ScopeCoreRead},
	"sale-listings-canonical-profile-project":    {auth.ScopeCoreRead},
	"sale-listings-house-overview-generate":      {auth.ScopeCoreRead},
	"sale-listings-manager-certificate-upload":   {auth.ScopeProfileWrite},
	"property-documents-download":                {auth.ScopeCoreRead},
	"property-documents-extract":                 {auth.ScopeCoreRead},
	"entity-detail":                              {},
	"search":                                     {},
	"auth-apple-web":                             {},
	"auth-passkey-authenticate-options":          {},
	"auth-passkey-authenticate":                  {},
	"auth-session-refresh":                       {},
	"auth-session-sign-out":                      {},
	"auth-passkey-register-options":              {auth.ScopeProfileWrite},
	"auth-passkey-register-finish":               {auth.ScopeProfileWrite},
}

var publicOperationIDs = map[string]struct{}{
	"entity-detail":                     {},
	"search":                            {},
	"auth-apple-web":                    {},
	"auth-passkey-authenticate-options": {},
	"auth-passkey-authenticate":         {},
	"auth-session-refresh":              {},
	"auth-session-sign-out":             {},
}
