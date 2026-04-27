package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"koditon/internal/platform/logging"

	"github.com/google/uuid"
)

type TaskHandler func(ctx context.Context, msg Message) error

type WorkerConfig struct {
	VisibilityTimeout time.Duration
	PollInterval      time.Duration
	TaskTimeout       time.Duration
	BaseRetryDelay    time.Duration
	MaxRetryDelay     time.Duration
	MaxAttempts       int32
	Logger            *slog.Logger
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
		logger:   logging.With(logger, slog.String("worker_id", workerID), slog.String("queue", queue.Name())),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.LogAttrs(ctx, slog.LevelInfo, "worker starting", logging.Op("worker.start"))
	if err := w.queue.EnsureQueue(ctx); err != nil {
		w.logger.LogAttrs(ctx, slog.LevelError, "ensure queue failed", logging.Op("worker.start"), logging.Outcome(logging.OutcomeError), logging.Error(err))
		close(w.doneCh)
		return
	}
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.LogAttrs(ctx, slog.LevelInfo, "worker stopping", logging.Op("worker.stop"), logging.Outcome(logging.OutcomeCancelled))
			close(w.doneCh)
			return
		case <-w.stopCh:
			w.logger.LogAttrs(ctx, slog.LevelInfo, "worker stopping", logging.Op("worker.stop"), logging.Outcome(logging.OutcomeSuccess))
			close(w.doneCh)
			return
		case <-ticker.C:
			if err := w.processNext(ctx); err != nil {
				w.logger.LogAttrs(ctx, slog.LevelWarn, "worker poll failed", logging.Op("worker.poll"), logging.Outcome(logging.OutcomeError), logging.Error(err))
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
		logging.Args(
			logging.Op("task.process"),
			slog.Any("sync_job_id", msg.Data.SyncJobID),
			slog.String("entity_id", msg.Data.EntityID),
			slog.String("task_type", msg.Data.TaskType),
			slog.Int("attempt", int(msg.Data.Attempt)),
			slog.Int64("message_id", msg.MessageID),
			slog.Int("read_count", int(msg.ReadCount)),
		)...,
	)
	taskLogger.InfoContext(ctx, "task received")
	taskCtx, cancel := context.WithTimeout(ctx, w.config.TaskTimeout)
	startTime := time.Now()
	processingErr := w.executeHandler(taskCtx, taskLogger, *msg)
	duration := time.Since(startTime)
	cancel()
	taskLogger = logging.With(taskLogger, logging.DurationMS(duration))

	if processingErr != nil {
		w.handleFailure(ctx, taskLogger, *msg, processingErr)
		return processingErr
	}

	taskLogger.LogAttrs(ctx, slog.LevelInfo, "task completed", logging.Outcome(logging.OutcomeSuccess))
	if err := w.queue.Delete(ctx, msg.MessageID); err != nil {
		taskLogger.LogAttrs(ctx, slog.LevelError, "task ack failed", logging.Outcome(logging.OutcomeError), logging.Error(err))
	}
	return nil
}

func (w *Worker) executeHandler(ctx context.Context, logger *slog.Logger, msg Message) (err error) { //nolint:nonamedreturns
	defer func() {
		if r := recover(); r != nil {
			logger.LogAttrs(ctx, slog.LevelError, "task handler panicked", logging.Outcome(logging.OutcomeError), slog.Any("panic", r))
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

	logger.LogAttrs(ctx, slog.LevelWarn, "task failed",
		logging.Error(processingErr),
		slog.Int("next_attempt", int(nextAttempt)),
		slog.Int("max_attempts", int(maxAttempts)),
		slog.Bool("is_permanent", isPermanent),
		slog.Bool("will_retry", shouldRetry),
		logging.Outcome(outcomeForTaskFailure(shouldRetry)),
	)

	if !shouldRetry || msg.Data.SyncJobID == nil {
		if err := w.queue.Delete(ctx, msg.MessageID); err != nil {
			logger.LogAttrs(ctx, slog.LevelWarn, "task ack failed", logging.Outcome(logging.OutcomeError), logging.Error(err))
		}
	}

	if shouldRetry {
		retryDelay := w.calculateRetryDelay(int(nextAttempt), processingErr)
		logger.LogAttrs(ctx, slog.LevelInfo, "task retry scheduled", slog.Int64("retry_delay_ms", retryDelay.Milliseconds()), logging.Outcome(logging.OutcomeRetry))
		if msg.Data.SyncJobID == nil {
			return
		}
		if _, err := w.queue.pgmqClient.SetVisibilityTimeout(ctx, w.queue.Name(), msg.MessageID, int64(retryDelay.Seconds())); err != nil {
			logger.LogAttrs(ctx, slog.LevelError, "task visibility update failed", logging.Outcome(logging.OutcomeError), logging.Error(err))
		}
	} else {
		logger.LogAttrs(ctx, slog.LevelError, "task failed permanently", logging.Outcome(logging.OutcomeError))
	}
}

func outcomeForTaskFailure(shouldRetry bool) string {
	if shouldRetry {
		return logging.OutcomeRetry
	}
	return logging.OutcomeError
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
	p.logger.LogAttrs(ctx, slog.LevelInfo, "worker pool starting", logging.Op("worker_pool.start"), slog.Int("worker_count", len(p.workers)))
	for _, worker := range p.workers {
		go worker.Start(ctx)
	}
}

func (p *WorkerPool) Stop() {
	p.logger.LogAttrs(context.Background(), slog.LevelInfo, "worker pool stopping", logging.Op("worker_pool.stop"))
	for _, worker := range p.workers {
		worker.Stop()
	}
}

func (p *WorkerPool) Wait() {
	for _, worker := range p.workers {
		worker.Wait()
	}
	p.logger.LogAttrs(context.Background(), slog.LevelInfo, "worker pool stopped", logging.Op("worker_pool.stop"), logging.Outcome(logging.OutcomeSuccess))
}
