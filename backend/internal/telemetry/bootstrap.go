package telemetry

import (
	"context"
	"io"
	"log/slog"

	"koditon-go/internal/logging"
	"koditon-go/internal/requestid"
	"koditon-go/internal/runtimecfg"
)

type BootstrapResult struct {
	Logger    *slog.Logger
	Shutdown  func(context.Context) error
	Enabled   bool
	Handler   slog.Handler
	InitError error
}

func Bootstrap(ctx context.Context, cfg *runtimecfg.TelemetryConfig, environment string, baseHandler slog.Handler, logger *slog.Logger) BootstrapResult {
	if logger == nil {
		logger = slog.Default()
	}
	if baseHandler == nil {
		baseHandler = slog.NewTextHandler(io.Discard, nil)
	}

	telemetryCfg := Config{
		Enabled:     cfg != nil,
		Environment: environment,
	}
	if cfg != nil {
		telemetryCfg.ServiceName = cfg.ServiceName
		telemetryCfg.OTLPProtocol = cfg.OTLPProtocol
		telemetryCfg.OTLPEndpoint = cfg.OTLPEndpoint
		telemetryCfg.OTLPInsecure = cfg.OTLPInsecure
		telemetryCfg.SampleRatio = cfg.SampleRatio
	}

	result, err := Init(ctx, telemetryCfg, logger)
	enabled := telemetryCfg.Enabled
	if err != nil {
		enabled = false
		result = InitResult{
			Shutdown:   func(context.Context) error { return nil },
			LogHandler: slog.NewTextHandler(io.Discard, nil),
		}
		logger.WarnContext(ctx, "telemetry init failed, continuing without telemetry", "error", err)
	}

	combinedHandler := logging.NewMultiHandler(WithTraceContext(baseHandler), result.LogHandler)
	combinedHandler = requestid.NewHandler(combinedHandler)
	combinedLogger := slog.New(combinedHandler)
	slog.SetDefault(combinedLogger)

	return BootstrapResult{
		Logger:    combinedLogger,
		Shutdown:  result.Shutdown,
		Enabled:   enabled,
		Handler:   combinedHandler,
		InitError: err,
	}
}
