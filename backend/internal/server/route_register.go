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

}
