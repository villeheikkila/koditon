package consumers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	frontdoorclient "koditon-go/internal/frontdoor/client"
	pricesclient "koditon-go/internal/prices/client"
	shortcutclient "koditon-go/internal/shortcut/client"
	"koditon-go/internal/taskqueue"
)

type HTTPStatusError struct {
	StatusCode int
	Message    string
	RetryAfter int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

type EntityParseError struct {
	EntityID string
	Reason   string
	Err      error
}

func (e *EntityParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parse entity %s: %s: %v", e.EntityID, e.Reason, e.Err)
	}
	return fmt.Sprintf("parse entity %s: %s", e.EntityID, e.Reason)
}

func (e *EntityParseError) Unwrap() error {
	return e.Err
}

func classifyError(err error, _ any) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return taskqueue.NewRetryableError(err)
	}
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) {
		return classifyHTTPStatus(err, httpErr.StatusCode, httpErr.RetryAfter)
	}
	var frontdoorHTTPErr *frontdoorclient.HTTPStatusError
	if errors.As(err, &frontdoorHTTPErr) {
		return classifyHTTPStatus(err, frontdoorHTTPErr.StatusCode, 0)
	}
	var shortcutHTTPErr *shortcutclient.HTTPStatusError
	if errors.As(err, &shortcutHTTPErr) {
		return classifyHTTPStatus(err, shortcutHTTPErr.StatusCode, 0)
	}
	var pricesHTTPErr *pricesclient.HTTPStatusError
	if errors.As(err, &pricesHTTPErr) {
		return classifyHTTPStatus(err, pricesHTTPErr.StatusCode, 0)
	}
	var parseErr *EntityParseError
	if errors.As(err, &parseErr) {
		return taskqueue.NewPermanentError(err, "invalid entity format")
	}
	return err
}

func classifyHTTPStatus(err error, statusCode, retryAfter int) error {
	switch {
	case statusCode == http.StatusNotFound:
		return taskqueue.NewPermanentError(err, "resource not found")
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return taskqueue.NewPermanentError(err, "authentication/authorization failed")
	case statusCode == http.StatusTooManyRequests:
		if retryAfter <= 0 {
			retryAfter = 60
		}
		return taskqueue.NewRetryableErrorWithDelay(err, retryAfter)
	case statusCode >= 400 && statusCode < 500:
		return taskqueue.NewPermanentError(err, fmt.Sprintf("client error: %d", statusCode))
	case statusCode >= 500:
		return taskqueue.NewRetryableError(err)
	}
	return err
}
