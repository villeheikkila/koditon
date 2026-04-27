package consumers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncflows "koditon/internal/sync/flows"
	"koditon/internal/sync/frontdoor"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
)

type Consumer struct {
	logger     *slog.Logger
	syncRunner *syncflows.Runner
	syncJobs   *syncjobs.Store
	queries    *db.Queries
	pool       *pgxpool.Pool

	frontdoorPool *taskqueue.WorkerPool
	shortcutPool  *taskqueue.WorkerPool
	pricesPool    *taskqueue.WorkerPool
	postalPool    *taskqueue.WorkerPool

	maintenanceCancel context.CancelFunc
	maintenanceDone   chan struct{}
	maintenanceOnce   sync.Once
}

type Config struct {
	WorkerCount           int
	MaintenanceEnabled    bool
	MaintenanceInterval   time.Duration
	MaintenanceStaleAfter time.Duration
	MaintenanceBatchLimit int32
}

func DefaultConfig() Config {
	return Config{
		WorkerCount:           1,
		MaintenanceEnabled:    true,
		MaintenanceInterval:   time.Minute,
		MaintenanceStaleAfter: 35 * time.Minute,
		MaintenanceBatchLimit: 25,
	}
}

func New(logger *slog.Logger, pool *pgxpool.Pool, pricesService *prices.Service, shortcutService *shortcut.Service, frontdoorService *frontdoor.Service, postalService *postal.Service) *Consumer {
	return &Consumer{
		logger:     logger,
		syncRunner: syncflows.NewRunner(logger, nil, pricesService, shortcutService, frontdoorService, postalService),
		syncJobs:   syncjobs.NewStore(logger, pool),
		queries:    db.New(pool),
		pool:       pool,
	}
}

func (c *Consumer) Start(ctx context.Context, cfg Config) error {
	frontdoorQueue := taskqueue.NewQueue(c.pool, "frontdoor")
	shortcutQueue := taskqueue.NewQueue(c.pool, "shortcut")
	pricesQueue := taskqueue.NewQueue(c.pool, "prices")
	postalQueue := taskqueue.NewQueue(c.pool, "postal")
	frontdoorConfig := c.baseWorkerConfig()
	shortcutConfig := c.baseWorkerConfig()
	pricesConfig := c.baseWorkerConfig()
	postalConfig := c.baseWorkerConfig()
	c.frontdoorPool = taskqueue.NewWorkerPool(cfg.WorkerCount, frontdoorQueue, c.handleFrontdoorTask, frontdoorConfig)
	c.shortcutPool = taskqueue.NewWorkerPool(cfg.WorkerCount, shortcutQueue, c.handleShortcutTask, shortcutConfig)
	c.pricesPool = taskqueue.NewWorkerPool(cfg.WorkerCount, pricesQueue, c.handlePricesTask, pricesConfig)
	c.postalPool = taskqueue.NewWorkerPool(cfg.WorkerCount, postalQueue, c.handlePostalTask, postalConfig)

	c.frontdoorPool.Start(ctx)
	c.shortcutPool.Start(ctx)
	c.pricesPool.Start(ctx)
	c.postalPool.Start(ctx)
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
	if c.frontdoorPool != nil {
		c.frontdoorPool.Stop()
	}
	if c.shortcutPool != nil {
		c.shortcutPool.Stop()
	}
	if c.pricesPool != nil {
		c.pricesPool.Stop()
	}
	if c.postalPool != nil {
		c.postalPool.Stop()
	}
	if c.frontdoorPool != nil {
		c.frontdoorPool.Wait()
	}
	if c.shortcutPool != nil {
		c.shortcutPool.Wait()
	}
	if c.pricesPool != nil {
		c.pricesPool.Wait()
	}
	if c.postalPool != nil {
		c.postalPool.Wait()
	}
	if c.maintenanceDone != nil {
		<-c.maintenanceDone
	}
	logging.With(c.logger, logging.Op("consumer.stop"), logging.Outcome(logging.OutcomeSuccess)).Info("consumer stopped")
}

func (c *Consumer) baseWorkerConfig() taskqueue.WorkerConfig {
	cfg := taskqueue.DefaultWorkerConfig()
	cfg.Logger = c.logger
	cfg.TaskTimeout = 30 * time.Minute
	cfg.VisibilityTimeout = 35 * time.Minute
	return cfg
}

func (c *Consumer) startMaintenance(ctx context.Context, cfg Config) {
	c.maintenanceOnce.Do(func() {
		if cfg.MaintenanceInterval <= 0 {
			cfg.MaintenanceInterval = time.Minute
		}
		if cfg.MaintenanceStaleAfter <= 0 {
			cfg.MaintenanceStaleAfter = 35 * time.Minute
		}
		if cfg.MaintenanceBatchLimit <= 0 {
			cfg.MaintenanceBatchLimit = 25
		}
		maintenanceCtx, cancel := context.WithCancel(ctx)
		c.maintenanceCancel = cancel
		c.maintenanceDone = make(chan struct{})
		go c.runMaintenance(maintenanceCtx, cfg)
	})
}

func (c *Consumer) runMaintenance(ctx context.Context, cfg Config) {
	defer close(c.maintenanceDone)
	logger := logging.With(c.logger, logging.Op("consumer.sync_job_maintenance"))
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
	reaped, err := c.syncJobs.ReapStaleClaimsWithAttempts(ctx, cfg.MaintenanceStaleAfter, cfg.MaintenanceBatchLimit)
	if err != nil {
		logger.WarnContext(ctx, "sync job stale claim recovery failed", "error", err, "outcome", logging.OutcomeError)
	} else if len(reaped.RecoveredJobs) > 0 || reaped.FinalizedAttempts > 0 {
		logger.InfoContext(ctx, "sync job stale claims recovered", "jobs", len(reaped.RecoveredJobs), "attempts", reaped.FinalizedAttempts, "outcome", logging.OutcomeSuccess)
	}
	reconciled, err := c.syncJobs.ReconcilePendingJobs(ctx, cfg.MaintenanceBatchLimit)
	if err != nil {
		logger.WarnContext(ctx, "sync job pending reconciliation failed", "error", err, "outcome", logging.OutcomeError)
		return
	}
	if reconciled.Scanned > 0 || reconciled.Reenqueued > 0 {
		logger.InfoContext(ctx, "sync job pending reconciliation completed", "scanned", reconciled.Scanned, "reenqueued", reconciled.Reenqueued, "outcome", logging.OutcomeSuccess)
	}
}
