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

	"koditon-go/internal/ads"
	"koditon-go/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	minPrice, ok, err := int64Arg(args, "min_price", 0, math.MaxInt64)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if ok {
		params.MinPrice = &minPrice
	}
	maxPrice, ok, err := int64Arg(args, "max_price", 0, math.MaxInt64)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if ok {
		params.MaxPrice = &maxPrice
	}
	if v, ok := numberArg(args, "min_area"); ok {
		if v < 0 {
			return mcp.NewToolResultError("min_area must be >= 0"), nil
		}
		params.MinArea = &v
	}
	if v, ok := numberArg(args, "max_area"); ok {
		if v < 0 {
			return mcp.NewToolResultError("max_area must be >= 0"), nil
		}
		params.MaxArea = &v
	}
	if params.MinPrice != nil && params.MaxPrice != nil && *params.MinPrice > *params.MaxPrice {
		return mcp.NewToolResultError("min_price cannot be greater than max_price"), nil
	}
	if params.MinArea != nil && params.MaxArea != nil && *params.MinArea > *params.MaxArea {
		return mcp.NewToolResultError("min_area cannot be greater than max_area"), nil
	}
	page, ok, err := int32Arg(args, "page", 1, math.MaxInt32)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if ok {
		params.Page = page
	}
	pageSize, ok, err := int32Arg(args, "page_size", 1, math.MaxInt32)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if ok {
		if pageSize != 25 && pageSize != 50 && pageSize != 100 {
			return mcp.NewToolResultError("page_size must be one of: 25, 50, 100"), nil
		}
		params.PageSize = pageSize
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
		if errors.Is(err, ads.ErrNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("listing not found: %s", canonicalID)), nil
		}
		return nil, fmt.Errorf("get listing detail: %w", err)
	}
	includeRawJSON := boolArg(args, "include_raw_json")
	return jsonResult(buildDetailResult(detail, includeRawJSON))
}

func (s *Server) handleSearchTransactions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	city := stringArg(args, "city")
	if city == "" {
		return mcp.NewToolResultError("city is required"), nil
	}

	address := stringArg(args, "address")
	var limit int32 = 200
	v, ok, err := int32Arg(args, "limit", 1, math.MaxInt32)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if ok {
		limit = v
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

func (s *Server) handleSearchTransactionsAdvanced(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	minPrice, ok, err := int32Arg(args, "min_price", 0, math.MaxInt32)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var minPricePtr *int32
	if ok {
		minPricePtr = &minPrice
	}
	maxPrice, ok, err := int32Arg(args, "max_price", 0, math.MaxInt32)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var maxPricePtr *int32
	if ok {
		maxPricePtr = &maxPrice
	}
	minArea, ok := numberArg(args, "min_area")
	if ok && minArea < 0 {
		return mcp.NewToolResultError("min_area must be >= 0"), nil
	}
	var minAreaPtr *float64
	if ok {
		minAreaPtr = &minArea
	}
	maxArea, ok := numberArg(args, "max_area")
	if ok && maxArea < 0 {
		return mcp.NewToolResultError("max_area must be >= 0"), nil
	}
	var maxAreaPtr *float64
	if ok {
		maxAreaPtr = &maxArea
	}
	if minPricePtr != nil && maxPricePtr != nil && *minPricePtr > *maxPricePtr {
		return mcp.NewToolResultError("min_price cannot be greater than max_price"), nil
	}
	if minAreaPtr != nil && maxAreaPtr != nil && *minAreaPtr > *maxAreaPtr {
		return mcp.NewToolResultError("min_area cannot be greater than max_area"), nil
	}
	limit, ok, err := int32Arg(args, "limit", 1, 5000)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if !ok {
		limit = 200
	}
	municipalityIDs, err := uuidArrayArg(args, "municipality_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	postalCodeIDs, err := uuidArrayArg(args, "postal_code_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	postalCodes, err := stringArrayArg(args, "postal_codes")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	categories, err := stringArrayArg(args, "categories")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	types, err := stringArrayArg(args, "types")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sortMode := normalizeTransactionSort(stringArg(args, "sort"))
	rows, err := s.searchTransactionsAdvanced(ctx, transactionsAdvancedParams{
		City:            stringArg(args, "city"),
		Query:           stringArg(args, "query"),
		MunicipalityIDs: municipalityIDs,
		PostalCodeIDs:   postalCodeIDs,
		PostalCodes:     postalCodes,
		Categories:      categories,
		Types:           types,
		MinPrice:        minPricePtr,
		MaxPrice:        maxPricePtr,
		MinArea:         minAreaPtr,
		MaxArea:         maxAreaPtr,
		Sort:            sortMode,
		Limit:           limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search transactions advanced: %w", err)
	}
	return jsonResult(map[string]any{
		"summary": map[string]any{
			"count": len(rows),
			"sort":  sortMode,
			"limit": limit,
		},
		"rows": rows,
	})
}

func (s *Server) handleMatchAdsFromTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	criteria := adsMatchCriteria{
		City:              stringArg(args, "city"),
		PostalCode:        stringArg(args, "postal_code"),
		RoomHint:          stringArg(args, "room_hint"),
		Query:             stringArg(args, "query"),
		Source:            stringArg(args, "source"),
		Kind:              stringArg(args, "kind"),
		ListingType:       stringArg(args, "listing_type"),
		AreaTolerance:     8.0,
		PriceTolerancePct: 0.35,
		AppliedPageSize:   100,
		AppliedMaxResults: 20,
	}
	if raw, ok := numberArg(args, "area"); ok {
		if raw < 0 {
			return mcp.NewToolResultError("area must be >= 0"), nil
		}
		criteria.Area = &raw
	}
	if raw, ok, err := int64Arg(args, "price", 0, math.MaxInt64); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		criteria.Price = &raw
	}
	if raw, ok := numberArg(args, "area_tolerance"); ok {
		if raw < 0 {
			return mcp.NewToolResultError("area_tolerance must be >= 0"), nil
		}
		criteria.AreaTolerance = raw
	}
	if raw, ok := numberArg(args, "price_tolerance_pct"); ok {
		if raw < 0 || raw > 1 {
			return mcp.NewToolResultError("price_tolerance_pct must be between 0 and 1"), nil
		}
		criteria.PriceTolerancePct = raw
	}
	if raw, ok, err := int32Arg(args, "max_candidates", 1, 100); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		if raw != 25 && raw != 50 && raw != 100 {
			return mcp.NewToolResultError("max_candidates must be one of: 25, 50, 100"), nil
		}
		criteria.AppliedPageSize = raw
	}
	if raw, ok, err := int32Arg(args, "max_results", 1, 100); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		criteria.AppliedMaxResults = raw
	}
	txIDRaw := stringArg(args, "transaction_id")
	var txRow *transactionsAdvancedRow
	if txIDRaw != "" {
		txID, err := uuid.Parse(txIDRaw)
		if err != nil {
			return mcp.NewToolResultError("transaction_id must be a valid UUID"), nil
		}
		row, err := s.getTransactionByID(ctx, txID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return mcp.NewToolResultError("transaction_id not found"), nil
			}
			return nil, fmt.Errorf("get transaction by id: %w", err)
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
		return mcp.NewToolResultError("provide transaction_id or at least one of postal_code, city, query, room_hint"), nil
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
	searchResult, err := s.adsSvc.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search ads for transaction: %w", err)
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
	return jsonResult(map[string]any{
		"summary": map[string]any{
			"candidates": len(searchResult.Rows),
			"returned":   len(ranked),
		},
		"criteria":    criteria,
		"transaction": txRow,
		"matches":     ranked,
	})
}

func (s *Server) handleListCities(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rows, err := s.queries.ListMunicipalitiesWithPostalCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list municipalities with postal codes: %w", err)
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

func (s *Server) searchTransactionsAdvanced(ctx context.Context, params transactionsAdvancedParams) ([]transactionsAdvancedRow, error) {
	query := strings.TrimSpace(params.Query)
	city := strings.TrimSpace(params.City)
	clauses := make([]string, 0, 16)
	clauses = append(clauses, "1=1")
	args := make([]any, 0, 16)
	add := func(template string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(template, len(args)))
	}
	if city != "" {
		add("lower(trim(pc.prices_city_name)) LIKE ('%%' || lower(trim($%d)) || '%%')", city)
	}
	if len(params.MunicipalityIDs) > 0 {
		add("pm.postal_municipality_id = ANY($%d::uuid[])", params.MunicipalityIDs)
	}
	if len(params.PostalCodeIDs) > 0 {
		add("ppc.postal_postal_code_id = ANY($%d::uuid[])", params.PostalCodeIDs)
	}
	if len(params.PostalCodes) > 0 {
		add("COALESCE(ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) = ANY($%d::text[])", params.PostalCodes)
	}
	if len(params.Categories) > 0 {
		add("ht.prices_transaction_category = ANY($%d::text[])", params.Categories)
	}
	if len(params.Types) > 0 {
		add("ht.prices_transaction_type = ANY($%d::text[])", params.Types)
	}
	if params.MinPrice != nil {
		add("ht.prices_transaction_price >= $%d::int", *params.MinPrice)
	}
	if params.MaxPrice != nil {
		add("ht.prices_transaction_price <= $%d::int", *params.MaxPrice)
	}
	if params.MinArea != nil {
		add("ht.prices_transaction_area >= $%d::double precision", *params.MinArea)
	}
	if params.MaxArea != nil {
		add("ht.prices_transaction_area <= $%d::double precision", *params.MaxArea)
	}
	if query != "" {
		normalized := compactText(query)
		args = append(args, query)
		qp := len(args)
		args = append(args, normalized)
		np := len(args)
		clauses = append(clauses, fmt.Sprintf(`(
			ht.prices_transaction_description ILIKE ('%%' || $%d || '%%')
			OR pn.prices_neighborhood_name ILIKE ('%%' || $%d || '%%')
			OR COALESCE(ppc.postal_postal_code_code, '') ILIKE ('%%' || $%d || '%%')
			OR COALESCE(ppc_prices.prices_postal_code_code, '') ILIKE ('%%' || $%d || '%%')
			OR COALESCE(ppc.postal_postal_code_name_fi, '') ILIKE ('%%' || $%d || '%%')
			OR COALESCE(pm.postal_municipality_name_fi, '') ILIKE ('%%' || $%d || '%%')
			OR ht.prices_transaction_category ILIKE ('%%' || $%d || '%%')
			OR ht.prices_transaction_type ILIKE ('%%' || $%d || '%%')
			OR lower(regexp_replace(COALESCE(ht.prices_transaction_description, ''), '[^[:alnum:]]+', '', 'g')) LIKE ('%%' || $%d || '%%')
		)`, qp, qp, qp, qp, qp, qp, qp, qp, np))
	}
	sortClause := transactionsOrderByClause(params.Sort)
	args = append(args, params.Limit)
	limitPos := len(args)
	sql := fmt.Sprintf(`SELECT
		ht.prices_transaction_id,
		ht.prices_transaction_description,
		ht.prices_transaction_type,
		ht.prices_transaction_category,
		ht.prices_transaction_area,
		ht.prices_transaction_price,
		ht.prices_transaction_price_per_square_meter,
		ht.prices_transaction_build_year,
		ht.prices_transaction_floor,
		ht.prices_transaction_elevator,
		ht.prices_transaction_condition,
		ht.prices_transaction_plot,
		ht.prices_transaction_energy_class,
		ht.prices_transaction_period_identifier,
		ht.prices_transaction_created_at,
		ht.prices_transaction_updated_at,
		pn.prices_neighborhood_id,
		pn.prices_neighborhood_name,
		ppc.postal_postal_code_id,
		COALESCE(ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_code,
		COALESCE(ppc.postal_postal_code_name_fi, '') AS postal_area_name_fi,
		pm.postal_municipality_id,
		COALESCE(pm.postal_municipality_name_fi, '') AS municipality_name_fi,
		pc.prices_city_name
	FROM public.prices_transactions AS ht
	JOIN public.prices_neighborhoods AS pn ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
	JOIN public.prices_cities AS pc ON pc.prices_city_id = pn.prices_city_id
	LEFT JOIN public.prices_postal_codes AS ppc_prices ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
	LEFT JOIN public.postal_postal_codes AS ppc ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
	LEFT JOIN public.postal_municipalities AS pm ON pm.postal_municipality_id = ppc.postal_municipality_id
	WHERE %s
	ORDER BY %s
	LIMIT $%d::int`, strings.Join(clauses, " AND "), sortClause, limitPos)
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]transactionsAdvancedRow, 0)
	for rows.Next() {
		var row transactionsAdvancedRow
		if err := rows.Scan(
			&row.TransactionID,
			&row.Description,
			&row.Type,
			&row.Category,
			&row.Area,
			&row.Price,
			&row.PricePerSquareMeter,
			&row.BuildYear,
			&row.Floor,
			&row.Elevator,
			&row.Condition,
			&row.Plot,
			&row.EnergyClass,
			&row.PeriodIdentifier,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.NeighborhoodID,
			&row.Neighborhood,
			&row.PostalCodeID,
			&row.PostalCode,
			&row.PostalArea,
			&row.MunicipalityID,
			&row.Municipality,
			&row.City,
		); err != nil {
			return nil, err
		}
		row.Display = formatTransactionDisplay(row)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) getTransactionByID(ctx context.Context, transactionID uuid.UUID) (transactionsAdvancedRow, error) {
	sql := `SELECT
		ht.prices_transaction_id,
		ht.prices_transaction_description,
		ht.prices_transaction_type,
		ht.prices_transaction_category,
		ht.prices_transaction_area,
		ht.prices_transaction_price,
		ht.prices_transaction_price_per_square_meter,
		ht.prices_transaction_build_year,
		ht.prices_transaction_floor,
		ht.prices_transaction_elevator,
		ht.prices_transaction_condition,
		ht.prices_transaction_plot,
		ht.prices_transaction_energy_class,
		ht.prices_transaction_period_identifier,
		ht.prices_transaction_created_at,
		ht.prices_transaction_updated_at,
		pn.prices_neighborhood_id,
		pn.prices_neighborhood_name,
		ppc.postal_postal_code_id,
		COALESCE(ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code) AS postal_code,
		COALESCE(ppc.postal_postal_code_name_fi, '') AS postal_area_name_fi,
		pm.postal_municipality_id,
		COALESCE(pm.postal_municipality_name_fi, '') AS municipality_name_fi,
		pc.prices_city_name
	FROM public.prices_transactions AS ht
	JOIN public.prices_neighborhoods AS pn ON pn.prices_neighborhood_id = ht.prices_neighborhood_id
	JOIN public.prices_cities AS pc ON pc.prices_city_id = pn.prices_city_id
	LEFT JOIN public.prices_postal_codes AS ppc_prices ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
	LEFT JOIN public.postal_postal_codes AS ppc ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
	LEFT JOIN public.postal_municipalities AS pm ON pm.postal_municipality_id = ppc.postal_municipality_id
	WHERE ht.prices_transaction_id = $1
	LIMIT 1`
	var row transactionsAdvancedRow
	err := s.pool.QueryRow(ctx, sql, transactionID).Scan(
		&row.TransactionID,
		&row.Description,
		&row.Type,
		&row.Category,
		&row.Area,
		&row.Price,
		&row.PricePerSquareMeter,
		&row.BuildYear,
		&row.Floor,
		&row.Elevator,
		&row.Condition,
		&row.Plot,
		&row.EnergyClass,
		&row.PeriodIdentifier,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.NeighborhoodID,
		&row.Neighborhood,
		&row.PostalCodeID,
		&row.PostalCode,
		&row.PostalArea,
		&row.MunicipalityID,
		&row.Municipality,
		&row.City,
	)
	if err != nil {
		return transactionsAdvancedRow{}, err
	}
	row.Display = formatTransactionDisplay(row)
	return row, nil
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

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
		rawJSON := parseJSON(detail.Raw.Pretty)
		if rawJSON != nil {
			result["raw_json"] = rawJSON
		}
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
		if v != 25 && v != 50 && v != 100 {
			return "", fmt.Errorf("max_candidates must be one of: 25, 50, 100")
		}
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

func compactText(value string) string {
	re := regexp.MustCompile(`[^[:alnum:]]+`)
	return strings.ToLower(strings.TrimSpace(re.ReplaceAllString(value, "")))
}

func transactionsOrderByClause(sortMode string) string {
	switch normalizeTransactionSort(sortMode) {
	case "price_asc":
		return "ht.prices_transaction_price ASC, ht.prices_transaction_created_at DESC"
	case "price_desc":
		return "ht.prices_transaction_price DESC, ht.prices_transaction_created_at DESC"
	case "area_asc":
		return "ht.prices_transaction_area ASC, ht.prices_transaction_created_at DESC"
	case "area_desc":
		return "ht.prices_transaction_area DESC, ht.prices_transaction_created_at DESC"
	case "date_asc":
		return "ht.prices_transaction_created_at ASC, ht.prices_transaction_price ASC"
	default:
		return "ht.prices_transaction_created_at DESC, ht.prices_transaction_price ASC"
	}
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
	return fmt.Sprintf("%s %s %s %.1fm2 €%d %s", row.CreatedAt.Format("2006-01-02"), firstNonEmpty(row.City, row.Municipality), row.PostalCode, row.Area, row.Price, row.Description)
}

func scoreAdsMatch(criteria adsMatchCriteria, row ads.UnifiedEntityRow) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 8)
	if criteria.PostalCode != "" {
		if strings.EqualFold(strings.TrimSpace(row.Postal), strings.TrimSpace(criteria.PostalCode)) {
			score += 40
			reasons = append(reasons, "postal code exact match")
		}
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
			local := 30 * (1 - diff/max(1, criteria.AreaTolerance))
			score += local
			reasons = append(reasons, fmt.Sprintf("area close (diff %.1fm2)", diff))
		}
	}
	if criteria.Price != nil && row.Price != nil {
		target := float64(*criteria.Price)
		if target > 0 {
			diffPct := math.Abs(float64(*row.Price)-target) / target
			if diffPct <= criteria.PriceTolerancePct {
				local := 20 * (1 - diffPct/max(0.01, criteria.PriceTolerancePct))
				score += local
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
	needleCompact := compactText(needle)
	hayCompact := compactText(haystack)
	compactScore := 0.0
	if needleCompact != "" && strings.Contains(hayCompact, needleCompact) {
		compactScore = 1
	}
	return max(tokenScore, compactScore)
}

func uniqueTokens(value string) []string {
	re := regexp.MustCompile(`[[:alnum:]]+`)
	raw := re.FindAllString(strings.ToLower(value), -1)
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, tok := range raw {
		if tok == "" {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
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
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringArg(args map[string]any, key string) string {
	v, ok := args[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func stringArrayArg(args map[string]any, key string) ([]string, error) {
	v, ok := args[key]
	if !ok {
		return nil, nil
	}
	switch typed := v.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s must contain only strings", key)
			}
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, nil
		}
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings or comma-separated string", key)
	}
}

func uuidArrayArg(args map[string]any, key string) ([]uuid.UUID, error) {
	values, err := stringArrayArg(args, key)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s contains invalid UUID: %s", key, value)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func numberArg(args map[string]any, key string) (float64, bool) {
	v, ok := args[key].(float64)
	return v, ok
}

func boolArg(args map[string]any, key string) bool {
	v, ok := args[key].(bool)
	if !ok {
		return false
	}
	return v
}

func int32Arg(args map[string]any, key string, min, max int64) (int32, bool, error) {
	raw, ok := numberArg(args, key)
	if !ok {
		return 0, false, nil
	}
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0, false, fmt.Errorf("%s must be a finite number", key)
	}
	if math.Trunc(raw) != raw {
		return 0, false, fmt.Errorf("%s must be an integer", key)
	}
	if raw < float64(min) || raw > float64(max) {
		return 0, false, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return int32(raw), true, nil
}

func int64Arg(args map[string]any, key string, min, max int64) (int64, bool, error) {
	raw, ok := numberArg(args, key)
	if !ok {
		return 0, false, nil
	}
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0, false, fmt.Errorf("%s must be a finite number", key)
	}
	if math.Trunc(raw) != raw {
		return 0, false, fmt.Errorf("%s must be an integer", key)
	}
	if raw < float64(min) || raw > float64(max) {
		return 0, false, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return int64(raw), true, nil
}
