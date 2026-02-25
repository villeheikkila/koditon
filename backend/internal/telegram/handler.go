package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	botToken string
	chatID   string
	minLevel slog.Level
	next     slog.Handler
	client   *http.Client
}

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func NewHandler(botToken, chatID string, minLevel slog.Level, next slog.Handler) *Handler {
	return &Handler{
		botToken: botToken,
		chatID:   chatID,
		minLevel: minLevel,
		next:     next,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.next.Handle(ctx, r); err != nil {
		return err
	}
	if r.Level >= h.minLevel {
		go h.sendToTelegram(r)
	}
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		botToken: h.botToken,
		chatID:   h.chatID,
		minLevel: h.minLevel,
		next:     h.next.WithAttrs(attrs),
		client:   h.client,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		botToken: h.botToken,
		chatID:   h.chatID,
		minLevel: h.minLevel,
		next:     h.next.WithGroup(name),
		client:   h.client,
	}
}

func (h *Handler) sendToTelegram(r slog.Record) {
	var buf strings.Builder
	levelEmoji := getLevelEmoji(r.Level)
	fmt.Fprintf(&buf, "%s *%s*\n\n", levelEmoji, r.Level.String())
	fmt.Fprintf(&buf, "`%s`\n", escapeMarkdown(r.Message))
	if r.NumAttrs() > 0 {
		buf.WriteString("\n*Attributes:*\n")
		r.Attrs(func(a slog.Attr) bool {
			fmt.Fprintf(&buf, "• `%s`: %s\n", a.Key, escapeMarkdown(a.Value.String()))
			return true
		})
	}
	msg := telegramMessage{
		ChatID:    h.chatID,
		Text:      buf.String(),
		ParseMode: "Markdown",
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.botToken)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
}

func getLevelEmoji(level slog.Level) string {
	switch level {
	case slog.LevelWarn:
		return "⚠️"
	case slog.LevelError:
		return "🚨"
	default:
		return "ℹ️"
	}
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}
