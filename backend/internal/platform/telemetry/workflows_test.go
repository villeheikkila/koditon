package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestInjectWorkflowTraceHeaders(t *testing.T) {
	t.Parallel()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))
	headers := InjectWorkflowTraceHeaders(ctx, map[string]any{"existing": "yes"})
	if headers["existing"] != "yes" {
		t.Fatalf("existing header = %v", headers["existing"])
	}
	if headers["traceparent"] == "" {
		t.Fatalf("traceparent missing from headers: %#v", headers)
	}
}
