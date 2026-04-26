package email

import (
	"context"
	"log/slog"
	"strings"

	"koditon-go/internal/logging"
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
	logging.With(s.logger, logging.Op("email.send.dev_logger")).WarnContext(
		ctx,
		"email delivery captured by development logger sender",
		"recipient_count", len(message.To),
		"to", strings.Join(message.To, ","),
		"subject", strings.TrimSpace(message.Subject),
		"has_text", strings.TrimSpace(message.Text) != "",
		"has_html", strings.TrimSpace(message.HTML) != "",
	)
	return nil
}
