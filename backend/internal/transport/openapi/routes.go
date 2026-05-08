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
		op.Description = "Fetch canonical detail for an ad or housing company by canonical ID or source URL"
		op.Tags = []string{"Entity"}
	})
	huma.Get(api, "/api/v1/search", a.searchHandler, func(op *huma.Operation) {
		op.OperationID = "search"
		op.Summary = "Search entities"
		op.Description = "Search ads and housing companies by free text, address, city, postal code, price, and area"
		op.Tags = []string{"Entity"}
	})
	huma.Get(api, "/api/v1/sale-listings", a.saleListingsSearchHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-search"
		op.Summary = "Search sale listings"
		op.Description = "Search sale listings using the shared provider-neutral sale listing model"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/sale-listings/map", a.saleListingsMapHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-map"
		op.Summary = "Map sale listing locations"
		op.Description = "Return grouped map markers for canonical sale offerings by exact housing company location"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/sale-listings/map-filter-options", a.saleListingsMapFilterOptionsHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-map-filter-options"
		op.Summary = "List map filter options"
		op.Description = "Return distinct city and postal values available in sale listing map filters"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/sale-listings/transaction-match-postals", a.transactionMatchPostalsHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-transaction-match-postals"
		op.Summary = "List postal codes with potential transaction matches"
		op.Description = "Returns postal codes where unresolved sale listing to price transaction candidates exist"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-transaction-match-postals"))
	})
	huma.Get(api, "/api/v1/sale-listings/transaction-match-candidates", a.transactionMatchCandidatesHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-transaction-match-candidates"
		op.Summary = "List potential transaction matches"
		op.Description = "Returns unresolved sale listing to price transaction candidate rows, optionally filtered by postal code"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-transaction-match-candidates"))
	})
	huma.Get(api, "/api/v1/sale-listings/{id}", a.saleListingDetailHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-detail"
		op.Summary = "Get sale listing detail"
		op.Description = "Fetch a canonical sale offering by UUID"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Get(api, "/api/v1/sale-listings/{id}/source-records/{sourceID}/raw", a.saleListingSourceRawHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-source-raw"
		op.Summary = "Get sale listing source raw payload"
		op.Description = "Fetch the original provider JSON payload for a source row linked to a canonical sale offering"
		op.Tags = []string{"Sale Listings"}
	})
	huma.Post(api, "/api/v1/sale-listings/{id}/renovations/extract", a.saleListingRenovationExtractHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-renovations-extract"
		op.Summary = "Extract structured sale listing renovations"
		op.Description = "Uses the configured Fantasy/OpenRouter model to extract structured renovation rows from listing renovation text"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-renovations-extract"))
	})
	huma.Post(api, "/api/v1/sale-listings/{id}/description/extract", a.saleListingDescriptionExtractHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-description-extract"
		op.Summary = "Extract structured sale listing description signals"
		op.Description = "Uses the configured Fantasy/OpenRouter model to extract offer-relevant signals from listing description text"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-description-extract"))
	})
	huma.Post(api, "/api/v1/sale-listings/{id}/valuation-inputs/extract", a.saleListingValuationInputsExtractHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-valuation-inputs-extract"
		op.Summary = "Extract canonical sale listing valuation inputs"
		op.Description = "Uses the configured Fantasy/OpenRouter model to parse layout and listing text into structured valuation facts"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-valuation-inputs-extract"))
	})
	huma.Post(api, "/api/v1/sale-listings/{id}/canonical-profile/project", a.saleListingCanonicalProfileProjectHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-canonical-profile-project"
		op.Summary = "Project canonical sale listing profile"
		op.Description = "Projects provider fields, extracted property claims, and typed renovation rows into the canonical apartment, building, housing-company, and quality-score tables"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-canonical-profile-project"))
	})
	huma.Post(api, "/api/v1/sale-listings/{id}/house-overview/generate", a.saleListingHouseOverviewGenerateHandler, func(op *huma.Operation) {
		op.OperationID = "sale-listings-house-overview-generate"
		op.Summary = "Generate sale listing house overview"
		op.Description = "Uses the configured Fantasy/OpenRouter model to summarize the building and renovation situation from already structured listing facts"
		op.Tags = []string{"Sale Listings"}
		applyAuth(op, makeMiddleware, resolveScopes("sale-listings-house-overview-generate"))
	})
	huma.Get(api, "/api/v1/rentals", a.rentalsSearchHandler, func(op *huma.Operation) {
		op.OperationID = "rentals-search"
		op.Summary = "Search rentals"
		op.Description = "Search rentals using the shared provider-neutral rental model"
		op.Tags = []string{"Rentals"}
	})
	huma.Get(api, "/api/v1/rentals/{id}", a.rentalDetailHandler, func(op *huma.Operation) {
		op.OperationID = "rentals-detail"
		op.Summary = "Get rental detail"
		op.Description = "Fetch a rental by public ID, canonical ID, or source URL"
		op.Tags = []string{"Rentals"}
	})
	huma.Get(api, "/api/v1/housing-companies/{id}", a.housingCompanyDetailHandler, func(op *huma.Operation) {
		op.OperationID = "housing-companies-detail"
		op.Summary = "Get housing company detail"
		op.Description = "Fetch housing company details by UUID, canonical ID, or source URL"
		op.Tags = []string{"Housing Companies"}
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
	huma.Post(api, "/auth/session/refresh", a.webSessionRefreshHandler, func(op *huma.Operation) {
		op.OperationID = "auth-session-refresh"
		op.Summary = "Refresh web session"
		op.Description = "Rotate the web refresh cookie and return a new access token"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/session/sign-out", a.webSessionSignOutHandler, func(op *huma.Operation) {
		op.OperationID = "auth-session-sign-out"
		op.Summary = "Sign out web session"
		op.Description = "Revoke the web refresh token and clear the session cookie"
		op.Tags = []string{"Authentication"}
	})
}
