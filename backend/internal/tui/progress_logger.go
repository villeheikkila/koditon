package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type reportSlogHandler struct {
	report reportFn
	attrs  []slog.Attr
}

func newProgressLogger(report reportFn) *slog.Logger {
	return slog.New(&reportSlogHandler{report: report})
}

func (h *reportSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *reportSlogHandler) Handle(_ context.Context, r slog.Record) error {
	if h.report == nil {
		return nil
	}
	parts := make([]string, 0, 8)
	parts = append(parts, strings.ToUpper(r.Level.String())+": "+r.Message)
	attrs := make([]slog.Attr, 0, len(h.attrs)+int(r.NumAttrs()))
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	for _, a := range attrs {
		if a.Equal(slog.Attr{}) || a.Key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
	}
	h.report(progressUpdate{Message: strings.Join(parts, " ")})
	return nil
}

func (h *reportSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := &reportSlogHandler{report: h.report, attrs: make([]slog.Attr, 0, len(h.attrs)+len(attrs))}
	next.attrs = append(next.attrs, h.attrs...)
	next.attrs = append(next.attrs, attrs...)
	return next
}

func (h *reportSlogHandler) WithGroup(_ string) slog.Handler {
	return h
}
