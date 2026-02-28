package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"koditon-go/internal/ads"
	"koditon-go/internal/db"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleSearchListings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	params := ads.SearchParams{
		Query:       stringArg(args, "query"),
		Source:      stringArg(args, "source"),
		Kind:        stringArg(args, "kind"),
		ListingType: stringArg(args, "listing_type"),
		City:        stringArg(args, "city"),
		Postal:      stringArg(args, "postal"),
		Sort:        stringArg(args, "sort"),
	}

	if v, ok := numberArg(args, "min_price"); ok {
		iv := int64(v)
		params.MinPrice = &iv
	}
	if v, ok := numberArg(args, "max_price"); ok {
		iv := int64(v)
		params.MaxPrice = &iv
	}
	if v, ok := numberArg(args, "min_area"); ok {
		params.MinArea = &v
	}
	if v, ok := numberArg(args, "max_area"); ok {
		params.MaxArea = &v
	}
	if v, ok := numberArg(args, "page"); ok {
		params.Page = int32(v)
	}
	if v, ok := numberArg(args, "page_size"); ok {
		params.PageSize = int32(v)
	}

	result, err := s.adsSvc.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search listings: %w", err)
	}

	return jsonResult(result)
}

func (s *Server) handleGetListingDetail(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	input := stringArg(args, "id")
	if input == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	canonicalID, err := ads.ResolveInput(input, s.cfg.Shortcut.BaseURL, s.cfg.Frontdoor.BaseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolve input: %v", err)), nil
	}

	detail, err := s.adsSvc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return mcp.NewToolResultError(fmt.Sprintf("listing not found: %s", canonicalID)), nil
		}
		return nil, fmt.Errorf("get listing detail: %w", err)
	}

	return jsonResult(detail)
}

func (s *Server) handleSearchTransactions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	city := stringArg(args, "city")
	if city == "" {
		return mcp.NewToolResultError("city is required"), nil
	}

	address := stringArg(args, "address")
	var limit int32 = 200
	if v, ok := numberArg(args, "limit"); ok {
		limit = int32(v)
	}

	rows, err := s.queries.SearchTransactionsByCityAndAddress(ctx, db.SearchTransactionsByCityAndAddressParams{
		CityName:   city,
		SearchTerm: address,
		LimitCount: &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search transactions: %w", err)
	}

	return jsonResult(rows)
}

func (s *Server) handleListCities(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rows, err := s.queries.ListAvailableMunicipalities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list municipalities: %w", err)
	}
	return jsonResult(rows)
}

func (s *Server) handleListAvailableLocations(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rows, err := s.queries.ListMunicipalitiesWithPriceData(ctx)
	if err != nil {
		return nil, fmt.Errorf("list municipalities with price data: %w", err)
	}
	return jsonResult(rows)
}

func (s *Server) handleListCategories(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rows, err := s.queries.ListDistinctCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return jsonResult(rows)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

func stringArg(args map[string]any, key string) string {
	v, ok := args[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func numberArg(args map[string]any, key string) (float64, bool) {
	v, ok := args[key].(float64)
	return v, ok
}
