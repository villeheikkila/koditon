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
	openrouter "github.com/revrost/go-openrouter"

	"koditon-go/cmd/tui/internal/tui"
	"koditon-go/internal/domain/ads"
	"koditon-go/internal/platform/config"
	"koditon-go/internal/sync/flows"
	"koditon-go/internal/sync/frontdoor"
	"koditon-go/internal/sync/postal"
	"koditon-go/internal/sync/prices"
	"koditon-go/internal/sync/shortcut"
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
	openRouterClient := openrouter.NewClient(cfg.OpenRouter.APIKey)
	pricesService, err := prices.NewService(pool, cfg.Prices.BaseURL, openRouterClient)
	if err != nil {
		return fmt.Errorf("create prices service: %w", err)
	}
	adsService := ads.NewService(pool)
	shortcutService := shortcut.NewService(pool, logger, cfg.Shortcut.BaseURL, cfg.Shortcut.DocsBaseURL, cfg.Shortcut.AdBaseURL, cfg.Shortcut.UserAgent, cfg.Shortcut.SitemapBase)
	frontdoorService := frontdoor.NewService(pool, logger, cfg.Frontdoor.BaseURL, cfg.Frontdoor.UserAgent, cfg.Frontdoor.Cookie, cfg.Frontdoor.SitemapBase)
	postalService := postal.NewService(pool)
	runner := syncflows.NewRunner(logger, adsService, pricesService, shortcutService, frontdoorService, postalService)
	p := tea.NewProgram(tui.NewApp(runner, tui.WithWebBaseURL(cfg.WebBaseURL)).Model(), tea.WithOutput(stderr))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
