package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

func StartSpan(ctx context.Context, tracerName, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	spanCtx, span := Tracer(tracerName).Start(ctx, spanName)
	if span.IsRecording() && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return spanCtx, span
}

func RecordSpanError(span trace.Span, err error, msg string) {
	if err == nil {
		return
	}
	span.RecordError(err, trace.WithStackTrace(true))
	if msg == "" {
		msg = err.Error()
	}
	span.SetStatus(codes.Error, msg)
}
