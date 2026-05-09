package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/cmd/tui/internal/tui"
	"koditon/internal/domain/ads"
	"koditon/internal/platform/config"
	syncflows "koditon/internal/sync/flows"
	"koditon/internal/sync/frontdoor"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
)

func main() {
	if err := run(context.Background(), os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, stderr io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.DiscardHandler)
	slog.SetDefault(logger)
	log.SetOutput(io.Discard)
	log.SetFlags(0)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	pricesService, err := prices.NewService(pool, cfg.Prices.BaseURL, cfg.OpenRouter.APIKey)
	if err != nil {
		return fmt.Errorf("create prices service: %w", err)
	}
	adsService := ads.NewService(pool)
	shortcutService := shortcut.NewService(pool, logger, cfg.Shortcut.BaseURL, cfg.Shortcut.DocsBaseURL, cfg.Shortcut.AdBaseURL, cfg.Shortcut.UserAgent, cfg.Shortcut.SitemapBase)
	frontdoorService := frontdoor.NewService(pool, logger, cfg.Frontdoor.BaseURL, cfg.Frontdoor.UserAgent, cfg.Frontdoor.Cookie, cfg.Frontdoor.SitemapBase)
	postalService := postal.NewService(pool)
	runner := syncflows.NewRunner(logger, adsService, pricesService, shortcutService, frontdoorService, postalService)
	syncJobStore := syncjobs.NewStore(logger, pool)
	p := tea.NewProgram(tui.NewApp(runner, tui.WithWebBaseURL(cfg.WebBaseURL), tui.WithSyncJobs(syncJobStore), tui.WithDBPool(pool)).Model(), tea.WithOutput(stderr))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
