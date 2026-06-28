package api

import (
	"context"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon/internal/domain/ads"
	"koditon/internal/platform/logging"
)

type searchInput struct {
	Query    string  `query:"q"         doc:"Free text search"`
	Source   string  `query:"source"    doc:"Source filter: shortcut, frontdoor, or all"`
	Kind     string  `query:"kind"      doc:"Kind filter: ad, announcement, building, or all"`
	Grouping string  `query:"grouping"  doc:"Grouping filter: grouped, ungrouped, or all"`
	City     string  `query:"city"      doc:"City / municipality filter"`
	Postal   string  `query:"postal"    doc:"Postal code prefix filter"`
	MinPrice int64   `query:"min_price" doc:"Minimum asking price (EUR, 0 = no minimum)"`
	MaxPrice int64   `query:"max_price" doc:"Maximum asking price (EUR, 0 = no maximum)"`
	MinArea  float64 `query:"min_area"  doc:"Minimum area (m², 0 = no minimum)"`
	MaxArea  float64 `query:"max_area"  doc:"Maximum area (m², 0 = no maximum)"`
	Sort     string  `query:"sort"      doc:"Sort order: price_asc, price_desc, area_asc, area_desc, seen_desc"`
	Page     int32   `query:"page"      doc:"Page number (1-based)" minimum:"1"`
	PageSize int32   `query:"page_size" doc:"Results per page: 25, 50, or 100"`
}

type searchResultRow struct {
	CanonicalID             string     `json:"canonical_id"`
	Source                  string     `json:"source"`
	Kind                    string     `json:"kind"`
	NativeID                string     `json:"native_id"`
	ListingID               string     `json:"listing_id,omitempty"`
	OfferingID              string     `json:"offering_id,omitempty"`
	HousingCompanyID        string     `json:"housing_company_id,omitempty"`
	HousingCompanyName      string     `json:"housing_company_name,omitempty"`
	LinkStatus              string     `json:"link_status,omitempty"`
	LinkMethod              string     `json:"link_method,omitempty"`
	LinkScore               *int32     `json:"link_score,omitempty"`
	PriceMatchTransactionID string     `json:"price_match_transaction_id,omitempty"`
	PriceMatchScope         string     `json:"price_match_scope,omitempty"`
	PriceMatchStatus        string     `json:"price_match_status,omitempty"`
	PriceMatchMethod        string     `json:"price_match_method,omitempty"`
	PriceMatchScore         *int32     `json:"price_match_score,omitempty"`
	PriceMatchPrice         *int64     `json:"price_match_price_eur,omitempty"`
	InsightCount            int32      `json:"insight_count,omitempty"`
	InsightTopSeverity      string     `json:"insight_top_severity,omitempty"`
	Headline                string     `json:"headline,omitempty"`
	Address                 string     `json:"address,omitempty"`
	City                    string     `json:"city,omitempty"`
	Postal                  string     `json:"postal,omitempty"`
	Price                   *int64     `json:"price,omitempty"`
	Area                    *float64   `json:"area,omitempty"`
	RoomLayout              string     `json:"room_layout,omitempty"`
	URL                     string     `json:"url,omitempty"`
	ExternalURLAvailable    bool       `json:"external_url_available"`
	LastSeenAt              *time.Time `json:"last_seen_at,omitempty"`
}

type searchOutput struct {
	Body struct {
		Rows     []searchResultRow `json:"rows"`
		Total    int64             `json:"total"`
		Page     int32             `json:"page"`
		PageSize int32             `json:"page_size"`
	}
}

type groupedOfferingSearchRow struct {
	OfferingID              string     `json:"offering_id"`
	HousingCompanyID        string     `json:"housing_company_id,omitempty"`
	HousingCompanyName      string     `json:"housing_company_name,omitempty"`
	Headline                string     `json:"headline,omitempty"`
	Address                 string     `json:"address,omitempty"`
	City                    string     `json:"city,omitempty"`
	Postal                  string     `json:"postal,omitempty"`
	Price                   *int64     `json:"price,omitempty"`
	Area                    *float64   `json:"area,omitempty"`
	RoomLayout              string     `json:"room_layout,omitempty"`
	LastSeenAt              *time.Time `json:"last_seen_at,omitempty"`
	SourceCount             int32      `json:"source_count"`
	Sources                 []string   `json:"sources"`
	PriceMatchTransactionID string     `json:"price_match_transaction_id,omitempty"`
	PriceMatchScope         string     `json:"price_match_scope,omitempty"`
	PriceMatchStatus        string     `json:"price_match_status,omitempty"`
	PriceMatchMethod        string     `json:"price_match_method,omitempty"`
	PriceMatchScore         *int32     `json:"price_match_score,omitempty"`
	PriceMatchPrice         *int64     `json:"price_match_price_eur,omitempty"`
	InsightCount            int32      `json:"insight_count,omitempty"`
	InsightTopSeverity      string     `json:"insight_top_severity,omitempty"`
}

type groupedOfferingSearchOutput struct {
	Body struct {
		Rows     []groupedOfferingSearchRow `json:"rows"`
		Total    int64                      `json:"total"`
		Page     int32                      `json:"page"`
		PageSize int32                      `json:"page_size"`
	}
}

type addressLookupInput struct {
	Address  string `query:"address"   doc:"Street address to inspect"`
	City     string `query:"city"      doc:"Optional city / municipality filter"`
	Postal   string `query:"postal"    doc:"Optional postal code filter"`
	Source   string `query:"source"    doc:"Source filter: shortcut, frontdoor, or all"`
	PageSize int32  `query:"page_size" doc:"Maximum listings to return: default 50, max 100"`
}

type addressLookupOutput struct {
	Body ads.AddressLookupResult
}

func (a *API) searchHandler(ctx context.Context, input *searchInput) (*searchOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.search"))
	page := max(input.Page, 1)
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}

	params := ads.SearchParams{
		Query:    input.Query,
		Source:   input.Source,
		Kind:     input.Kind,
		Grouping: input.Grouping,
		City:     input.City,
		Postal:   input.Postal,
		MinPrice: positiveInt64Ptr(input.MinPrice),
		MaxPrice: positiveInt64Ptr(input.MaxPrice),
		MinArea:  positiveFloat64Ptr(input.MinArea),
		MaxArea:  positiveFloat64Ptr(input.MaxArea),
		Sort:     input.Sort,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := a.adsService.Search(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "search failed", "error", err, "query", input.Query, "page", page, "page_size", pageSize, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("search failed")
	}

	rows := make([]searchResultRow, len(result.Rows))
	for i, r := range result.Rows {
		t := r.LastSeenAt
		var lastSeen *time.Time
		if !t.IsZero() {
			lastSeen = &t
		}
		rows[i] = searchResultRow{
			CanonicalID:             r.CanonicalID,
			Source:                  r.Source,
			Kind:                    r.Kind,
			NativeID:                r.NativeID,
			ListingID:               r.ListingID,
			OfferingID:              r.OfferingID,
			HousingCompanyID:        r.HousingCompanyID,
			HousingCompanyName:      r.HousingCompanyName,
			LinkStatus:              r.LinkStatus,
			LinkMethod:              r.LinkMethod,
			LinkScore:               r.LinkScore,
			PriceMatchTransactionID: r.PriceMatchTransactionID,
			PriceMatchScope:         r.PriceMatchScope,
			PriceMatchStatus:        r.PriceMatchStatus,
			PriceMatchMethod:        r.PriceMatchMethod,
			PriceMatchScore:         r.PriceMatchScore,
			PriceMatchPrice:         r.PriceMatchPrice,
			InsightCount:            r.InsightCount,
			InsightTopSeverity:      r.InsightTopSeverity,
			Headline:                r.Headline,
			Address:                 r.Address,
			City:                    r.City,
			Postal:                  r.Postal,
			Price:                   r.Price,
			Area:                    r.Area,
			RoomLayout:              r.RoomLayout,
			URL:                     r.URL,
			ExternalURLAvailable:    r.ExternalURLAvailable,
			LastSeenAt:              lastSeen,
		}
	}

	out := &searchOutput{}
	out.Body.Rows = rows
	out.Body.Total = result.Total
	out.Body.Page = result.Page
	out.Body.PageSize = result.PageSize
	return out, nil
}

func (a *API) groupedOfferingSearchHandler(ctx context.Context, input *searchInput) (*groupedOfferingSearchOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.search_grouped_offerings"))
	page := max(input.Page, 1)
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	params := ads.SearchParams{
		Query:    input.Query,
		Source:   input.Source,
		Kind:     input.Kind,
		City:     input.City,
		Postal:   input.Postal,
		MinPrice: positiveInt64Ptr(input.MinPrice),
		MaxPrice: positiveInt64Ptr(input.MaxPrice),
		MinArea:  positiveFloat64Ptr(input.MinArea),
		MaxArea:  positiveFloat64Ptr(input.MaxArea),
		Sort:     input.Sort,
		Page:     page,
		PageSize: pageSize,
	}
	result, err := a.adsService.SearchGroupedOfferings(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "grouped offering search failed", "error", err, "query", input.Query, "page", page, "page_size", pageSize, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("grouped offering search failed")
	}
	rows := make([]groupedOfferingSearchRow, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = groupedOfferingSearchRow{
			OfferingID:              r.OfferingID,
			HousingCompanyID:        r.HousingCompanyID,
			HousingCompanyName:      r.HousingCompanyName,
			Headline:                r.Headline,
			Address:                 r.Address,
			City:                    r.City,
			Postal:                  r.Postal,
			Price:                   r.Price,
			Area:                    r.Area,
			RoomLayout:              r.RoomLayout,
			LastSeenAt:              r.LastSeenAt,
			SourceCount:             r.SourceCount,
			Sources:                 r.Sources,
			PriceMatchTransactionID: r.PriceMatchTransactionID,
			PriceMatchScope:         r.PriceMatchScope,
			PriceMatchStatus:        r.PriceMatchStatus,
			PriceMatchMethod:        r.PriceMatchMethod,
			PriceMatchScore:         r.PriceMatchScore,
			PriceMatchPrice:         r.PriceMatchPrice,
			InsightCount:            r.InsightCount,
			InsightTopSeverity:      r.InsightTopSeverity,
		}
	}
	out := &groupedOfferingSearchOutput{}
	out.Body.Rows = rows
	out.Body.Total = result.Total
	out.Body.Page = result.Page
	out.Body.PageSize = result.PageSize
	return out, nil
}

func (a *API) addressLookupHandler(ctx context.Context, input *addressLookupInput) (*addressLookupOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.address_lookup"))
	if strings.TrimSpace(input.Address) == "" {
		return nil, huma.Error400BadRequest("address is required")
	}
	result, err := a.adsService.LookupAddress(ctx, ads.AddressLookupParams{Address: input.Address, City: input.City, Postal: input.Postal, Source: input.Source, PageSize: input.PageSize})
	if err != nil {
		logger.ErrorContext(ctx, "address lookup failed", "address", input.Address, "city", input.City, "postal", input.Postal, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("address lookup failed")
	}
	return &addressLookupOutput{Body: result}, nil
}
