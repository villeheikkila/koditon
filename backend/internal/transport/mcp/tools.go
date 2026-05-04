package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"koditon/internal/db"
	"koditon/internal/domain/ads"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- input/output types --------------------------------------------------------

type searchListingsInput struct {
	Query       string   `json:"query,omitempty"`
	Address     string   `json:"address,omitempty"`
	Source      string   `json:"source,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	ListingType string   `json:"listing_type,omitempty"`
	City        string   `json:"city,omitempty"`
	Postal      string   `json:"postal,omitempty"`
	MinPrice    *int64   `json:"min_price,omitempty"`
	MaxPrice    *int64   `json:"max_price,omitempty"`
	MinArea     *float64 `json:"min_area,omitempty"`
	MaxArea     *float64 `json:"max_area,omitempty"`
	Sort        string   `json:"sort,omitempty"`
	Page        *int32   `json:"page,omitempty"`
	PageSize    *int32   `json:"page_size,omitempty"`
}

type getListingDetailInput struct {
	ID             string `json:"id,omitempty"`
	Input          string `json:"input,omitempty"`
	Source         string `json:"source,omitempty"`
	Kind           string `json:"kind,omitempty"`
	City           string `json:"city,omitempty"`
	Postal         string `json:"postal,omitempty"`
	MaxCandidates  *int32 `json:"max_candidates,omitempty"`
	IncludeRawJSON bool   `json:"include_raw_json,omitempty"`
}

type searchTransactionsInput struct {
	City    string `json:"city"`
	Address string `json:"address,omitempty"`
	Limit   *int32 `json:"limit,omitempty"`
}

type searchTransactionsAdvancedInput struct {
	City            string   `json:"city,omitempty"`
	Query           string   `json:"query,omitempty"`
	MunicipalityIDs []string `json:"municipality_ids,omitempty"`
	PostalCodeIDs   []string `json:"postal_code_ids,omitempty"`
	PostalCodes     []string `json:"postal_codes,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Types           []string `json:"types,omitempty"`
	MinPrice        *int32   `json:"min_price,omitempty"`
	MaxPrice        *int32   `json:"max_price,omitempty"`
	MinArea         *float64 `json:"min_area,omitempty"`
	MaxArea         *float64 `json:"max_area,omitempty"`
	Sort            string   `json:"sort,omitempty"`
	Limit           *int32   `json:"limit,omitempty"`
}

type matchAdsFromTransactionInput struct {
	TransactionID     string   `json:"transaction_id,omitempty"`
	City              string   `json:"city,omitempty"`
	PostalCode        string   `json:"postal_code,omitempty"`
	Area              *float64 `json:"area,omitempty"`
	Price             *int64   `json:"price,omitempty"`
	RoomHint          string   `json:"room_hint,omitempty"`
	Query             string   `json:"query,omitempty"`
	Source            string   `json:"source,omitempty"`
	Kind              string   `json:"kind,omitempty"`
	ListingType       string   `json:"listing_type,omitempty"`
	AreaTolerance     *float64 `json:"area_tolerance,omitempty"`
	PriceTolerancePct *float64 `json:"price_tolerance_pct,omitempty"`
	MaxCandidates     *int32   `json:"max_candidates,omitempty"`
	MaxResults        *int32   `json:"max_results,omitempty"`
}

type emptyInput struct{}

// ---- tool definitions ----------------------------------------------------------

func (t *toolImpl) searchListingsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_search_listings",
		Title:       "Search Listings",
		Description: "Search Finnish property listings (ads, buildings, announcements) with filters. Returns matching listings with address, price, area, and canonical IDs for further lookup.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Search Listings",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) getListingDetailTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_get_listing_detail",
		Title:       "Get Listing Detail",
		Description: "Get full listing detail by canonical ID, URL, or address text. Returns normalized fields for valuation and optional raw_json payload.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Get Listing Detail",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) searchTransactionsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_search_transactions",
		Title:       "Search Transactions",
		Description: "Search Finnish property price transactions by city and optional address.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Search Transactions",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) searchTransactionsAdvancedTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_search_transactions_advanced",
		Title:       "Search Transactions (Advanced)",
		Description: "Advanced Finnish property price transaction search with exact filters and flexible free-text. Supports filtering by municipality, postal code, category, type, price range, and area range.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Search Transactions (Advanced)",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) matchAdsFromTransactionTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_match_ads_from_transaction",
		Title:       "Match Listings from Transaction",
		Description: "Find matching Koditon listings for a price transaction using postal code, area, and room-layout hints. Useful for cross-referencing sold prices with current listings.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Match Listings from Transaction",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) listCitiesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_list_cities",
		Title:       "List Cities",
		Description: "List all Finnish municipalities with their postal codes.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "List Cities",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) listAvailableLocationsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_list_available_locations",
		Title:       "List Available Locations",
		Description: "List municipalities that have property price transaction data.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "List Available Locations",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

func (t *toolImpl) listCategoriesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "koditon_list_categories",
		Title:       "List Categories",
		Description: "List distinct property building categories (e.g. Kerrostalo, Rivitalo, Omakotitalo).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "List Categories",
			ReadOnlyHint:    true,
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}
}

// ---- typed tool handlers -------------------------------------------------------

func (t *toolImpl) searchListings(ctx context.Context, _ *mcp.CallToolRequest, in searchListingsInput) (*mcp.CallToolResult, struct{}, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		query = strings.TrimSpace(in.Address)
	}
	params := ads.SearchParams{
		Query:       query,
		Source:      strings.TrimSpace(in.Source),
		Kind:        strings.TrimSpace(in.Kind),
		ListingType: strings.TrimSpace(in.ListingType),
		City:        strings.TrimSpace(in.City),
		Postal:      strings.TrimSpace(in.Postal),
		Sort:        strings.TrimSpace(in.Sort),
	}
	if in.MinPrice != nil && in.MaxPrice != nil && *in.MinPrice > *in.MaxPrice {
		return newToolResultError("min_price cannot be greater than max_price"), struct{}{}, nil
	}
	if in.MinArea != nil && in.MaxArea != nil && *in.MinArea > *in.MaxArea {
		return newToolResultError("min_area cannot be greater than max_area"), struct{}{}, nil
	}
	params.MinPrice = in.MinPrice
	params.MaxPrice = in.MaxPrice
	params.MinArea = in.MinArea
	params.MaxArea = in.MaxArea
	if in.Page != nil {
		params.Page = *in.Page
	}
	if in.PageSize != nil {
		if *in.PageSize != 25 && *in.PageSize != 50 && *in.PageSize != 100 {
			return newToolResultError("page_size must be one of: 25, 50, 100"), struct{}{}, nil
		}
		params.PageSize = *in.PageSize
	}
	result, err := t.adsSvc.Search(ctx, params)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("search listings: %w", err)
	}
	res, err := jsonResult(result)
	return res, struct{}{}, err
}

func (t *toolImpl) getListingDetail(ctx context.Context, _ *mcp.CallToolRequest, in getListingDetailInput) (*mcp.CallToolResult, struct{}, error) {
	input := strings.TrimSpace(in.ID)
	if input == "" {
		input = strings.TrimSpace(in.Input)
	}
	if input == "" {
		return newToolResultError("id or input is required"), struct{}{}, nil
	}
	canonicalID, err := t.resolveDetailInput(ctx, input, in)
	if err != nil {
		return newToolResultError(fmt.Sprintf("resolve input: %v", err)), struct{}{}, nil
	}
	detail, err := t.adsSvc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return newToolResultError(fmt.Sprintf("listing not found: %s", canonicalID)), struct{}{}, nil
		}
		return nil, struct{}{}, fmt.Errorf("get listing detail: %w", err)
	}
	res, err := jsonResult(buildDetailResult(detail, in.IncludeRawJSON))
	return res, struct{}{}, err
}

func (t *toolImpl) searchTransactions(ctx context.Context, _ *mcp.CallToolRequest, in searchTransactionsInput) (*mcp.CallToolResult, struct{}, error) {
	city := strings.TrimSpace(in.City)
	if city == "" {
		return newToolResultError("city is required"), struct{}{}, nil
	}
	var limit int32 = 200
	if in.Limit != nil {
		if *in.Limit < 1 {
			return newToolResultError("limit must be >= 1"), struct{}{}, nil
		}
		limit = *in.Limit
	}
	rows, err := t.queries.SearchTransactionsByCityAndAddress(ctx, db.SearchTransactionsByCityAndAddressParams{
		CityName:   city,
		SearchTerm: strings.TrimSpace(in.Address),
		LimitCount: &limit,
	})
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("search transactions: %w", err)
	}
	res, err := jsonResult(rows)
	return res, struct{}{}, err
}

func (t *toolImpl) searchTransactionsAdvanced(ctx context.Context, _ *mcp.CallToolRequest, in searchTransactionsAdvancedInput) (*mcp.CallToolResult, struct{}, error) {
	if in.MinPrice != nil && in.MaxPrice != nil && *in.MinPrice > *in.MaxPrice {
		return newToolResultError("min_price cannot be greater than max_price"), struct{}{}, nil
	}
	if in.MinArea != nil && in.MaxArea != nil && *in.MinArea > *in.MaxArea {
		return newToolResultError("min_area cannot be greater than max_area"), struct{}{}, nil
	}
	limit := int32(200)
	if in.Limit != nil {
		if *in.Limit < 1 || *in.Limit > 5000 {
			return newToolResultError("limit must be between 1 and 5000"), struct{}{}, nil
		}
		limit = *in.Limit
	}
	municipalityIDs, err := parseUUIDs(in.MunicipalityIDs, "municipality_ids")
	if err != nil {
		return newToolResultError(err.Error()), struct{}{}, nil
	}
	postalCodeIDs, err := parseUUIDs(in.PostalCodeIDs, "postal_code_ids")
	if err != nil {
		return newToolResultError(err.Error()), struct{}{}, nil
	}
	sortMode := normalizeTransactionSort(in.Sort)
	rows, err := t.runSearchTransactionsAdvanced(ctx, transactionsAdvancedParams{
		City:            in.City,
		Query:           in.Query,
		MunicipalityIDs: municipalityIDs,
		PostalCodeIDs:   postalCodeIDs,
		PostalCodes:     in.PostalCodes,
		Categories:      in.Categories,
		Types:           in.Types,
		MinPrice:        in.MinPrice,
		MaxPrice:        in.MaxPrice,
		MinArea:         in.MinArea,
		MaxArea:         in.MaxArea,
		Sort:            sortMode,
		Limit:           limit,
	})
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("search transactions advanced: %w", err)
	}
	res, err := jsonResult(map[string]any{
		"summary": map[string]any{"count": len(rows), "sort": sortMode, "limit": limit},
		"rows":    rows,
	})
	return res, struct{}{}, err
}

func (t *toolImpl) matchAdsFromTransaction(ctx context.Context, _ *mcp.CallToolRequest, in matchAdsFromTransactionInput) (*mcp.CallToolResult, struct{}, error) {
	criteria := adsMatchCriteria{
		City:              strings.TrimSpace(in.City),
		PostalCode:        strings.TrimSpace(in.PostalCode),
		RoomHint:          strings.TrimSpace(in.RoomHint),
		Query:             strings.TrimSpace(in.Query),
		Source:            strings.TrimSpace(in.Source),
		Kind:              strings.TrimSpace(in.Kind),
		ListingType:       strings.TrimSpace(in.ListingType),
		AreaTolerance:     8.0,
		PriceTolerancePct: 0.35,
		AppliedPageSize:   100,
		AppliedMaxResults: 20,
	}
	criteria.Area = in.Area
	criteria.Price = in.Price
	if in.AreaTolerance != nil {
		if *in.AreaTolerance < 0 {
			return newToolResultError("area_tolerance must be >= 0"), struct{}{}, nil
		}
		criteria.AreaTolerance = *in.AreaTolerance
	}
	if in.PriceTolerancePct != nil {
		if *in.PriceTolerancePct < 0 || *in.PriceTolerancePct > 1 {
			return newToolResultError("price_tolerance_pct must be between 0 and 1"), struct{}{}, nil
		}
		criteria.PriceTolerancePct = *in.PriceTolerancePct
	}
	if in.MaxCandidates != nil {
		if *in.MaxCandidates != 25 && *in.MaxCandidates != 50 && *in.MaxCandidates != 100 {
			return newToolResultError("max_candidates must be one of: 25, 50, 100"), struct{}{}, nil
		}
		criteria.AppliedPageSize = *in.MaxCandidates
	}
	if in.MaxResults != nil {
		if *in.MaxResults < 1 || *in.MaxResults > 100 {
			return newToolResultError("max_results must be between 1 and 100"), struct{}{}, nil
		}
		criteria.AppliedMaxResults = *in.MaxResults
	}

	var txRow *transactionsAdvancedRow
	if txIDRaw := strings.TrimSpace(in.TransactionID); txIDRaw != "" {
		txID, err := uuid.Parse(txIDRaw)
		if err != nil {
			return newToolResultError("transaction_id must be a valid UUID"), struct{}{}, nil
		}
		row, err := t.getTransactionByID(ctx, txID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return newToolResultError("transaction_id not found"), struct{}{}, nil
			}
			return nil, struct{}{}, fmt.Errorf("get transaction by id: %w", err)
		}
		txRow = &row
		criteria.TransactionID = &txID
		criteria.DerivedFromTxn = true
		criteria.DerivedDescription = row.Description
		if criteria.City == "" {
			criteria.City = row.City
		}
		if criteria.PostalCode == "" {
			criteria.PostalCode = row.PostalCode
		}
		if criteria.Area == nil {
			criteria.Area = &row.Area
		}
		if criteria.Price == nil {
			price := int64(row.Price)
			criteria.Price = &price
		}
		if criteria.RoomHint == "" {
			criteria.RoomHint = extractRoomHint(row.Description)
		}
	}
	if criteria.PostalCode == "" && criteria.City == "" && criteria.Query == "" && criteria.RoomHint == "" {
		return newToolResultError("provide transaction_id or at least one of postal_code, city, query, room_hint"), struct{}{}, nil
	}

	adsQuery := strings.TrimSpace(strings.Join([]string{criteria.Query, criteria.RoomHint}, " "))
	params := ads.SearchParams{
		Query:       adsQuery,
		Source:      criteria.Source,
		Kind:        criteria.Kind,
		ListingType: criteria.ListingType,
		City:        criteria.City,
		Postal:      criteria.PostalCode,
		Page:        1,
		PageSize:    criteria.AppliedPageSize,
		Sort:        "seen_desc",
	}
	if criteria.Area != nil {
		minArea := max(0, *criteria.Area-criteria.AreaTolerance)
		maxArea := *criteria.Area + criteria.AreaTolerance
		params.MinArea = &minArea
		params.MaxArea = &maxArea
	}
	if criteria.Price != nil {
		minPrice := int64(float64(*criteria.Price) * (1 - criteria.PriceTolerancePct))
		maxPrice := int64(float64(*criteria.Price) * (1 + criteria.PriceTolerancePct))
		if minPrice < 0 {
			minPrice = 0
		}
		params.MinPrice = &minPrice
		params.MaxPrice = &maxPrice
	}
	searchResult, err := t.adsSvc.Search(ctx, params)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("search ads for transaction: %w", err)
	}
	ranked := make([]adsRankedMatch, 0, len(searchResult.Rows))
	for _, row := range searchResult.Rows {
		score, reasons := scoreAdsMatch(criteria, row)
		ranked = append(ranked, adsRankedMatch{
			Score:      score,
			Confidence: scoreConfidence(score),
			Reasons:    reasons,
			Row:        row,
			Display:    formatAdsMatchDisplay(score, row),
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Row.LastSeenAt.After(ranked[j].Row.LastSeenAt)
		}
		return ranked[i].Score > ranked[j].Score
	})
	if len(ranked) > int(criteria.AppliedMaxResults) {
		ranked = ranked[:criteria.AppliedMaxResults]
	}
	res, err := jsonResult(map[string]any{
		"summary":     map[string]any{"candidates": len(searchResult.Rows), "returned": len(ranked)},
		"criteria":    criteria,
		"transaction": txRow,
		"matches":     ranked,
	})
	return res, struct{}{}, err
}

func (t *toolImpl) listCities(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, struct{}, error) {
	rows, err := t.queries.ListMunicipalitiesWithPostalCodes(ctx)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("list municipalities with postal codes: %w", err)
	}
	res, err := jsonResult(rows)
	return res, struct{}{}, err
}

func (t *toolImpl) listAvailableLocations(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, struct{}, error) {
	rows, err := t.queries.ListMunicipalitiesWithPriceData(ctx)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("list municipalities with price data: %w", err)
	}
	res, err := jsonResult(rows)
	return res, struct{}{}, err
}

func (t *toolImpl) listCategories(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, struct{}, error) {
	rows, err := t.queries.ListDistinctCategories(ctx)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("list categories: %w", err)
	}
	res, err := jsonResult(rows)
	return res, struct{}{}, err
}

// ---- detail resolution ---------------------------------------------------------

type toolImplConfig struct {
	shortcutSitemapBase  string
	frontdoorSitemapBase string
}

func (t *toolImpl) resolveDetailInput(ctx context.Context, input string, in getListingDetailInput) (string, error) {
	canonicalID, err := ads.ResolveInput(input, t.config.shortcutSitemapBase, t.config.frontdoorSitemapBase)
	if err == nil {
		return canonicalID, nil
	}
	query := strings.TrimSpace(input)
	if query == "" {
		return "", fmt.Errorf("input is empty")
	}
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = "ad"
	}
	pageSize := int32(25)
	if in.MaxCandidates != nil {
		if *in.MaxCandidates != 25 && *in.MaxCandidates != 50 && *in.MaxCandidates != 100 {
			return "", fmt.Errorf("max_candidates must be one of: 25, 50, 100")
		}
		pageSize = *in.MaxCandidates
	}
	params := ads.SearchParams{
		Query:    query,
		Source:   strings.TrimSpace(in.Source),
		Kind:     kind,
		City:     strings.TrimSpace(in.City),
		Postal:   strings.TrimSpace(in.Postal),
		Page:     1,
		PageSize: pageSize,
		Sort:     "seen_desc",
	}
	search, err := t.adsSvc.Search(ctx, params)
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

// ---- internal DB types --------------------------------------------------------

type transactionsAdvancedParams struct {
	City            string
	Query           string
	MunicipalityIDs []uuid.UUID
	PostalCodeIDs   []uuid.UUID
	PostalCodes     []string
	Categories      []string
	Types           []string
	MinPrice        *int32
	MaxPrice        *int32
	MinArea         *float64
	MaxArea         *float64
	Sort            string
	Limit           int32
}

type transactionsAdvancedRow struct {
	TransactionID       uuid.UUID  `json:"transaction_id"`
	Description         string     `json:"description"`
	Type                string     `json:"type"`
	Category            string     `json:"category"`
	Area                float64    `json:"area"`
	Price               int32      `json:"price"`
	PricePerSquareMeter int32      `json:"price_per_square_meter"`
	BuildYear           int32      `json:"build_year"`
	Floor               *string    `json:"floor"`
	Elevator            bool       `json:"elevator"`
	Condition           *string    `json:"condition"`
	Plot                *string    `json:"plot"`
	EnergyClass         *string    `json:"energy_class"`
	PeriodIdentifier    string     `json:"period_identifier"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	NeighborhoodID      uuid.UUID  `json:"neighborhood_id"`
	Neighborhood        string     `json:"neighborhood"`
	PostalCodeID        *uuid.UUID `json:"postal_code_id"`
	PostalCode          string     `json:"postal_code"`
	PostalArea          string     `json:"postal_area"`
	MunicipalityID      *uuid.UUID `json:"municipality_id"`
	Municipality        string     `json:"municipality"`
	City                string     `json:"city"`
	Display             string     `json:"display"`
}

type adsMatchCriteria struct {
	TransactionID      *uuid.UUID `json:"transaction_id,omitempty"`
	City               string     `json:"city"`
	PostalCode         string     `json:"postal_code"`
	Area               *float64   `json:"area"`
	Price              *int64     `json:"price"`
	RoomHint           string     `json:"room_hint"`
	Query              string     `json:"query"`
	Source             string     `json:"source"`
	Kind               string     `json:"kind"`
	ListingType        string     `json:"listing_type"`
	AreaTolerance      float64    `json:"area_tolerance"`
	PriceTolerancePct  float64    `json:"price_tolerance_pct"`
	AppliedPageSize    int32      `json:"applied_page_size"`
	AppliedMaxResults  int32      `json:"applied_max_results"`
	DerivedFromTxn     bool       `json:"derived_from_transaction"`
	DerivedDescription string     `json:"derived_description,omitempty"`
}

type adsRankedMatch struct {
	Score      float64              `json:"score"`
	Confidence string               `json:"confidence"`
	Reasons    []string             `json:"reasons"`
	Row        ads.UnifiedEntityRow `json:"row"`
	Display    string               `json:"display"`
}

// ---- DB helpers ----------------------------------------------------------------

func (t *toolImpl) runSearchTransactionsAdvanced(ctx context.Context, params transactionsAdvancedParams) ([]transactionsAdvancedRow, error) {
	query := strings.TrimSpace(params.Query)
	rows, err := t.queries.SearchTransactionsAdvanced(ctx, db.SearchTransactionsAdvancedParams{
		City:            strings.TrimSpace(params.City),
		MunicipalityIds: params.MunicipalityIDs,
		PostalCodeIds:   params.PostalCodeIDs,
		PostalCodes:     params.PostalCodes,
		Categories:      params.Categories,
		Types:           params.Types,
		MinPrice:        params.MinPrice,
		MaxPrice:        params.MaxPrice,
		MinArea:         params.MinArea,
		MaxArea:         params.MaxArea,
		Query:           query,
		NormalizedQuery: compactText(query),
		SortMode:        normalizeTransactionSort(params.Sort),
		LimitCount:      params.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]transactionsAdvancedRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSearchTransactionsAdvancedRow(row))
	}
	return out, nil
}

func (t *toolImpl) getTransactionByID(ctx context.Context, transactionID uuid.UUID) (transactionsAdvancedRow, error) {
	row, err := t.queries.GetTransactionAdvancedByID(ctx, transactionID)
	if err != nil {
		return transactionsAdvancedRow{}, err
	}
	return mapGetTransactionAdvancedByIDRow(row), nil
}

func mapSearchTransactionsAdvancedRow(row db.SearchTransactionsAdvancedRow) transactionsAdvancedRow {
	postalCodeID := row.PostalCodeID
	municipalityID := row.MunicipalityID
	out := transactionsAdvancedRow{
		TransactionID:       row.TransactionID,
		Description:         row.Description,
		Type:                row.Type,
		Category:            row.Category,
		Area:                row.Area,
		Price:               row.Price,
		PricePerSquareMeter: row.PricePerSquareMeter,
		BuildYear:           row.BuildYear,
		Floor:               row.Floor,
		Elevator:            row.Elevator,
		Condition:           row.Condition,
		Plot:                row.Plot,
		EnergyClass:         row.EnergyClass,
		PeriodIdentifier:    row.PeriodIdentifier,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		NeighborhoodID:      row.NeighborhoodID,
		Neighborhood:        row.Neighborhood,
		PostalCodeID:        &postalCodeID,
		PostalCode:          row.PostalCode,
		PostalArea:          row.PostalArea,
		MunicipalityID:      &municipalityID,
		Municipality:        row.Municipality,
		City:                row.City,
	}
	out.Display = formatTransactionDisplay(out)
	return out
}

func mapGetTransactionAdvancedByIDRow(row db.GetTransactionAdvancedByIDRow) transactionsAdvancedRow {
	out := transactionsAdvancedRow{
		TransactionID:       row.TransactionID,
		Description:         row.Description,
		Type:                row.Type,
		Category:            row.Category,
		Area:                row.Area,
		Price:               row.Price,
		PricePerSquareMeter: row.PricePerSquareMeter,
		BuildYear:           row.BuildYear,
		Floor:               row.Floor,
		Elevator:            row.Elevator,
		Condition:           row.Condition,
		Plot:                row.Plot,
		EnergyClass:         row.EnergyClass,
		PeriodIdentifier:    row.PeriodIdentifier,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		NeighborhoodID:      row.NeighborhoodID,
		Neighborhood:        row.Neighborhood,
		PostalCodeID:        row.PostalCodeID,
		PostalCode:          row.PostalCode,
		PostalArea:          row.PostalArea,
		MunicipalityID:      row.MunicipalityID,
		Municipality:        row.Municipality,
		City:                row.City,
	}
	out.Display = formatTransactionDisplay(out)
	return out
}

// ---- result helpers ------------------------------------------------------------

func buildDetailResult(detail ads.UnifiedEntityDetail, includeRawJSON bool) any {
	result := map[string]any{
		"canonical":       detail.Canonical,
		"canonical_extra": detail.CanonicalExtra,
		"source_specific": detail.SourceSpecific,
		"related":         detail.Related,
		"normalized":      detail.Normalized,
		"raw":             detail.Raw,
	}
	if includeRawJSON {
		if rawJSON := parseJSON(detail.Raw.Pretty); rawJSON != nil {
			result["raw_json"] = rawJSON
		}
	}
	return result
}

func chooseBestRows(input string, rows []ads.UnifiedEntityRow) []ads.UnifiedEntityRow {
	if len(rows) == 0 {
		return nil
	}
	needle := normalizeText(input)
	if needle == "" {
		return rows
	}
	var exact []ads.UnifiedEntityRow
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

func formatCandidates(rows []ads.UnifiedEntityRow, maxCount int) string {
	if len(rows) == 0 {
		return ""
	}
	if maxCount <= 0 {
		maxCount = 1
	}
	limit := min(maxCount, len(rows))
	parts := make([]string, 0, limit)
	for _, row := range rows[:limit] {
		label := firstNonEmpty(row.Address, row.Headline, row.CanonicalID)
		parts = append(parts, fmt.Sprintf("%s (%s)", row.CanonicalID, label))
	}
	return strings.Join(parts, "; ")
}

// ---- scoring -------------------------------------------------------------------

func scoreAdsMatch(criteria adsMatchCriteria, row ads.UnifiedEntityRow) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 8)
	if criteria.PostalCode != "" && strings.EqualFold(strings.TrimSpace(row.Postal), strings.TrimSpace(criteria.PostalCode)) {
		score += 40
		reasons = append(reasons, "postal code exact match")
	}
	if criteria.City != "" {
		if strings.EqualFold(strings.TrimSpace(row.City), strings.TrimSpace(criteria.City)) {
			score += 20
			reasons = append(reasons, "city exact match")
		} else if strings.Contains(strings.ToLower(row.City), strings.ToLower(criteria.City)) {
			score += 8
			reasons = append(reasons, "city partial match")
		}
	}
	if criteria.Area != nil && row.Area != nil {
		diff := math.Abs(*row.Area - *criteria.Area)
		if diff <= criteria.AreaTolerance {
			score += 30 * (1 - diff/max(1, criteria.AreaTolerance))
			reasons = append(reasons, fmt.Sprintf("area close (diff %.1fm2)", diff))
		}
	}
	if criteria.Price != nil && row.Price != nil {
		target := float64(*criteria.Price)
		if target > 0 {
			diffPct := math.Abs(float64(*row.Price)-target) / target
			if diffPct <= criteria.PriceTolerancePct {
				score += 20 * (1 - diffPct/max(0.01, criteria.PriceTolerancePct))
				reasons = append(reasons, fmt.Sprintf("price close (diff %.0f%%)", diffPct*100))
			}
		}
	}
	if criteria.RoomHint != "" {
		sim := textSimilarity(criteria.RoomHint, strings.Join([]string{row.RoomLayout, row.Headline, row.Address}, " "))
		if sim > 0 {
			score += 25 * sim
			reasons = append(reasons, fmt.Sprintf("room hint similarity %.2f", sim))
		}
	}
	if criteria.Query != "" {
		sim := textSimilarity(criteria.Query, strings.Join([]string{row.Headline, row.Address, row.RoomLayout}, " "))
		if sim > 0 {
			score += 10 * sim
			reasons = append(reasons, fmt.Sprintf("query similarity %.2f", sim))
		}
	}
	if row.Kind == "ad" || row.Kind == "announcement" {
		score += 3
	}
	return score, reasons
}

func scoreConfidence(score float64) string {
	switch {
	case score >= 70:
		return "high"
	case score >= 45:
		return "medium"
	default:
		return "low"
	}
}

func formatAdsMatchDisplay(score float64, row ads.UnifiedEntityRow) string {
	price := ""
	if row.Price != nil {
		price = " €" + strconv.FormatInt(*row.Price, 10)
	}
	area := ""
	if row.Area != nil {
		area = fmt.Sprintf(" %.1fm2", *row.Area)
	}
	return fmt.Sprintf("[%.1f] %s/%s %s %s%s%s", score, row.Source, row.Kind, firstNonEmpty(row.Address, row.Headline, row.CanonicalID), row.Postal, area, price)
}

// ---- text utilities ------------------------------------------------------------

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func compactText(value string) string {
	re := regexp.MustCompile(`[^[:alnum:]]+`)
	return strings.ToLower(strings.TrimSpace(re.ReplaceAllString(value, "")))
}

func textSimilarity(needle, haystack string) float64 {
	needleTokens := uniqueTokens(needle)
	if len(needleTokens) == 0 {
		return 0
	}
	hayTokens := uniqueTokens(haystack)
	if len(hayTokens) == 0 {
		return 0
	}
	overlap := 0
	for _, tok := range needleTokens {
		if slices.Contains(hayTokens, tok) {
			overlap++
		}
	}
	tokenScore := float64(overlap) / float64(len(needleTokens))
	compactScore := 0.0
	if needleCompact := compactText(needle); needleCompact != "" && strings.Contains(compactText(haystack), needleCompact) {
		compactScore = 1
	}
	return max(tokenScore, compactScore)
}

func uniqueTokens(value string) []string {
	re := regexp.MustCompile(`[[:alnum:]]+`)
	raw := re.FindAllString(strings.ToLower(value), -1)
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, tok := range raw {
		if tok == "" {
			continue
		}
		if _, ok := seen[tok]; !ok {
			seen[tok] = struct{}{}
			out = append(out, tok)
		}
	}
	return out
}

func extractRoomHint(value string) string {
	re := regexp.MustCompile(`(?i)\b[0-9]{1,2}\s*h(?:\s*[+x]\s*[\w]+)?`)
	matches := re.FindAllString(strings.TrimSpace(value), -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(matches[0])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
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

func parseUUIDs(values []string, fieldName string) ([]uuid.UUID, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		parsed, err := uuid.Parse(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%s contains invalid UUID: %s", fieldName, v)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func normalizeTransactionSort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "price_asc", "price_desc", "area_asc", "area_desc", "date_asc", "date_desc":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "date_desc"
	}
}

func formatTransactionDisplay(row transactionsAdvancedRow) string {
	return fmt.Sprintf("%s %s %s %.1fm2 €%d %s",
		row.CreatedAt.Format("2006-01-02"),
		firstNonEmpty(row.City, row.Municipality),
		row.PostalCode, row.Area, row.Price, row.Description)
}
