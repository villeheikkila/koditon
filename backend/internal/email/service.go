package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrSenderNotConfigured = errors.New("email sender is not configured")

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type Message struct {
	To      []string
	Subject string
	Text    string
	HTML    string
}

type Service struct {
	sender Sender
}

func NewService(sender Sender) *Service {
	return &Service{sender: sender}
}

func (s *Service) Send(ctx context.Context, message Message) error {
	if s == nil || s.sender == nil {
		return ErrSenderNotConfigured
	}
	if len(message.To) == 0 {
		return errors.New("email recipients are required")
	}
	subject := strings.TrimSpace(message.Subject)
	if subject == "" {
		return errors.New("email subject is required")
	}
	if strings.TrimSpace(message.Text) == "" && strings.TrimSpace(message.HTML) == "" {
		return errors.New("email content is required")
	}
	if err := s.sender.Send(ctx, Message{
		To:      message.To,
		Subject: subject,
		Text:    message.Text,
		HTML:    message.HTML,
	}); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
