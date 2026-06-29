package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var (
	metricsOnce       sync.Once
	httpLatency       metric.Float64Histogram
	httpInFlight      metric.Int64UpDownCounter
	workflowTasks     metric.Int64Counter
	workflowDuration  metric.Float64Histogram
	noopMeterProvider = noop.NewMeterProvider()
)

type operationHolder struct {
	operationID string
	route       string
}

type operationKey struct{}

func initMetrics() {
	meter := otel.Meter("koditon-backend")
	noopMeter := noopMeterProvider.Meter("koditon-backend-noop")
	var err error
	httpLatency, err = meter.Float64Histogram("http.server.request.duration", metric.WithDescription("HTTP request duration."), metric.WithUnit("s"))
	if err != nil {
		httpLatency, _ = noopMeter.Float64Histogram("http.server.request.duration")
	}
	httpInFlight, err = meter.Int64UpDownCounter("http.server.active_requests", metric.WithDescription("In-flight HTTP requests."))
	if err != nil {
		httpInFlight, _ = noopMeter.Int64UpDownCounter("http.server.active_requests")
	}
	workflowTasks, err = meter.Int64Counter("koditon.workflow.tasks", metric.WithDescription("Absurd workflow task executions."), metric.WithUnit("{task}"))
	if err != nil {
		workflowTasks, _ = noopMeter.Int64Counter("koditon.workflow.tasks")
	}
	workflowDuration, err = meter.Float64Histogram("koditon.workflow.task.duration", metric.WithDescription("Absurd workflow task duration."), metric.WithUnit("s"))
	if err != nil {
		workflowDuration, _ = noopMeter.Float64Histogram("koditon.workflow.task.duration")
	}
}

func ensureMetrics() {
	metricsOnce.Do(initMetrics)
}

func WithOperationHolder(ctx context.Context) context.Context {
	return context.WithValue(ctx, operationKey{}, &operationHolder{})
}

func SetOperationID(ctx context.Context, operationID string) {
	if operationID == "" {
		return
	}
	if holder, ok := ctx.Value(operationKey{}).(*operationHolder); ok && holder != nil {
		holder.operationID = operationID
	}
}

func SetOperationRoute(ctx context.Context, route string) {
	if route == "" {
		return
	}
	if holder, ok := ctx.Value(operationKey{}).(*operationHolder); ok && holder != nil {
		holder.route = route
	}
}

func RecordHTTPStart(ctx context.Context) {
	ensureMetrics()
	httpInFlight.Add(ctx, 1)
}

func RecordHTTPEnd(ctx context.Context, status int, duration time.Duration, method string) {
	ensureMetrics()
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", method),
		attribute.Int("http.response.status_code", status),
	}
	if holder, ok := ctx.Value(operationKey{}).(*operationHolder); ok && holder != nil {
		if holder.operationID != "" {
			attrs = append(attrs, Attrs.OperationID(holder.operationID))
		}
		if holder.route != "" {
			attrs = append(attrs, attribute.String("http.route", holder.route))
		}
	}
	recordOpts := metric.WithAttributes(attrs...)
	httpInFlight.Add(ctx, -1)
	httpLatency.Record(ctx, duration.Seconds(), recordOpts)
}

func RecordWorkflowTask(ctx context.Context, consumer, queueName, taskName, status string, duration time.Duration) {
	ensureMetrics()
	attrs := metric.WithAttributes(
		attribute.String("messaging.system", "absurd"),
		attribute.String("messaging.destination.name", queueName),
		attribute.String("koditon.workflow.consumer", consumer),
		attribute.String("koditon.workflow.task_name", taskName),
		attribute.String("koditon.workflow.status", status),
	)
	workflowTasks.Add(ctx, 1, attrs)
	workflowDuration.Record(ctx, duration.Seconds(), attrs)
}

func RegisterPGXPoolMetrics(pool *pgxpool.Pool) (func() error, error) {
	if pool == nil {
		return func() error { return nil }, nil
	}
	if err := otelpgx.RecordStats(pool, otelpgx.WithMinimumReadDBStatsInterval(15*time.Second)); err != nil {
		return func() error { return nil }, err
	}
	return func() error { return nil }, nil
}
