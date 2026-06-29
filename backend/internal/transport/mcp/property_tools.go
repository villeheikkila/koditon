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

type propertyQueryInput struct {
	Entity      string                 `json:"entity,omitempty" jsonschema:"Entity to query: property, listing, address, transaction, or all. Defaults to property/listing search."`
	Query       string                 `json:"query,omitempty" jsonschema:"Free text search across listing headlines, addresses, source native IDs, and transaction descriptions."`
	Address     string                 `json:"address,omitempty" jsonschema:"Exact or pasted address for address lookup mode."`
	Source      string                 `json:"source,omitempty" jsonschema:"Source filter: shortcut, frontdoor, or all."`
	Kind        string                 `json:"kind,omitempty" jsonschema:"Entity kind filter: ad, building, announcement, rental, or all."`
	ListingType string                 `json:"listing_type,omitempty" jsonschema:"Listing type filter: listing, rental, or all."`
	City        string                 `json:"city,omitempty" jsonschema:"Municipality or city filter."`
	Postal      string                 `json:"postal,omitempty" jsonschema:"Finnish postal code filter."`
	MinPrice    *int64                 `json:"min_price,omitempty" jsonschema:"Minimum asking or sale price in EUR."`
	MaxPrice    *int64                 `json:"max_price,omitempty" jsonschema:"Maximum asking or sale price in EUR."`
	MinArea     *float64               `json:"min_area,omitempty" jsonschema:"Minimum area in square meters."`
	MaxArea     *float64               `json:"max_area,omitempty" jsonschema:"Maximum area in square meters."`
	Sort        string                 `json:"sort,omitempty" jsonschema:"Sort mode such as seen_desc, price_asc, price_desc, area_asc, area_desc, newest, cheapest, or expensive."`
	Page        *int32                 `json:"page,omitempty" jsonschema:"One-based result page for listings."`
	PageSize    *int32                 `json:"page_size,omitempty" jsonschema:"Listing result count. Must be 25, 50, or 100 for search mode; address lookup is capped at 100."`
	Limit       *int32                 `json:"limit,omitempty" jsonschema:"Transaction result count, between 1 and 5000."`
	Include     propertyIncludeOptions `json:"include,omitempty" jsonschema:"Controls optional linked records and raw payloads."`
}

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

type propertyQueryResult struct {
	Summary      string                   `json:"summary"`
	Mode         string                   `json:"mode"`
	Entity       string                   `json:"entity"`
	Query        propertyQueryEcho        `json:"query"`
	Rows         []propertySummary        `json:"rows"`
	Transactions []propertyTransaction    `json:"transactions,omitempty"`
	Facets       propertyQueryFacets      `json:"facets,omitempty"`
	Diagnostics  propertyQueryDiagnostics `json:"diagnostics,omitempty"`
	WebURL       string                   `json:"web_url,omitempty"`
	Total        int64                    `json:"total"`
	Page         int32                    `json:"page"`
	PageSize     int32                    `json:"page_size"`
}

type propertyQueryEcho struct {
	Text        string                 `json:"text,omitempty"`
	Address     string                 `json:"address,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Kind        string                 `json:"kind,omitempty"`
	ListingType string                 `json:"listing_type,omitempty"`
	City        string                 `json:"city,omitempty"`
	Postal      string                 `json:"postal,omitempty"`
	MinPrice    *int64                 `json:"min_price,omitempty"`
	MaxPrice    *int64                 `json:"max_price,omitempty"`
	MinArea     *float64               `json:"min_area,omitempty"`
	MaxArea     *float64               `json:"max_area,omitempty"`
	Sort        string                 `json:"sort,omitempty"`
	Include     propertyIncludeOptions `json:"include,omitempty"`
}

type propertySummary struct {
	EntityID             string                  `json:"entity_id"`
	EntityType           string                  `json:"entity_type"`
	CanonicalID          string                  `json:"canonical_id,omitempty"`
	OfferingID           string                  `json:"offering_id,omitempty"`
	ListingID            string                  `json:"listing_id,omitempty"`
	NativeID             string                  `json:"native_id,omitempty"`
	Source               string                  `json:"source,omitempty"`
	Kind                 string                  `json:"kind,omitempty"`
	Title                string                  `json:"title"`
	Address              string                  `json:"address,omitempty"`
	City                 string                  `json:"city,omitempty"`
	Postal               string                  `json:"postal,omitempty"`
	Price                *int64                  `json:"price,omitempty"`
	Area                 *float64                `json:"area,omitempty"`
	RoomLayout           string                  `json:"room_layout,omitempty"`
	URL                  string                  `json:"url,omitempty"`
	ExternalURLAvailable bool                    `json:"external_url_available,omitempty"`
	WebURL               string                  `json:"web_url,omitempty"`
	LastSeenAt           *time.Time              `json:"last_seen_at,omitempty"`
	Links                propertyEntityLinks     `json:"links"`
	Match                propertyMatchSummary    `json:"match,omitempty"`
	Insights             propertyInsightSummary  `json:"insights,omitempty"`
	Transactions         []propertyTransaction   `json:"transactions,omitempty"`
	SourceRecords        []propertySourceSummary `json:"source_records,omitempty"`
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

type propertyTransaction struct {
	TransactionID       string     `json:"transaction_id,omitempty"`
	ID                  string     `json:"id,omitempty"`
	Description         string     `json:"description,omitempty"`
	Category            string     `json:"category,omitempty"`
	Type                string     `json:"type,omitempty"`
	Area                *float64   `json:"area,omitempty"`
	Price               *int64     `json:"price,omitempty"`
	PricePerSquareMeter *int64     `json:"price_per_square_meter,omitempty"`
	BuildYear           *int32     `json:"build_year,omitempty"`
	Floor               string     `json:"floor,omitempty"`
	Elevator            *bool      `json:"elevator,omitempty"`
	Condition           string     `json:"condition,omitempty"`
	Plot                string     `json:"plot,omitempty"`
	EnergyClass         string     `json:"energy_class,omitempty"`
	PeriodIdentifier    string     `json:"period_identifier,omitempty"`
	City                string     `json:"city,omitempty"`
	Neighborhood        string     `json:"neighborhood,omitempty"`
	Postal              string     `json:"postal,omitempty"`
	Confidence          string     `json:"confidence,omitempty"`
	LinkStatus          string     `json:"link_status,omitempty"`
	LinkMethod          string     `json:"link_method,omitempty"`
	Score               *int32     `json:"score,omitempty"`
	CreatedAt           *time.Time `json:"created_at,omitempty"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
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

type propertyDetailResult struct {
	Summary        string                     `json:"summary"`
	Canonical      propertyCanonicalFields    `json:"canonical"`
	CanonicalExtra []propertyDetailField      `json:"canonical_extra,omitempty"`
	SourceSpecific []propertyDetailField      `json:"source_specific,omitempty"`
	Related        []propertyDetailField      `json:"related,omitempty"`
	Normalized     ads.NormalizedDetailFields `json:"normalized"`
	Links          propertyEntityLinks        `json:"links"`
	Report         []propertyReportSection    `json:"report"`
	Markdown       string                     `json:"markdown"`
	Raw            *propertyRawPayload        `json:"raw,omitempty"`
	RawJSON        any                        `json:"raw_json,omitempty"`
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

func (t *toolImpl) queryProperties(ctx context.Context, _ *mcp.CallToolRequest, in propertyQueryInput) (*mcp.CallToolResult, *propertyQueryResult, error) {
	if in.MinPrice != nil && in.MaxPrice != nil && *in.MinPrice > *in.MaxPrice {
		return newToolResultError("min_price cannot be greater than max_price"), nil, nil
	}
	if in.MinArea != nil && in.MaxArea != nil && *in.MinArea > *in.MaxArea {
		return newToolResultError("min_area cannot be greater than max_area"), nil, nil
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

func (t *toolImpl) getPropertyDetail(ctx context.Context, _ *mcp.CallToolRequest, in propertyDetailInput) (*mcp.CallToolResult, *propertyDetailResult, error) {
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

func (t *toolImpl) buildPropertyListingQueryResult(ctx context.Context, in propertyQueryInput) (propertyQueryResult, string, error) {
	appResult, toolErr, err := t.buildListingsAppResult(ctx, findListingsInput{Query: in.Query, Address: in.Address, Source: in.Source, Kind: in.Kind, ListingType: in.ListingType, City: in.City, Postal: in.Postal, MinPrice: in.MinPrice, MaxPrice: in.MaxPrice, MinArea: in.MinArea, MaxArea: in.MaxArea, Sort: in.Sort, Page: in.Page, PageSize: in.PageSize, IncludePrices: in.Include.LinkedTransactions})
	if toolErr != "" || err != nil {
		return propertyQueryResult{}, toolErr, err
	}
	rows := make([]propertySummary, 0, len(appResult.Rows))
	for _, row := range appResult.Rows {
		rows = append(rows, t.propertySummaryFromListingAppRow(row))
	}
	result := propertyQueryResult{Summary: propertyQuerySummary(appResult.Mode, len(rows), len(appResult.Transactions)), Mode: appResult.Mode, Entity: "property", Query: propertyQueryEcho{Text: strings.TrimSpace(in.Query), Address: strings.TrimSpace(in.Address), Source: normalizeAppFilter(in.Source), Kind: normalizeAppFilter(in.Kind), ListingType: normalizeAppFilter(in.ListingType), City: strings.TrimSpace(in.City), Postal: strings.TrimSpace(in.Postal), MinPrice: in.MinPrice, MaxPrice: in.MaxPrice, MinArea: in.MinArea, MaxArea: in.MaxArea, Sort: strings.TrimSpace(in.Sort), Include: in.Include}, Rows: rows, Transactions: propertyTransactionsFromAppRows(appResult.Transactions), Facets: propertyFacets(rows), Diagnostics: propertyQueryDiagnostics{ReturnedTransactions: len(appResult.Transactions)}, WebURL: appResult.WebURL, Total: appResult.Total, Page: appResult.Page, PageSize: appResult.PageSize}
	return result, "", nil
}

func (t *toolImpl) buildPropertyTransactionQueryResult(ctx context.Context, in propertyQueryInput) (propertyQueryResult, string, error) {
	limit := int32(200)
	if in.Limit != nil {
		if *in.Limit < 1 || *in.Limit > 5000 {
			return propertyQueryResult{}, "limit must be between 1 and 5000", nil
		}
		limit = *in.Limit
	}
	rows, err := t.runSearchTransactionsAdvanced(ctx, transactionsAdvancedParams{City: in.City, Query: in.Query, PostalCodes: stringList(in.Postal), MinPrice: int64PtrToInt32Ptr(in.MinPrice), MaxPrice: int64PtrToInt32Ptr(in.MaxPrice), MinArea: in.MinArea, MaxArea: in.MaxArea, Sort: in.Sort, Limit: limit})
	if err != nil {
		return propertyQueryResult{}, "", fmt.Errorf("query property transactions: %w", err)
	}
	transactions := make([]propertyTransaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, propertyTransactionFromAdvancedRow(row))
	}
	result := propertyQueryResult{Summary: fmt.Sprintf("Koditon transaction query returned %d sale price rows.", len(transactions)), Mode: "transaction", Entity: "transaction", Query: propertyQueryEcho{Text: strings.TrimSpace(in.Query), City: strings.TrimSpace(in.City), Postal: strings.TrimSpace(in.Postal), MinPrice: in.MinPrice, MaxPrice: in.MaxPrice, MinArea: in.MinArea, MaxArea: in.MaxArea, Sort: strings.TrimSpace(in.Sort), Include: in.Include}, Rows: []propertySummary{}, Transactions: transactions, Diagnostics: propertyQueryDiagnostics{ReturnedTransactions: len(transactions)}, WebURL: t.webPath("/prices"), Total: int64(len(transactions)), Page: 1, PageSize: limit}
	return result, "", nil
}

func (t *toolImpl) propertySummaryFromListingAppRow(row listingAppRow) propertySummary {
	transactions := propertyTransactionsFromAppRows(row.Transactions)
	return propertySummary{EntityID: row.CanonicalID, EntityType: "listing", CanonicalID: row.CanonicalID, Source: row.Source, Kind: row.Kind, Title: firstNonEmpty(row.Title, row.Address, row.CanonicalID), Address: row.Address, City: row.City, Postal: row.Postal, Price: row.Price, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, WebURL: row.WebURL, Links: propertyEntityLinks{Web: row.WebURL, Source: row.URL, DetailID: row.CanonicalID}, Transactions: transactions}
}

func propertyTransactionsFromAppRows(rows []priceTransactionAppRow) []propertyTransaction {
	out := make([]propertyTransaction, 0, len(rows))
	for _, row := range rows {
		out = append(out, propertyTransaction{TransactionID: firstNonEmpty(row.TransactionID, row.ID), ID: firstNonEmpty(row.ID, row.TransactionID), Description: row.Description, Category: row.Category, Type: row.Type, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Postal: row.Postal, Confidence: row.Confidence})
	}
	return out
}

func propertyTransactionFromAdvancedRow(row transactionsAdvancedRow) propertyTransaction {
	price := int64(row.Price)
	priceM2 := int64(row.PricePerSquareMeter)
	buildYear := row.BuildYear
	floor := ""
	if row.Floor != nil {
		floor = *row.Floor
	}
	return propertyTransaction{TransactionID: row.TransactionID.String(), ID: row.TransactionID.String(), Description: row.Description, Category: row.Category, Type: row.Type, Area: &row.Area, Price: &price, PricePerSquareMeter: &priceM2, BuildYear: &buildYear, Floor: floor, Elevator: &row.Elevator, Condition: stringPtrValue(row.Condition), Plot: stringPtrValue(row.Plot), EnergyClass: stringPtrValue(row.EnergyClass), PeriodIdentifier: row.PeriodIdentifier, City: row.City, Neighborhood: row.Neighborhood, Postal: row.PostalCode, CreatedAt: &row.CreatedAt, UpdatedAt: &row.UpdatedAt}
}

func (t *toolImpl) buildPropertyDetailResult(detail ads.UnifiedEntityDetail, includeRaw bool, includeRawJSON bool) propertyDetailResult {
	webURL := t.listingWebURL(detail.Canonical.CanonicalID, detail.Canonical.Kind, "")
	result := propertyDetailResult{Canonical: propertyCanonicalFromDomain(detail.Canonical), CanonicalExtra: propertyDetailFieldsFromDomain(detail.CanonicalExtra), SourceSpecific: propertyDetailFieldsFromDomain(detail.SourceSpecific), Related: propertyDetailFieldsFromDomain(detail.Related), Normalized: detail.Normalized, Links: propertyEntityLinks{Web: webURL, Source: detail.Canonical.URL, DetailID: detail.Canonical.CanonicalID}}
	result.Report = propertyDetailReport(result)
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
	return propertyCanonicalFields{CanonicalID: value.CanonicalID, Source: value.Source, Kind: value.Kind, NativeID: value.NativeID, Headline: value.Headline, Address: value.Address, City: value.City, Postal: value.Postal, Price: value.Price, Area: value.Area, RoomLayout: value.RoomLayout, URL: value.URL, ExternalURLAvailable: value.ExternalURLAvailable, LastSeenAt: &lastSeenAt}
}

func propertyDetailFieldsFromDomain(fields []ads.DetailField) []propertyDetailField {
	out := make([]propertyDetailField, 0, len(fields))
	for _, field := range fields {
		out = append(out, propertyDetailField{Label: field.Label, Value: field.Value})
	}
	return out
}

func propertyDetailReport(result propertyDetailResult) []propertyReportSection {
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

func propertyDetailSummary(result propertyDetailResult) string {
	title := firstNonEmpty(result.Canonical.Headline, result.Canonical.Address, result.Canonical.CanonicalID)
	return fmt.Sprintf("Koditon property detail for %s (%s/%s).", title, result.Canonical.Source, result.Canonical.Kind)
}

func propertyDetailMarkdown(result propertyDetailResult) string {
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

func propertyFacets(rows []propertySummary) propertyQueryFacets {
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

func stringList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{strings.TrimSpace(value)}
}

func int64PtrToInt32Ptr(value *int64) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
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
