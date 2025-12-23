package taskqueue

import (
	"errors"
	"fmt"
	"log/slog"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrEntityNotFound    = errors.New("entity not found")
	ErrInvalidEntityID   = errors.New("invalid entity ID format")
	ErrInvalidTaskType   = errors.New("invalid task type")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrMaxRetriesReached = errors.New("max retries reached")
	ErrTaskCancelled     = errors.New("task cancelled")
	ErrTaskTimeout       = errors.New("task timeout")
	ErrTaskPanicked      = errors.New("task handler panicked")
	ErrQueueFull         = errors.New("queue is full")
	ErrWorkerStopped     = errors.New("worker stopped")
)

type TaskError struct {
	Op       string      // operation that failed (e.g., "Worker.processTask", "Client.EnqueueTask")
	TaskID   int64       // task ID if available
	EntityID string      // entity ID if available
	TaskType string      // task type if available
	Attempt  int         // current attempt number
	Err      error       // underlying error
	Attrs    []slog.Attr // additional structured attributes
}

func (e *TaskError) Error() string {
	if e.TaskID > 0 {
		return fmt.Sprintf("%s: task %d: %v", e.Op, e.TaskID, e.Err)
	}
	if e.EntityID != "" {
		return fmt.Sprintf("%s: entity %s: %v", e.Op, e.EntityID, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *TaskError) Unwrap() error {
	return e.Err
}

func (e *TaskError) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(e.Attrs)+5)
	attrs = append(attrs, slog.String("op", e.Op))
	if e.TaskID > 0 {
		attrs = append(attrs, slog.Int64("task_id", e.TaskID))
	}
	if e.EntityID != "" {
		attrs = append(attrs, slog.String("entity_id", e.EntityID))
	}
	if e.TaskType != "" {
		attrs = append(attrs, slog.String("task_type", e.TaskType))
	}
	if e.Attempt > 0 {
		attrs = append(attrs, slog.Int("attempt", e.Attempt))
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("cause", e.Err.Error()))
	}
	attrs = append(attrs, e.Attrs...)
	return slog.GroupValue(attrs...)
}

type TaskErrorBuilder struct {
	err *TaskError
}

func NewTaskError(op string, cause error) *TaskErrorBuilder {
	return &TaskErrorBuilder{
		err: &TaskError{
			Op:  op,
			Err: cause,
		},
	}
}

func (b *TaskErrorBuilder) WithTaskID(taskID int64) *TaskErrorBuilder {
	b.err.TaskID = taskID
	return b
}

func (b *TaskErrorBuilder) WithEntityID(entityID string) *TaskErrorBuilder {
	b.err.EntityID = entityID
	return b
}

func (b *TaskErrorBuilder) WithTaskType(taskType string) *TaskErrorBuilder {
	b.err.TaskType = taskType
	return b
}

func (b *TaskErrorBuilder) WithAttempt(attempt int) *TaskErrorBuilder {
	b.err.Attempt = attempt
	return b
}

func (b *TaskErrorBuilder) WithAttr(key string, value any) *TaskErrorBuilder {
	b.err.Attrs = append(b.err.Attrs, slog.Any(key, value))
	return b
}

func (b *TaskErrorBuilder) WithAttrs(attrs ...slog.Attr) *TaskErrorBuilder {
	b.err.Attrs = append(b.err.Attrs, attrs...)
	return b
}

func (b *TaskErrorBuilder) Build() error {
	return b.err
}

type RetryableError struct {
	Err        error
	RetryAfter int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

func NewRetryableErrorWithDelay(err error, retryAfterSeconds int) *RetryableError {
	return &RetryableError{Err: err, RetryAfter: retryAfterSeconds}
}

type PermanentError struct {
	Err    error
	Reason string
}

func (e *PermanentError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("permanent (%s): %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("permanent: %v", e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewPermanentError wraps an error as permanent (should not be retried)
func NewPermanentError(err error, reason string) *PermanentError {
	return &PermanentError{Err: err, Reason: reason}
}

func IsRetryable(err error) bool {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}
	var permanent *PermanentError
	if errors.As(err, &permanent) {
		return false
	}
	return true
}

func IsPermanent(err error) bool {
	var permanent *PermanentError
	return errors.As(err, &permanent)
}

func GetRetryDelay(err error) int {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return retryable.RetryAfter
	}
	return 0
}

func IsTaskNotFound(err error) bool {
	return errors.Is(err, ErrTaskNotFound)
}

func IsEntityNotFound(err error) bool {
	return errors.Is(err, ErrEntityNotFound)
}

func IsMaxRetriesReached(err error) bool {
	return errors.Is(err, ErrMaxRetriesReached)
}

func LogError(logger *slog.Logger, msg string, err error, additionalAttrs ...slog.Attr) {
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		attrs := make([]any, 0, len(additionalAttrs)+1)
		attrs = append(attrs, slog.Any("error", taskErr))
		for _, attr := range additionalAttrs {
			attrs = append(attrs, attr)
		}
		logger.Error(msg, attrs...)
		return
	}
	attrs := make([]any, 0, len(additionalAttrs)+1)
	attrs = append(attrs, slog.String("error", err.Error()))
	for _, attr := range additionalAttrs {
		attrs = append(attrs, attr)
	}
	logger.Error(msg, attrs...)
}

func LogWarn(logger *slog.Logger, msg string, err error, additionalAttrs ...slog.Attr) {
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		attrs := make([]any, 0, len(additionalAttrs)+1)
		attrs = append(attrs, slog.Any("error", taskErr))
		for _, attr := range additionalAttrs {
			attrs = append(attrs, attr)
		}
		logger.Warn(msg, attrs...)
		return
	}
	attrs := make([]any, 0, len(additionalAttrs)+1)
	attrs = append(attrs, slog.String("error", err.Error()))
	for _, attr := range additionalAttrs {
		attrs = append(attrs, attr)
	}
	logger.Warn(msg, attrs...)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
