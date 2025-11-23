package server

import (
	"log/slog"
	"net/http"
	"time"

	"koditon-go/internal/config"
	"koditon-go/internal/db"
	"koditon-go/internal/hintatiedot"
)

type Server struct {
	logger         *slog.Logger
	cfg            config.Config
	db             *db.Queries
	hintatiedotAPI *hintatiedot.Client
}

func New(logger *slog.Logger, cfg config.Config, queries *db.Queries, hintatiedotClient *hintatiedot.Client) *Server {
	return &Server{
		logger:         logger.With("component", "server"),
		cfg:            cfg,
		db:             queries,
		hintatiedotAPI: hintatiedotClient,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.addRoutes(mux)

	var handler http.Handler = mux
	handler = s.loggingMiddleware(handler)

	return handler
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		s.logger.InfoContext(
			r.Context(),
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
