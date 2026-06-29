package telemetry

import (
	"context"
	"log/slog"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"go.opentelemetry.io/otel/trace"
)

type traceContextHandler struct {
	next slog.Handler
}

func WithTraceContext(next slog.Handler) slog.Handler {
	if next == nil {
		return nil
	}
	return traceContextHandler{next: next}
}

func (h traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h traceContextHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	if task, ok := absurd.TaskFromContext(ctx); ok {
		record.AddAttrs(
			slog.String("workflow_queue", task.QueueName()),
			slog.String("workflow_task_name", task.TaskName()),
			slog.String("workflow_task_id", task.TaskID()),
			slog.String("workflow_run_id", task.RunID()),
			slog.Int("workflow_attempt", task.Attempt()),
		)
	}
	return h.next.Handle(ctx, record)
}

func (h traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceContextHandler) WithGroup(name string) slog.Handler {
	return traceContextHandler{next: h.next.WithGroup(name)}
}
