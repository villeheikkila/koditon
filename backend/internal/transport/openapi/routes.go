package api

import (
	"koditon/internal/domain/auth"

	"github.com/danielgtaylor/huma/v2"
)

func addRoutes(a *API, api huma.API) {
	var makeMiddleware func([]string) func(huma.Context, func(huma.Context))
	if a.authService != nil {
		makeMiddleware = auth.ScopedAuthMiddlewareFactory(api, a.authService)
	}
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
	huma.Get(api, "/api/v1/sale-listings", a.saleListingsSearchHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-search"
		op.Summary = "Search sale listings"
		op.Description = "Search sale listings using the shared provider-neutral sale listing model"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/sale-listings/{canonical_id}", a.saleListingDetailHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-detail"
		op.Summary = "Get sale listing detail"
		op.Description = "Fetch a sale listing by canonical ID"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/rentals", a.rentalsSearchHandler, func(op *huma.Operation) {
		op.OperationID = "rentals-search"
		op.Summary = "Search rentals"
		op.Description = "Search rentals using the shared provider-neutral rental model"
		op.Tags = []string{"Rentals"}
	})
	huma.Get(api, "/api/v1/rentals/{canonical_id}", a.rentalDetailHandler, func(op *huma.Operation) {
		op.OperationID = "rentals-detail"
		op.Summary = "Get rental detail"
		op.Description = "Fetch a rental by canonical ID"
		op.Tags = []string{"Rentals"}
	})
	huma.Get(api, "/api/v1/buildings/{canonical_id}", a.buildingDetailHandler, func(op *huma.Operation) {
		op.OperationID = "buildings-detail"
		op.Summary = "Get building detail"
		op.Description = "Fetch building details by canonical ID"
		op.Tags = []string{"Buildings"}
	})
	huma.Get(api, "/api/v1/resolve", a.resolveCanonicalIDHandler, func(op *huma.Operation) {
		op.OperationID = "resolve-canonical-id"
		op.Summary = "Resolve source URL"
		op.Description = "Resolve a source URL into a canonical ID for use with detail endpoints"
		op.Tags = []string{"Entity"}
	})
	huma.Post(api, "/auth/apple", a.appleWebAuthHandler, func(op *huma.Operation) {
		op.OperationID = "auth-apple-web"
		op.Summary = "Sign in with Apple (web)"
		op.Description = "Exchange an Apple authorization code for access tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/email/request", a.emailAuthRequestHandler, func(op *huma.Operation) {
		op.OperationID = "auth-email-request"
		op.Summary = "Request email sign-in link"
		op.Description = "Send a sign-in link to the requested email address"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/email/confirm", a.emailAuthConfirmHandler, func(op *huma.Operation) {
		op.OperationID = "auth-email-confirm"
		op.Summary = "Confirm email sign-in"
		op.Description = "Exchange an email sign-in token for access tokens"
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
