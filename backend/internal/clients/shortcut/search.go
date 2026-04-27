package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type CardType string

const (
	CardTypeSale CardType = "100"
	CardTypeRent CardType = "101"
)

type SearchParams struct {
	Location LocationResponse
	MinSize  *int
	MaxSize  *int
	CardType CardType
	Page     int
	PageSize int
}

func (c *Client) SearchApartments(ctx context.Context, params SearchParams) (*SearchResult, error) {
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 30
	}
	if params.Page < 0 {
		params.Page = 0
	}
	q := url.Values{
		"cardType": {string(params.CardType)},
		"locations": {
			params.Location.LocationString(),
		},
		"limit":  {strconv.Itoa(pageSize)},
		"offset": {strconv.Itoa(params.Page * pageSize)},
	}
	if params.MinSize != nil {
		q.Set("size[min]", strconv.Itoa(*params.MinSize))
	}
	if params.MaxSize != nil {
		q.Set("size[max]", strconv.Itoa(*params.MaxSize))
	}
	var result SearchResult
	cardsURL := joinURL(c.baseURL, "/api/5.0/cards")
	if err := c.doRequestWithRetry(ctx, cardsURL, q, &result); err != nil {
		return nil, fmt.Errorf("fetch cards: %w", err)
	}
	return &result, nil
}
