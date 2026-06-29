package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type Config struct {
	Enabled      bool
	ServiceName  string
	Environment  string
	OTLPProtocol string
	OTLPEndpoint string
	OTLPInsecure bool
	OTLPHeaders  map[string]string
	Version      string
	InstanceID   string
	Sampler      string
	SamplerArg   string
	SampleRatio  float64
}

type InitResult struct {
	Shutdown   func(context.Context) error
	LogHandler slog.Handler
}

func Init(ctx context.Context, cfg Config, logger *slog.Logger) (InitResult, error) {
	if !cfg.Enabled {
		return InitResult{Shutdown: func(context.Context) error { return nil }, LogHandler: slog.NewTextHandler(io.Discard, nil)}, nil
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "koditon-backend"
	}
	protocol := normalizeOTLPProtocol(cfg.OTLPProtocol)
	if protocol == "" {
		protocol = "http"
	}
	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		if protocol == "grpc" {
			endpoint = "localhost:4317"
		} else {
			endpoint = "localhost:4318"
		}
	}
	sampler, sampleRatio := traceSampler(cfg, logger, ctx)
	if sampleRatio < 0 || sampleRatio > 1 {
		if logger != nil {
			logger.WarnContext(ctx, "invalid telemetry sample ratio, defaulting to 1.0", "sample_ratio", sampleRatio)
		}
		sampleRatio = 1
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))
	}
	res, err := telemetryResource(ctx, cfg, serviceName)
	if err != nil {
		return InitResult{}, err
	}
	var logProvider *sdklog.LoggerProvider
	var meterProvider *sdkmetric.MeterProvider
	var tracerProvider *sdktrace.TracerProvider
	cleanup := func(ctx context.Context) {
		if meterProvider != nil {
			_ = meterProvider.Shutdown(ctx)
		}
		if logProvider != nil {
			_ = logProvider.Shutdown(ctx)
		}
		if tracerProvider != nil {
			_ = tracerProvider.Shutdown(ctx)
		}
	}
	traceExporter, err := newTraceExporter(ctx, protocol, endpoint, cfg)
	if err != nil {
		return InitResult{}, err
	}
	tracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler), sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	logExporter, err := newLogExporter(ctx, protocol, endpoint, cfg)
	if err != nil {
		cleanup(ctx)
		return InitResult{}, err
	}
	logProvider = sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)), sdklog.WithResource(res))
	logglobal.SetLoggerProvider(logProvider)
	logHandler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(logProvider))
	metricExporter, err := newMetricExporter(ctx, protocol, endpoint, cfg)
	if err != nil {
		cleanup(ctx)
		return InitResult{}, err
	}
	meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)), sdkmetric.WithResource(res))
	otel.SetMeterProvider(meterProvider)
	if err := otelruntime.Start(otelruntime.WithMeterProvider(meterProvider)); err != nil && logger != nil {
		logger.WarnContext(ctx, "runtime telemetry init failed", "error", err)
	}
	if logger != nil {
		logger.InfoContext(ctx, "telemetry enabled", "service_name", serviceName, "otlp_protocol", protocol, "otlp_endpoint", endpoint, "sample_ratio", sampleRatio)
	}
	shutdown := func(ctx context.Context) error {
		if err := logProvider.Shutdown(ctx); err != nil {
			return err
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return tracerProvider.Shutdown(ctx)
	}
	return InitResult{Shutdown: shutdown, LogHandler: logHandler}, nil
}

func telemetryResource(ctx context.Context, cfg Config, serviceName string) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceNameKey.String(serviceName)}
	if env := strings.TrimSpace(cfg.Environment); env != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(env))
	}
	if version := firstNonEmpty(cfg.Version, os.Getenv("OTEL_SERVICE_VERSION")); version != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(version))
	}
	if instanceID := firstNonEmpty(cfg.InstanceID, os.Getenv("OTEL_SERVICE_INSTANCE_ID")); instanceID != "" {
		attrs = append(attrs, semconv.ServiceInstanceIDKey.String(instanceID))
	}
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithProcess(), resource.WithOS(), resource.WithHost(), resource.WithTelemetrySDK(), resource.WithAttributes(attrs...))
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}
	return res, nil
}

func newTraceExporter(ctx context.Context, protocol, endpoint string, cfg Config) (sdktrace.SpanExporter, error) {
	switch protocol {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(grpcEndpoint(endpoint))}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create trace exporter: %w", err)
		}
		return exporter, nil
	case "http":
		opts := []otlptracehttp.Option{traceHTTPExporterEndpointOption(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create trace exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
}

func newLogExporter(ctx context.Context, protocol, endpoint string, cfg Config) (sdklog.Exporter, error) {
	switch protocol {
	case "grpc":
		opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(grpcEndpoint(endpoint))}
		if cfg.OTLPInsecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlploggrpc.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create log exporter: %w", err)
		}
		return exporter, nil
	case "http":
		opts := []otlploghttp.Option{logHTTPExporterEndpointOption(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create log exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
}

func newMetricExporter(ctx context.Context, protocol, endpoint string, cfg Config) (sdkmetric.Exporter, error) {
	switch protocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(grpcEndpoint(endpoint))}
		if cfg.OTLPInsecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create metric exporter: %w", err)
		}
		return exporter, nil
	case "http":
		opts := []otlpmetrichttp.Option{metricHTTPExporterEndpointOption(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(cfg.OTLPHeaders))
		}
		exporter, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create metric exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
}

func traceSampler(cfg Config, logger *slog.Logger, ctx context.Context) (sdktrace.Sampler, float64) {
	sampler := strings.ToLower(strings.TrimSpace(cfg.Sampler))
	arg := strings.TrimSpace(cfg.SamplerArg)
	if sampler == "" {
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio)), cfg.SampleRatio
	}
	switch sampler {
	case "always_on":
		return sdktrace.AlwaysSample(), 1
	case "always_off":
		return sdktrace.NeverSample(), 0
	case "traceidratio", "parentbased_traceidratio":
		ratio := cfg.SampleRatio
		if arg != "" {
			parsed, err := strconv.ParseFloat(arg, 64)
			if err != nil {
				if logger != nil {
					logger.WarnContext(ctx, "invalid OTEL_TRACES_SAMPLER_ARG, using legacy sample ratio", "sampler_arg", arg)
				}
			} else {
				ratio = parsed
			}
		}
		if sampler == "traceidratio" {
			return sdktrace.TraceIDRatioBased(ratio), ratio
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), ratio
	default:
		if logger != nil {
			logger.WarnContext(ctx, "unsupported OTEL_TRACES_SAMPLER, using parentbased_traceidratio", "sampler", sampler)
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio)), cfg.SampleRatio
	}
}

func normalizeOTLPProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "http", "http/protobuf":
		return "http"
	case "grpc":
		return "grpc"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func traceHTTPExporterEndpointOption(endpoint string) otlptracehttp.Option {
	if endpointURL := httpSignalEndpointURL(endpoint, "/v1/traces"); endpointURL != "" {
		return otlptracehttp.WithEndpointURL(endpointURL)
	}
	return otlptracehttp.WithEndpoint(endpoint)
}

func logHTTPExporterEndpointOption(endpoint string) otlploghttp.Option {
	if endpointURL := httpSignalEndpointURL(endpoint, "/v1/logs"); endpointURL != "" {
		return otlploghttp.WithEndpointURL(endpointURL)
	}
	return otlploghttp.WithEndpoint(endpoint)
}

func metricHTTPExporterEndpointOption(endpoint string) otlpmetrichttp.Option {
	if endpointURL := httpSignalEndpointURL(endpoint, "/v1/metrics"); endpointURL != "" {
		return otlpmetrichttp.WithEndpointURL(endpointURL)
	}
	return otlpmetrichttp.WithEndpoint(endpoint)
}

func httpSignalEndpointURL(endpoint, signalPath string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch strings.TrimRight(parsed.Path, "/") {
	case "", "/api/otel":
		parsed.Path = strings.TrimRight(parsed.Path, "/") + signalPath
		return parsed.String()
	default:
		return endpoint
	}
}

func grpcEndpoint(endpoint string) string {
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return parsed.Host
	}
	return endpoint
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
