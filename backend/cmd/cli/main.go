package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	openrouter "github.com/revrost/go-openrouter"

	"koditon/cmd/cli/internal/cli"
	"koditon/internal/domain/ads"
	"koditon/internal/platform/config"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/prices"
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
	case "sync":
		return runSync(ctx, subArgs)
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
	fmt.Fprintln(os.Stderr, "  sync           Enqueue durable sync jobs")
	fmt.Fprintln(os.Stderr, "  api-query      Run raw Frontdoor and Shortcut provider client queries")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Run 'cli <command> --help' for command-specific flags.")
}

func runSync(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	watch := fs.Bool("watch", false, "Watch the sync job until it reaches a final status")
	interval := fs.Duration("interval", 2*time.Second, "Watch polling interval")
	providerFlag := fs.String("provider", "", "Provider for generic enqueue: frontdoor, shortcut, prices, postal")
	kindFlag := fs.String("kind", "", "Sync job kind for generic enqueue")
	entityFlag := fs.String("entity", "", "Sync job entity id for generic enqueue")
	if err := fs.Parse(args); err != nil {
		return err
	}
	flags, err := resolveSyncFlags(fs.Args(), cli.SyncFlags{
		Provider: *providerFlag,
		Kind:     *kindFlag,
		EntityID: *entityFlag,
		Watch:    *watch,
		Interval: *interval,
	})
	if err != nil {
		return err
	}
	pool, _, err := setup(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	return cli.RunSync(ctx, syncjobs.NewStore(nil, pool), flags)
}

func resolveSyncFlags(args []string, flags cli.SyncFlags) (cli.SyncFlags, error) {
	if strings.TrimSpace(flags.Provider) != "" || strings.TrimSpace(flags.Kind) != "" || strings.TrimSpace(flags.EntityID) != "" {
		if flags.Provider == "" || flags.Kind == "" || flags.EntityID == "" {
			return cli.SyncFlags{}, fmt.Errorf("--provider, --kind, and --entity must be provided together")
		}
		return flags, nil
	}
	if len(args) < 1 {
		return cli.SyncFlags{}, errors.New(syncUsage())
	}
	provider := args[0]
	action := ""
	if len(args) > 1 {
		action = args[1]
	}
	value := ""
	if len(args) > 2 {
		value = args[2]
	}
	flags.Provider = provider
	switch provider {
	case "frontdoor":
		switch action {
		case "sitemap":
			flags.Kind = "frontdoor_sitemap_sync"
			flags.EntityID = "frontdoor:sitemap"
		case "buildings":
			flags.Kind = "frontdoor_buildings_sitemap_sync"
			flags.EntityID = "frontdoor:buildings_sitemap"
		case "ad":
			if value == "" {
				return cli.SyncFlags{}, fmt.Errorf("usage: cli sync frontdoor ad <friendly-id>")
			}
			flags.Kind = "frontdoor_sync"
			flags.EntityID = "ad:" + value
		case "building":
			if value == "" {
				return cli.SyncFlags{}, fmt.Errorf("usage: cli sync frontdoor building <housing-company-id-or-uuid>")
			}
			flags.Kind = "frontdoor_sync"
			flags.EntityID = "building:" + value
		default:
			return cli.SyncFlags{}, errors.New(syncUsage())
		}
	case "shortcut":
		switch action {
		case "sitemap":
			flags.Kind = "shortcut_sitemap_sync"
			flags.EntityID = "shortcut:sitemap"
		case "buildings":
			flags.Kind = "shortcut_buildings_sitemap_sync"
			flags.EntityID = "shortcut:buildings_sitemap"
		case "ad":
			if value == "" {
				return cli.SyncFlags{}, fmt.Errorf("usage: cli sync shortcut ad <id>")
			}
			flags.Kind = "shortcut_scraper_sync"
			flags.EntityID = "ad:" + value
		case "building":
			if value == "" {
				return cli.SyncFlags{}, fmt.Errorf("usage: cli sync shortcut building <uuid>")
			}
			flags.Kind = "shortcut_scraper_sync"
			flags.EntityID = "building:" + value
		default:
			return cli.SyncFlags{}, errors.New(syncUsage())
		}
	case "prices":
		switch action {
		case "cities":
			flags.Kind = "prices_cities_init"
			flags.EntityID = "prices:cities"
		case "all":
			flags.Kind = "prices_sync_all"
			flags.EntityID = "prices:sync_all"
		case "neighborhood-postal-codes":
			flags.Kind = "prices_neighborhood_postal_code_sync"
			flags.EntityID = "prices:neighborhood_postal_codes"
		case "city":
			if value == "" {
				return cli.SyncFlags{}, fmt.Errorf("usage: cli sync prices city <name>")
			}
			flags.Kind = "prices_sync"
			flags.EntityID = "city:" + value
		default:
			return cli.SyncFlags{}, errors.New(syncUsage())
		}
	case "postal":
		if action != "all" && action != "sync" {
			return cli.SyncFlags{}, errors.New("usage: cli sync postal all")
		}
		flags.Kind = "postal_sync"
		flags.EntityID = "postal:all"
	default:
		return cli.SyncFlags{}, errors.New(syncUsage())
	}
	return flags, nil
}

func syncUsage() string {
	return `usage:
  cli sync [--watch] frontdoor sitemap
  cli sync [--watch] frontdoor buildings
  cli sync [--watch] frontdoor ad <friendly-id>
  cli sync [--watch] frontdoor building <housing-company-id-or-uuid>
  cli sync [--watch] shortcut sitemap
  cli sync [--watch] shortcut buildings
  cli sync [--watch] shortcut ad <id>
  cli sync [--watch] shortcut building <uuid>
  cli sync [--watch] prices cities
  cli sync [--watch] prices all
  cli sync [--watch] prices neighborhood-postal-codes
  cli sync [--watch] prices city <name>
  cli sync [--watch] postal all
  cli sync [--watch] --provider <provider> --kind <kind> --entity <entity-id>`
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
