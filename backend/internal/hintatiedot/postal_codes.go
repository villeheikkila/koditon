package hintatiedot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type postalCodesResponse struct {
	PostalCodes string `json:"postalCodes"`
}

type neighborhoodsResponse struct {
	Neighborhoods string `json:"neighborhoods"`
}

func (c *Client) FetchPostalCodes(ctx context.Context, city string) ([]string, error) {
	endpoint, err := c.baseURL.Parse("/haku/searchForm/fetchPostalCodes")
	if err != nil {
		return nil, fmt.Errorf("build fetchPostalCodes URL: %w", err)
	}
	body := strings.NewReader(url.Values{
		"lang": {"fi_FI"},
		"city": {city},
		"type": {""},
	}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	var payload postalCodesResponse
	if err := c.doRequest(ctx, req, &payload); err != nil {
		return nil, err
	}
	return parseList(payload.PostalCodes), nil
}

func (c *Client) FetchNeighborhoods(ctx context.Context, city string) ([]string, error) {
	endpoint, err := c.baseURL.Parse("/haku/searchForm/fetchNeighborhoods")
	if err != nil {
		return nil, fmt.Errorf("build fetchNeighborhoods URL: %w", err)
	}
	body := strings.NewReader(url.Values{
		"lang": {"fi_FI"},
		"city": {city},
		"type": {""},
	}.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	var payload neighborhoodsResponse
	if err := c.doRequest(ctx, req, &payload); err != nil {
		return nil, err
	}
	return parseList(payload.Neighborhoods), nil
}
