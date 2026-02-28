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
	query := stringArg(args, "query")
	if query == "" {
		query = stringArg(args, "address")
	}
	params := ads.SearchParams{
		Query:       query,
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
		input = stringArg(args, "input")
	}
	if input == "" {
		return mcp.NewToolResultError("id or input is required"), nil
	}
	canonicalID, err := s.resolveDetailInput(ctx, input, args)
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

	return jsonResult(buildDetailResult(detail))
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

func buildDetailResult(detail ads.UnifiedEntityDetail) any {
	result := map[string]any{
		"canonical":       detail.Canonical,
		"canonical_extra": detail.CanonicalExtra,
		"source_specific": detail.SourceSpecific,
		"related":         detail.Related,
		"raw":             detail.Raw,
	}
	rawJSON := parseJSON(detail.Raw.Pretty)
	if rawJSON != nil {
		result["raw_json"] = rawJSON
	}
	return result
}

func (s *Server) resolveDetailInput(ctx context.Context, input string, args map[string]any) (string, error) {
	canonicalID, err := ads.ResolveInput(input, s.cfg.Shortcut.SitemapBase, s.cfg.Frontdoor.SitemapBase)
	if err == nil {
		return canonicalID, nil
	}
	query := strings.TrimSpace(input)
	if query == "" {
		return "", fmt.Errorf("input is empty")
	}
	kind := stringArg(args, "kind")
	if kind == "" {
		kind = "ad"
	}
	params := ads.SearchParams{
		Query:    query,
		Source:   stringArg(args, "source"),
		Kind:     kind,
		City:     stringArg(args, "city"),
		Postal:   stringArg(args, "postal"),
		Page:     1,
		PageSize: 25,
		Sort:     "seen_desc",
	}
	if v, ok := numberArg(args, "max_candidates"); ok {
		params.PageSize = int32(v)
	}
	search, err := s.adsSvc.Search(ctx, params)
	if err != nil {
		return "", fmt.Errorf("search by text: %w", err)
	}
	if len(search.Rows) == 0 {
		return "", fmt.Errorf("no listing found for input: %s", query)
	}
	matches := chooseBestRows(query, search.Rows)
	if len(matches) == 1 {
		return matches[0].CanonicalID, nil
	}
	return "", fmt.Errorf("input matched multiple listings (%d). top candidates: %s", len(matches), formatCandidates(matches, 5))
}

func chooseBestRows(input string, rows []ads.UnifiedEntityRow) []ads.UnifiedEntityRow {
	if len(rows) == 0 {
		return nil
	}
	needle := normalizeText(input)
	if needle == "" {
		return rows
	}
	exact := make([]ads.UnifiedEntityRow, 0)
	for _, row := range rows {
		if normalizeText(row.Address) == needle || normalizeText(row.Headline) == needle || normalizeText(row.URL) == needle {
			exact = append(exact, row)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return rows
}

func formatCandidates(rows []ads.UnifiedEntityRow, max int) string {
	if len(rows) == 0 {
		return ""
	}
	if max <= 0 {
		max = 1
	}
	limit := min(max, len(rows))
	parts := make([]string, 0, limit)
	for _, row := range rows[:limit] {
		label := strings.TrimSpace(row.Address)
		if label == "" {
			label = strings.TrimSpace(row.Headline)
		}
		if label == "" {
			label = row.CanonicalID
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", row.CanonicalID, label))
	}
	return strings.Join(parts, "; ")
}

func parseJSON(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil
	}
	return out
}

func normalizeText(value string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
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
