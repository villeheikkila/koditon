package consumers

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	syncflows "koditon/internal/sync/flows"
	"koditon/internal/sync/frontdoor"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/shortcut"
)

type Consumer struct {
	logger     *slog.Logger
	syncRunner *syncflows.Runner
	queries    *db.Queries
	pool       *pgxpool.Pool

	propertiesService *properties.Service

	frontdoorWorkflowClient       *absurd.Client
	shortcutAPIWorkflowClient     *absurd.Client
	shortcutScraperWorkflowClient *absurd.Client
	canonicalWorkflowClient       *absurd.Client
	canonicalLLMWorkflowClient    *absurd.Client
	postalWorkflowClient          *absurd.Client
	pricesWorkflowClient          *absurd.Client
	frontdoorService              *frontdoor.Service
	shortcutService               *shortcut.Service
	postalService                 *postal.Service
	frontdoorWorkflowCancel       context.CancelFunc
	frontdoorWorkflowDone         chan struct{}
	shortcutAPIWorkflowCancel     context.CancelFunc
	shortcutAPIWorkflowDone       chan struct{}
	shortcutScraperWorkflowCancel context.CancelFunc
	shortcutScraperWorkflowDone   chan struct{}
	canonicalWorkflowCancel       context.CancelFunc
	canonicalWorkflowDone         chan struct{}
	canonicalLLMWorkflowCancel    context.CancelFunc
	canonicalLLMWorkflowDone      chan struct{}
	postalWorkflowCancel          context.CancelFunc
	postalWorkflowDone            chan struct{}
	pricesWorkflowCancel          context.CancelFunc
	pricesWorkflowDone            chan struct{}

	maintenanceCancel context.CancelFunc
	maintenanceDone   chan struct{}
	maintenanceOnce   sync.Once
}

type Config struct {
	WorkerCount         int
	MaintenanceEnabled  bool
	MaintenanceInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		WorkerCount:         1,
		MaintenanceEnabled:  true,
		MaintenanceInterval: time.Minute,
	}
}

func New(logger *slog.Logger, pool *pgxpool.Pool, shortcutService *shortcut.Service, frontdoorService *frontdoor.Service, postalService *postal.Service, propertiesService *properties.Service, frontdoorWorkflowClient *absurd.Client, shortcutAPIWorkflowClient *absurd.Client, shortcutScraperWorkflowClient *absurd.Client, canonicalWorkflowClient *absurd.Client, canonicalLLMWorkflowClient *absurd.Client, postalWorkflowClient *absurd.Client, pricesWorkflowClient *absurd.Client) *Consumer {
	return &Consumer{
		logger:                        logger,
		syncRunner:                    syncflows.NewRunner(logger, nil, nil, shortcutService, frontdoorService, postalService),
		queries:                       db.New(pool),
		pool:                          pool,
		propertiesService:             propertiesService,
		frontdoorWorkflowClient:       frontdoorWorkflowClient,
		shortcutAPIWorkflowClient:     shortcutAPIWorkflowClient,
		shortcutScraperWorkflowClient: shortcutScraperWorkflowClient,
		canonicalWorkflowClient:       canonicalWorkflowClient,
		canonicalLLMWorkflowClient:    canonicalLLMWorkflowClient,
		postalWorkflowClient:          postalWorkflowClient,
		pricesWorkflowClient:          pricesWorkflowClient,
		frontdoorService:              frontdoorService,
		shortcutService:               shortcutService,
		postalService:                 postalService,
	}
}

func (c *Consumer) Start(ctx context.Context, cfg Config) error {
	if err := c.startPostalWorkflowWorker(ctx, cfg); err != nil {
		return err
	}
	if err := c.startPricesWorkflowWorker(ctx, cfg); err != nil {
		return err
	}
	if err := c.startFrontdoorWorkflowWorker(ctx, cfg); err != nil {
		return err
	}
	if err := c.startShortcutWorkflowWorkers(ctx, cfg); err != nil {
		return err
	}
	if err := c.startCanonicalWorkflowWorker(ctx, cfg); err != nil {
		return err
	}
	if err := c.startCanonicalLLMWorkflowWorker(ctx, cfg); err != nil {
		return err
	}
	if cfg.MaintenanceEnabled {
		c.startMaintenance(ctx, cfg)
	}

	logging.With(c.logger, logging.Op("consumer.start")).InfoContext(ctx, "consumer started", "worker_count_per_domain", cfg.WorkerCount, "domains", 4)
	return nil
}

func (c *Consumer) Stop() {
	logging.With(c.logger, logging.Op("consumer.stop")).Info("consumer stopping")
	if c.maintenanceCancel != nil {
		c.maintenanceCancel()
	}
	if c.postalWorkflowCancel != nil {
		c.postalWorkflowCancel()
	}
	if c.pricesWorkflowCancel != nil {
		c.pricesWorkflowCancel()
	}
	if c.frontdoorWorkflowCancel != nil {
		c.frontdoorWorkflowCancel()
	}
	if c.shortcutAPIWorkflowCancel != nil {
		c.shortcutAPIWorkflowCancel()
	}
	if c.shortcutScraperWorkflowCancel != nil {
		c.shortcutScraperWorkflowCancel()
	}
	if c.canonicalWorkflowCancel != nil {
		c.canonicalWorkflowCancel()
	}
	if c.canonicalLLMWorkflowCancel != nil {
		c.canonicalLLMWorkflowCancel()
	}
	if c.postalWorkflowDone != nil {
		<-c.postalWorkflowDone
	}
	if c.pricesWorkflowDone != nil {
		<-c.pricesWorkflowDone
	}
	if c.frontdoorWorkflowDone != nil {
		<-c.frontdoorWorkflowDone
	}
	if c.shortcutAPIWorkflowDone != nil {
		<-c.shortcutAPIWorkflowDone
	}
	if c.shortcutScraperWorkflowDone != nil {
		<-c.shortcutScraperWorkflowDone
	}
	if c.canonicalWorkflowDone != nil {
		<-c.canonicalWorkflowDone
	}
	if c.canonicalLLMWorkflowDone != nil {
		<-c.canonicalLLMWorkflowDone
	}
	if c.maintenanceDone != nil {
		<-c.maintenanceDone
	}
	logging.With(c.logger, logging.Op("consumer.stop"), logging.Outcome(logging.OutcomeSuccess)).Info("consumer stopped")
}

func (c *Consumer) startMaintenance(ctx context.Context, cfg Config) {
	c.maintenanceOnce.Do(func() {
		if cfg.MaintenanceInterval <= 0 {
			cfg.MaintenanceInterval = time.Minute
		}
		maintenanceCtx, cancel := context.WithCancel(ctx)
		c.maintenanceCancel = cancel
		c.maintenanceDone = make(chan struct{})
		go c.runMaintenance(maintenanceCtx, cfg)
	})
}

func (c *Consumer) runMaintenance(ctx context.Context, cfg Config) {
	defer close(c.maintenanceDone)
	logger := logging.With(c.logger, logging.Op("consumer.sync_maintenance"))
	c.runMaintenanceOnce(ctx, logger, cfg)
	ticker := time.NewTicker(cfg.MaintenanceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runMaintenanceOnce(ctx, logger, cfg)
		}
	}
}

func (c *Consumer) runMaintenanceOnce(ctx context.Context, logger *slog.Logger, cfg Config) {
	if err := c.enqueueDirtyDimensionTargetFanout(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return
		}
		logger.WarnContext(ctx, "dirty dimension target fanout enqueue failed", "error", err, "outcome", logging.OutcomeError)
	}
}
