package logging

import (
	"log/slog"
	"time"
)

const (
	OutcomeSuccess   = "success"
	OutcomeError     = "error"
	OutcomeRetry     = "retry"
	OutcomeCancelled = "cancelled"
	OutcomeRejected  = "rejected"
)

func Args(attrs ...slog.Attr) []any {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return args
}

func With(logger *slog.Logger, attrs ...slog.Attr) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(Args(attrs...)...)
}

func Op(name string) slog.Attr {
	return slog.String("op", name)
}

func Outcome(value string) slog.Attr {
	return slog.String("outcome", value)
}

func Error(err error) slog.Attr {
	return slog.Any("error", err)
}

func DurationMS(duration time.Duration) slog.Attr {
	return slog.Int64("duration_ms", duration.Milliseconds())
}
