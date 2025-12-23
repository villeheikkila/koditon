package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/config"
	frontdoorclient "koditon-go/internal/frontdoor/client"
	pricesclient "koditon-go/internal/prices/client"
	pricesdb "koditon-go/internal/prices/db"
	shortcutclient "koditon-go/internal/shortcut/client"
	shortcutdb "koditon-go/internal/shortcut/db"
	"koditon-go/internal/taskqueue"
)

type Server struct {
	logger        *slog.Logger
	cfg           config.Config
	pricesQueries *pricesdb.Queries
	pricesAPI     *pricesclient.Client
	taskQueue     *taskqueue.Client
	shortcutAPI   *shortcutclient.Client
	frontdoorAPI  *frontdoorclient.Client
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, taskQueueClient *taskqueue.Client) *Server {
	pricesQueries := pricesdb.New(pool)
	shortcutQueries := shortcutdb.New(pool)

	pricesClient, _ := pricesclient.NewClient(cfg.Prices.BaseURL)

	tokenLoad := func(ctx context.Context) (*shortcutclient.Tokens, error) {
		dbToken, err := shortcutQueries.GetValidShortcutToken(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("no valid token found")
			}
			return nil, err
		}
		tokens := &shortcutclient.Tokens{
			CUID:   dbToken.ShortcutTokensCuid,
			Token:  dbToken.ShortcutTokensToken,
			Loaded: dbToken.ShortcutTokensLoaded,
		}
		return tokens, nil
	}
	tokenStore := func(ctx context.Context, tokens *shortcutclient.Tokens, expiresAt time.Time) error {
		_, err := shortcutQueries.InsertShortcutToken(ctx, &shortcutdb.InsertShortcutTokenParams{
			ShortcutTokensCuid:      tokens.CUID,
			ShortcutTokensToken:     tokens.Token,
			ShortcutTokensLoaded:    tokens.Loaded,
			ShortcutTokensExpiresAt: expiresAt,
		})
		return err
	}
	shortcutClient := shortcutclient.NewClient(
		logger,
		tokenLoad,
		tokenStore,
		cfg.Shortcut.BaseURL,
		cfg.Shortcut.DocsBaseURL,
		cfg.Shortcut.AdBaseURL,
		cfg.Shortcut.UserAgent,
		cfg.Shortcut.SitemapBase,
	)
	frontdoorClient := frontdoorclient.New(
		cfg.Frontdoor.BaseURL,
		cfg.Frontdoor.UserAgent,
		cfg.Frontdoor.Cookie,
		cfg.Frontdoor.SitemapBase,
	)
	return &Server{
		logger:        logger.With("component", "server"),
		cfg:           cfg,
		pricesQueries: pricesQueries,
		pricesAPI:     pricesClient,
		taskQueue:     taskQueueClient,
		shortcutAPI:   shortcutClient,
		frontdoorAPI:  frontdoorClient,
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
