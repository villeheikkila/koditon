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
}
