package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon-go/internal/config"
	"koditon-go/internal/db"
	"koditon-go/internal/hintatiedot"
)

type Server struct {
	logger          *slog.Logger
	cfg             config.Config
	db              *db.Queries
	hintatiedotAPI  *hintatiedot.Client
	hintatiedotSync *hintatiedot.SyncService
}

func New(logger *slog.Logger, cfg config.Config, queries *db.Queries, hintatiedotClient *hintatiedot.Client) *Server {
	return &Server{
		logger:          logger.With("component", "server"),
		cfg:             cfg,
		db:              queries,
		hintatiedotAPI:  hintatiedotClient,
		hintatiedotSync: hintatiedot.NewSyncService(queries, hintatiedotClient, logger.With("component", "hintatiedot-sync")),
	}
}

func (s *Server) Handler(mux *http.ServeMux, api huma.API) http.Handler {
	s.addRoutes(api)
	var handler http.Handler = mux
	handler = s.loggingMiddleware(handler)
	return handler
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
