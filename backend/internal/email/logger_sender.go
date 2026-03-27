package email

import (
	"context"
	"log/slog"
	"strings"
)

type LoggerSender struct {
	logger *slog.Logger
}

func NewLoggerSender(logger *slog.Logger) *LoggerSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggerSender{logger: logger}
}

func (s *LoggerSender) Send(ctx context.Context, message Message) error {
	if s == nil || s.logger == nil {
		return ErrSenderNotConfigured
	}
	s.logger.WarnContext(
		ctx,
		"email delivery (development logger sender)",
		"to", strings.Join(message.To, ","),
		"subject", strings.TrimSpace(message.Subject),
		"text", strings.TrimSpace(message.Text),
		"html", strings.TrimSpace(message.HTML),
	)
	return nil
}
