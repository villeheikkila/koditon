package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/frontdoor"
	"koditon-go/internal/prices"
	"koditon-go/internal/shortcut"
	"koditon-go/internal/taskqueue"
	taskqueuedb "koditon-go/internal/taskqueue/db"
)

type Consumer struct {
	logger           *slog.Logger
	taskQueueClient  *taskqueue.Client
	pricesService    *prices.Service
	shortcutService  *shortcut.Service
	frontdoorService *frontdoor.Service
	workerPool       *taskqueue.WorkerPool
}

type Config struct {
	WorkerCount int
}

func DefaultConfig() Config {
	return Config{
		WorkerCount: 1,
	}
}

func New(
	logger *slog.Logger,
	taskQueueClient *taskqueue.Client,
	pricesService *prices.Service,
	shortcutService *shortcut.Service,
	frontdoorService *frontdoor.Service,
) *Consumer {
	return &Consumer{
		logger:           logger,
		taskQueueClient:  taskQueueClient,
		pricesService:    pricesService,
		shortcutService:  shortcutService,
		frontdoorService: frontdoorService,
	}
}

func (c *Consumer) Start(ctx context.Context, cfg Config, pool *pgxpool.Pool) error {
	if err := c.taskQueueClient.EnsureQueue(ctx); err != nil {
		return fmt.Errorf("ensure task queue: %w", err)
	}
	workerConfig := taskqueue.DefaultWorkerConfig()
	workerConfig.Logger = c.logger
	c.workerPool = taskqueue.NewWorkerPool(
		cfg.WorkerCount,
		pool,
		c.handleTask,
		workerConfig,
	)
	c.workerPool.Start(ctx)
	c.logger.InfoContext(ctx, "consumer started", "worker_count", cfg.WorkerCount)
	return nil
}

func (c *Consumer) Stop() {
	if c.workerPool != nil {
		c.logger.Info("stopping consumer worker pool")
		c.workerPool.Stop()
		c.workerPool.Wait()
		c.logger.Info("consumer stopped")
	}
}

func (c *Consumer) handleTask(taskCtx context.Context, task taskqueuedb.TaskQueueTask) error {
	taskLogger := c.logger.With(
		"task_id", task.TaskID,
		"task_type", task.TaskType,
		"entity_id", task.EntityID,
		"attempt", task.Attempt,
		"priority", task.Priority,
	)
	var err error
	switch task.TaskType {
	case taskqueue.TaskTypeFrontdoorSitemapSync:
		err = c.handleFrontdoorSitemapSync(taskCtx, taskLogger)
	case taskqueue.TaskTypeFrontdoorSync:
		err = c.handleFrontdoorSync(taskCtx, taskLogger, task)
	case taskqueue.TaskTypeShortcutSitemapSync:
		err = c.handleShortcutSitemapSync(taskCtx, taskLogger)
	case taskqueue.TaskTypeShortcutScraperSync:
		err = c.handleShortcutScraperSync(taskCtx, taskLogger, task)
	case taskqueue.TaskTypeShortcutAPISync:
		err = c.handleShortcutAPISync(taskCtx, taskLogger, task)
	case taskqueue.TaskTypePricesCitiesInit:
		err = c.handlePricesCitiesInit(taskCtx, taskLogger)
	case taskqueue.TaskTypePricesSync:
		err = c.handlePricesSync(taskCtx, taskLogger, task)
	default:
		return taskqueue.NewPermanentError(
			fmt.Errorf("unknown task type: %s", task.TaskType),
			"unrecognized task type",
		)
	}
	if err != nil {
		return classifyError(err, task)
	}
	return nil
}
