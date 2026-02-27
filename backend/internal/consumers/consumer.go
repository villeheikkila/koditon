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
			return c.queries.UpdateFrontdoorPendingTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdateFrontdoorPendingTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdateFrontdoorPendingTaskToFailed(ctx, db.UpdateFrontdoorPendingTaskToFailedParams{
				FrontdoorPendingTaskID:        id,
				FrontdoorPendingTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetFrontdoorPendingTaskToPending(ctx, db.ResetFrontdoorPendingTaskToPendingParams{
				FrontdoorPendingTaskID:        id,
				FrontdoorPendingTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) shortcutCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdateShortcutPendingTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdateShortcutPendingTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdateShortcutPendingTaskToFailed(ctx, db.UpdateShortcutPendingTaskToFailedParams{
				ShortcutPendingTaskID:        id,
				ShortcutPendingTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetShortcutPendingTaskToPending(ctx, db.ResetShortcutPendingTaskToPendingParams{
				ShortcutPendingTaskID:        id,
				ShortcutPendingTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) pricesCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePricesPendingTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePricesPendingTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdatePricesPendingTaskToFailed(ctx, db.UpdatePricesPendingTaskToFailedParams{
				PricesPendingTaskID:        id,
				PricesPendingTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetPricesPendingTaskToPending(ctx, db.ResetPricesPendingTaskToPendingParams{
				PricesPendingTaskID:        id,
				PricesPendingTaskLastError: &errMsg,
			})
		},
	}
}

func (c *Consumer) postalCallbacks() *taskqueue.StatusCallbacks {
	return &taskqueue.StatusCallbacks{
		OnProcessing: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePostalPendingTaskToProcessing(ctx, id)
		},
		OnCompleted: func(ctx context.Context, id int64) error {
			return c.queries.UpdatePostalPendingTaskToCompleted(ctx, id)
		},
		OnFailed: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.UpdatePostalPendingTaskToFailed(ctx, db.UpdatePostalPendingTaskToFailedParams{
				PostalPendingTaskID:        id,
				PostalPendingTaskLastError: &errMsg,
			})
		},
		OnRetry: func(ctx context.Context, id int64, errMsg string) error {
			return c.queries.ResetPostalPendingTaskToPending(ctx, db.ResetPostalPendingTaskToPendingParams{
				PostalPendingTaskID:        id,
				PostalPendingTaskLastError: &errMsg,
			})
		},
	}
}
