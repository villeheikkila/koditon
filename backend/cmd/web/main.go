package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/ads"
	"koditon-go/internal/config"
	"koditon-go/internal/web"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.SlogLevel()}))

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	adsService := ads.NewService(pool)
	handler := web.NewHandler(adsService, cfg.Shortcut.SitemapBase, cfg.Frontdoor.SitemapBase, logger)

	mux := http.NewServeMux()
	handler.Register(mux)

	addr := cfg.Host + ":" + cfg.Port
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", "err", err)
		}
	}()

	logger.Info("web server starting", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}
