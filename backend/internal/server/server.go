package server

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"koditon-go/internal/ads"
	"koditon-go/internal/api"
	"koditon-go/internal/auth"
	"koditon-go/internal/config"
	"koditon-go/internal/web"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	logger     *slog.Logger
	cfg        config.Config
	api        *api.API
	webHandler *web.Handler
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, authService *auth.Service) *Server {
	adsService := ads.NewService(pool)
	a := api.New(logger, cfg, pool, authService, adsService)
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
