package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultTokenExpiry    = 365 * 24 * time.Hour // 1 year - rely on 401 for actual expiry
	defaultRequestTimeout = 30 * time.Second
)

var (
	ErrInvalidTokens = errors.New("invalid tokens")
	ErrAuthFailed    = errors.New("authentication failed")
)

// HTTPStatusError represents an HTTP error response with status code for proper error classification.
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("shortcut: HTTP %d: %s", e.StatusCode, e.Body)
}

func IsHTTPStatusError(err error) (*HTTPStatusError, bool) {
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) {
		return httpErr, true
	}
	return nil, false
}

type Tokens struct {
	Loaded string
	CUID   string
	Token  string
}

type TokenStore func(ctx context.Context, tokens *Tokens, expiresAt time.Time) error
type TokenLoader func(ctx context.Context) (*Tokens, error)

type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
	tokenStore TokenStore
	tokenLoad  TokenLoader

	mu                 sync.RWMutex
	cachedTokens       *Tokens
	tokenFetchGroup    singleflight.Group
	tokenExpiry        time.Duration
	requestTimeout     time.Duration
	userAgent          string
	tokenRefreshRandom func(context.Context) (string, error)
	baseURL            string
	docsBaseURL        string
	adBaseURL          string
	refererURL         string
	sitemapBaseURL     string
}

func NewClient(logger *slog.Logger, tokenLoad TokenLoader, tokenStore TokenStore, baseURL, docsBaseURL, adBaseURL, userAgent, sitemapBaseURL string) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	httpClient := &http.Client{
		Transport: transport,
	}
	return &Client{
		httpClient:         httpClient,
		logger:             logger.With("component", "shortcut-client"),
		tokenLoad:          tokenLoad,
		tokenStore:         tokenStore,
		tokenExpiry:        defaultTokenExpiry,
		requestTimeout:     defaultRequestTimeout,
		userAgent:          userAgent,
		tokenRefreshRandom: defaultTokenRandom,
		baseURL:            baseURL,
		docsBaseURL:        docsBaseURL,
		adBaseURL:          adBaseURL,
		refererURL:         joinURL(baseURL, "/myytavat-asunnot"),
		sitemapBaseURL:     sitemapBaseURL,
	}
}

func defaultTokenRandom(context.Context) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return n.String(), nil
}

func (c *Client) getValidTokens(ctx context.Context) (*Tokens, error) {
	c.mu.RLock()
	if c.cachedTokens != nil {
		cached := c.cachedTokens
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()
	v, err, _ := c.tokenFetchGroup.Do("fetch-token", func() (any, error) {
		tokenCtx := context.Background()
		if deadline, ok := ctx.Deadline(); ok {
			var cancel context.CancelFunc
			tokenCtx, cancel = context.WithDeadline(context.Background(), deadline)
			defer cancel()
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.cachedTokens != nil {
			return c.cachedTokens, nil
		}
		if c.tokenLoad != nil {
			tokens, err := c.tokenLoad(tokenCtx)
			if err == nil {
				c.cachedTokens = tokens
				return tokens, nil
			}
		}
		return c.refreshTokens(tokenCtx)
	})
	if err != nil {
		return nil, err
	}
	return v.(*Tokens), nil
}

func (c *Client) refreshTokens(ctx context.Context) (*Tokens, error) {
	tokens, err := c.createAnonymousUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("create anonymous user: %w", err)
	}
	expiresAt := time.Now().Add(c.tokenExpiry)
	if c.tokenStore != nil {
		_ = c.tokenStore(ctx, tokens, expiresAt)
	}
	c.cachedTokens = tokens
	return tokens, nil
}

func (c *Client) invalidateTokens() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cachedTokens = nil
}

func (c *Client) createAnonymousUser(ctx context.Context) (*Tokens, error) {
	randomValue, err := c.tokenRefreshRandom(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate random value: %w", err)
	}
	tokenURL := joinURL(c.baseURL, "/user/get")
	parsed, err := url.Parse(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("parse token endpoint: %w", err)
	}
	query := parsed.Query()
	query.Set("format", "json")
	query.Set("rand", randomValue)
	parsed.RawQuery = query.Encode()
	reqCtx := ctx
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, "token request failed", "url", parsed.String(), "error", err)
		return nil, fmt.Errorf("perform token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Body: bytesTrim(body)}
	}
	var tokenResponse struct {
		User struct {
			CUID  string `json:"cuid"`
			Token string `json:"token"`
			Time  int64  `json:"time"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResponse.User.CUID == "" || tokenResponse.User.Token == "" || tokenResponse.User.Time == 0 {
		return nil, fmt.Errorf("%w: missing fields", ErrInvalidTokens)
	}
	return &Tokens{
		Loaded: strconv.FormatInt(tokenResponse.User.Time, 10),
		CUID:   tokenResponse.User.CUID,
		Token:  tokenResponse.User.Token,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, endpoint string, params url.Values, tokens *Tokens) (*http.Request, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", endpoint, err)
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", c.refererURL)
	req.Header.Set("User-Agent", c.userAgent)
	if tokens != nil {
		req.Header.Set("OTA-loaded", tokens.Loaded)
		req.Header.Set("OTA-cuid", tokens.CUID)
		req.Header.Set("OTA-token", tokens.Token)
	}
	return req, nil
}

func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, tokens *Tokens, target any) error {
	reqCtx := ctx
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	req, err := c.newRequest(reqCtx, endpoint, params, tokens)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_, _ = io.Copy(io.Discard, resp.Body)
		c.invalidateTokens()
		return fmt.Errorf("%w: %d: %s", ErrAuthFailed, resp.StatusCode, bytesTrim(body))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_, _ = io.Copy(io.Discard, resp.Body)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Body: bytesTrim(body)}
	}
	if target == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, endpoint string, params url.Values, target any) error {
	tokens, err := c.getValidTokens(ctx)
	if err != nil {
		return fmt.Errorf("get valid tokens: %w", err)
	}
	err = c.doRequest(ctx, endpoint, params, tokens, target)
	if err != nil && errors.Is(err, ErrAuthFailed) {
		c.mu.Lock()
		newTokens, refreshErr := c.refreshTokens(ctx)
		c.mu.Unlock()
		if refreshErr != nil {
			return fmt.Errorf("refresh tokens after auth failure: %w", refreshErr)
		}
		err = c.doRequest(ctx, endpoint, params, newTokens, target)
	}
	return err
}

func bytesTrim(b []byte) string {
	return string(bytes.TrimSpace(b))
}

func joinURL(base, path string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return base + path
	}
	rel, err := url.Parse(path)
	if err != nil {
		return base + path
	}
	return baseURL.ResolveReference(rel).String()
}
