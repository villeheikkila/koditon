package httpratelimit

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type Config struct {
	RequestsPerSecond float64
	Burst             int
}

type limiterKey struct {
	name              string
	requestsPerSecond float64
	burst             int
}

var registry = struct {
	sync.Mutex
	limiters map[limiterKey]*rate.Limiter
}{
	limiters: make(map[limiterKey]*rate.Limiter),
}

func Transport(name string, next http.RoundTripper, cfg Config) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	if cfg.RequestsPerSecond <= 0 {
		return next
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 1
	}
	key := limiterKey{
		name:              name,
		requestsPerSecond: cfg.RequestsPerSecond,
		burst:             cfg.Burst,
	}
	registry.Lock()
	limiter := registry.limiters[key]
	if limiter == nil {
		limiter = rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)
		registry.limiters[key] = limiter
	}
	registry.Unlock()
	return roundTripper{
		next:    next,
		limiter: limiter,
	}
}

type roundTripper struct {
	next    http.RoundTripper
	limiter *rate.Limiter
}

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := r.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return r.next.RoundTrip(req)
}
