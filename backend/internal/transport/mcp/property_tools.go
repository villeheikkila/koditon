package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"koditon/internal/domain/ads"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type propertyIncludeOptions struct {
	LinkedTransactions bool `json:"linked_transactions,omitempty" jsonschema:"Include linked actual sale prices when available."`
	SourceRecords      bool `json:"source_records,omitempty" jsonschema:"Include source record evidence where available."`
	Insights           bool `json:"insights,omitempty" jsonschema:"Include computed insight summaries where available."`
	Evidence           bool `json:"evidence,omitempty" jsonschema:"Include compact match and provenance evidence."`
	Raw                bool `json:"raw,omitempty" jsonschema:"Include raw source payloads in detail responses."`
}

type propertyDetailInput struct {
	ID             string                 `json:"id,omitempty" jsonschema:"Canonical ID, source URL, or exact listing/address text."`
	Input          string                 `json:"input,omitempty" jsonschema:"Fallback canonical ID, source URL, or exact listing/address text."`
	Source         string                 `json:"source,omitempty" jsonschema:"Source hint: shortcut or frontdoor."`
	Kind           string                 `json:"kind,omitempty" jsonschema:"Kind hint when resolving text: ad, building, announcement, or rental."`
	City           string                 `json:"city,omitempty" jsonschema:"City hint when resolving text."`
	Postal         string                 `json:"postal,omitempty" jsonschema:"Postal code hint when resolving text."`
	MaxCandidates  *int32                 `json:"max_candidates,omitempty" jsonschema:"Candidate count when resolving text. Must be 25, 50, or 100."`
	Include        propertyIncludeOptions `json:"include,omitempty" jsonschema:"Controls optional raw payload and evidence fields."`
	IncludeRawJSON bool                   `json:"include_raw_json,omitempty" jsonschema:"Compatibility flag for including parsed raw source JSON."`
}

type propertyEntityLinks struct {
	Web      string `json:"web,omitempty"`
	Source   string `json:"source,omitempty"`
	DetailID string `json:"detail_id,omitempty"`
}

type propertyMatchSummary struct {
	Status     string   `json:"status,omitempty"`
	Method     string   `json:"method,omitempty"`
	Score      *int32   `json:"score,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Reasons    []string `json:"reasons,omitempty"`
}

type propertyInsightSummary struct {
	Count       int32  `json:"count,omitempty"`
	TopSeverity string `json:"top_severity,omitempty"`
}

type propertySourceSummary struct {
	CanonicalID string `json:"canonical_id,omitempty"`
	Source      string `json:"source,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
}

type propertyQueryFacets struct {
	Sources      map[string]int `json:"sources,omitempty"`
	Kinds        map[string]int `json:"kinds,omitempty"`
	Cities       map[string]int `json:"cities,omitempty"`
	WithSales    int            `json:"with_sales,omitempty"`
	WithInsights int            `json:"with_insights,omitempty"`
}

type propertyQueryDiagnostics struct {
	ReturnedTransactions int      `json:"returned_transactions,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
}

type propertyCanonicalFields struct {
	CanonicalID          string     `json:"canonical_id"`
	Source               string     `json:"source"`
	Kind                 string     `json:"kind"`
	NativeID             string     `json:"native_id,omitempty"`
	Headline             string     `json:"headline,omitempty"`
	Address              string     `json:"address,omitempty"`
	City                 string     `json:"city,omitempty"`
	Postal               string     `json:"postal,omitempty"`
	Latitude             *float64   `json:"latitude,omitempty"`
	Longitude            *float64   `json:"longitude,omitempty"`
	Price                *int64     `json:"price,omitempty"`
	Area                 *float64   `json:"area,omitempty"`
	RoomLayout           string     `json:"room_layout,omitempty"`
	URL                  string     `json:"url,omitempty"`
	ExternalURLAvailable bool       `json:"external_url_available,omitempty"`
	LastSeenAt           *time.Time `json:"last_seen_at,omitempty"`
}

type propertyDetailField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type propertyRawPayload struct {
	Pretty        string `json:"pretty"`
	OriginalBytes int    `json:"original_bytes"`
}

type propertyReportSection struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func (t *toolImpl) queryPropertiesTool() *mcp.Tool {
	return &mcp.Tool{Name: "koditon_query_properties", Title: "Query Properties", Description: "Canonical Koditon property query for precise listing, address, and transaction search. Returns typed rows, linked sale prices, web UI links, and a readable summary.", Annotations: &mcp.ToolAnnotations{Title: "Query Properties", ReadOnlyHint: true, DestructiveHint: new(false), IdempotentHint: true, OpenWorldHint: new(false)}, Meta: mcp.Meta{"ui": map[string]any{"resourceUri": listingsAppResourceURI}}}
}

func (t *toolImpl) getPropertyDetailTool() *mcp.Tool {
	return &mcp.Tool{Name: "koditon_get_property_detail", Title: "Get Property Detail", Description: "Canonical Koditon property detail lookup by canonical ID, source URL, or listing/address text. Returns typed canonical fields, normalized fields, source facts, related records, and a markdown report.", Annotations: &mcp.ToolAnnotations{Title: "Get Property Detail", ReadOnlyHint: true, DestructiveHint: new(false), IdempotentHint: true, OpenWorldHint: new(false)}, Meta: mcp.Meta{"ui": map[string]any{"resourceUri": listingsAppResourceURI}}}
}

func (t *toolImpl) queryProperties(ctx context.Context, _ *mcp.CallToolRequest, in PropertyQueryInput) (*mcp.CallToolResult, *PropertyQueryResult, error) {
	in = normalizePropertyQueryInput(in)
	if in.Price.Min != nil && in.Price.Max != nil && *in.Price.Min > *in.Price.Max {
		return newToolResultError("price.min cannot be greater than price.max"), nil, nil
	}
	if in.AreaM2.Min != nil && in.AreaM2.Max != nil && *in.AreaM2.Min > *in.AreaM2.Max {
		return newToolResultError("area_m2.min cannot be greater than area_m2.max"), nil, nil
	}
	entity := strings.ToLower(strings.TrimSpace(in.Entity))
	if entity == "" || entity == "property" || entity == "listing" || entity == "all" || entity == "address" {
		result, toolErr, err := t.buildPropertyListingQueryResult(ctx, in)
		if toolErr != "" {
			return newToolResultError(toolErr), nil, nil
		}
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Summary}}}, &result, nil
	}
	if entity == "transaction" || entity == "transactions" || entity == "sale" || entity == "sales" {
		result, toolErr, err := t.buildPropertyTransactionQueryResult(ctx, in)
		if toolErr != "" {
			return newToolResultError(toolErr), nil, nil
		}
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Summary}}}, &result, nil
	}
	return newToolResultError("entity must be property, listing, address, transaction, or all"), nil, nil
}

func (t *toolImpl) getPropertyDetail(ctx context.Context, _ *mcp.CallToolRequest, in propertyDetailInput) (*mcp.CallToolResult, *PropertyDetail, error) {
	input := strings.TrimSpace(in.ID)
	if input == "" {
		input = strings.TrimSpace(in.Input)
	}
	if input == "" {
		return newToolResultError("id or input is required"), nil, nil
	}
	canonicalID, err := t.resolveDetailInput(ctx, input, getListingDetailInput{Source: in.Source, Kind: in.Kind, City: in.City, Postal: in.Postal, MaxCandidates: in.MaxCandidates})
	if err != nil {
		return newToolResultError(fmt.Sprintf("resolve input: %v", err)), nil, nil
	}
	detail, err := t.adsSvc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return newToolResultError(fmt.Sprintf("property not found: %s", canonicalID)), nil, nil
		}
		return nil, nil, fmt.Errorf("get property detail: %w", err)
	}
	result := t.buildPropertyDetailResult(detail, in.Include.Raw || in.IncludeRawJSON, in.IncludeRawJSON)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Markdown}}}, &result, nil
}

func (t *toolImpl) buildPropertyListingQueryResult(ctx context.Context, in PropertyQueryInput) (PropertyQueryResult, string, error) {
	appResult, toolErr, err := t.buildListingsAppResult(ctx, findListingsInput{Query: in.Query, Address: in.Address, Source: in.Source, Kind: in.Kind, ListingType: in.ListingType, City: in.City, Postal: in.Postal, MinPrice: in.Price.Min, MaxPrice: in.Price.Max, MinArea: in.AreaM2.Min, MaxArea: in.AreaM2.Max, Sort: in.Sort, Page: in.Page, PageSize: in.PageSize, IncludePrices: in.Include.LinkedTransactions})
	if toolErr != "" || err != nil {
		return PropertyQueryResult{}, toolErr, err
	}
	rows := make([]PropertySummary, 0, len(appResult.Rows))
	for _, row := range appResult.Rows {
		rows = append(rows, t.propertySummaryFromListingAppRow(row))
	}
	transactions := propertyTransactionsFromAppRows(appResult.Transactions)
	result := PropertyQueryResult{Schema: propertySchema("koditon.property.query_result"), View: "search_results", Summary: propertyQuerySummary(appResult.Mode, len(rows), len(appResult.Transactions)), Mode: appResult.Mode, Entity: "property", Query: propertyQueryEchoFromInput(in), Rows: rows, Transactions: transactions, Facets: propertyFacets(rows), DataQuality: propertyCollectionDataQuality(rows, transactions), Diagnostics: propertyQueryDiagnostics{ReturnedTransactions: len(appResult.Transactions)}, WebURL: appResult.WebURL, Total: appResult.Total, Page: appResult.Page, PageSize: appResult.PageSize}
	return result, "", nil
}

func (t *toolImpl) buildPropertyTransactionQueryResult(ctx context.Context, in PropertyQueryInput) (PropertyQueryResult, string, error) {
	limit := int32(200)
	if in.Limit != nil {
		if *in.Limit < 1 || *in.Limit > 5000 {
			return PropertyQueryResult{}, "limit must be between 1 and 5000", nil
		}
		limit = *in.Limit
	}
	rows, err := t.runSearchTransactionsAdvanced(ctx, transactionsAdvancedParams{City: in.City, Query: in.Query, PostalCodes: stringList(in.Postal), MinPrice: int64PtrToInt32Ptr(in.Price.Min), MaxPrice: int64PtrToInt32Ptr(in.Price.Max), MinArea: in.AreaM2.Min, MaxArea: in.AreaM2.Max, Sort: in.Sort, Limit: limit})
	if err != nil {
		return PropertyQueryResult{}, "", fmt.Errorf("query property transactions: %w", err)
	}
	transactions := make([]ComparableSale, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, propertyTransactionFromAdvancedRow(row))
	}
	result := PropertyQueryResult{Schema: propertySchema("koditon.property.query_result"), View: "market_context", Summary: fmt.Sprintf("Koditon transaction query returned %d sale price rows.", len(transactions)), Mode: "transaction", Entity: "transaction", Query: propertyQueryEchoFromInput(in), Rows: []PropertySummary{}, Transactions: transactions, DataQuality: propertyDataQuality([]string{"subject property"}, nil), Diagnostics: propertyQueryDiagnostics{ReturnedTransactions: len(transactions)}, WebURL: t.webPath("/prices"), Total: int64(len(transactions)), Page: 1, PageSize: limit}
	return result, "", nil
}

func (t *toolImpl) propertySummaryFromListingAppRow(row listingAppRow) PropertySummary {
	transactions := propertyTransactionsFromAppRows(row.Transactions)
	summary := PropertySummary{Schema: propertySchema("koditon.property.summary"), ID: row.CanonicalID, EntityID: row.CanonicalID, EntityType: "listing", CanonicalID: row.CanonicalID, SourceIDs: compactStrings([]string{row.CanonicalID, row.NativeID, row.ListingID}), OfferingID: row.OfferingID, GroupingID: row.GroupingID, ListingID: row.ListingID, NativeID: row.NativeID, Source: row.Source, Kind: row.Kind, Title: firstNonEmpty(row.Title, row.Address, row.CanonicalID), Subtitle: compactJoin([]string{row.Address, row.City, row.Postal}, ", "), Badges: propertySummaryBadges(row, transactions), Address: row.Address, City: row.City, Postal: row.Postal, Price: row.Price, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, WebURL: row.WebURL, LastSeenAt: row.LastSeenAt, Links: propertyEntityLinks{Web: row.WebURL, Source: row.URL, DetailID: row.CanonicalID}, Transactions: transactions}
	summary.Location = PropertyLocation{Address: row.Address, City: row.City, Postal: row.Postal, Coordinates: coordinates(row.Longitude, row.Latitude)}
	summary.Facts = PropertyFacts{Price: row.Price, AreaM2: row.Area, Rooms: row.RoomLayout}
	summary.Costs = PropertyCosts{AskingPrice: row.Price, PricePerSquareMeter: pricePerM2Float(row.Price, row.Area)}
	summary.Lifecycle = PropertyLifecycle{FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, PriceChanged: row.PriceChanged}
	summary.Media = PropertyMedia{}
	summary.Market = deriveMarketSummary(row.Price, row.Area, transactions)
	summary.Match = propertyMatchSummary{Status: row.MatchStatus, Method: row.MatchMethod, Score: row.MatchScore, Confidence: row.MatchStatus}
	summary.Insights = propertyInsightSummary{Count: row.InsightCount, TopSeverity: row.InsightTopSeverity}
	summary.Actions = deriveAvailableActions(summary.ID, summary.Links)
	summary.DataQuality = propertySummaryDataQuality(summary)
	return summary
}

func propertyTransactionsFromAppRows(rows []priceTransactionAppRow) []ComparableSale {
	out := make([]ComparableSale, 0, len(rows))
	for _, row := range rows {
		out = append(out, ComparableSale{Schema: propertySchema("koditon.property.comparable_sale"), TransactionID: firstNonEmpty(row.TransactionID, row.ID), ID: firstNonEmpty(row.ID, row.TransactionID), Description: row.Description, Category: row.Category, Type: row.Type, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Postal: row.Postal, Confidence: row.Confidence})
	}
	return out
}

func propertyTransactionFromAdvancedRow(row transactionsAdvancedRow) ComparableSale {
	price := int64(row.Price)
	priceM2 := int64(row.PricePerSquareMeter)
	buildYear := row.BuildYear
	floor := ""
	if row.Floor != nil {
		floor = *row.Floor
	}
	return ComparableSale{Schema: propertySchema("koditon.property.comparable_sale"), TransactionID: row.TransactionID.String(), ID: row.TransactionID.String(), Description: row.Description, Category: row.Category, Type: row.Type, Area: &row.Area, Price: &price, PricePerSquareMeter: &priceM2, BuildYear: &buildYear, Floor: floor, Elevator: &row.Elevator, Condition: stringPtrValue(row.Condition), Plot: stringPtrValue(row.Plot), EnergyClass: stringPtrValue(row.EnergyClass), PeriodIdentifier: row.PeriodIdentifier, City: row.City, Neighborhood: row.Neighborhood, Postal: row.PostalCode, CreatedAt: &row.CreatedAt, UpdatedAt: &row.UpdatedAt}
}

func (t *toolImpl) buildPropertyDetailResult(detail ads.UnifiedEntityDetail, includeRaw bool, includeRawJSON bool) PropertyDetail {
	webURL := t.listingWebURL(detail.Canonical.CanonicalID, detail.Canonical.Kind, "")
	result := PropertyDetail{Schema: propertySchema("koditon.property.detail"), View: "detail", ID: detail.Canonical.CanonicalID, EntityType: "listing", Title: firstNonEmpty(detail.Canonical.Headline, detail.Canonical.Address, detail.Canonical.CanonicalID), Canonical: propertyCanonicalFromDomain(detail.Canonical), CanonicalExtra: propertyDetailFieldsFromDomain(detail.CanonicalExtra), SourceSpecific: propertyDetailFieldsFromDomain(detail.SourceSpecific), Related: propertyDetailFieldsFromDomain(detail.Related), Normalized: normalizedPropertyFields(detail.Normalized), Links: propertyEntityLinks{Web: webURL, Source: detail.Canonical.URL, DetailID: detail.Canonical.CanonicalID}}
	result.Location = derivePropertyLocation(result.Canonical, result.Normalized)
	result.Facts = derivePropertyFacts(result.Canonical, result.Normalized)
	result.Costs = derivePropertyCosts(result.Canonical, result.Normalized)
	result.Features = derivePropertyFeatures(result.Normalized)
	result.Lifecycle = PropertyLifecycle{LastSeenAt: result.Canonical.LastSeenAt}
	result.Media = PropertyMedia{}
	result.MarketContext = deriveMarketSummary(result.Costs.AskingPrice, result.Facts.AreaM2, nil)
	result.Overview = propertyDetailOverview(result)
	result.HousingCompany = filterDetailFields(result.Related, []string{"housing", "company", "taloyhtiö", "yhtiö"})
	result.Building = filterDetailFields(result.SourceSpecific, []string{"building", "build", "floor", "energy", "heating", "elevator"})
	result.Renovations = filterDetailFields(result.SourceSpecific, []string{"renovation", "repair", "korjaus", "remont"})
	result.RawEvidence = propertyEvidenceFromDetail(result)
	result.Actions = deriveAvailableActions(result.ID, result.Links)
	result.DataQuality = propertyDetailDataQuality(result)
	result.Report = propertyDetailReport(result)
	result.Reports = propertyReportsFromSections(result.Report)
	result.Summary = propertyDetailSummary(result)
	result.Markdown = propertyDetailMarkdown(result)
	if includeRaw {
		raw := propertyRawPayload{Pretty: detail.Raw.Pretty, OriginalBytes: detail.Raw.OriginalBytes}
		result.Raw = &raw
	}
	if includeRawJSON {
		result.RawJSON = parseJSON(detail.Raw.Pretty)
	}
	return result
}

func propertyCanonicalFromDomain(value ads.UnifiedCanonicalFields) propertyCanonicalFields {
	lastSeenAt := value.LastSeenAt
	return propertyCanonicalFields{CanonicalID: value.CanonicalID, Source: value.Source, Kind: value.Kind, NativeID: value.NativeID, Headline: value.Headline, Address: value.Address, City: value.City, Postal: value.Postal, Latitude: value.Latitude, Longitude: value.Longitude, Price: value.Price, Area: value.Area, RoomLayout: value.RoomLayout, URL: value.URL, ExternalURLAvailable: value.ExternalURLAvailable, LastSeenAt: &lastSeenAt}
}

func propertyDetailFieldsFromDomain(fields []ads.DetailField) []propertyDetailField {
	out := make([]propertyDetailField, 0, len(fields))
	for _, field := range fields {
		out = append(out, propertyDetailField{Label: field.Label, Value: field.Value})
	}
	return out
}

func propertyDetailReport(result PropertyDetail) []propertyReportSection {
	overview := compactStrings([]string{labelValue("Address", firstNonEmpty(result.Canonical.Address, result.Normalized.StreetAddress)), labelValue("City", firstNonEmpty(result.Canonical.City, result.Normalized.City)), labelValue("Postal", firstNonEmpty(result.Canonical.Postal, result.Normalized.Postal)), labelValue("Price", formatInt64Value(result.Canonical.Price)), labelValue("Area", formatFloatValue(result.Canonical.Area)), labelValue("Rooms", firstNonEmpty(result.Canonical.RoomLayout, result.Normalized.RoomLayout))})
	source := detailFieldsToItems(result.SourceSpecific, 8)
	normalized := compactStrings([]string{labelValue("Debt-free price", formatInt64Value(result.Normalized.DebtFreePrice)), labelValue("Price per m2", formatFloatValue(result.Normalized.PricePerSquareMeter)), labelValue("Build year", formatInt32Value(result.Normalized.BuildYear)), labelValue("Condition", result.Normalized.Condition), labelValue("Energy class", result.Normalized.EnergyClass), labelValue("Elevator", formatBoolValue(result.Normalized.Elevator)), labelValue("Sauna", formatBoolValue(result.Normalized.Sauna))})
	sections := []propertyReportSection{{Title: "Overview", Items: overview}}
	if len(normalized) > 0 {
		sections = append(sections, propertyReportSection{Title: "Normalized facts", Items: normalized})
	}
	if len(source) > 0 {
		sections = append(sections, propertyReportSection{Title: "Source facts", Items: source})
	}
	if related := detailFieldsToItems(result.Related, 8); len(related) > 0 {
		sections = append(sections, propertyReportSection{Title: "Related records", Items: related})
	}
	return sections
}

func propertyDetailSummary(result PropertyDetail) string {
	title := firstNonEmpty(result.Canonical.Headline, result.Canonical.Address, result.Canonical.CanonicalID)
	return fmt.Sprintf("Koditon property detail for %s (%s/%s).", title, result.Canonical.Source, result.Canonical.Kind)
}

func propertyDetailMarkdown(result PropertyDetail) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(propertyDetailSummary(result))
	b.WriteString("\n")
	for _, section := range result.Report {
		if len(section.Items) == 0 {
			continue
		}
		b.WriteString("\n## ")
		b.WriteString(section.Title)
		b.WriteString("\n")
		for _, item := range section.Items {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
	}
	if result.Links.Web != "" {
		b.WriteString("\nOpen in Koditon: ")
		b.WriteString(result.Links.Web)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func propertyQuerySummary(mode string, rows int, transactions int) string {
	label := "property query"
	if mode == "address" {
		label = "address property query"
	}
	return fmt.Sprintf("Koditon %s returned %d property rows and %d linked sale prices.", label, rows, transactions)
}

func propertySchema(name string) PropertySchema {
	return PropertySchema{Name: name, Version: propertySchemaVersion}
}

func normalizePropertyQueryInput(in PropertyQueryInput) PropertyQueryInput {
	if in.Price.Min == nil {
		in.Price.Min = in.MinPrice
	}
	if in.Price.Max == nil {
		in.Price.Max = in.MaxPrice
	}
	if in.AreaM2.Min == nil {
		in.AreaM2.Min = in.MinArea
	}
	if in.AreaM2.Max == nil {
		in.AreaM2.Max = in.MaxArea
	}
	if in.Location != "" && in.City == "" && in.Postal == "" && in.Address == "" {
		in.Query = firstNonEmpty(in.Query, in.Location)
	}
	if len(in.ListingTypes) > 0 && in.ListingType == "" {
		in.ListingType = in.ListingTypes[0]
	}
	return in
}

func propertyQueryEchoFromInput(in PropertyQueryInput) PropertyQueryEcho {
	return PropertyQueryEcho{Text: strings.TrimSpace(in.Query), Address: strings.TrimSpace(in.Address), Location: strings.TrimSpace(in.Location), Source: normalizeAppFilter(in.Source), Kind: normalizeAppFilter(in.Kind), ListingType: normalizeAppFilter(in.ListingType), ListingTypes: in.ListingTypes, City: strings.TrimSpace(in.City), Postal: strings.TrimSpace(in.Postal), Price: in.Price, DebtFreePrice: in.DebtFreePrice, AreaM2: in.AreaM2, PropertyTypes: in.PropertyTypes, OwnershipTypes: in.OwnershipTypes, Features: in.Features, Sort: strings.TrimSpace(in.Sort), Include: in.Include}
}

func derivePropertyLocation(canonical propertyCanonicalFields, normalized normalizedPropertyFields) PropertyLocation {
	lng := firstFloat64Ptr(canonical.Longitude, normalized.Longitude)
	lat := firstFloat64Ptr(canonical.Latitude, normalized.Latitude)
	return PropertyLocation{Address: firstNonEmpty(canonical.Address, normalized.StreetAddress), Street: firstNonEmpty(normalized.StreetAddress, canonical.Address), City: firstNonEmpty(canonical.City, normalized.City), Postal: firstNonEmpty(canonical.Postal, normalized.Postal), Coordinates: coordinates(lng, lat)}
}

func derivePropertyFacts(canonical propertyCanonicalFields, normalized normalizedPropertyFields) PropertyFacts {
	return PropertyFacts{Price: firstInt64Ptr(canonical.Price, normalized.AskingPrice), AreaM2: firstFloat64Ptr(canonical.Area, normalized.AreaM2), Rooms: firstNonEmpty(canonical.RoomLayout, normalized.RoomLayout), RoomsCount: normalized.RoomsCount, BuildYear: normalized.BuildYear, Floor: normalized.FloorLevel, TotalFloors: normalized.TotalFloors, Condition: normalized.Condition, EnergyClass: normalized.EnergyClass}
}

func derivePropertyCosts(canonical propertyCanonicalFields, normalized normalizedPropertyFields) PropertyCosts {
	askingPrice := firstInt64Ptr(normalized.AskingPrice, canonical.Price)
	area := firstFloat64Ptr(normalized.AreaM2, canonical.Area)
	return PropertyCosts{AskingPrice: askingPrice, DebtFreePrice: normalized.DebtFreePrice, DebtShareAmount: normalized.DebtShareAmount, PricePerSquareMeter: firstFloat64Ptr(normalized.PricePerSquareMeter, pricePerM2Float(askingPrice, area)), MaintenanceChargeMonthly: normalized.MaintenanceChargeMonthly, TotalChargeMonthly: normalized.TotalChargeMonthly, WaterCharge: normalized.WaterCharge}
}

func derivePropertyFeatures(normalized normalizedPropertyFields) PropertyFeatures {
	return PropertyFeatures{Sauna: normalized.Sauna, Elevator: normalized.Elevator, PlotType: normalized.PlotType}
}

func deriveMarketSummary(price *int64, area *float64, sales []ComparableSale) MarketContext {
	market := MarketContext{LinkedSalesCount: len(sales), ComparableSales: sales, Confidence: "low"}
	if len(sales) > 0 {
		market.NearestComparable = &sales[0]
		market.Confidence = "medium"
	}
	if price != nil && area != nil && *area > 0 {
		subject := float64(*price) / *area
		market.SubjectPricePerM2 = &subject
	}
	values := comparablePricePerM2Values(sales)
	if len(values) > 0 {
		median := medianFloat64(values)
		market.MedianPricePerM2 = &median
		if market.SubjectPricePerM2 != nil && median > 0 {
			diff := ((*market.SubjectPricePerM2 - median) / median) * 100
			market.SubjectVsMarketPct = &diff
			market.OverUnderMarketHint = marketHint(diff)
		}
	}
	if len(sales) == 0 {
		market.Explanation = "No linked comparable sales were available in the current result set."
		market.RecommendedFollowUps = []string{"Call koditon_get_property_market_context with the listing id or city/postal filters."}
	}
	return market
}

func deriveAvailableActions(id string, links propertyEntityLinks) []PropertyAction {
	actions := []PropertyAction{}
	if links.Web != "" {
		actions = append(actions, PropertyAction{ID: "open_web", Label: "Open in Koditon", Type: "link", Target: links.Web})
	}
	if links.Source != "" {
		actions = append(actions, PropertyAction{ID: "open_source", Label: "Open source listing", Type: "link", Target: links.Source})
	}
	if id != "" {
		actions = append(actions, PropertyAction{ID: "get_detail", Label: "Get detail", Type: "tool", Target: "koditon_get_property_detail", Params: map[string]any{"id": id}})
		actions = append(actions, PropertyAction{ID: "market_context", Label: "Get market context", Type: "tool", Target: "koditon_get_property_market_context", Params: map[string]any{"id": id}})
	}
	return actions
}

func propertyDetailOverview(result PropertyDetail) []propertyDetailField {
	return []propertyDetailField{{Label: "Title", Value: result.Title}, {Label: "Address", Value: result.Location.Address}, {Label: "City", Value: result.Location.City}, {Label: "Postal", Value: result.Location.Postal}, {Label: "Asking price", Value: formatInt64Value(result.Costs.AskingPrice)}, {Label: "Debt-free price", Value: formatInt64Value(result.Costs.DebtFreePrice)}, {Label: "Area m2", Value: formatFloatValue(result.Facts.AreaM2)}, {Label: "Rooms", Value: result.Facts.Rooms}}
}

func propertySummaryBadges(row listingAppRow, transactions []ComparableSale) []string {
	badges := compactStrings([]string{sourceLabel(row.Source), row.Kind})
	if len(transactions) > 0 {
		badges = append(badges, fmt.Sprintf("%d linked sales", len(transactions)))
	}
	if row.PriceChanged != nil && *row.PriceChanged {
		badges = append(badges, "price changed")
	}
	if row.InsightTopSeverity != "" {
		badges = append(badges, row.InsightTopSeverity+" insight")
	}
	return badges
}

func propertyReportsFromSections(sections []propertyReportSection) []PropertyReport {
	out := make([]PropertyReport, 0, len(sections))
	for _, section := range sections {
		out = append(out, PropertyReport{Title: section.Title, Items: section.Items})
	}
	return out
}

func propertyEvidenceFromDetail(result PropertyDetail) []PropertyEvidence {
	evidence := []PropertyEvidence{}
	for _, field := range append(result.CanonicalExtra, result.SourceSpecific...) {
		evidence = append(evidence, PropertyEvidence{Field: field.Label, Value: field.Value, Source: result.Canonical.Source, Confidence: "source"})
	}
	return evidence
}

func filterDetailFields(fields []propertyDetailField, needles []string) []propertyDetailField {
	out := []propertyDetailField{}
	for _, field := range fields {
		haystack := strings.ToLower(field.Label + " " + field.Value)
		for _, needle := range needles {
			if strings.Contains(haystack, strings.ToLower(needle)) {
				out = append(out, field)
				break
			}
		}
	}
	return out
}

func propertySummaryDataQuality(summary PropertySummary) PropertyDataQuality {
	missing := []string{}
	if summary.Location.Address == "" {
		missing = append(missing, "location.address")
	}
	if summary.Facts.AreaM2 == nil {
		missing = append(missing, "facts.area_m2")
	}
	if summary.Costs.AskingPrice == nil {
		missing = append(missing, "costs.asking_price")
	}
	if summary.Facts.EnergyClass == "" {
		missing = append(missing, "facts.energy_class")
	}
	warnings := []string{}
	if len(summary.Transactions) == 0 {
		warnings = append(warnings, "No linked sale prices found")
	}
	return propertyDataQuality(missing, warnings)
}

func propertyDetailDataQuality(detail PropertyDetail) PropertyDataQuality {
	missing := []string{}
	if detail.Location.Address == "" {
		missing = append(missing, "location.address")
	}
	if detail.Facts.AreaM2 == nil {
		missing = append(missing, "facts.area_m2")
	}
	if detail.Costs.AskingPrice == nil && detail.Costs.DebtFreePrice == nil {
		missing = append(missing, "costs.price")
	}
	if detail.Facts.EnergyClass == "" {
		missing = append(missing, "facts.energy_class")
	}
	if len(detail.Renovations) == 0 && detail.Normalized.RenovationsDoneText == "" && detail.Normalized.RenovationsPlannedText == "" {
		missing = append(missing, "renovations")
	}
	return propertyDataQuality(missing, nil)
}

func propertyCollectionDataQuality(rows []PropertySummary, transactions []ComparableSale) PropertyDataQuality {
	missing := []string{}
	warnings := []string{}
	if len(rows) == 0 {
		missing = append(missing, "rows")
	}
	if len(transactions) == 0 {
		warnings = append(warnings, "No linked sale prices found")
	}
	return propertyDataQuality(missing, warnings)
}

func propertyDataQuality(missing []string, warnings []string) PropertyDataQuality {
	total := 8.0
	completeness := (total - float64(len(missing))) / total
	if completeness < 0 {
		completeness = 0
	}
	return PropertyDataQuality{Completeness: completeness, MissingFields: missing, Warnings: warnings, SourceConflicts: []PropertyConflict{}}
}

func propertyFacets(rows []PropertySummary) propertyQueryFacets {
	facets := propertyQueryFacets{Sources: map[string]int{}, Kinds: map[string]int{}, Cities: map[string]int{}}
	for _, row := range rows {
		incrementFacet(facets.Sources, row.Source)
		incrementFacet(facets.Kinds, row.Kind)
		incrementFacet(facets.Cities, row.City)
		if len(row.Transactions) > 0 {
			facets.WithSales++
		}
		if row.Insights.Count > 0 {
			facets.WithInsights++
		}
	}
	return facets
}

func incrementFacet(values map[string]int, key string) {
	key = strings.TrimSpace(key)
	if key != "" {
		values[key]++
	}
}

func detailFieldsToItems(fields []propertyDetailField, limit int) []string {
	out := []string{}
	for _, field := range fields {
		if len(out) >= limit {
			break
		}
		if item := labelValue(field.Label, field.Value); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func labelValue(label string, value string) string {
	label = strings.TrimSpace(label)
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if label == "" {
		return value
	}
	return label + ": " + value
}

func compactStrings(values []string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func compactJoin(values []string, sep string) string {
	return strings.Join(compactStrings(values), sep)
}

func stringList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{strings.TrimSpace(value)}
}

func firstFloat64Ptr(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func coordinates(longitude *float64, latitude *float64) []float64 {
	if longitude == nil || latitude == nil {
		return nil
	}
	if *longitude < -180 || *longitude > 180 || *latitude < -90 || *latitude > 90 {
		return nil
	}
	return []float64{*longitude, *latitude}
}

func pricePerM2Float(price *int64, area *float64) *float64 {
	if price == nil || area == nil || *area <= 0 {
		return nil
	}
	value := float64(*price) / *area
	return &value
}

func int64PtrToInt32Ptr(value *int64) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func comparablePricePerM2Values(sales []ComparableSale) []float64 {
	values := []float64{}
	for _, sale := range sales {
		if sale.PricePerSquareMeter != nil {
			values = append(values, float64(*sale.PricePerSquareMeter))
			continue
		}
		if sale.Price != nil && sale.Area != nil && *sale.Area > 0 {
			values = append(values, float64(*sale.Price)/(*sale.Area))
		}
	}
	return values
}

func medianFloat64(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func marketHint(diff float64) string {
	if diff > 10 {
		return "over_market"
	}
	if diff < -10 {
		return "under_market"
	}
	return "near_market"
}

func sourceLabel(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "frontdoor":
		return "Frontdoor"
	case "shortcut":
		return "Shortcut"
	default:
		return strings.TrimSpace(source)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func formatInt64Value(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatInt32Value(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
}

func formatFloatValue(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func formatBoolValue(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "yes"
	}
	return "no"
}
