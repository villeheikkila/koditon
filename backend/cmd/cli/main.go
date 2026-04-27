package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	openrouter "github.com/revrost/go-openrouter"

	"koditon-go/cmd/cli/internal/cli"
	"koditon-go/internal/domain/ads"
	"koditon-go/internal/platform/config"
	"koditon-go/internal/sync/prices"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	subArgs := os.Args[2:]

	switch subcommand {
	case "api-query":
		return runAPIQuery(ctx, subArgs, os.Stdout, os.Getenv)
	case "search":
		return runSearch(ctx, subArgs)
	case "detail":
		return runDetail(ctx, subArgs)
	case "transactions":
		return runTransactions(ctx, subArgs)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
		return nil
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: cli <command> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  search         Search ads, buildings, and announcements")
	fmt.Fprintln(os.Stderr, "  detail         Show entity detail by canonical ID or URL")
	fmt.Fprintln(os.Stderr, "  transactions   Search price transactions")
	fmt.Fprintln(os.Stderr, "  api-query      Run raw Frontdoor and Shortcut provider client queries")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Run 'cli <command> --help' for command-specific flags.")
}

func runSearch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	var f cli.SearchFlags
	fs.StringVar(&f.Query, "query", "", "Free text search")
	fs.StringVar(&f.Source, "source", "all", "Source filter: shortcut, frontdoor, all")
	fs.StringVar(&f.Kind, "kind", "all", "Kind filter: ad, building, announcement, all")
	fs.StringVar(&f.ListingType, "type", "all", "Listing type: listing, rental, all")
	fs.StringVar(&f.City, "city", "", "City filter")
	fs.StringVar(&f.Postal, "postal", "", "Postal code filter")
	fs.IntVar(&f.MinPrice, "min-price", 0, "Minimum price")
	fs.IntVar(&f.MaxPrice, "max-price", 0, "Maximum price")
	fs.Float64Var(&f.MinArea, "min-area", 0, "Minimum area (m²)")
	fs.Float64Var(&f.MaxArea, "max-area", 0, "Maximum area (m²)")
	fs.StringVar(&f.Sort, "sort", "seen_desc", "Sort: price_asc, price_desc, area_asc, area_desc, seen_desc")
	fs.IntVar(&f.Limit, "limit", 25, "Results per page (25, 50, 100)")
	fs.IntVar(&f.Page, "page", 1, "Page number")
	fs.StringVar(&f.PublishedAfter, "after", "", "Published after date (YYYY-MM-DD)")
	fs.StringVar(&f.PublishedBefore, "before", "", "Published before date (YYYY-MM-DD)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pool, cfg, err := setup(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	_ = cfg
	adsService := ads.NewService(pool)
	return cli.RunSearch(ctx, adsService, f)
}

func runDetail(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("detail", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: cli detail <canonical-id-or-url>\n  example: cli detail shortcut:ad:12345\n  example: cli detail https://example.com/myytavat-asunnot/12345")
	}
	input := fs.Arg(0)

	pool, cfg, err := setup(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	adsService := ads.NewService(pool)
	return cli.RunDetail(ctx, adsService, input, cfg.Shortcut.SitemapBase, cfg.Frontdoor.SitemapBase, cfg.WebBaseURL)
}

func runTransactions(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("transactions", flag.ExitOnError)
	var f cli.TransactionsFlags
	fs.StringVar(&f.City, "city", "", "City name (required)")
	fs.StringVar(&f.Search, "search", "", "Address/description search term")
	fs.IntVar(&f.Limit, "limit", 50, "Maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pool, cfg, err := setup(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	openRouterClient := openrouter.NewClient(cfg.OpenRouter.APIKey)
	pricesService, err := prices.NewService(pool, cfg.Prices.BaseURL, openRouterClient)
	if err != nil {
		return fmt.Errorf("create prices service: %w", err)
	}
	return cli.RunTransactions(ctx, pricesService, f)
}

func setup(ctx context.Context) (*pgxpool.Pool, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("load config: %w", err)
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, config.Config{}, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, config.Config{}, fmt.Errorf("ping database: %w", err)
	}
	return pool, cfg, nil
}
