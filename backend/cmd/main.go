package main

import (
	"context"
	"fmt"
	"io"
	"koditon-go/internal/config"
	"koditon-go/internal/db"
	"koditon-go/internal/hintatiedot"
	"koditon-go/internal/server"
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
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := config.Load(args, getenv, stderr)
	if err != nil {
		return err
	}
	logger := slog.New(tint.NewHandler(stderr, nil))
	slog.SetDefault(slog.New(
		tint.NewHandler(stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
	appLogger := logger.With("component", "app")
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	queries := db.New(pool)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	hintatiedotClient, err := hintatiedot.NewClient(httpClient, cfg.HintatiedotBaseURL, logger)
	if err != nil {
		return fmt.Errorf("create hintatiedot client: %w", err)
	}
	srv := server.New(logger, cfg, queries, hintatiedotClient)
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Koditon API", "0.1.0"))
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
		Handler: srv.Handler(mux, api),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error, 1)
	go func() {
		appLogger.InfoContext(ctx, "listening", "addr", httpServer.Addr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		appLogger.Info("shutting down", "timeout", cfg.ShutdownTimeout)
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			return fmt.Errorf("server shutdown: %w", shutdownErr)
		}
		if serveErr := <-errCh; serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
