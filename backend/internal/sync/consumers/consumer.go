package consumers

import (
	"context"
	"log/slog"
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
}

type Config struct {
	WorkerCount int
}

func DefaultConfig() Config {
	return Config{WorkerCount: 1}
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

	logging.With(c.logger, logging.Op("consumer.start")).InfoContext(ctx, "consumer started", "worker_count_per_domain", cfg.WorkerCount, "domains", 4)
	return nil
}

func (c *Consumer) Stop() {
	logging.With(c.logger, logging.Op("consumer.stop")).Info("consumer stopping")
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
	logging.With(c.logger, logging.Op("consumer.stop"), logging.Outcome(logging.OutcomeSuccess)).Info("consumer stopped")
}

func (c *Consumer) baseWorkerConfig() taskqueue.WorkerConfig {
	cfg := taskqueue.DefaultWorkerConfig()
	cfg.Logger = c.logger
	cfg.TaskTimeout = 30 * time.Minute
	cfg.VisibilityTimeout = 35 * time.Minute
	return cfg
}
