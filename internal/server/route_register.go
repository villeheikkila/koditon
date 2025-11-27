package server

import "github.com/danielgtaylor/huma/v2"

func (s *Server) addRoutes(api huma.API) {
	huma.Get(api, "/healthz", s.healthHandler, func(op *huma.Operation) {
		op.OperationID = "healthz"
		op.Summary = "Health check"
	})
	huma.Post(api, "/api/v1/ping", s.pingHandler, func(op *huma.Operation) {
		op.OperationID = "ping"
		op.Summary = "Echo a message"
	})
	huma.Get(api, "/api/v1/cities", s.listCitiesHandler, func(op *huma.Operation) {
		op.OperationID = "listCities"
		op.Summary = "List cities with neighborhoods"
	})
	huma.Get(api, "/api/v1/transactions", s.listTransactionsHandler, func(op *huma.Operation) {
		op.OperationID = "listTransactions"
		op.Summary = "List transactions for neighborhoods"
	})
	huma.Post(api, "/api/v1/transactions/fetch", s.fetchTransactionsHandler, func(op *huma.Operation) {
		op.OperationID = "fetchTransactions"
		op.Summary = "Fetch transactions from Hintatiedot for Helsinki"
	})
	huma.Get(api, "/api/v1/hintatiedot/cities", s.fetchHintatiedotCitiesHandler, func(op *huma.Operation) {
		op.OperationID = "fetchHintatiedotCities"
		op.Summary = "Fetch cities from Hintatiedot"
	})
	huma.Post(api, "/api/v1/hintatiedot/sync", s.syncHintatiedotCityHandler, func(op *huma.Operation) {
		op.OperationID = "syncHintatiedotCity"
		op.Summary = "Sync a city from Hintatiedot"
	})
}
