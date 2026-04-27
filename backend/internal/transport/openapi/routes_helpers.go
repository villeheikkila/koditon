package api

import (
	"koditon-go/internal/platform/openapiutil"

	"github.com/danielgtaylor/huma/v2"
)

func applyAuth(op *huma.Operation, makeMiddleware func([]string) func(huma.Context, func(huma.Context)), scopes []string) {
	openapiutil.ApplyBearerSecurity(op, scopes)
	if makeMiddleware != nil {
		op.Middlewares = append(op.Middlewares, makeMiddleware(scopes))
	}
}

func resolveScopes(operationID string) []string {
	if _, ok := publicOperationIDs[operationID]; ok {
		return nil
	}
	if scopes, ok := scopesForOperationID(operationID); ok {
		return scopes
	}
	return nil
}
