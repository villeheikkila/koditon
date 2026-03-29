package api

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon-go/internal/ads"
)

type searchInput struct {
	Query    string  `query:"q"         doc:"Free text search"`
	Source   string  `query:"source"    doc:"Source filter: shortcut, frontdoor, or all"`
	Kind     string  `query:"kind"      doc:"Kind filter: ad, announcement, building, or all"`
	City     string  `query:"city"      doc:"City / municipality filter"`
	Postal   string  `query:"postal"    doc:"Postal code prefix filter"`
	MinPrice *int64  `query:"min_price" doc:"Minimum asking price (EUR)"`
	MaxPrice *int64  `query:"max_price" doc:"Maximum asking price (EUR)"`
	MinArea  *float64 `query:"min_area"  doc:"Minimum area (m²)"`
	MaxArea  *float64 `query:"max_area"  doc:"Maximum area (m²)"`
	Sort     string  `query:"sort"      doc:"Sort order: price_asc, price_desc, area_asc, area_desc, seen_desc"`
	Page     int32   `query:"page"      doc:"Page number (1-based)" minimum:"1"`
	PageSize int32   `query:"page_size" doc:"Results per page: 25, 50, or 100"`
}

type searchResultRow struct {
	CanonicalID string     `json:"canonical_id"`
	Source      string     `json:"source"`
	Kind        string     `json:"kind"`
	Headline    string     `json:"headline,omitempty"`
	Address     string     `json:"address,omitempty"`
	City        string     `json:"city,omitempty"`
	Postal      string     `json:"postal,omitempty"`
	Price       *int64     `json:"price,omitempty"`
	Area        *float64   `json:"area,omitempty"`
	RoomLayout  string     `json:"room_layout,omitempty"`
	URL         string     `json:"url,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

type searchOutput struct {
	Body struct {
		Rows     []searchResultRow `json:"rows"`
		Total    int64             `json:"total"`
		Page     int32             `json:"page"`
		PageSize int32             `json:"page_size"`
	}
}

func (a *API) searchHandler(ctx context.Context, input *searchInput) (*searchOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
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
		MinPrice: input.MinPrice,
		MaxPrice: input.MaxPrice,
		MinArea:  input.MinArea,
		MaxArea:  input.MaxArea,
		Sort:     input.Sort,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := a.adsService.Search(ctx, params)
	if err != nil {
		a.logger.ErrorContext(ctx, "search failed", "error", err)
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
			CanonicalID: r.CanonicalID,
			Source:      r.Source,
			Kind:        r.Kind,
			Headline:    r.Headline,
			Address:     r.Address,
			City:        r.City,
			Postal:      r.Postal,
			Price:       r.Price,
			Area:        r.Area,
			RoomLayout:  r.RoomLayout,
			URL:         r.URL,
			LastSeenAt:  lastSeen,
		}
	}

	out := &searchOutput{}
	out.Body.Rows = rows
	out.Body.Total = result.Total
	out.Body.Page = result.Page
	out.Body.PageSize = result.PageSize
	return out, nil
}
