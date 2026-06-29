package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const WorkflowTracerName = "koditon/internal/sync/workflows"

func WorkflowTaskAttrs(queueName, taskName, taskID, runID string, attempt int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", "absurd"),
		attribute.String("messaging.destination.name", queueName),
		attribute.String("koditon.workflow.queue", queueName),
		attribute.String("koditon.workflow.task_name", taskName),
		attribute.String("koditon.workflow.task_id", taskID),
		attribute.String("koditon.workflow.run_id", runID),
		attribute.Int("koditon.workflow.attempt", attempt),
	}
}

func InjectWorkflowTraceHeaders(ctx context.Context, headers map[string]any) map[string]any {
	if headers == nil {
		headers = map[string]any{}
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for key, value := range carrier {
		headers[key] = value
	}
	return headers
}

func ExtractWorkflowTraceContext(ctx context.Context, headers map[string]any) context.Context {
	carrier := propagation.MapCarrier{}
	for key, value := range headers {
		if stringValue, ok := value.(string); ok {
			carrier[key] = stringValue
		}
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

func WrapAbsurdTaskExecution(logger *slog.Logger, consumer string) absurd.WrapTaskExecutionHook {
	return func(task *absurd.TaskContext, execute func() (any, error)) (result any, err error) {
		extracted := ExtractWorkflowTraceContext(task.Context, task.Headers())
		task.Context = extracted
		ctx, span := Tracer(WorkflowTracerName).Start(
			task.Context,
			fmt.Sprintf("workflow.task %s", task.TaskName()),
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(WorkflowTaskAttrs(task.QueueName(), task.TaskName(), task.TaskID(), task.RunID(), task.Attempt())...),
		)
		task.Context = ctx
		startedAt := time.Now()
		status := "completed"
		defer func() {
			duration := time.Since(startedAt)
			if recovered := recover(); recovered != nil {
				status = "panicked"
				err = fmt.Errorf("panic: %v", recovered)
				RecordSpanError(span, err, "workflow task panicked")
				finishWorkflowTaskTelemetry(task, logger, consumer, span, status, duration, err)
				panic(recovered)
			}
			if err != nil {
				status = "failed"
				RecordSpanError(span, err, "workflow task failed")
			} else {
				span.SetStatus(codes.Ok, "")
			}
			finishWorkflowTaskTelemetry(task, logger, consumer, span, status, duration, err)
		}()
		return execute()
	}
}

func finishWorkflowTaskTelemetry(task *absurd.TaskContext, logger *slog.Logger, consumer string, span trace.Span, status string, duration time.Duration, err error) {
	span.SetAttributes(attribute.String("koditon.workflow.status", status))
	RecordWorkflowTask(task.Context, consumer, task.QueueName(), task.TaskName(), status, duration)
	if logger != nil {
		attrs := []any{"queue", task.QueueName(), "task_name", task.TaskName(), "task_id", task.TaskID(), "run_id", task.RunID(), "attempt", task.Attempt(), "status", status, "duration_ms", duration.Milliseconds()}
		if err != nil {
			attrs = append(attrs, "error", err)
			logger.WarnContext(task.Context, "workflow task finished", attrs...)
		} else {
			logger.DebugContext(task.Context, "workflow task finished", attrs...)
		}
	}
	span.End()
}
