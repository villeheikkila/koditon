package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"koditon-go/internal/ads"
	"koditon-go/internal/api"
	"koditon-go/internal/auth"
	"koditon-go/internal/config"
	"koditon-go/internal/web"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	logger     *slog.Logger
	cfg        config.Config
	api        *api.API
	webHandler *web.Handler
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, redisClient *redis.Client, authService *auth.Service) *Server {
	adsService := ads.NewService(pool)
	a := api.New(logger, cfg, pool, redisClient, authService, adsService)
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
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
			ip, _, _ := strings.Cut(r.RemoteAddr, ":")
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip, _, _ = strings.Cut(fwd, ",")
				ip = strings.TrimSpace(ip)
			}
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
		s.logger.InfoContext(
			r.Context(),
			"request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		logLevel := slog.LevelInfo
		if rw.status >= 500 {
			logLevel = slog.LevelError
		} else if rw.status >= 400 {
			logLevel = slog.LevelWarn
		}
		s.logger.Log(
			r.Context(),
			logLevel,
			"request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
