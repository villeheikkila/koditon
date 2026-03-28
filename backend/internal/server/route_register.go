package server

import (
	"koditon-go/internal/auth"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) addRoutes(api huma.API) {
	var authMiddleware func(huma.Context, func(huma.Context))
	if s.authService != nil {
		authMiddleware = auth.AuthMiddlewareFactory(api, s.authService)
	}
	huma.Get(api, "/healthz", s.healthHandler, func(op *huma.Operation) {
		op.OperationID = "healthz"
		op.Summary = "Health check"
	})
	huma.Post(api, "/api/v1/ping", s.pingHandler, func(op *huma.Operation) {
		op.OperationID = "ping"
		op.Summary = "Echo a message"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/postal/cities", s.postalCitiesHandler, func(op *huma.Operation) {
		op.OperationID = "postal-cities"
		op.Summary = "List postal municipalities with postal codes"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/prices/transactions", s.pricesTransactionsHandler, func(op *huma.Operation) {
		op.OperationID = "prices-transactions"
		op.Summary = "List price transactions by municipality and postal code"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/prices/transactions/filtered", s.pricesTransactionsFilteredHandler, func(op *huma.Operation) {
		op.OperationID = "prices-transactions-filtered"
		op.Summary = "List price transactions with advanced filters"
		op.Description = "Query transactions with multiple postal codes, categories, types, and area ranges"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/availability/locations", s.availableLocationsHandler, func(op *huma.Operation) {
		op.OperationID = "availability-locations"
		op.Summary = "List municipalities and postal codes with price data"
		op.Description = "Returns only locations that have price transaction data available"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/availability/categories", s.availableCategoriesHandler, func(op *huma.Operation) {
		op.OperationID = "availability-categories"
		op.Summary = "List available building categories"
		op.Description = "Returns distinct building categories (e.g., Kerrostalo, Rivitalo, Omakotitalo)"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/availability/types", s.availableTypesHandler, func(op *huma.Operation) {
		op.OperationID = "availability-types"
		op.Summary = "List available apartment types"
		op.Description = "Returns distinct apartment types (e.g., Yksiö, Kaksio, Kolmio)"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/availability/plots", s.availablePlotsHandler, func(op *huma.Operation) {
		op.OperationID = "availability-plots"
		op.Summary = "List available plot ownership types"
		op.Description = "Returns distinct plot ownership types (e.g., Oma, Vuokra)"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Get(api, "/api/v1/entity", s.entityDetailHandler, func(op *huma.Operation) {
		op.OperationID = "entity-detail"
		op.Summary = "Get entity detail"
		op.Description = "Fetch canonical detail for an ad or building by canonical ID or source URL"
		op.Tags = []string{"Entity"}
	})
	huma.Get(api, "/api/v1/search", s.searchHandler, func(op *huma.Operation) {
		op.OperationID = "search"
		op.Summary = "Search entities"
		op.Description = "Search ads and buildings by free text, address, city, postal code, price, and area"
		op.Tags = []string{"Entity"}
	})
	huma.Post(api, "/auth/apple", s.appleWebAuthHandler, func(op *huma.Operation) {
		op.OperationID = "auth-apple-web"
		op.Summary = "Sign in with Apple (web)"
		op.Description = "Exchange an Apple authorization code for access tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/authenticate/options", s.passkeyAuthOptionsHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-authenticate-options"
		op.Summary = "Begin passkey authentication"
		op.Description = "Returns a WebAuthn challenge and options for passkey sign-in"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/authenticate", s.passkeyAuthHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-authenticate"
		op.Summary = "Complete passkey authentication"
		op.Description = "Verify the passkey credential and return access tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/passkey/register/options", s.passkeyRegisterOptionsHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-register-options"
		op.Summary = "Begin passkey registration"
		op.Description = "Returns a WebAuthn challenge and options to register a new passkey"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
	huma.Post(api, "/auth/passkey/register/finish", s.passkeyRegisterFinishHandler, func(op *huma.Operation) {
		op.OperationID = "auth-passkey-register-finish"
		op.Summary = "Complete passkey registration"
		op.Description = "Save the new passkey credential for the authenticated user"
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		if authMiddleware != nil {
			op.Middlewares = huma.Middlewares{authMiddleware}
		}
	})
}
