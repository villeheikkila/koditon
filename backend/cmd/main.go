package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"koditon-go/internal/config"
	"koditon-go/internal/consumers"
	"koditon-go/internal/frontdoor"
	"koditon-go/internal/prices"
	"koditon-go/internal/server"
	"koditon-go/internal/shortcut"
	"koditon-go/internal/taskqueue"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	_ []string,
	_ func(string) string,
	_ io.Reader,
	_, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(stderr, cfg)
	slog.SetDefault(logger)
	appLogger := logger.With("component", "app")
	appLogger.Info("starting application",
		"env", cfg.Environment,
		"log_level", cfg.LogLevel,
	)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	appLogger.Debug("database connection established")
	taskQueueClient := taskqueue.NewClient(pool)
	pricesService, err := prices.NewService(
		pool,
		cfg.Prices.BaseURL,
	)
	if err != nil {
		return fmt.Errorf("create prices service: %w", err)
	}
	shortcutService := shortcut.NewService(
		pool,
		logger,
		cfg.Shortcut.BaseURL,
		cfg.Shortcut.DocsBaseURL,
		cfg.Shortcut.AdBaseURL,
		cfg.Shortcut.UserAgent,
		cfg.Shortcut.SitemapBase,
	)
	frontdoorService := frontdoor.NewService(
		pool,
		cfg.Frontdoor.BaseURL,
		cfg.Frontdoor.UserAgent,
		cfg.Frontdoor.Cookie,
		cfg.Frontdoor.SitemapBase,
	)
	consumer := consumers.New(
		logger,
		taskQueueClient,
		pricesService,
		shortcutService,
		frontdoorService,
	)
	consumerConfig := consumers.DefaultConfig()
	if err := consumer.Start(ctx, consumerConfig, pool); err != nil {
		return fmt.Errorf("start consumer: %w", err)
	}
	srv := server.New(logger, cfg, pool, taskQueueClient)
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Koditon API", "0.1.0"))
	httpServer := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           srv.Handler(mux, api),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	errCh := make(chan error, 1)
	go func() {
		appLogger.Info("server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		appLogger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			consumer.Stop()
			return fmt.Errorf("http server: %w", err)
		}
		consumer.Stop()
		return nil
	}
	// graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	var shutdownErrs []error
	appLogger.Debug("stopping consumer")
	consumer.Stop()
	appLogger.Debug("consumer stopped")
	appLogger.Debug("shutting down http server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("http server shutdown failed", tint.Err(err))
		shutdownErrs = append(shutdownErrs, fmt.Errorf("http server shutdown: %w", err))
	} else {
		appLogger.Debug("http server stopped")
	}
	if err := <-errCh; err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("http server: %w", err))
	}
	appLogger.Debug("closing database pool")
	pool.Close()
	appLogger.Debug("database pool closed")
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	appLogger.Info("graceful shutdown complete")
	return nil
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return os.Getenv("TERM") != "dumb" &&
		(fd == os.Stdout.Fd() || fd == os.Stderr.Fd())
}

func newLogger(w io.Writer, cfg config.Config) *slog.Logger {
	isTTY := isTerminal(w)
	opts := &tint.Options{
		Level:      cfg.SlogLevel(),
		TimeFormat: "15:04:05",
		NoColor:    !isTTY,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey && len(groups) == 0 {
				return slog.Attr{Key: a.Key, Value: slog.StringValue(formatLevel(a.Value.Any().(slog.Level)))}
			}
			return a
		},
	}
	if cfg.Environment.IsDevelopment() {
		opts.AddSource = true
	}
	return slog.New(tint.NewHandler(w, opts))
}

func formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DBG"
	case slog.LevelInfo:
		return "INF"
	case slog.LevelWarn:
		return "WRN"
	case slog.LevelError:
		return "ERR"
	default:
		return level.String()
	}
}
