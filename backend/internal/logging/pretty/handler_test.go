package pretty

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestHandlerOrdersPriorityAttributes(t *testing.T) {
	var out bytes.Buffer
	handler := NewHandler(&out, &Options{
		DisableColor: true,
		TimeFormat:   time.RFC3339,
	})
	logger := slog.New(handler).With("component", "server")
	logger.Info("request completed",
		"path", "/v1/products",
		"duration", 150*time.Millisecond,
		"status", 201,
		"request_id", "req-1",
		"span_id", "span-1",
		"trace_id", "trace-1",
	)
	logLine := strings.TrimSpace(out.String())
	if strings.Contains(logLine, "component=") {
		t.Fatalf("component key should be rendered in the component column, got: %s", logLine)
	}
	wantOrder := []string{
		"request_id=req-1",
		"trace_id=trace-1",
		"span_id=span-1",
		"path=/v1/products",
		"status=201",
		"duration=150ms",
	}
	assertInOrder(t, logLine, wantOrder)
}

func TestHandlerFlattensGroups(t *testing.T) {
	var out bytes.Buffer
	handler := NewHandler(&out, &Options{
		DisableColor: true,
		TimeFormat:   time.RFC3339,
	})
	logger := slog.New(handler).With("component", "storage").WithGroup("db")
	logger.Info("query completed", slog.Group("query",
		slog.String("table", "users"),
		slog.Int("rows", 3),
	))
	logLine := strings.TrimSpace(out.String())
	if !strings.Contains(logLine, "db.query.rows=3") {
		t.Fatalf("expected flattened key db.query.rows, got: %s", logLine)
	}
	if !strings.Contains(logLine, "db.query.table=users") {
		t.Fatalf("expected flattened key db.query.table, got: %s", logLine)
	}
}

func TestHandlerEscapesMultilineValues(t *testing.T) {
	var out bytes.Buffer
	handler := NewHandler(&out, &Options{
		DisableColor: true,
		TimeFormat:   time.RFC3339,
	})
	logger := slog.New(handler).With("component", "worker")
	logger.Error("task failed", "error", "line1\nline2")
	logLine := strings.TrimSpace(out.String())
	if !strings.Contains(logLine, "error=line1\\nline2") {
		t.Fatalf("expected escaped multiline value, got: %s", logLine)
	}
}

func TestHandlerMultilineModeIncludesSeparator(t *testing.T) {
	var out bytes.Buffer
	handler := NewHandler(&out, &Options{
		DisableColor:   true,
		TimeFormat:     time.RFC3339,
		Multiline:      true,
		Separator:      true,
		SeparatorWidth: 20,
	})
	logger := slog.New(handler).With("component", "worker.scheduler")
	logger.Warn("dropping unknown scheduler kind",
		"kind", "scan_missing_brand_logo_matches",
		"msg_id", 1570,
	)
	logOutput := out.String()
	if !strings.Contains(logOutput, "\n  message=\"dropping unknown scheduler kind\"\n") {
		t.Fatalf("expected message row in multiline output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "\n  kind=scan_missing_brand_logo_matches\n") {
		t.Fatalf("expected dedicated kind row in multiline output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "\n  msg_id=1570\n") {
		t.Fatalf("expected dedicated msg_id row in multiline output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "\n────────────────────\n") {
		t.Fatalf("expected separator row in multiline output, got: %s", logOutput)
	}
}

func assertInOrder(t *testing.T, line string, parts []string) {
	t.Helper()
	index := -1
	for _, part := range parts {
		next := strings.Index(line, part)
		if next < 0 {
			t.Fatalf("expected %q in line: %s", part, line)
		}
		if next <= index {
			t.Fatalf("expected %q after previous segment in line: %s", part, line)
		}
		index = next
	}
}
