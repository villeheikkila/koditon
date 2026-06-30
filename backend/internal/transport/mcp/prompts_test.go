package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPropertyPromptHandlerBuildsBuyerWorkflow(t *testing.T) {
	t.Parallel()
	result, err := propertyPromptHandler(context.Background(), &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Name: "analyze_property_for_buyer", Arguments: map[string]string{"property_id": "frontdoor:ad:123", "buyer_profile": "first home"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(result.Messages))
	}
	text := result.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(text, "koditon_get_property_detail") || !strings.Contains(text, "frontdoor:ad:123") {
		t.Fatalf("prompt text missing workflow details: %s", text)
	}
}
