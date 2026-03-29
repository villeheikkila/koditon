package api

import (
	"koditon-go/internal/auth"

	"github.com/danielgtaylor/huma/v2"
)

func addRoutes(a *API, api huma.API) {
	var makeMiddleware func([]string) func(huma.Context, func(huma.Context))
	if a.authService != nil {
		makeMiddleware = auth.ScopedAuthMiddlewareFactory(api, a.authService)
	}
	huma.Get(api, "/livez", a.livezHandler, func(op *huma.Operation) {
		op.OperationID = "livez"
		op.Summary = "Liveness probe"
	})
	huma.Get(api, "/readyz", a.readyzHandler, func(op *huma.Operation) {
		op.OperationID = "readyz"
		op.Summary = "Readiness probe"
	})
	huma.Post(api, "/api/v1/ping", a.pingHandler, func(op *huma.Operation) {
		op.OperationID = "ping"
		op.Summary = "Echo a message"
		applyAuth(op, makeMiddleware, resolveScopes("ping"))
	})
	huma.Get(api, "/api/v1/postal/cities", a.postalCitiesHandler, func(op *huma.Operation) {
		op.OperationID = "postal-cities"
		op.Summary = "List postal municipalities with postal codes"
		applyAuth(op, makeMiddleware, resolveScopes("postal-cities"))
	})
	huma.Get(api, "/api/v1/prices/transactions", a.pricesTransactionsHandler, func(op *huma.Operation) {
		op.OperationID = "prices-transactions"
		op.Summary = "List price transactions by municipality and postal code"
		applyAuth(op, makeMiddleware, resolveScopes("prices-transactions"))
	})
	huma.Get(api, "/api/v1/prices/transactions/filtered", a.pricesTransactionsFilteredHandler, func(op *huma.Operation) {
		op.OperationID = "prices-transactions-filtered"
		op.Summary = "List price transactions with advanced filters"
		op.Description = "Query transactions with multiple postal codes, categories, types, and area ranges"
		applyAuth(op, makeMiddleware, resolveScopes("prices-transactions-filtered"))
	})
	huma.Get(api, "/api/v1/availability/locations", a.availableLocationsHandler, func(op *huma.Operation) {
		op.OperationID = "availability-locations"
		op.Summary = "List municipalities and postal codes with price data"
		op.Description = "Returns only locations that have price transaction data available"
		applyAuth(op, makeMiddleware, resolveScopes("availability-locations"))
	})
	huma.Get(api, "/api/v1/availability/categories", a.availableCategoriesHandler, func(op *huma.Operation) {
		op.OperationID = "availability-categories"
		op.Summary = "List available building categories"
		op.Description = "Returns distinct building categories (e.g., Kerrostalo, Rivitalo, Omakotitalo)"
		applyAuth(op, makeMiddleware, resolveScopes("availability-categories"))
	})
	huma.Get(api, "/api/v1/availability/types", a.availableTypesHandler, func(op *huma.Operation) {
		op.OperationID = "availability-types"
		op.Summary = "List available apartment types"
		op.Description = "Returns distinct apartment types (e.g., Yksiö, Kaksio, Kolmio)"
		applyAuth(op, makeMiddleware, resolveScopes("availability-types"))
	})
	huma.Get(api, "/api/v1/availability/plots", a.availablePlotsHandler, func(op *huma.Operation) {
		op.OperationID = "availability-plots"
		op.Summary = "List available plot ownership types"
		op.Description = "Returns distinct plot ownership types (e.g., Oma, Vuokra)"
		applyAuth(op, makeMiddleware, resolveScopes("availability-plots"))
	})
	huma.Get(api, "/api/v1/entity", a.entityDetailHandler, func(op *huma.Operation) {
		op.OperationID = "entity-detail"
		op.Summary = "Get entity detail"
		op.Description = "Fetch canonical detail for an ad or building by canonical ID or source URL"
		op.Tags = []string{"Entity"}
	})
	huma.Get(api, "/api/v1/search", a.searchHandler, func(op *huma.Operation) {
		op.OperationID = "search"
		op.Summary = "Search entities"
		op.Description = "Search ads and buildings by free text, address, city, postal code, price, and area"
		op.Tags = []string{"Entity"}
	})
	huma.Post(api, "/auth/apple", a.appleWebAuthHandler, func(op *huma.Operation) {
		op.OperationID = "auth-apple-web"
		op.Summary = "Sign in with Apple (web)"
		op.Description = "Exchange an Apple authorization code for access tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/authenticate/options", a.passkeyAuthOptionsHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-authenticate-options"
		op.Summary = "Begin passkey authentication"
		op.Description = "Returns a WebAuthn challenge and options for passkey sign-in"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/authenticate", a.passkeyAuthHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-authenticate"
		op.Summary = "Complete passkey authentication"
		op.Description = "Verify the passkey credential and return access tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/register/options", a.passkeyRegisterOptionsHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-register-options"
		op.Summary = "Begin passkey registration"
		op.Description = "Returns a WebAuthn challenge and options to register a new passkey"
		op.Tags = []string{"Authentication"}
		applyAuth(op, makeMiddleware, resolveScopes("auth-passkey-register-options"))
	})
	huma.Post(api, "/auth/passkey/register/finish", a.passkeyRegisterFinishHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-register-finish"
		op.Summary = "Complete passkey registration"
		op.Description = "Save the new passkey credential for the authenticated user"
		op.Tags = []string{"Authentication"}
		applyAuth(op, makeMiddleware, resolveScopes("auth-passkey-register-finish"))
	})
}
