package consumers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/db"
	"koditon-go/internal/frontdoor"
	"koditon-go/internal/postal"
	"koditon-go/internal/prices"
	"koditon-go/internal/shortcut"
	"koditon-go/internal/syncflows"
	"koditon-go/internal/taskqueue"
)

type Consumer struct {
	logger     *slog.Logger
	syncRunner *syncflows.Runner
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
		queries:    db.New(pool),
		pool:       pool,
	}
}

func (c *Consumer) Start(ctx context.Context, cfg Config) error {
	workerConfig := taskqueue.DefaultWorkerConfig()
	workerConfig.Logger = c.logger
	workerConfig.TaskTimeout = 30 * time.Minute
	workerConfig.VisibilityTimeout = 35 * time.Minute

	frontdoorQueue := taskqueue.NewQueue(c.pool, "frontdoor")
	shortcutQueue := taskqueue.NewQueue(c.pool, "shortcut")
	pricesQueue := taskqueue.NewQueue(c.pool, "prices")
	postalQueue := taskqueue.NewQueue(c.pool, "postal")

	c.frontdoorPool = taskqueue.NewWorkerPool(cfg.WorkerCount, frontdoorQueue, c.handleFrontdoorTask, workerConfig)
	c.shortcutPool = taskqueue.NewWorkerPool(cfg.WorkerCount, shortcutQueue, c.handleShortcutTask, workerConfig)
	c.pricesPool = taskqueue.NewWorkerPool(cfg.WorkerCount, pricesQueue, c.handlePricesTask, workerConfig)
	c.postalPool = taskqueue.NewWorkerPool(cfg.WorkerCount, postalQueue, c.handlePostalTask, workerConfig)

	c.frontdoorPool.Start(ctx)
	c.shortcutPool.Start(ctx)
	c.pricesPool.Start(ctx)
	c.postalPool.Start(ctx)

	c.logger.InfoContext(ctx, "consumer started", "worker_count_per_domain", cfg.WorkerCount, "domains", 4)
	return nil
}

func (c *Consumer) Stop() {
	c.logger.Info("stopping consumer worker pools")
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
	c.logger.Info("consumer stopped")
}

func (c *Consumer) deletePendingTask(ctx context.Context, logger *slog.Logger, msg taskqueue.Message, deleteFunc func(context.Context, int64) error) {
	if msg.Data.PendingTaskID > 0 {
		if err := deleteFunc(ctx, msg.Data.PendingTaskID); err != nil {
			logger.WarnContext(ctx, "failed to delete pending task", "error", err, "pending_task_id", msg.Data.PendingTaskID)
		}
	}
}
