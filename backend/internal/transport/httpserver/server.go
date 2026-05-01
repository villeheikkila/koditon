package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"koditon/internal/domain/ads"
	"koditon/internal/domain/auth"
	"koditon/internal/domain/emailauth"
	"koditon/internal/platform/config"
	"koditon/internal/platform/logging"
	api "koditon/internal/transport/openapi"
	"koditon/internal/transport/web"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	logger     *slog.Logger
	cfg        config.Config
	api        *api.API
	webHandler *web.Handler
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, authService *auth.Service, emailAuthService *emailauth.Service) *Server {
	adsService := ads.NewService(pool)
	a := api.New(logger, cfg, pool, authService, emailAuthService, adsService)
	webHandler := web.NewHandler(adsService, cfg.Shortcut.SitemapBase, cfg.Frontdoor.SitemapBase, logger)
	return &Server{
		logger:     logger.With("component", "server"),
		cfg:        cfg,
		api:        a,
		webHandler: webHandler,
	}
}

func (s *Server) Handler(mux *http.ServeMux, humaAPI huma.API) http.Handler {
	s.api.AddRoutes(humaAPI)
	if s.cfg.WebStaticDir != "" {
		registerSPA(mux, s.cfg.WebStaticDir)
	} else {
		s.webHandler.Register(mux)
	}
	var handler http.Handler = mux
	handler = s.recoveryMiddleware(handler)
	handler = s.loggingMiddleware(handler)
	handler = s.bodySizeLimitMiddleware(handler)
	rl := newRateLimiter(300, time.Minute)
	handler = s.rateLimitMiddleware(rl)(handler)
	if s.cfg.CORSAllowedOrigins != "" {
		origins := strings.Split(s.cfg.CORSAllowedOrigins, ",")
		for i, o := range origins {
			origins[i] = strings.TrimSpace(o)
		}
		handler = corsMiddleware(origins)(handler)
	}
	return handler
}

func registerSPA(mux *http.ServeMux, dir string) {
	fs := http.FileServer(http.Dir(dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := dir + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, dir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				}
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-ID, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.ErrorContext(
					r.Context(),
					"request panicked",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"outcome", logging.OutcomeError,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status": http.StatusInternalServerError,
					"title":  http.StatusText(http.StatusInternalServerError),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) bodySizeLimitMiddleware(next http.Handler) http.Handler {
	const maxBodyBytes = 4 << 20 // 4 MB
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateBucket
	limit   int
	window  time.Duration
}

type rateBucket struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		clients: make(map[string]*rateBucket),
		limit:   limit,
		window:  window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.clients[key]
	if !ok || now.After(b.reset) {
		rl.clients[key] = &rateBucket{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

func (s *Server) rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := requestClientIP(r)
			if !rl.allow(ip) {
				http.Error(w, `{"status":429,"title":"Too Many Requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestLogger := logging.With(
			s.logger,
			logging.Op("request.handle"),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("route", requestRoute(r)),
			slog.String("client_ip", requestClientIP(r)),
		)
		requestLogger.InfoContext(r.Context(), "request started")
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		logLevel := slog.LevelInfo
		if rw.status >= 500 {
			logLevel = slog.LevelError
		} else if rw.status >= 400 {
			logLevel = slog.LevelWarn
		}
		requestLogger.LogAttrs(
			r.Context(),
			logLevel,
			"request completed",
			slog.Int("status", rw.status),
			slog.Int64("response_bytes", rw.bytes),
			logging.DurationMS(duration),
			logging.Outcome(httpOutcome(rw.status)),
		)
	})
}

func requestRoute(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.URL.Path
}

func requestClientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		if ip := normalizeIP(first); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		if ip := normalizeIP(realIP); ip != "" {
			return ip
		}
	}
	host, err := remoteHost(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	if ip := normalizeIP(host); ip != "" {
		return ip
	}
	return host
}

func remoteHost(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host, nil
	}
	if strings.Contains(err.Error(), "missing port in address") {
		return remoteAddr, nil
	}
	return "", fmt.Errorf("split remote addr: %w", err)
}

func normalizeIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return raw
	}
	return addr.String()
}

func httpOutcome(status int) string {
	switch {
	case status >= 500:
		return logging.OutcomeError
	case status >= 400:
		return logging.OutcomeRejected
	default:
		return logging.OutcomeSuccess
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(p)
	rw.bytes += int64(n)
	return n, err
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
