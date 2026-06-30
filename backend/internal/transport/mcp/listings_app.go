package mcpserver

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"koditon/internal/domain/ads"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	listingsAppResourceURI  = "ui://koditon/listings.html"
	listingsAppResourceMime = "text/html;profile=mcp-app"
)

//go:embed appdist/mcp-app.html
var listingsAppHTML string

type findListingsInput struct {
	Query         string   `json:"query,omitempty" jsonschema:"Free text search across headlines, addresses, source native IDs, and listing facts."`
	Address       string   `json:"address,omitempty" jsonschema:"Exact or pasted address for address lookup mode. Supports values like 'Askvägen 4, 22100 Maarianhamina'."`
	Source        string   `json:"source,omitempty" jsonschema:"Source filter: shortcut, frontdoor, or all."`
	Kind          string   `json:"kind,omitempty" jsonschema:"Entity kind filter: ad, building, announcement, or all."`
	ListingType   string   `json:"listing_type,omitempty" jsonschema:"Listing type filter: listing, rental, or all."`
	City          string   `json:"city,omitempty" jsonschema:"Municipality/city filter."`
	Postal        string   `json:"postal,omitempty" jsonschema:"Finnish postal code filter."`
	MinPrice      *int64   `json:"min_price,omitempty" jsonschema:"Minimum asking price in EUR."`
	MaxPrice      *int64   `json:"max_price,omitempty" jsonschema:"Maximum asking price in EUR."`
	MinArea       *float64 `json:"min_area,omitempty" jsonschema:"Minimum area in square meters."`
	MaxArea       *float64 `json:"max_area,omitempty" jsonschema:"Maximum area in square meters."`
	Sort          string   `json:"sort,omitempty" jsonschema:"Sort mode such as seen_desc, price_asc, price_desc, area_asc, or area_desc."`
	Page          *int32   `json:"page,omitempty" jsonschema:"One-based result page."`
	PageSize      *int32   `json:"page_size,omitempty" jsonschema:"Result count. Must be 25, 50, or 100 for search mode; address lookup is capped at 100."`
	IncludePrices bool     `json:"include_prices,omitempty" jsonschema:"Include linked actual sale prices when available."`
}

type listingsAppResult struct {
	Mode         string                   `json:"mode"`
	WebURL       string                   `json:"web_url"`
	Rows         []listingAppRow          `json:"rows"`
	Transactions []priceTransactionAppRow `json:"transactions"`
	Total        int64                    `json:"total"`
	Page         int32                    `json:"page"`
	PageSize     int32                    `json:"page_size"`
	Summary      string                   `json:"summary"`
}

type listingAppRow struct {
	CanonicalID          string                   `json:"canonical_id"`
	NativeID             string                   `json:"native_id,omitempty"`
	ListingID            string                   `json:"listing_id,omitempty"`
	OfferingID           string                   `json:"offering_id,omitempty"`
	GroupingID           string                   `json:"grouping_id,omitempty"`
	Source               string                   `json:"source"`
	Kind                 string                   `json:"kind"`
	Title                string                   `json:"title"`
	Address              string                   `json:"address,omitempty"`
	City                 string                   `json:"city,omitempty"`
	Postal               string                   `json:"postal,omitempty"`
	Latitude             *float64                 `json:"latitude,omitempty"`
	Longitude            *float64                 `json:"longitude,omitempty"`
	Price                *int64                   `json:"price,omitempty"`
	Area                 *float64                 `json:"area,omitempty"`
	RoomLayout           string                   `json:"room_layout,omitempty"`
	URL                  string                   `json:"url,omitempty"`
	ExternalURLAvailable bool                     `json:"external_url_available,omitempty"`
	WebURL               string                   `json:"web_url"`
	FirstSeenAt          *time.Time               `json:"first_seen_at,omitempty"`
	LastSeenAt           *time.Time               `json:"last_seen_at,omitempty"`
	PriceChanged         *bool                    `json:"price_changed,omitempty"`
	MatchStatus          string                   `json:"match_status,omitempty"`
	MatchMethod          string                   `json:"match_method,omitempty"`
	MatchScore           *int32                   `json:"match_score,omitempty"`
	InsightCount         int32                    `json:"insight_count,omitempty"`
	InsightTopSeverity   string                   `json:"insight_top_severity,omitempty"`
	Transactions         []priceTransactionAppRow `json:"transactions"`
}

type priceTransactionAppRow struct {
	TransactionID       string   `json:"transaction_id,omitempty"`
	ID                  string   `json:"id,omitempty"`
	Description         string   `json:"description,omitempty"`
	Category            string   `json:"category,omitempty"`
	Type                string   `json:"type,omitempty"`
	Area                *float64 `json:"area,omitempty"`
	Price               *int64   `json:"price,omitempty"`
	PricePerSquareMeter *int64   `json:"price_per_square_meter,omitempty"`
	PeriodIdentifier    string   `json:"period_identifier,omitempty"`
	City                string   `json:"city,omitempty"`
	Postal              string   `json:"postal,omitempty"`
	Confidence          string   `json:"confidence,omitempty"`
}

func (t *toolImpl) findListingsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_find_listings",
		Title:       "Find Listings",
		Description: "Search Koditon listings with precise filters and render them in an MCP App. Use address for exact address lookup with linked sale prices, or query/filter fields for broader listing search.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Find Listings",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
		Meta: mcp.Meta{"ui": map[string]any{"resourceUri": listingsAppResourceURI}},
	}
}

func (t *toolImpl) findListings(ctx context.Context, _ *mcp.CallToolRequest, in findListingsInput) (*mcp.CallToolResult, *listingsAppResult, error) {
	result, toolErr, err := t.buildListingsAppResult(ctx, in)
	if toolErr != "" {
		return newToolResultError(toolErr), nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: result.Summary}}}, &result, nil
}

func (t *toolImpl) buildListingsAppResult(ctx context.Context, in findListingsInput) (listingsAppResult, string, error) {
	if in.MinPrice != nil && in.MaxPrice != nil && *in.MinPrice > *in.MaxPrice {
		return listingsAppResult{}, "min_price cannot be greater than max_price", nil
	}
	if in.MinArea != nil && in.MaxArea != nil && *in.MinArea > *in.MaxArea {
		return listingsAppResult{}, "min_area cannot be greater than max_area", nil
	}
	if strings.TrimSpace(in.Address) != "" {
		return t.buildAddressListingsAppResult(ctx, in)
	}
	return t.buildSearchListingsAppResult(ctx, in)
}

func (t *toolImpl) buildSearchListingsAppResult(ctx context.Context, in findListingsInput) (listingsAppResult, string, error) {
	params := ads.SearchParams{Query: strings.TrimSpace(in.Query), Source: normalizeAppFilter(in.Source), Kind: normalizeAppFilter(in.Kind), ListingType: normalizeAppFilter(in.ListingType), City: strings.TrimSpace(in.City), Postal: strings.TrimSpace(in.Postal), Sort: strings.TrimSpace(in.Sort), MinPrice: in.MinPrice, MaxPrice: in.MaxPrice, MinArea: in.MinArea, MaxArea: in.MaxArea}
	if in.Page != nil {
		params.Page = *in.Page
	}
	if in.PageSize != nil {
		if *in.PageSize != 25 && *in.PageSize != 50 && *in.PageSize != 100 {
			return listingsAppResult{}, "page_size must be one of: 25, 50, 100", nil
		}
		params.PageSize = *in.PageSize
	}
	page, err := t.adsSvc.Search(ctx, params)
	if err != nil {
		return listingsAppResult{}, "", fmt.Errorf("search listings for app: %w", err)
	}
	rows := make([]listingAppRow, 0, len(page.Rows))
	transactions := []priceTransactionAppRow{}
	for _, row := range page.Rows {
		mapped := t.mapUnifiedListingAppRow(row, in.IncludePrices)
		rows = append(rows, mapped)
		transactions = append(transactions, mapped.Transactions...)
	}
	result := listingsAppResult{Mode: "search", WebURL: t.searchWebURL(in), Rows: rows, Transactions: dedupePriceTransactionAppRows(transactions), Total: page.Total, Page: page.Page, PageSize: page.PageSize}
	result.Summary = listingsAppSummary(result)
	return result, "", nil
}

func (t *toolImpl) buildAddressListingsAppResult(ctx context.Context, in findListingsInput) (listingsAppResult, string, error) {
	pageSize := int32(50)
	if in.PageSize != nil {
		if *in.PageSize < 1 || *in.PageSize > 100 {
			return listingsAppResult{}, "page_size must be between 1 and 100 for address lookup", nil
		}
		pageSize = *in.PageSize
	}
	result, err := t.adsSvc.LookupAddress(ctx, ads.AddressLookupParams{Address: strings.TrimSpace(in.Address), City: strings.TrimSpace(in.City), Postal: strings.TrimSpace(in.Postal), Source: normalizeAppFilter(in.Source), PageSize: pageSize})
	if err != nil {
		return listingsAppResult{}, "", fmt.Errorf("lookup address for app: %w", err)
	}
	rows := make([]listingAppRow, 0, len(result.Listings))
	transactions := make([]priceTransactionAppRow, 0, len(result.RawTransactions))
	for _, row := range result.Listings {
		mapped := t.mapAddressListingAppRow(row)
		rows = append(rows, mapped)
		transactions = append(transactions, mapped.Transactions...)
	}
	for _, row := range result.RawTransactions {
		transactions = append(transactions, mapRawAddressTransactionAppRow(row))
	}
	out := listingsAppResult{Mode: "address", WebURL: t.addressWebURL(in), Rows: rows, Transactions: dedupePriceTransactionAppRows(transactions), Total: int64(result.ListingCount), Page: 1, PageSize: pageSize}
	out.Summary = listingsAppSummary(out)
	return out, "", nil
}

func (t *toolImpl) mapUnifiedListingAppRow(row ads.UnifiedEntityRow, includePrices bool) listingAppRow {
	transactions := []priceTransactionAppRow{}
	if includePrices && strings.TrimSpace(row.PriceMatchTransactionID) != "" {
		transactions = append(transactions, priceTransactionAppRow{TransactionID: strings.TrimSpace(row.PriceMatchTransactionID), ID: strings.TrimSpace(row.PriceMatchTransactionID), Price: row.PriceMatchPrice, Confidence: firstNonEmpty(row.PriceMatchStatus, row.PriceMatchScope, row.PriceMatchMethod)})
	}
	return listingAppRow{CanonicalID: row.CanonicalID, NativeID: row.NativeID, ListingID: row.ListingID, OfferingID: row.OfferingID, GroupingID: firstNonEmpty(row.HousingCompanyID, row.HousingCompanyName), Source: row.Source, Kind: row.Kind, Title: firstNonEmpty(row.Headline, row.Address, row.CanonicalID), Address: row.Address, City: row.City, Postal: row.Postal, Latitude: row.Latitude, Longitude: row.Longitude, Price: row.Price, Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, WebURL: t.listingWebURL(row.CanonicalID, row.Kind, row.OfferingID), LastSeenAt: &row.LastSeenAt, MatchStatus: firstNonEmpty(row.LinkStatus, row.PriceMatchStatus), MatchMethod: firstNonEmpty(row.LinkMethod, row.PriceMatchMethod), MatchScore: firstInt32Ptr(row.LinkScore, row.PriceMatchScore), InsightCount: row.InsightCount, InsightTopSeverity: row.InsightTopSeverity, Transactions: transactions}
}

func (t *toolImpl) mapAddressListingAppRow(row ads.AddressListing) listingAppRow {
	transactions := make([]priceTransactionAppRow, 0, len(row.Transactions))
	for _, transaction := range row.Transactions {
		transactions = append(transactions, mapLinkedAddressTransactionAppRow(transaction))
	}
	priceChanged := boolPtr(row.PreviousAskingPrice != nil || row.PreviousDebtFreePrice != nil)
	return listingAppRow{CanonicalID: row.CanonicalID, NativeID: row.NativeID, ListingID: row.ListingID, OfferingID: row.OfferingID, GroupingID: firstNonEmpty(row.HousingCompanyID, row.HousingCompanyName), Source: row.Source, Kind: row.Kind, Title: firstNonEmpty(row.Headline, row.Address, row.CanonicalID), Address: row.Address, City: row.City, Postal: row.Postal, Latitude: row.Latitude, Longitude: row.Longitude, Price: firstInt64Ptr(row.AskingPrice, row.DebtFreePrice), Area: row.Area, RoomLayout: row.RoomLayout, URL: row.URL, ExternalURLAvailable: row.ExternalURLAvailable, WebURL: t.listingWebURL(row.CanonicalID, row.Kind, row.OfferingID), FirstSeenAt: row.FirstSeenAt, LastSeenAt: row.LastSeenAt, PriceChanged: priceChanged, MatchStatus: firstNonEmpty(row.SourceMatchStatus, row.PriceMatchStatus), InsightCount: int32(len(row.Insights)), Transactions: transactions}
}

func mapLinkedAddressTransactionAppRow(row ads.AddressTransactionLink) priceTransactionAppRow {
	return priceTransactionAppRow{TransactionID: row.TransactionID, ID: row.TransactionID, Description: row.Description, Category: row.Category, Type: row.Type, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Postal: row.Postal, Confidence: firstNonEmpty(row.Confidence, row.LinkStatus, row.LinkMethod)}
}

func mapRawAddressTransactionAppRow(row ads.AddressRawTransaction) priceTransactionAppRow {
	return priceTransactionAppRow{TransactionID: row.TransactionID, ID: row.TransactionID, Description: row.Description, Category: row.Category, Type: row.Type, Area: row.Area, Price: row.Price, PricePerSquareMeter: row.PricePerSquareMeter, PeriodIdentifier: row.PeriodIdentifier, City: row.City, Postal: row.Postal}
}

func (t *toolImpl) listingWebURL(canonicalID, kind, offeringID string) string {
	if strings.TrimSpace(offeringID) != "" {
		return t.webPath("/target/offering/" + url.PathEscape(strings.TrimSpace(offeringID)))
	}
	route := sourceEntityRoute(kind)
	if strings.TrimSpace(canonicalID) == "" {
		return t.webPath("/search")
	}
	return t.webPath("/" + route + "/" + url.PathEscape(strings.TrimSpace(canonicalID)))
}

func (t *toolImpl) searchWebURL(in findListingsInput) string {
	params := url.Values{}
	setQueryValue(params, "q", in.Query)
	setQueryValue(params, "source", omitAllFilter(in.Source))
	setQueryValue(params, "kind", omitAllFilter(in.Kind))
	setQueryValue(params, "listing_type", omitAllFilter(in.ListingType))
	setQueryValue(params, "city", in.City)
	setQueryValue(params, "postal", in.Postal)
	setQueryValue(params, "sort", in.Sort)
	setInt64QueryValue(params, "min_price", in.MinPrice)
	setInt64QueryValue(params, "max_price", in.MaxPrice)
	setFloatQueryValue(params, "min_area", in.MinArea)
	setFloatQueryValue(params, "max_area", in.MaxArea)
	path := "/search"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return t.webPath(path)
}

func (t *toolImpl) addressWebURL(in findListingsInput) string {
	params := url.Values{}
	setQueryValue(params, "address", in.Address)
	setQueryValue(params, "city", in.City)
	setQueryValue(params, "postal", in.Postal)
	setQueryValue(params, "source", omitAllFilter(in.Source))
	if in.PageSize != nil && *in.PageSize != 50 {
		params.Set("page_size", strconv.FormatInt(int64(*in.PageSize), 10))
	}
	path := "/address"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return t.webPath(path)
}

func (t *toolImpl) webPath(path string) string {
	base := strings.TrimRight(strings.TrimSpace(t.config.webBaseURL), "/")
	if base == "" {
		return path
	}
	return base + path
}

func registerListingsAppResource(server *mcp.Server, apiBaseURL, webBaseURL string) {
	server.AddResource(&mcp.Resource{URI: listingsAppResourceURI, Name: "koditon-listings-app", Title: "Koditon Listings", Description: "Interactive listing search results for Koditon MCP tools.", MIMEType: listingsAppResourceMime}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: listingsAppResourceURI, MIMEType: listingsAppResourceMime, Text: listingsAppHTML, Meta: mcp.Meta{"ui": map[string]any{"csp": listingsAppCSP(apiBaseURL, webBaseURL), "prefersBorder": true}}}}}, nil
	})
}

func listingsAppCSP(apiBaseURL, webBaseURL string) map[string]any {
	domains := uniqueOrigins(apiBaseURL, webBaseURL)
	return map[string]any{"connectDomains": domains, "resourceDomains": domains}
}

func uniqueOrigins(values ...string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		origin := originFromURL(value)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		out = append(out, origin)
	}
	return out
}

func originFromURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func listingsAppSummary(result listingsAppResult) string {
	mode := "listing search"
	if result.Mode == "address" {
		mode = "address lookup"
	}
	return fmt.Sprintf("Koditon %s returned %d listing rows and %d linked sale prices. Open the rendered MCP App to inspect listings and jump to the web UI.", mode, len(result.Rows), len(result.Transactions))
}

func dedupePriceTransactionAppRows(rows []priceTransactionAppRow) []priceTransactionAppRow {
	seen := map[string]struct{}{}
	out := make([]priceTransactionAppRow, 0, len(rows))
	for _, row := range rows {
		key := firstNonEmpty(row.TransactionID, row.ID, row.Description)
		if key == "" {
			out = append(out, row)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, row)
	}
	return out
}

func sourceEntityRoute(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "rental":
		return "rental"
	case "announcement", "building":
		return "housing-company"
	default:
		return "listing"
	}
}

func normalizeAppFilter(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "all" {
		return ""
	}
	return trimmed
}

func omitAllFilter(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "all") {
		return ""
	}
	return trimmed
}

func setQueryValue(params url.Values, key, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		params.Set(key, trimmed)
	}
}

func setInt64QueryValue(params url.Values, key string, value *int64) {
	if value != nil {
		params.Set(key, strconv.FormatInt(*value, 10))
	}
}

func setFloatQueryValue(params url.Values, key string, value *float64) {
	if value != nil {
		params.Set(key, strconv.FormatFloat(*value, 'f', -1, 64))
	}
}

func firstInt64Ptr(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstInt32Ptr(values ...*int32) *int32 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}
