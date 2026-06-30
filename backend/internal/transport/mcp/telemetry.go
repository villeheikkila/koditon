package mcpserver

import (
	"context"
	"fmt"
	"time"

	"koditon/internal/platform/telemetry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const mcpTracerName = "koditon/internal/transport/mcp"

func addTracedTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, traceToolHandler(tool, handler))
}

func traceToolHandler[In, Out any](tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	toolName := "unknown"
	if tool != nil && tool.Name != "" {
		toolName = tool.Name
	}
	return func(ctx context.Context, request *mcp.CallToolRequest, input In) (result *mcp.CallToolResult, output Out, err error) {
		ctx, span := telemetry.Tracer(mcpTracerName).Start(ctx, "mcp.tool "+toolName, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("mcp.operation", "tools/call"), attribute.String("mcp.tool.name", toolName)))
		startedAt := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic: %v", recovered)
				telemetry.RecordSpanError(span, err, "mcp tool panicked")
				span.SetAttributes(attribute.String("mcp.tool.status", "panic"), attribute.Int64("mcp.tool.duration_ms", time.Since(startedAt).Milliseconds()))
				span.End()
				panic(recovered)
			}
			status := "ok"
			if err != nil {
				status = "error"
				telemetry.RecordSpanError(span, err, "mcp tool failed")
			} else if result != nil && result.IsError {
				status = "tool_error"
				span.SetStatus(codes.Error, "mcp tool returned an error result")
			} else {
				span.SetStatus(codes.Ok, "")
			}
			span.SetAttributes(attribute.String("mcp.tool.status", status), attribute.Int64("mcp.tool.duration_ms", time.Since(startedAt).Milliseconds()))
			span.End()
		}()
		return handler(ctx, request, input)
	}
}
