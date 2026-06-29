package telemetry

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type captureHandler struct {
	attrs []slog.Attr
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	record.Attrs(func(attr slog.Attr) bool {
		h.attrs = append(h.attrs, attr)
		return true
	})
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &captureHandler{attrs: slices.Clone(h.attrs)}
	next.attrs = append(next.attrs, attrs...)
	return next
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return h
}

func TestWithTraceContextAddsSpanFields(t *testing.T) {
	t.Parallel()
	provider := sdktrace.NewTracerProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()
	otel.SetTracerProvider(provider)
	ctx, span := otel.Tracer("test").Start(context.Background(), "test-span")
	defer span.End()
	capture := &captureHandler{}
	if err := WithTraceContext(capture).Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !hasAttr(capture.attrs, "trace_id") {
		t.Fatalf("trace_id missing from attrs: %#v", capture.attrs)
	}
	if !hasAttr(capture.attrs, "span_id") {
		t.Fatalf("span_id missing from attrs: %#v", capture.attrs)
	}
}

func hasAttr(attrs []slog.Attr, key string) bool {
	for _, attr := range attrs {
		if attr.Key == key && attr.Value.String() != "" {
			return true
		}
	}
	return false
}
