package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRequestTimeout = 30 * time.Second
	maxRetries            = 3
	initialBackoff        = 1 * time.Second
)

var (
	ErrInitialStateNotFound    = errors.New("frontdoor: initial state not found")
	ErrInitialStateEndNotFound = errors.New("frontdoor: initial state end not found")
	ErrDecodeInitialState      = errors.New("frontdoor: failed to decode initial state")
)

// HTTPStatusError represents an HTTP error response with status code for proper error classification.
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("frontdoor: HTTP %d: %s", e.StatusCode, e.Body)
}

// IsNotFound returns true if the error is a 404 Not Found.
func (e *HTTPStatusError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsHTTPStatusError checks if an error is an HTTPStatusError and returns it.
func IsHTTPStatusError(err error) (*HTTPStatusError, bool) {
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) {
		return httpErr, true
	}
	return nil, false
}

type EntryType string

const (
	EntryTypeAd       EntryType = "ad"
	EntryTypeBuilding EntryType = "building"
)

type SitemapEntry struct {
	ID   string
	Type EntryType
	URL  *url.URL
}

type Client struct {
	httpClient     *http.Client
	baseURL        string
	userAgent      string
	cookie         string
	timeout        time.Duration
	sitemapBaseURL string
}

func New(baseURL, userAgent, cookie, sitemapBaseURL string) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	httpClient := &http.Client{
		Timeout:   defaultRequestTimeout,
		Transport: transport,
	}
	return &Client{
		httpClient:     httpClient,
		baseURL:        baseURL,
		userAgent:      userAgent,
		cookie:         cookie,
		sitemapBaseURL: sitemapBaseURL,
	}
}

func (c *Client) GetAdByFriendlyID(ctx context.Context, friendlyID string) (*AdResponse, error) {
	reqCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	u.Path = "/api/announcement/details"
	q := u.Query()
	q.Set("friendlyId", friendlyID)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.applyDefaultHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	var ad AdResponse
	if err := json.Unmarshal(body, &ad); err != nil {
		return nil, fmt.Errorf("decode ad response: %w", err)
	}
	return &ad, nil
}

func (c *Client) GetBuildingPageData(ctx context.Context, pageURL string) (*HousingCompanyResponse, error) {
	reqCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	if pageURL == "" {
		return nil, fmt.Errorf("build request: pageURL is required")
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.applyDefaultHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	raw, err := extractInitialState(body)
	if err != nil {
		return nil, err
	}
	var respPayload HousingCompanyResponse
	if err := json.Unmarshal(raw, &respPayload); err != nil {
		return nil, fmt.Errorf("decode housing company response: %w", err)
	}
	return &respPayload, nil
}

func (c *Client) GetSitemapEntries(ctx context.Context) ([]SitemapEntry, error) {
	sitemapURLs := []string{
		c.joinSitemap("sitemap_row_house.xml"),
		c.joinSitemap("sitemap_detached_house.xml"),
		c.joinSitemap("sitemap_semi_detached_house.xml"),
		c.joinSitemap("sitemap_separate_house.xml"),
		c.joinSitemap("sitemap_apartment_house.xml"),
		c.joinSitemap("sitemap_wooden_house_apartment.xml"),
		c.joinSitemap("sitemap_balcony_access_block.xml"),
		c.joinSitemap("sitemap_hca.xml"),
	}
	var entries []SitemapEntry
	var fetchErrors []error
	for _, sitemapURL := range sitemapURLs {
		sitemapXML, err := c.fetchXMLWithRetry(ctx, sitemapURL)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("fetch %s: %w", sitemapURL, err))
			continue
		}
		for _, loc := range extractLocs(sitemapXML) {
			if entry, ok := c.parseEntry(loc); ok {
				entries = append(entries, *entry)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(entries) == 0 && len(fetchErrors) > 0 {
		return nil, fmt.Errorf("all sitemap fetches failed: %w", errors.Join(fetchErrors...))
	}
	return entries, nil
}

func (c *Client) applyDefaultHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", c.userAgent)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
}

func extractInitialState(body []byte) (json.RawMessage, error) {
	const prefix = "window.__INITIAL_STATE__ = "
	html := string(body)
	start := strings.Index(html, prefix)
	if start == -1 {
		return nil, ErrInitialStateNotFound
	}
	end := strings.Index(html[start:], "})();")
	if end == -1 {
		return nil, ErrInitialStateEndNotFound
	}
	rawJSON := html[start+len(prefix) : start+end]
	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimSuffix(rawJSON, ";")
	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.ReplaceAll(rawJSON, "undefined", "null")
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeInitialState, err)
	}
	return raw, nil
}

func (c *Client) parseEntry(raw string) (*SitemapEntry, bool) {
	adPrefix := c.sitemapBaseURL + "/kohde/"
	buildingPrefix := c.sitemapBaseURL + "/talo/"
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	if remaining, ok := strings.CutPrefix(raw, adPrefix); ok {
		id := trimAfterSeparators(remaining)
		return &SitemapEntry{ID: id, Type: EntryTypeAd, URL: u}, true
	}

	if remaining, ok := strings.CutPrefix(raw, buildingPrefix); ok {
		id := trimAfterSeparators(remaining)
		// Skip building entries with hyphens in the ID
		if strings.Contains(id, "-") {
			return nil, false
		}
		return &SitemapEntry{ID: id, Type: EntryTypeBuilding, URL: u}, true
	}

	return nil, false
}

var locPattern = regexp.MustCompile(`<loc>([^<]+)</loc>`)

func (c *Client) fetchXMLWithRetry(ctx context.Context, url string) (string, error) {
	var lastErr error
	backoff := initialBackoff
	for attempt := range maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
		xml, err := c.fetchXML(ctx, url)
		if err == nil {
			return xml, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func extractLocs(xml string) []string {
	matches := locPattern.FindAllStringSubmatch(xml, -1)
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}
	return results
}

func (c *Client) fetchXML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/xml, text/xml, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return "", &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)) + retryAfterSuffix(retryAfter),
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(body), nil
}

func trimAfterSeparators(value string) string {
	for i, r := range value {
		if r == '/' || r == '?' || r == '#' {
			return value[:i]
		}
	}
	return value
}

func (c *Client) joinSitemap(path string) string {
	base, err := url.Parse(c.sitemapBaseURL)
	if err != nil {
		return c.sitemapBaseURL + "/" + path
	}
	ref, err := url.Parse(path)
	if err != nil {
		return c.sitemapBaseURL + "/" + path
	}
	return base.ResolveReference(ref).String()
}

func parseRetryAfter(value string) int {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return seconds
}

func retryAfterSuffix(seconds int) string {
	if seconds > 0 {
		return fmt.Sprintf(" (retry-after: %ds)", seconds)
	}
	return ""
}
