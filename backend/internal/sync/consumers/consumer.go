package consumers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	"koditon/internal/sync/flows"
	"koditon/internal/sync/frontdoor"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
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
	frontdoorQueue := taskqueue.NewQueue(c.pool, "frontdoor")
	shortcutQueue := taskqueue.NewQueue(c.pool, "shortcut")
	pricesQueue := taskqueue.NewQueue(c.pool, "prices")
	postalQueue := taskqueue.NewQueue(c.pool, "postal")

	frontdoorConfig := c.baseWorkerConfig()
	frontdoorConfig.Callbacks = c.frontdoorCallbacks()

	shortcutConfig := c.baseWorkerConfig()
	shortcutConfig.Callbacks = c.shortcutCallbacks()

	pricesConfig := c.baseWorkerConfig()
	pricesConfig.Callbacks = c.pricesCallbacks()

	postalConfig := c.baseWorkerConfig()
	postalConfig.Callbacks = c.postalCallbacks()

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

func (c *Consumer) frontdoorCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdateFrontdoorSyncTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdateFrontdoorSyncTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdateFrontdoorSyncTaskToFailed(ctx, db.UpdateFrontdoorSyncTaskToFailedParams{
				FrontdoorSyncTaskID:        id,
				FrontdoorSyncTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetFrontdoorSyncTaskToPending(ctx, db.ResetFrontdoorSyncTaskToPendingParams{
				FrontdoorSyncTaskID:        id,
				FrontdoorSyncTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) shortcutCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdateShortcutSyncTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdateShortcutSyncTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdateShortcutSyncTaskToFailed(ctx, db.UpdateShortcutSyncTaskToFailedParams{
				ShortcutSyncTaskID:        id,
				ShortcutSyncTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetShortcutSyncTaskToPending(ctx, db.ResetShortcutSyncTaskToPendingParams{
				ShortcutSyncTaskID:        id,
				ShortcutSyncTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) pricesCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePricesSyncTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePricesSyncTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdatePricesSyncTaskToFailed(ctx, db.UpdatePricesSyncTaskToFailedParams{
				PricesSyncTaskID:        id,
				PricesSyncTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetPricesSyncTaskToPending(ctx, db.ResetPricesSyncTaskToPendingParams{
				PricesSyncTaskID:        id,
				PricesSyncTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) postalCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePostalSyncTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePostalSyncTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdatePostalSyncTaskToFailed(ctx, db.UpdatePostalSyncTaskToFailedParams{
				PostalSyncTaskID:        id,
				PostalSyncTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetPostalSyncTaskToPending(ctx, db.ResetPostalSyncTaskToPendingParams{
				PostalSyncTaskID:        id,
				PostalSyncTaskLastError: &errMsg,
			})
		},
	}
}
