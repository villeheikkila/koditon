package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var (
	metricsOnce sync.Once

	httpRequests      metric.Int64Counter
	httpLatencyMs     metric.Float64Histogram
	httpInFlight      metric.Int64UpDownCounter
	noopMeterProvider = noop.NewMeterProvider()
)

type operationIDHolder struct {
	value string
}

type operationIDKey struct{}

func initMetrics() {
	meter := otel.Meter("koditon-go")
	noopMeter := noopMeterProvider.Meter("koditon-go-noop")

	var err error
	httpRequests, err = meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total HTTP requests."),
	)
	if err != nil {
		httpRequests, _ = noopMeter.Int64Counter("http.server.requests")
	}

	httpLatencyMs, err = meter.Float64Histogram(
		"http.server.latency_ms",
		metric.WithDescription("HTTP request latency in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		httpLatencyMs, _ = noopMeter.Float64Histogram("http.server.latency_ms")
	}

	httpInFlight, err = meter.Int64UpDownCounter(
		"http.server.in_flight",
		metric.WithDescription("In-flight HTTP requests."),
	)
	if err != nil {
		httpInFlight, _ = noopMeter.Int64UpDownCounter("http.server.in_flight")
	}
}

func ensureMetrics() {
	metricsOnce.Do(initMetrics)
}

func WithOperationIDHolder(ctx context.Context) (context.Context, *operationIDHolder) {
	holder := &operationIDHolder{}
	return context.WithValue(ctx, operationIDKey{}, holder), holder
}

func SetOperationID(ctx context.Context, operationID string) {
	if operationID == "" {
		return
	}
	if holder, ok := ctx.Value(operationIDKey{}).(*operationIDHolder); ok && holder != nil {
		holder.value = operationID
	}
}

func OperationIDFromContext(ctx context.Context) (string, bool) {
	holder, ok := ctx.Value(operationIDKey{}).(*operationIDHolder)
	if !ok || holder == nil || holder.value == "" {
		return "", false
	}
	return holder.value, true
}

func RecordHTTPStart(ctx context.Context) {
	ensureMetrics()
	httpInFlight.Add(ctx, 1)
}

func RecordHTTPEnd(ctx context.Context, status int, duration time.Duration, method, path string) {
	ensureMetrics()
	attrs := []attribute.KeyValue{
		Attrs.OperationMethod(method),
		attribute.Int("http.status_code", status),
	}
	if operationID, ok := OperationIDFromContext(ctx); ok {
		attrs = append(attrs, Attrs.OperationID(operationID))
	} else if path != "" {
		attrs = append(attrs, Attrs.OperationPath(path))
	}
	recordOpts := metric.WithAttributes(attrs...)
	httpInFlight.Add(ctx, -1)
	httpRequests.Add(ctx, 1, recordOpts)
	httpLatencyMs.Record(ctx, float64(duration.Milliseconds()), recordOpts)
}

func RegisterPGXPoolMetrics(pool *pgxpool.Pool) (func() error, error) {
	if pool == nil {
		return func() error { return nil }, nil
	}
	meter := otel.Meter("koditon-go")

	totalGauge, err := meter.Int64ObservableGauge(
		"db.pool.total",
		metric.WithDescription("Total database connections."),
	)
	if err != nil {
		return func() error { return nil }, err
	}
	idleGauge, err := meter.Int64ObservableGauge(
		"db.pool.idle",
		metric.WithDescription("Idle database connections."),
	)
	if err != nil {
		return func() error { return nil }, err
	}
	acquiredGauge, err := meter.Int64ObservableGauge(
		"db.pool.acquired",
		metric.WithDescription("Acquired database connections."),
	)
	if err != nil {
		return func() error { return nil }, err
	}
	constructingGauge, err := meter.Int64ObservableGauge(
		"db.pool.constructing",
		metric.WithDescription("Connections being constructed."),
	)
	if err != nil {
		return func() error { return nil }, err
	}
	maxGauge, err := meter.Int64ObservableGauge(
		"db.pool.max",
		metric.WithDescription("Max connections allowed in pool."),
	)
	if err != nil {
		return func() error { return nil }, err
	}

	reg, err := meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		stats := pool.Stat()
		observer.ObserveInt64(totalGauge, int64(stats.TotalConns()))
		observer.ObserveInt64(idleGauge, int64(stats.IdleConns()))
		observer.ObserveInt64(acquiredGauge, int64(stats.AcquiredConns()))
		observer.ObserveInt64(constructingGauge, int64(stats.ConstructingConns()))
		observer.ObserveInt64(maxGauge, int64(stats.MaxConns()))
		return nil
	}, totalGauge, idleGauge, acquiredGauge, constructingGauge, maxGauge)
	if err != nil {
		return func() error { return nil }, err
	}

	return reg.Unregister, nil
}
