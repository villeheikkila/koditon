package hintatiedot

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type citiesResponse struct {
	Cities string `json:"cities"`
}

func (c *Client) FetchCities(ctx context.Context) ([]string, error) {
	endpoint, err := c.baseURL.Parse("/haku/searchForm/fetchCities")
	if err != nil {
		return nil, fmt.Errorf("build fetchCities URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader("lang=fi_FI"))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	var payload citiesResponse
	if err := c.doRequest(ctx, req, &payload); err != nil {
		return nil, err
	}
	return parseList(payload.Cities), nil
}
