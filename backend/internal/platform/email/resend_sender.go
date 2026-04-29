package email

import (
	"context"
	"fmt"
	"strings"

	resend "github.com/resend/resend-go/v3"
)

type ResendSender struct {
	client   *resend.Client
	from     string
	fromName string
}

func NewResendSender(apiKey, fromEmail, fromName string) (*ResendSender, error) {
	key := strings.TrimSpace(apiKey)
	from := strings.TrimSpace(fromEmail)
	name := strings.TrimSpace(fromName)
	if key == "" {
		return nil, fmt.Errorf("resend api key is required")
	}
	if from == "" {
		return nil, fmt.Errorf("resend from email is required")
	}
	if name == "" {
		name = "Koditon"
	}
	return &ResendSender{
		client:   resend.NewClient(key),
		from:     from,
		fromName: name,
	}, nil
}

func (s *ResendSender) Send(ctx context.Context, message Message) error {
	from := s.from
	if s.fromName != "" {
		from = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}
	params := &resend.SendEmailRequest{
		From:    from,
		To:      message.To,
		Subject: message.Subject,
		Text:    message.Text,
		Html:    message.HTML,
	}
	if _, err := s.client.Emails.SendWithContext(ctx, params); err != nil {
		return fmt.Errorf("send email via resend: %w", err)
	}
	return nil
}
