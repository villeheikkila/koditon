package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	openrouter "github.com/revrost/go-openrouter"
	"github.com/spf13/cobra"

	"koditon/cmd/cli/internal/cli"
	"koditon/internal/db"
	"koditon/internal/domain/ads"
	"koditon/internal/platform/schema"
	"koditon/internal/sync/consumers"
	"koditon/internal/sync/frontdoor"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
)

type commandOptions struct {
	ctx     context.Context
	stdout  io.Writer
	stderr  io.Writer
	getenv  func(string) string
	json    bool
	noColor bool
}

func newRootCommand(ctx context.Context, stdout, stderr io.Writer, getenv func(string) string) *cobra.Command {
	opts := &commandOptions{ctx: ctx, stdout: stdout, stderr: stderr, getenv: getenv}
	cmd := &cobra.Command{
		Use:           "cli",
		Short:         "Operate the koditon backend",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			cli.SetColorEnabled(!opts.noColor)
		},
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().BoolVar(&opts.json, "json", false, "Emit machine-readable JSON when supported")
	cmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "Disable styled terminal output")
	cmd.AddCommand(newSearchCommand(opts))
	cmd.AddCommand(newDetailCommand(opts))
	cmd.AddCommand(newTransactionsCommand(opts))
	cmd.AddCommand(newPricesCommand(opts))
	cmd.AddCommand(newSyncCommand(opts))
	cmd.AddCommand(newAPIQueryCommand(opts))
	return cmd
}

func newSearchCommand(opts *commandOptions) *cobra.Command {
	var f cli.SearchFlags
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search ads, buildings, and announcements",
		RunE: func(_ *cobra.Command, _ []string) error {
			pool, _, err := setup(opts.ctx)
			if err != nil {
				return err
			}
			defer pool.Close()
			f.Out = opts.stdout
			return cli.RunSearch(opts.ctx, ads.NewService(pool), f)
		},
	}
	cmd.Flags().StringVar(&f.Query, "query", "", "Free text search")
	cmd.Flags().StringVar(&f.Source, "source", "all", "Source filter: shortcut, frontdoor, all")
	cmd.Flags().StringVar(&f.Kind, "kind", "all", "Kind filter: ad, building, announcement, all")
	cmd.Flags().StringVar(&f.ListingType, "type", "all", "Listing type: listing, rental, all")
	cmd.Flags().StringVar(&f.City, "city", "", "City filter")
	cmd.Flags().StringVar(&f.Postal, "postal", "", "Postal code filter")
	cmd.Flags().IntVar(&f.MinPrice, "min-price", 0, "Minimum price")
	cmd.Flags().IntVar(&f.MaxPrice, "max-price", 0, "Maximum price")
	cmd.Flags().Float64Var(&f.MinArea, "min-area", 0, "Minimum area (m²)")
	cmd.Flags().Float64Var(&f.MaxArea, "max-area", 0, "Maximum area (m²)")
	cmd.Flags().StringVar(&f.Sort, "sort", "seen_desc", "Sort: price_asc, price_desc, area_asc, area_desc, seen_desc")
	cmd.Flags().IntVar(&f.Limit, "limit", 25, "Results per page (25, 50, 100)")
	cmd.Flags().IntVar(&f.Page, "page", 1, "Page number")
	cmd.Flags().StringVar(&f.PublishedAfter, "after", "", "Published after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.PublishedBefore, "before", "", "Published before date (YYYY-MM-DD)")
	return cmd
}

func newDetailCommand(opts *commandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "detail <canonical-id-or-url>",
		Short: "Show entity detail by canonical ID or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			pool, cfg, err := setup(opts.ctx)
			if err != nil {
				return err
			}
			defer pool.Close()
			return cli.RunDetail(opts.ctx, ads.NewService(pool), args[0], cfg.Shortcut.SitemapBase, cfg.Frontdoor.SitemapBase, cfg.WebBaseURL, opts.stdout)
		},
	}
}

func newTransactionsCommand(opts *commandOptions) *cobra.Command {
	var f cli.TransactionsFlags
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "Search price transactions",
		RunE: func(_ *cobra.Command, _ []string) error {
			pool, cfg, err := setup(opts.ctx)
			if err != nil {
				return err
			}
			defer pool.Close()
			openRouterClient := openrouter.NewClient(cfg.OpenRouter.APIKey)
			pricesService, err := prices.NewService(pool, cfg.Prices.BaseURL, openRouterClient)
			if err != nil {
				return fmt.Errorf("create prices service: %w", err)
			}
			f.Out = opts.stdout
			return cli.RunTransactions(opts.ctx, pricesService, f)
		},
	}
	cmd.Flags().StringVar(&f.City, "city", "", "City name (required)")
	cmd.Flags().StringVar(&f.Search, "search", "", "Address/description search term")
	cmd.Flags().IntVar(&f.Limit, "limit", 50, "Maximum results")
	return cmd
}

func newPricesCommand(opts *commandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices",
		Short: "Operate prices data and listing matches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	match := &cobra.Command{
		Use:   "match",
		Short: "Generate and apply prices matching candidates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	var f cli.PricesMatchSaleListingsFlags
	saleListings := &cobra.Command{
		Use:   "sale-listings",
		Short: "Match prices transactions to sale listings",
		RunE: func(_ *cobra.Command, _ []string) error {
			pool, _, err := setup(opts.ctx)
			if err != nil {
				return err
			}
			defer pool.Close()
			f.JSON = opts.json
			f.Out = opts.stdout
			return cli.RunPricesMatchSaleListings(opts.ctx, pool, f)
		},
	}
	saleListings.Flags().BoolVar(&f.AutoLinkSafe, "auto-link-safe", false, "Apply only unique high-confidence matches")
	saleListings.Flags().IntVar(&f.ScoreThreshold, "threshold", 90, "Minimum score for high-confidence matches")
	saleListings.Flags().IntVar(&f.CompetitorMargin, "margin", 15, "Minimum score gap to competing candidates")
	match.AddCommand(saleListings)
	cmd.AddCommand(match)
	return cmd
}

func newAPIQueryCommand(opts *commandOptions) *cobra.Command {
	return &cobra.Command{
		Use:                "api-query <frontdoor|shortcut> <query> [flags]",
		Short:              "Run raw Frontdoor and Shortcut provider client queries",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
				return cmd.Help()
			}
			return runAPIQuery(opts.ctx, args, opts.stdout, opts.getenv)
		},
	}
}

func newSyncCommand(opts *commandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Enqueue and inspect durable sync jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newSyncEnqueueCommand(opts))
	cmd.AddCommand(newSyncStatusCommand(opts))
	cmd.AddCommand(newSyncListCommand(opts))
	cmd.AddCommand(newSyncMaintenanceCommand(opts))
	cmd.AddCommand(newSyncRunCommand(opts))
	return cmd
}

func newSyncEnqueueCommand(opts *commandOptions) *cobra.Command {
	var watch bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "enqueue <provider> <kind> <entity-id>",
		Short: "Enqueue a durable sync job",
		Example: strings.Join([]string{
			"cli sync enqueue frontdoor frontdoor_sitemap_sync frontdoor:sitemap --json --watch",
			"cli sync enqueue frontdoor frontdoor_sync ad:12345 --json --watch",
			"cli sync enqueue shortcut shortcut_buildings_sitemap_sync shortcut:buildings_sitemap --json",
			"cli sync enqueue prices prices_sync city:Helsinki --json --watch",
			"cli sync enqueue postal postal_sync postal:all --json",
		}, "\n"),
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := validateSyncJobTarget(args[0], args[1]); err != nil {
				return err
			}
			store, closeFn, err := setupSyncStore(opts.ctx)
			if err != nil {
				return err
			}
			defer closeFn()
			return cli.RunSync(opts.ctx, store, cli.SyncFlags{
				Provider: args[0],
				Kind:     args[1],
				EntityID: args[2],
				Watch:    watch,
				Interval: interval,
				JSON:     opts.json,
				Out:      opts.stdout,
			})
		},
	}
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch the sync job until it reaches a final status")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Watch polling interval")
	return cmd
}

func newSyncStatusCommand(opts *commandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status <job-id>",
		Short: "Show sync job status and attempts",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			jobID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("parse job id: %w", err)
			}
			store, closeFn, err := setupSyncStore(opts.ctx)
			if err != nil {
				return err
			}
			defer closeFn()
			return cli.RunSyncStatus(opts.ctx, store, jobID, opts.json, opts.stdout)
		},
	}
}

func newSyncListCommand(opts *commandOptions) *cobra.Command {
	var f cli.SyncListFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sync jobs",
		RunE: func(_ *cobra.Command, _ []string) error {
			store, closeFn, err := setupSyncStore(opts.ctx)
			if err != nil {
				return err
			}
			defer closeFn()
			f.JSON = opts.json
			f.Out = opts.stdout
			return cli.RunSyncList(opts.ctx, store, f)
		},
	}
	cmd.Flags().StringVar(&f.Status, "status", "", "Filter by job status")
	cmd.Flags().StringVar(&f.Provider, "provider", "", "Filter by provider")
	cmd.Flags().StringVar(&f.Kind, "kind", "", "Filter by job kind")
	cmd.Flags().IntVar(&f.Limit, "limit", 25, "Maximum jobs to return")
	return cmd
}

func newSyncMaintenanceCommand(opts *commandOptions) *cobra.Command {
	var f cli.SyncMaintenanceFlags
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "Recover stale sync job claims and reconcile pending jobs",
		RunE: func(_ *cobra.Command, _ []string) error {
			store, closeFn, err := setupSyncStore(opts.ctx)
			if err != nil {
				return err
			}
			defer closeFn()
			f.JSON = opts.json
			f.Out = opts.stdout
			return cli.RunSyncMaintenance(opts.ctx, store, f)
		},
	}
	cmd.Flags().DurationVar(&f.StaleAfter, "stale-after", 35*time.Minute, "Recover in-progress jobs older than this")
	cmd.Flags().IntVar(&f.Limit, "limit", 25, "Maximum jobs to process")
	return cmd
}

func newSyncRunCommand(opts *commandOptions) *cobra.Command {
	cfg := consumers.DefaultConfig()
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sync queue consumers until interrupted",
		RunE: func(_ *cobra.Command, _ []string) error {
			if cfg.WorkerCount <= 0 {
				return fmt.Errorf("--workers must be greater than zero")
			}
			if cfg.MaintenanceBatchLimit <= 0 {
				return fmt.Errorf("--maintenance-limit must be greater than zero")
			}
			return runSyncConsumers(opts, cfg)
		},
	}
	cmd.Flags().IntVar(&cfg.WorkerCount, "workers", cfg.WorkerCount, "Workers per sync queue")
	cmd.Flags().BoolVar(&cfg.MaintenanceEnabled, "maintenance", cfg.MaintenanceEnabled, "Run periodic sync job maintenance")
	cmd.Flags().DurationVar(&cfg.MaintenanceInterval, "maintenance-interval", cfg.MaintenanceInterval, "Periodic maintenance interval")
	cmd.Flags().DurationVar(&cfg.MaintenanceStaleAfter, "stale-after", cfg.MaintenanceStaleAfter, "Recover in-progress jobs older than this")
	cmd.Flags().Int32Var(&cfg.MaintenanceBatchLimit, "maintenance-limit", cfg.MaintenanceBatchLimit, "Maximum jobs to process per maintenance pass")
	return cmd
}

func runSyncConsumers(opts *commandOptions, consumerConfig consumers.Config) error {
	ctx, stop := signal.NotifyContext(opts.ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, cfg, err := setup(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := schema.Check(ctx, db.New(pool)); err != nil {
		return fmt.Errorf("check schema: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(opts.stderr, &slog.HandlerOptions{}))
	openRouterClient := openrouter.NewClient(cfg.OpenRouter.APIKey)
	pricesService, err := prices.NewService(pool, cfg.Prices.BaseURL, openRouterClient)
	if err != nil {
		return fmt.Errorf("create prices service: %w", err)
	}
	shortcutService := shortcut.NewService(pool, logger, cfg.Shortcut.BaseURL, cfg.Shortcut.DocsBaseURL, cfg.Shortcut.AdBaseURL, cfg.Shortcut.UserAgent, cfg.Shortcut.SitemapBase)
	frontdoorService := frontdoor.NewService(pool, logger, cfg.Frontdoor.BaseURL, cfg.Frontdoor.UserAgent, cfg.Frontdoor.Cookie, cfg.Frontdoor.SitemapBase)
	postalService := postal.NewService(pool)
	consumer := consumers.New(logger, pool, pricesService, shortcutService, frontdoorService, postalService)
	if err := consumer.Start(ctx, consumerConfig); err != nil {
		return fmt.Errorf("start sync consumers: %w", err)
	}
	if opts.json {
		if err := cli.WriteJSON(opts.stdout, map[string]any{"event": "sync_consumers_started", "workers_per_queue": consumerConfig.WorkerCount, "maintenance": consumerConfig.MaintenanceEnabled}); err != nil {
			consumer.Stop()
			return err
		}
	} else {
		fmt.Fprintf(opts.stdout, "sync consumers running workers_per_queue=%d maintenance=%t\n", consumerConfig.WorkerCount, consumerConfig.MaintenanceEnabled)
	}
	<-ctx.Done()
	consumer.Stop()
	if opts.json {
		return cli.WriteJSON(opts.stdout, map[string]any{"event": "sync_consumers_stopped"})
	}
	fmt.Fprintln(opts.stdout, "sync consumers stopped")
	return nil
}

func setupSyncStore(ctx context.Context) (*syncjobs.Store, func(), error) {
	pool, _, err := setup(ctx)
	if err != nil {
		return nil, nil, err
	}
	return syncjobs.NewStore(nil, pool), pool.Close, nil
}

func validateSyncJobTarget(provider, kind string) error {
	validKinds := map[string]map[string]struct{}{
		"frontdoor": {
			"frontdoor_sitemap_sync":           {},
			"frontdoor_buildings_sitemap_sync": {},
			"frontdoor_sync":                   {},
			"frontdoor_ad_data_hash_backfill":  {},
		},
		"shortcut": {
			"shortcut_sitemap_sync":           {},
			"shortcut_buildings_sitemap_sync": {},
			"shortcut_scraper_sync":           {},
			"shortcut_api_sync":               {},
			"shortcut_ad_data_hash_backfill":  {},
		},
		"prices": {
			"prices_cities_init":                   {},
			"prices_sync":                          {},
			"prices_postal_code_sync":              {},
			"prices_postal_code_page_sync":         {},
			"prices_neighborhood_postal_code_sync": {},
			"prices_sync_all":                      {},
			"prices_match_sale_listings_backfill":  {},
			"prices_match_sale_listings_fanout":    {},
			"prices_match_sale_listing":            {},
		},
		"postal": {
			"postal_sync": {},
		},
	}
	kinds, ok := validKinds[strings.TrimSpace(provider)]
	if !ok {
		return fmt.Errorf("unknown sync provider %q", provider)
	}
	if _, ok := kinds[strings.TrimSpace(kind)]; !ok {
		return fmt.Errorf("sync kind %q is not valid for provider %q", kind, provider)
	}
	return nil
}
