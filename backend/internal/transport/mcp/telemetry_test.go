package mcpserver

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceToolHandlerRecordsToolSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})
	handler := traceToolHandler(&mcp.Tool{Name: "koditon_test_tool"}, func(context.Context, *mcp.CallToolRequest, emptyInput) (*mcp.CallToolResult, struct{}, error) {
		return &mcp.CallToolResult{}, struct{}{}, nil
	})
	if _, _, err := handler(context.Background(), &mcp.CallToolRequest{}, emptyInput{}); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "mcp.tool koditon_test_tool" {
		t.Fatalf("span name = %q, want mcp.tool koditon_test_tool", spans[0].Name)
	}
	if spans[0].Status.Code != codes.Ok {
		t.Fatalf("span status = %v, want Ok", spans[0].Status.Code)
	}
}

func TestTraceToolHandlerMarksToolError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})
	handler := traceToolHandler(&mcp.Tool{Name: "koditon_error_tool"}, func(context.Context, *mcp.CallToolRequest, emptyInput) (*mcp.CallToolResult, struct{}, error) {
		result := &mcp.CallToolResult{}
		result.SetError(errors.New("bad input"))
		return result, struct{}{}, nil
	})
	if _, _, err := handler(context.Background(), &mcp.CallToolRequest{}, emptyInput{}); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Fatalf("span status = %v, want Error", spans[0].Status.Code)
	}
}
