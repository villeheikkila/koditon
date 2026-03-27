package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
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
	SampleRatio  float64
}

type InitResult struct {
	Shutdown   func(context.Context) error
	LogHandler slog.Handler
}

func Init(ctx context.Context, cfg Config, logger *slog.Logger) (InitResult, error) {
	if !cfg.Enabled {
		return InitResult{
			Shutdown:   func(context.Context) error { return nil },
			LogHandler: slog.NewTextHandler(io.Discard, nil),
		}, nil
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "koditon-api"
	}
	protocol := strings.ToLower(strings.TrimSpace(cfg.OTLPProtocol))
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

	sampleRatio := cfg.SampleRatio
	if sampleRatio < 0 || sampleRatio > 1 {
		if logger != nil {
			logger.WarnContext(ctx, "invalid telemetry sample ratio, defaulting to 1.0", "sample_ratio", sampleRatio)
		}
		sampleRatio = 1
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
	}
	if env := strings.TrimSpace(cfg.Environment); env != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(env))
	}
	if version := strings.TrimSpace(os.Getenv("OTEL_SERVICE_VERSION")); version != "" {
		attrs = append(attrs, semconv.ServiceVersionKey.String(version))
	}
	if instanceID := strings.TrimSpace(os.Getenv("OTEL_SERVICE_INSTANCE_ID")); instanceID != "" {
		attrs = append(attrs, semconv.ServiceInstanceIDKey.String(instanceID))
	}
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return InitResult{}, fmt.Errorf("create resource: %w", err)
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
	var exporter sdktrace.SpanExporter
	switch protocol {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case "http":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		return InitResult{}, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
	if err != nil {
		return InitResult{}, fmt.Errorf("create exporter: %w", err)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	var logExporter sdklog.Exporter
	switch protocol {
	case "grpc":
		opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		logExporter, err = otlploggrpc.New(ctx, opts...)
	case "http":
		opts := []otlploghttp.Option{otlploghttp.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		logExporter, err = otlploghttp.New(ctx, opts...)
	default:
		cleanup(ctx)
		return InitResult{}, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
	if err != nil {
		cleanup(ctx)
		return InitResult{}, fmt.Errorf("create log exporter: %w", err)
	}

	logProcessor := sdklog.NewBatchProcessor(logExporter)
	logProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(logProcessor),
		sdklog.WithResource(res),
	)
	logglobal.SetLoggerProvider(logProvider)
	logHandler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(logProvider))

	var metricExporter sdkmetric.Exporter
	switch protocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		metricExporter, err = otlpmetricgrpc.New(ctx, opts...)
	case "http":
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if cfg.OTLPInsecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		metricExporter, err = otlpmetrichttp.New(ctx, opts...)
	default:
		cleanup(ctx)
		return InitResult{}, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}
	if err != nil {
		cleanup(ctx)
		return InitResult{}, fmt.Errorf("create metric exporter: %w", err)
	}

	metricReader := sdkmetric.NewPeriodicReader(metricExporter)
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	logger.InfoContext(ctx, "telemetry enabled",
		"service_name", serviceName,
		"otlp_protocol", protocol,
		"otlp_endpoint", endpoint,
		"sample_ratio", sampleRatio,
	)

	shutdown := func(ctx context.Context) error {
		if err := logProvider.Shutdown(ctx); err != nil {
			return err
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return tracerProvider.Shutdown(ctx)
	}

	return InitResult{
		Shutdown:   shutdown,
		LogHandler: logHandler,
	}, nil
}
