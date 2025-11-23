package server

import (
	"log"
	"net/http"
	"time"

	"koditon-go/internal/config"
	"koditon-go/internal/db"
)

func New(logger *log.Logger, cfg config.Config, queries *db.Queries) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, logger, cfg, queries)

	var handler http.Handler = mux
	handler = loggingMiddleware(logger, handler)

	return handler
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
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
