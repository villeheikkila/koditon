package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) FetchLocationIDs(ctx context.Context, postalCode string) ([]LocationResponse, error) {
	locationURL := joinURL(c.baseURL, "/api/5.0/location")
	q := url.Values{
		"query":     {postalCode},
		"card_type": {"5"},
	}
	var result []LocationResponse
	if err := c.doRequestWithRetry(ctx, locationURL, q, &result); err != nil {
		return nil, fmt.Errorf("fetch location ids: %w", err)
	}
	return result, nil
}

func (c *Client) FetchBuildings(ctx context.Context, query string) ([]LocationResponse, error) {
	searchURL := joinURL(c.baseURL, "/api/3.0/location")
	q := url.Values{
		"query":       {query},
		"cardTypes[]": {"3"},
	}
	q.Add("cardTypes[]", "2")
	var result []LocationResponse
	if err := c.doRequestWithRetry(ctx, searchURL, q, &result); err != nil {
		return nil, fmt.Errorf("fetch buildings: %w", err)
	}
	return result, nil
}

func (c *Client) GetBuildingData(ctx context.Context, locationID string) ([]BuildingResponse, error) {
	buildingURL := joinURL(c.baseURL, "/api/3.0/building")
	q := url.Values{"locationId": {locationID}}
	var result []BuildingResponse
	if err := c.doRequestWithRetry(ctx, buildingURL, q, &result); err != nil {
		return nil, fmt.Errorf("fetch building data: %w", err)
	}
	return result, nil
}

func (c *Client) GetAdByID(ctx context.Context, id int) (json.RawMessage, error) {
	adEndpoint := fmt.Sprintf("%s/%s/%d", c.adBaseURL, "api/v5/5/apartments/items", id)
	var raw json.RawMessage
	if err := c.doRequestWithRetry(ctx, adEndpoint, nil, &raw); err != nil {
		return nil, fmt.Errorf("fetch ad %d: %w", id, err)
	}
	return raw, nil
}
