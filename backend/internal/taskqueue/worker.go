package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type TaskHandler func(ctx context.Context, msg Message) error

// StatusCallbacks allows the consumer to hook into task lifecycle events
// to update the pending_tasks table status accordingly.
type StatusCallbacks struct {
	OnProcessing func(ctx context.Context, pendingTaskID int64) error
	OnCompleted  func(ctx context.Context, pendingTaskID int64) error
	OnFailed     func(ctx context.Context, pendingTaskID int64, errMsg string) error
	OnRetry      func(ctx context.Context, pendingTaskID int64, errMsg string) error
}

type WorkerConfig struct {
	VisibilityTimeout time.Duration
	PollInterval      time.Duration
	TaskTimeout       time.Duration
	BaseRetryDelay    time.Duration
	MaxRetryDelay     time.Duration
	MaxAttempts       int32
	Logger            *slog.Logger
	Callbacks         *StatusCallbacks
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		VisibilityTimeout: 5 * time.Minute,
		PollInterval:      1 * time.Second,
		TaskTimeout:       5 * time.Minute,
		BaseRetryDelay:    30 * time.Second,
		MaxRetryDelay:     30 * time.Minute,
		MaxAttempts:       3,
		Logger:            slog.Default(),
	}
}

type Worker struct {
	queue    *Queue
	workerID string
	handler  TaskHandler
	config   WorkerConfig
	logger   *slog.Logger
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
	stopped  atomic.Bool
}

func NewWorker(queue *Queue, handler TaskHandler, config WorkerConfig) *Worker {
	workerID := fmt.Sprintf("worker-%s-%s", queue.Name(), uuid.New().String()[:8])
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		queue:    queue,
		workerID: workerID,
		handler:  handler,
		config:   config,
		logger:   logger.With("worker_id", workerID, "queue", queue.Name()),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.InfoContext(ctx, "worker starting")
	if err := w.queue.EnsureQueue(ctx); err != nil {
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
			if err := w.processNext(ctx); err != nil {
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

func (w *Worker) processNext(ctx context.Context) (err error) {
	msg, err := w.queue.Read(ctx, w.config.VisibilityTimeout)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if msg == nil {
		return nil
	}
	taskLogger := w.logger.With(
		"pending_task_id", msg.Data.PendingTaskID,
		"entity_id", msg.Data.EntityID,
		"task_type", msg.Data.TaskType,
		"attempt", msg.Data.Attempt,
		"message_id", msg.MessageID,
	)
	taskLogger.InfoContext(ctx, "received task")

	// Mark pending task as processing
	if cb := w.config.Callbacks; cb != nil && cb.OnProcessing != nil && msg.Data.PendingTaskID > 0 {
		if err := cb.OnProcessing(ctx, msg.Data.PendingTaskID); err != nil {
			taskLogger.WarnContext(ctx, "failed to mark task as processing", "error", err)
		}
	}

	taskCtx, cancel := context.WithTimeout(ctx, w.config.TaskTimeout)
	startTime := time.Now()
	processingErr := w.executeHandler(taskCtx, taskLogger, *msg)
	duration := time.Since(startTime)
	cancel()

	taskLogger = taskLogger.With("duration_ms", duration.Milliseconds())

	if processingErr != nil {
		w.handleFailure(ctx, taskLogger, *msg, processingErr)
		return processingErr
	}

	taskLogger.InfoContext(ctx, "task completed successfully")
	if err := w.queue.Delete(ctx, msg.MessageID); err != nil {
		taskLogger.ErrorContext(ctx, "failed to delete message from queue", "error", err)
	}

	// Mark pending task as completed
	if cb := w.config.Callbacks; cb != nil && cb.OnCompleted != nil && msg.Data.PendingTaskID > 0 {
		if err := cb.OnCompleted(ctx, msg.Data.PendingTaskID); err != nil {
			taskLogger.WarnContext(ctx, "failed to mark task as completed", "error", err)
		}
	}

	return nil
}

func (w *Worker) executeHandler(ctx context.Context, logger *slog.Logger, msg Message) (err error) { //nolint:nonamedreturns
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, "task handler panicked", "panic", r)
			err = NewPermanentError(fmt.Errorf("handler panicked: %v", r), "panic")
		}
	}()
	return w.handler(ctx, msg)
}

func (w *Worker) handleFailure(ctx context.Context, logger *slog.Logger, msg Message, processingErr error) {
	nextAttempt := msg.Data.Attempt + 1
	isPermanent := IsPermanent(processingErr)
	maxAttempts := w.config.MaxAttempts
	shouldRetry := !isPermanent && nextAttempt < maxAttempts && IsRetryable(processingErr)

	logger.WarnContext(ctx, "task failed",
		"error", processingErr,
		"next_attempt", nextAttempt,
		"max_attempts", maxAttempts,
		"is_permanent", isPermanent,
		"will_retry", shouldRetry,
	)

	// Always delete the current message from pgmq
	_ = w.queue.Delete(ctx, msg.MessageID)

	errMsg := processingErr.Error()
	cb := w.config.Callbacks

	if shouldRetry {
		retryDelay := w.calculateRetryDelay(int(nextAttempt), processingErr)
		logger.InfoContext(ctx, "scheduling retry", "retry_delay", retryDelay.String())

		// Reset pending task to pending so it remains the source of truth
		if cb != nil && cb.OnRetry != nil && msg.Data.PendingTaskID > 0 {
			if err := cb.OnRetry(ctx, msg.Data.PendingTaskID, errMsg); err != nil {
				logger.WarnContext(ctx, "failed to reset task to pending", "error", err)
			}
		}

		retryData := msg.Data
		retryData.Attempt = nextAttempt
		if _, err := w.queue.SendWithDelay(ctx, retryData, retryDelay); err != nil {
			logger.ErrorContext(ctx, "failed to enqueue retry", "error", err)
		}
	} else {
		logger.ErrorContext(ctx, "task permanently failed, not retrying")

		// Mark pending task as failed
		if cb != nil && cb.OnFailed != nil && msg.Data.PendingTaskID > 0 {
			if err := cb.OnFailed(ctx, msg.Data.PendingTaskID, errMsg); err != nil {
				logger.WarnContext(ctx, "failed to mark task as failed", "error", err)
			}
		}
	}
}

func (w *Worker) calculateRetryDelay(attempt int, err error) time.Duration {
	if suggestedDelay := GetRetryDelay(err); suggestedDelay > 0 {
		delay := time.Duration(suggestedDelay) * time.Second
		return min(delay, w.config.MaxRetryDelay)
	}
	delay := w.config.BaseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	return min(delay, w.config.MaxRetryDelay)
}

type WorkerPool struct {
	workers []*Worker
	logger  *slog.Logger
}

func NewWorkerPool(numWorkers int, queue *Queue, handler TaskHandler, config WorkerConfig) *WorkerPool {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	workers := make([]*Worker, numWorkers)
	for i := range numWorkers {
		workers[i] = NewWorker(queue, handler, config)
	}
	return &WorkerPool{
		workers: workers,
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
