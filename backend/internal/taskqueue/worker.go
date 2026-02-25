package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/db"
)

// Task types
const (
	TaskTypeFrontdoorSitemapSync             = "frontdoor_sitemap_sync"
	TaskTypeFrontdoorSync                    = "frontdoor_sync"
	TaskTypeShortcutSitemapSync              = "shortcut_sitemap_sync"
	TaskTypeShortcutScraperSync              = "shortcut_scraper_sync"
	TaskTypeShortcutAPISync                  = "shortcut_api_sync"
	TaskTypePricesCitiesInit                 = "prices_cities_init"
	TaskTypePricesSync                       = "prices_sync"
	TaskTypePricesNeighborhoodPostalCodeSync = "prices_neighborhood_postal_code_sync"
	TaskTypePricesSyncAll                    = "prices_sync_all"
	TaskTypePostalSync                       = "postal_sync"
)

const (
	EntityPrefixAd       = "ad:"
	EntityPrefixBuilding = "building:"
	EntityPrefixCity     = "city:"
)

const (
	PriorityLow      = -10
	PriorityNormal   = 0
	PriorityHigh     = 10
	PriorityCritical = 100
)

type Worker struct {
	client   *Client
	queries  *db.Queries
	workerID string
	handler  TaskHandler
	config   WorkerConfig
	logger   *slog.Logger
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
	stopped  atomic.Bool
}

type TaskHandler func(ctx context.Context, task db.TaskQueueTask) error

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

type WorkerConfig struct {
	VisibilityTimeout time.Duration
	PollInterval      time.Duration
	TaskTimeout       time.Duration
	BaseRetryDelay    time.Duration
	MaxRetryDelay     time.Duration
	Logger            *slog.Logger
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		VisibilityTimeout: 5 * time.Minute,
		PollInterval:      1 * time.Second,
		TaskTimeout:       5 * time.Minute,
		BaseRetryDelay:    30 * time.Second,
		MaxRetryDelay:     30 * time.Minute,
		Logger:            slog.Default(),
	}
}

func NewWorker(pool *pgxpool.Pool, handler TaskHandler, config WorkerConfig) *Worker {
	client := NewClient(pool)
	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		client:   client,
		queries:  db.New(pool),
		workerID: workerID,
		handler:  handler,
		config:   config,
		logger:   logger.With("worker_id", workerID),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.InfoContext(ctx, "worker starting")
	if err := w.client.EnsureQueue(ctx); err != nil {
		w.logger.ErrorContext(ctx, "failed to ensure queue exists", "error", err)
		close(w.doneCh)
		return
	}
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "context cancelled, shutting down")
			close(w.doneCh)
			return
		case <-w.stopCh:
			w.logger.InfoContext(ctx, "stop signal received, shutting down")
			close(w.doneCh)
			return
		case <-ticker.C:
			if err := w.processNextTask(ctx); err != nil {
				w.logger.WarnContext(ctx, "error processing task", "error", err)
			}
		}
	}
}

func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		w.stopped.Store(true)
		close(w.stopCh)
	})
}

func (w *Worker) Wait() {
	<-w.doneCh
}

func (w *Worker) processNextTask(ctx context.Context) (err error) {
	vtSeconds := int(w.config.VisibilityTimeout.Seconds())
	msg, err := w.client.ReadTask(ctx, vtSeconds)
	if err != nil {
		if err == ErrNoRows {
			return nil
		}
		return NewTaskError("Worker.processNextTask", err).Build()
	}
	if msg == nil {
		return nil
	}
	taskLogger := w.logger.With(
		"task_id", msg.Message.TaskID,
		"entity_id", msg.Message.EntityID,
		"attempt", msg.Message.Attempt,
		"message_id", msg.MessageID,
	)
	taskLogger.InfoContext(ctx, "received task")
	task, err := w.queries.GetTask(ctx, msg.Message.TaskID)
	if err != nil {
		taskLogger.ErrorContext(ctx, "failed to get task from database", "error", err)
		_ = w.client.ArchiveTaskFromQueue(ctx, msg.MessageID)
		return NewTaskError("Worker.GetTask", err).
			WithTaskID(msg.Message.TaskID).
			WithEntityID(msg.Message.EntityID).
			Build()
	}
	taskLogger = taskLogger.With(
		"task_type", task.TaskType,
		"max_attempts", task.MaxAttempts,
		"priority", task.Priority,
	)
	workerID := strPtr(w.workerID)
	if err := w.queries.UpdateTaskToProcessing(ctx, &db.UpdateTaskToProcessingParams{TaskID: task.TaskID, WorkerID: workerID}); err != nil {
		taskLogger.ErrorContext(ctx, "failed to update task to processing", "error", err)
		return NewTaskError("Worker.UpdateTaskToProcessing", err).
			WithTaskID(task.TaskID).
			WithEntityID(task.EntityID).
			WithTaskType(task.TaskType).
			Build()
	}
	taskCtx, cancel := context.WithTimeout(ctx, w.config.TaskTimeout)
	startTime := time.Now()
	processingErr := w.executeHandler(taskCtx, taskLogger, task)
	duration := time.Since(startTime)
	cancel()
	taskLogger = taskLogger.With("duration_ms", duration.Milliseconds())
	if processingErr != nil {
		w.handleTaskFailure(ctx, taskLogger, task, msg.MessageID, processingErr, duration)
		return processingErr
	}
	taskLogger.InfoContext(ctx, "task completed successfully")
	if err := w.queries.UpdateTaskToCompleted(ctx, task.TaskID); err != nil {
		taskLogger.ErrorContext(ctx, "failed to mark task as completed", "error", err)
		return NewTaskError("Worker.UpdateTaskToCompleted", err).
			WithTaskID(task.TaskID).
			WithEntityID(task.EntityID).
			WithTaskType(task.TaskType).
			Build()
	}
	if err := w.client.DeleteTaskFromQueue(ctx, msg.MessageID); err != nil {
		taskLogger.ErrorContext(ctx, "failed to delete message from queue", "error", err)
		return NewTaskError("Worker.DeleteTaskFromQueue", err).
			WithTaskID(task.TaskID).
			WithAttr("message_id", msg.MessageID).
			Build()
	}
	return nil
}

func (w *Worker) executeHandler(ctx context.Context, logger *slog.Logger, task db.TaskQueueTask) (err error) { //nolint:nonamedreturns
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, "task handler panicked",
				"panic", r,
				"task_id", task.TaskID,
				"entity_id", task.EntityID,
				"task_type", task.TaskType,
			)
			err = NewTaskError("Worker.executeHandler", ErrTaskPanicked).
				WithTaskID(task.TaskID).
				WithEntityID(task.EntityID).
				WithTaskType(task.TaskType).
				WithAttr("panic_value", fmt.Sprintf("%v", r)).
				Build()
		}
	}()
	return w.handler(ctx, task)
}

func (w *Worker) handleTaskFailure(ctx context.Context, logger *slog.Logger, task db.TaskQueueTask, messageID int64, processingErr error, duration time.Duration) {
	currentAttempt := int64(task.Attempt) + 1
	isPermanent := IsPermanent(processingErr)
	shouldRetry := !isPermanent && currentAttempt < int64(task.MaxAttempts) && IsRetryable(processingErr)
	logger.WarnContext(ctx, "task failed",
		"error", processingErr,
		"current_attempt", currentAttempt,
		"is_permanent", isPermanent,
		"will_retry", shouldRetry,
	)
	if shouldRetry {
		w.scheduleRetry(ctx, logger, task, currentAttempt, processingErr)
	} else {
		w.moveToDLQ(ctx, logger, task, currentAttempt, processingErr, duration)
	}
	_ = w.client.DeleteTaskFromQueue(ctx, messageID)
}

func (w *Worker) scheduleRetry(ctx context.Context, logger *slog.Logger, task db.TaskQueueTask, currentAttempt int64, processingErr error) {
	retryDelay := w.calculateRetryDelay(int(currentAttempt), processingErr)
	retryAt := time.Now().Add(retryDelay)
	logger.InfoContext(ctx, "scheduling task for retry",
		"current_attempt", currentAttempt,
		"retry_delay", retryDelay.String(),
		"retry_at", retryAt,
	)
	if err := w.queries.UpdateTaskToPendingForRetry(ctx, &db.UpdateTaskToPendingForRetryParams{TaskID: task.TaskID, ScheduledFor: retryAt}); err != nil {
		logger.ErrorContext(ctx, "failed to update task for retry", "error", err)
		return
	}
	msgID, err := w.client.EnqueueTask(ctx, task.TaskID, task.EntityID, int32(currentAttempt), retryAt)
	if err != nil {
		logger.ErrorContext(ctx, "failed to enqueue retry", "error", err)
		return
	}
	_ = w.queries.UpdateTaskQueueMessageId(ctx, &db.UpdateTaskQueueMessageIdParams{TaskID: task.TaskID, QueueMessageID: int64Ptr(msgID)})
}

func (w *Worker) moveToDLQ(ctx context.Context, logger *slog.Logger, task db.TaskQueueTask, totalAttempts int64, lastErr error, duration time.Duration) {
	logger.WarnContext(ctx, "moving task to dead letter queue",
		"total_attempts", totalAttempts,
		"reason", w.getDLQReason(task, totalAttempts, lastErr),
	)
	errorEntry := map[string]any{
		"attempt":     totalAttempts,
		"error":       lastErr.Error(),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"duration_ms": duration.Milliseconds(),
		"worker_id":   w.workerID,
	}
	if IsPermanent(lastErr) {
		var permErr *PermanentError
		if errors.As(lastErr, &permErr) && permErr.Reason != "" {
			errorEntry["permanent_reason"] = permErr.Reason
		}
	}
	errorHistoryJSON, _ := json.Marshal([]any{errorEntry})
	var firstError *string
	if task.LastError != nil {
		firstError = task.LastError
	} else {
		firstError = strPtr(lastErr.Error())
	}
	var taskMetadata []byte
	entity, err := w.queries.GetEntity(ctx, task.EntityID)
	if err == nil {
		taskMetadata = entity.Metadata
	} else {
		taskMetadata = []byte("{}")
	}
	_, dlqErr := w.queries.InsertIntoDLQ(ctx, &db.InsertIntoDLQParams{
		OriginalTaskID:    task.TaskID,
		EntityID:          task.EntityID,
		TaskType:          task.TaskType,
		Priority:          task.Priority,
		TotalAttempts:     int32(totalAttempts),
		FirstError:        firstError,
		LastError:         lastErr.Error(),
		ErrorHistory:      errorHistoryJSON,
		TaskMetadata:      taskMetadata,
		OriginalCreatedAt: task.CreatedAt,
		FirstAttemptedAt:  task.StartedAt,
		LastAttemptedAt:   time.Now(),
	})
	if dlqErr != nil {
		logger.ErrorContext(ctx, "failed to insert task into DLQ", "error", dlqErr)
	}
	// Mark original task as failed
	lastErrorText := strPtr(lastErr.Error())
	if err := w.queries.UpdateTaskToFailed(ctx, &db.UpdateTaskToFailedParams{TaskID: task.TaskID, LastError: lastErrorText}); err != nil {
		logger.ErrorContext(ctx, "failed to mark task as failed", "error", err)
	}
}

func (w *Worker) getDLQReason(task db.TaskQueueTask, totalAttempts int64, lastErr error) string {
	if IsPermanent(lastErr) {
		var permErr *PermanentError
		if errors.As(lastErr, &permErr) && permErr.Reason != "" {
			return fmt.Sprintf("permanent error: %s", permErr.Reason)
		}
		return "permanent error"
	}
	if totalAttempts >= int64(task.MaxAttempts) {
		return fmt.Sprintf("exhausted all %d retry attempts", task.MaxAttempts)
	}
	return "unknown"
}

func (w *Worker) calculateRetryDelay(attempt int, err error) time.Duration {
	// check if error suggests a specific delay
	if suggestedDelay := GetRetryDelay(err); suggestedDelay > 0 {
		delay := time.Duration(suggestedDelay) * time.Second
		if delay > w.config.MaxRetryDelay {
			return w.config.MaxRetryDelay
		}
		return delay
	}
	// exponential backoff: baseDelay * 2^(attempt-1)
	delay := w.config.BaseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	delay = min(delay, w.config.MaxRetryDelay)
	return delay
}

type WorkerPool struct {
	workers []*Worker
	config  WorkerConfig
	logger  *slog.Logger
}

func NewWorkerPool(numWorkers int, pool *pgxpool.Pool, handler TaskHandler, config WorkerConfig) *WorkerPool {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	workers := make([]*Worker, numWorkers)
	for i := range numWorkers {
		workers[i] = NewWorker(pool, handler, config)
	}
	return &WorkerPool{
		workers: workers,
		config:  config,
		logger:  logger,
	}
}

func (p *WorkerPool) Start(ctx context.Context) {
	p.logger.InfoContext(ctx, "starting worker pool", "worker_count", len(p.workers))
	for _, worker := range p.workers {
		go worker.Start(ctx)
	}
}

func (p *WorkerPool) Stop() {
	p.logger.Info("stopping worker pool")
	for _, worker := range p.workers {
		worker.Stop()
	}
}

func (p *WorkerPool) Wait() {
	for _, worker := range p.workers {
		worker.Wait()
	}
	p.logger.Info("worker pool stopped")
}
