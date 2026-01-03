package server

import (
	"koditon-go/internal/auth"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) addRoutes(api huma.API) {
	var authMiddleware func(huma.Context, func(huma.Context))
	if s.authService != nil {
		authHandlers := auth.NewHandlers(s.authService)
		authMiddleware = auth.AuthMiddlewareFactory(api, s.authService)
		authHandlers.RegisterRoutes(api, authMiddleware)
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
}
