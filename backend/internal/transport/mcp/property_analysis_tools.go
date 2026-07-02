package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"koditon/internal/domain/ads"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (t *toolImpl) comparePropertiesTool() *mcp.Tool {
	return &mcp.Tool{Name: "koditon_compare_properties", Title: "Compare Properties", Description: "Compare canonical Koditon properties by ID, source URL, or listing/address text. Returns normalized rows, ranking, tradeoffs, missing-data warnings, market comparison, and recommended detail calls.", Annotations: &mcp.ToolAnnotations{Title: "Compare Properties", ReadOnlyHint: true, DestructiveHint: new(false), IdempotentHint: true, OpenWorldHint: new(false)}, Meta: mcp.Meta{"ui": map[string]any{"resourceUri": listingsAppResourceURI}}}
}

func (t *toolImpl) getPropertyMarketContextTool() *mcp.Tool {
	return &mcp.Tool{Name: "koditon_get_property_market_context", Title: "Get Property Market Context", Description: "Get comparable sales, median price per square meter, subject-vs-market hint, confidence, and explanation for a property or location/facts query.", Annotations: &mcp.ToolAnnotations{Title: "Get Property Market Context", ReadOnlyHint: true, DestructiveHint: new(false), IdempotentHint: true, OpenWorldHint: new(false)}, Meta: mcp.Meta{"ui": map[string]any{"resourceUri": listingsAppResourceURI}}}
}

func (t *toolImpl) compareProperties(ctx context.Context, _ *mcp.CallToolRequest, in PropertyComparisonInput) (*mcp.CallToolResult, *PropertyComparisonResult, error) {
	if len(in.IDs) < 2 {
		return newToolResultError("ids must contain at least two properties"), nil, nil
	}
	rows := make([]PropertySummary, 0, len(in.IDs))
	warnings := []string{}
	for _, id := range in.IDs {
		summary, err := t.propertySummaryForInput(ctx, id)
		if err != nil {
			if errors.Is(err, ads.ErrNotFound) {
				warnings = append(warnings, fmt.Sprintf("Property not found: %s", id))
				continue
			}
			return nil, nil, fmt.Errorf("compare property %q: %w", id, err)
		}
		rows = append(rows, summary)
	}
	if len(rows) == 0 {
		return newToolResultError("no comparable properties were resolved"), nil, nil
	}
	ranking := rankPropertySummaries(rows)
	result := PropertyComparisonResult{Schema: propertySchema("koditon.property.comparison"), View: "comparison", Summary: fmt.Sprintf("Compared %d Koditon properties.", len(rows)), Rows: rows, Ranking: ranking, Tradeoffs: propertyComparisonTradeoffs(rows, ranking, in), MissingDataWarnings: append(warnings, propertyMissingDataWarnings(rows)...), MarketComparison: aggregateMarketContext(rows), RecommendedFollowUpCalls: comparisonFollowUps(rows), DataQuality: propertyDataQuality(nil, warnings)}
	result.Markdown = propertyComparisonMarkdown(result)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Markdown}}}, &result, nil
}

func (t *toolImpl) getPropertyMarketContext(ctx context.Context, _ *mcp.CallToolRequest, in PropertyMarketContextInput) (*mcp.CallToolResult, *PropertyMarketContextResult, error) {
	subject, city, postal, price, area, err := t.resolveMarketSubject(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	limit := int32(50)
	if in.Limit != nil {
		if *in.Limit < 1 || *in.Limit > 500 {
			return newToolResultError("limit must be between 1 and 500"), nil, nil
		}
		limit = *in.Limit
	}
	query := strings.TrimSpace(in.Location)
	rows, err := t.runSearchTransactionsAdvanced(ctx, transactionsAdvancedParams{City: city, Query: query, PostalCodes: stringList(postal), Limit: limit})
	if err != nil {
		return nil, nil, fmt.Errorf("get market context transactions: %w", err)
	}
	sales := make([]ComparableSale, 0, len(rows))
	for _, row := range rows {
		sales = append(sales, propertyTransactionFromAdvancedRow(row))
	}
	market := deriveMarketSummary(price, area, sales)
	market.Confidence = marketContextConfidence(subject, sales)
	if market.Explanation == "" {
		market.Explanation = fmt.Sprintf("Computed from %d comparable sale price rows matching the available location filters.", len(sales))
	}
	result := PropertyMarketContextResult{Schema: propertySchema("koditon.property.market_context"), View: "market_context", Summary: fmt.Sprintf("Koditon market context returned %d comparable sales.", len(sales)), Subject: subject, Market: market, DataQuality: propertyDataQuality(nil, marketDataWarnings(subject, sales))}
	result.Markdown = propertyMarketContextMarkdown(result)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Markdown}}}, &result, nil
}

func (t *toolImpl) propertySummaryForInput(ctx context.Context, input string) (PropertySummary, error) {
	canonicalID, err := t.resolveDetailInput(ctx, strings.TrimSpace(input), getListingDetailInput{})
	if err != nil {
		return PropertySummary{}, err
	}
	detail, err := t.adsSvc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		return PropertySummary{}, err
	}
	return t.propertySummaryFromDetail(detail), nil
}

func (t *toolImpl) propertySummaryFromDetail(detail ads.UnifiedEntityDetail) PropertySummary {
	result := t.buildPropertyDetailResult(detail, false, false)
	summary := PropertySummary{Schema: propertySchema("koditon.property.summary"), ID: result.ID, EntityID: result.ID, EntityType: result.EntityType, CanonicalID: result.ID, SourceIDs: compactStrings([]string{result.Canonical.NativeID, result.ID}), NativeID: result.Canonical.NativeID, Source: result.Canonical.Source, Kind: result.Canonical.Kind, Title: result.Title, Subtitle: compactJoin([]string{result.Location.Address, result.Location.City, result.Location.Postal}, ", "), Location: result.Location, Facts: result.Facts, Costs: result.Costs, Features: result.Features, Lifecycle: result.Lifecycle, Media: result.Media, Market: result.MarketContext, Links: result.Links, Actions: result.Actions, DataQuality: result.DataQuality, Evidence: result.RawEvidence, Address: result.Location.Address, City: result.Location.City, Postal: result.Location.Postal, Price: result.Costs.AskingPrice, Area: result.Facts.AreaM2, RoomLayout: result.Facts.Rooms, URL: result.Canonical.URL, ExternalURLAvailable: result.Canonical.ExternalURLAvailable, WebURL: result.Links.Web, LastSeenAt: result.Canonical.LastSeenAt}
	return summary
}

func rankPropertySummaries(rows []PropertySummary) []PropertyRank {
	type scored struct {
		row   PropertySummary
		score float64
	}
	scoredRows := make([]scored, 0, len(rows))
	for _, row := range rows {
		score := 0.0
		reasons := propertyRankReasons(row)
		score += float64(len(reasons)) * 10
		if row.Costs.PricePerSquareMeter != nil {
			score += 10_000 / *row.Costs.PricePerSquareMeter
		}
		if row.DataQuality.Completeness > 0 {
			score += row.DataQuality.Completeness * 20
		}
		scoredRows = append(scoredRows, scored{row: row, score: score})
	}
	sort.SliceStable(scoredRows, func(i, j int) bool { return scoredRows[i].score > scoredRows[j].score })
	out := make([]PropertyRank, 0, len(scoredRows))
	for i, scoredRow := range scoredRows {
		out = append(out, PropertyRank{ID: scoredRow.row.ID, Rank: i + 1, Score: scoredRow.score, Reasons: propertyRankReasons(scoredRow.row)})
	}
	return out
}

func propertyRankReasons(row PropertySummary) []string {
	reasons := []string{}
	if row.Costs.PricePerSquareMeter != nil {
		reasons = append(reasons, fmt.Sprintf("price per m2 %.0f", *row.Costs.PricePerSquareMeter))
	}
	if row.Facts.EnergyClass != "" {
		reasons = append(reasons, "energy class available")
	}
	if row.Facts.Condition != "" {
		reasons = append(reasons, "condition available")
	}
	if len(row.Transactions) > 0 || row.Market.LinkedSalesCount > 0 {
		reasons = append(reasons, "has linked sale evidence")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "ranking is data-quality based because key comparison fields are missing")
	}
	return reasons
}

func propertyComparisonTradeoffs(rows []PropertySummary, ranking []PropertyRank, in PropertyComparisonInput) []string {
	tradeoffs := []string{}
	if in.BuyerProfile != "" {
		tradeoffs = append(tradeoffs, "Buyer profile considered: "+strings.TrimSpace(in.BuyerProfile))
	}
	if len(ranking) > 0 {
		tradeoffs = append(tradeoffs, "Top ranked property: "+ranking[0].ID)
	}
	for _, row := range rows {
		if row.Costs.AskingPrice == nil {
			tradeoffs = append(tradeoffs, row.ID+": asking price missing")
		}
		if row.Facts.AreaM2 == nil {
			tradeoffs = append(tradeoffs, row.ID+": area missing")
		}
	}
	return tradeoffs
}

func propertyMissingDataWarnings(rows []PropertySummary) []string {
	warnings := []string{}
	for _, row := range rows {
		for _, field := range row.DataQuality.MissingFields {
			warnings = append(warnings, row.ID+" missing "+field)
		}
	}
	return warnings
}

func aggregateMarketContext(rows []PropertySummary) MarketContext {
	sales := []ComparableSale{}
	for _, row := range rows {
		sales = append(sales, row.Transactions...)
	}
	market := deriveMarketSummary(nil, nil, sales)
	market.Explanation = "Aggregated from linked sale evidence on compared rows."
	return market
}

func comparisonFollowUps(rows []PropertySummary) []PropertyAction {
	actions := make([]PropertyAction, 0, len(rows))
	for _, row := range rows {
		actions = append(actions, PropertyAction{ID: "get_detail_" + row.ID, Label: "Get detail for " + row.ID, Type: "tool", Target: "koditon_get_property_detail", Params: map[string]any{"id": row.ID}})
	}
	return actions
}

func propertyComparisonMarkdown(result PropertyComparisonResult) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(result.Summary)
	for _, rank := range result.Ranking {
		b.WriteString("\n- #")
		fmt.Fprintf(&b, "%d %s", rank.Rank, rank.ID)
		if len(rank.Reasons) > 0 {
			b.WriteString(": ")
			b.WriteString(strings.Join(rank.Reasons, "; "))
		}
	}
	return strings.TrimSpace(b.String())
}

func (t *toolImpl) resolveMarketSubject(ctx context.Context, in PropertyMarketContextInput) (*PropertySummary, string, string, *int64, *float64, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, strings.TrimSpace(in.City), strings.TrimSpace(in.Postal), nil, nil, nil
	}
	summary, err := t.propertySummaryForInput(ctx, in.ID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, "", "", nil, nil, fmt.Errorf("property not found: %s", in.ID)
		}
		return nil, "", "", nil, nil, fmt.Errorf("resolve market subject: %w", err)
	}
	city := firstNonEmpty(in.City, summary.Location.City)
	postal := firstNonEmpty(in.Postal, summary.Location.Postal)
	return &summary, city, postal, summary.Costs.AskingPrice, summary.Facts.AreaM2, nil
}

func marketContextConfidence(subject *PropertySummary, sales []ComparableSale) string {
	if subject != nil && len(sales) >= 10 {
		return "high"
	}
	if len(sales) >= 3 {
		return "medium"
	}
	return "low"
}

func marketDataWarnings(subject *PropertySummary, sales []ComparableSale) []string {
	warnings := []string{}
	if subject == nil {
		warnings = append(warnings, "No subject property was resolved; market context is location-only")
	}
	if len(sales) == 0 {
		warnings = append(warnings, "No comparable sales found")
	}
	return warnings
}

func propertyMarketContextMarkdown(result PropertyMarketContextResult) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(result.Summary)
	if result.Subject != nil {
		b.WriteString("\nSubject: ")
		b.WriteString(result.Subject.Title)
	}
	if result.Market.MedianPricePerM2 != nil {
		b.WriteString("\nMedian price/m2: ")
		fmt.Fprintf(&b, "%.0f", *result.Market.MedianPricePerM2)
	}
	if result.Market.OverUnderMarketHint != "" {
		b.WriteString("\nSubject vs market: ")
		b.WriteString(result.Market.OverUnderMarketHint)
	}
	return strings.TrimSpace(b.String())
}
