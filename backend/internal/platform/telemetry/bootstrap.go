package telemetry

import (
	"context"
	"io"
	"log/slog"
)

type BootstrapResult struct {
	Logger    *slog.Logger
	Shutdown  func(context.Context) error
	Enabled   bool
	Handler   slog.Handler
	InitError error
}

func Bootstrap(ctx context.Context, cfg *Config, environment string, baseHandler slog.Handler, logger *slog.Logger) BootstrapResult {
	if logger == nil {
		logger = slog.Default()
	}
	if baseHandler == nil {
		baseHandler = slog.NewTextHandler(io.Discard, nil)
	}
	telemetryCfg := Config{Enabled: cfg != nil, Environment: environment}
	if cfg != nil {
		telemetryCfg.ServiceName = cfg.ServiceName
		telemetryCfg.Version = cfg.Version
		telemetryCfg.InstanceID = cfg.InstanceID
		telemetryCfg.OTLPProtocol = cfg.OTLPProtocol
		telemetryCfg.OTLPEndpoint = cfg.OTLPEndpoint
		telemetryCfg.OTLPInsecure = cfg.OTLPInsecure
		telemetryCfg.OTLPHeaders = cfg.OTLPHeaders
		telemetryCfg.Sampler = cfg.Sampler
		telemetryCfg.SamplerArg = cfg.SamplerArg
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
	combinedHandler := multiHandler{handlers: []slog.Handler{WithTraceContext(baseHandler), result.LogHandler}}
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

type multiHandler struct {
	handlers []slog.Handler
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler != nil && handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if handler == nil || !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		if handler != nil {
			next = append(next, handler.WithAttrs(attrs))
		}
	}
	return multiHandler{handlers: next}
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		if handler != nil {
			next = append(next, handler.WithGroup(name))
		}
	}
	return multiHandler{handlers: next}
}
