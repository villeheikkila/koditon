package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPropertyPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "analyze_property_for_buyer",
		Title:       "Analyze Property For Buyer",
		Description: "Guide a buyer-focused due-diligence analysis using Koditon property tools and resources.",
		Arguments: []*mcp.PromptArgument{
			{Name: "property_id", Title: "Property ID", Description: "Koditon canonical property ID or source URL.", Required: true},
			{Name: "buyer_profile", Title: "Buyer Profile", Description: "Optional buyer goals, budget, constraints, or risk tolerance."},
		},
	}, propertyPromptHandler)
	server.AddPrompt(&mcp.Prompt{
		Name:        "compare_property_shortlist",
		Title:       "Compare Property Shortlist",
		Description: "Compare a shortlist with ranking, tradeoffs, market evidence, and missing-data warnings.",
		Arguments: []*mcp.PromptArgument{
			{Name: "property_ids", Title: "Property IDs", Description: "Comma-separated Koditon canonical IDs.", Required: true},
			{Name: "buyer_profile", Title: "Buyer Profile", Description: "Optional buyer goals, budget, constraints, or risk tolerance."},
		},
	}, propertyPromptHandler)
	server.AddPrompt(&mcp.Prompt{
		Name:        "prepare_viewing_questions",
		Title:       "Prepare Viewing Questions",
		Description: "Generate practical viewing questions from missing fields, market context, and property evidence.",
		Arguments: []*mcp.PromptArgument{
			{Name: "property_id", Title: "Property ID", Description: "Koditon canonical property ID or source URL.", Required: true},
		},
	}, propertyPromptHandler)
}

func propertyPromptHandler(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	switch req.Params.Name {
	case "analyze_property_for_buyer":
		propertyID := strings.TrimSpace(args["property_id"])
		profile := strings.TrimSpace(args["buyer_profile"])
		return promptResult("Buyer due-diligence workflow", fmt.Sprintf("Analyze property %s for a buyer. First call koditon_get_property_detail, then call koditon_get_property_market_context, then summarize facts, costs, lifecycle, market evidence, missing data, red flags, and follow-up questions. Buyer profile: %s", propertyID, firstNonEmpty(profile, "not provided"))), nil
	case "compare_property_shortlist":
		ids := strings.TrimSpace(args["property_ids"])
		profile := strings.TrimSpace(args["buyer_profile"])
		return promptResult("Property shortlist comparison workflow", fmt.Sprintf("Compare these Koditon properties: %s. Use koditon_compare_properties with the buyer profile, then inspect any missing detail or market context before recommending a ranked shortlist. Buyer profile: %s", ids, firstNonEmpty(profile, "not provided"))), nil
	case "prepare_viewing_questions":
		propertyID := strings.TrimSpace(args["property_id"])
		return promptResult("Viewing questions workflow", fmt.Sprintf("Prepare viewing questions for property %s. Use koditon_get_property_detail and focus on missing data, renovations, housing-company risks, monthly costs, debt share, source conflicts, and market evidence gaps.", propertyID)), nil
	default:
		return nil, fmt.Errorf("unknown property prompt: %s", req.Params.Name)
	}
}

func promptResult(description, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{Description: description, Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}}}
}
