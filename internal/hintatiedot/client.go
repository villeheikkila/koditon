package hintatiedot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const (
	defaultBaseURL   = "https://asuntojen.hintatiedot.fi"
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.1 Safari/605.1.15"
	defaultTimeout   = 30 * time.Second
)

var (
	ErrUnexpectedStatus = errors.New("unexpected HTTP status")
	ErrInvalidResponse  = errors.New("invalid response format")
)

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	userAgent  string
}

func NewClient(httpClient *http.Client, baseURL, userAgent string) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultTimeout,
		}
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    parsedBaseURL,
		userAgent:  userAgent,
	}, nil
}

type citiesResponse struct {
	Cities string `json:"cities"`
}

type postalCodesResponse struct {
	PostalCodes string `json:"postalCodes"`
}

func (c *Client) setCommonHeaders(req *http.Request) {
	headers := map[string]string{
		"Accept":           "*/*",
		"Accept-Language":  "en-US,en;q=0.9",
		"Cache-Control":    "no-cache",
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Pragma":           "no-cache",
		"Priority":         "u=5, i",
		"User-Agent":       c.userAgent,
		"X-Requested-With": "XMLHttpRequest",
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

func (c *Client) doRequest(ctx context.Context, req *http.Request, target interface{}) error {
	c.setCommonHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%w: %d: %s", ErrUnexpectedStatus, resp.StatusCode, bytes.TrimSpace(body))
	}
	reader := c.getBodyReader(resp)
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) getBodyReader(resp *http.Response) io.Reader {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "charset=iso-8859-1") {
		return charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	}
	return resp.Body
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

func parseList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	if trimmed == "" {
		return []string{}
	}

	parts := strings.Split(trimmed, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
