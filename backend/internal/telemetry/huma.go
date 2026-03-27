package telemetry

import (
	"github.com/danielgtaylor/huma/v2"
	"go.opentelemetry.io/otel/trace"
)

func HumaOperationMiddleware() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		span := trace.SpanFromContext(ctx.Context())
		if span != nil && span.IsRecording() {
			operation := ctx.Operation()
			if operation != nil {
				if operation.OperationID != "" {
					span.SetName(operation.OperationID)
					span.SetAttributes(Attrs.OperationID(operation.OperationID))
					SetOperationID(ctx.Context(), operation.OperationID)
				}
				if operation.Summary != "" {
					span.SetAttributes(Attrs.OperationSummary(operation.Summary))
				}
				if operation.Path != "" {
					span.SetAttributes(Attrs.OperationPath(operation.Path))
				}
			}
			if method := ctx.Method(); method != "" {
				span.SetAttributes(Attrs.OperationMethod(method))
			}
		}
		next(ctx)
	}
}
