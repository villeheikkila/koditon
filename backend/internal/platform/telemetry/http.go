package telemetry

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func HTTPTransport(base http.RoundTripper, opts ...otelhttp.Option) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base, opts...)
}

func HTTPClient(timeout time.Duration, transport http.RoundTripper, opts ...otelhttp.Option) *http.Client {
	return &http.Client{Timeout: timeout, Transport: HTTPTransport(transport, opts...)}
}
