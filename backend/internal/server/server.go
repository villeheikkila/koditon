package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/ads"
	"koditon-go/internal/auth"
	"koditon-go/internal/config"
	"koditon-go/internal/db"
	frontdoorclient "koditon-go/internal/frontdoor/client"
	pricesclient "koditon-go/internal/prices/client"
	shortcutclient "koditon-go/internal/shortcut/client"
	"koditon-go/internal/web"
)

type Server struct {
	logger        *slog.Logger
	cfg           config.Config
	pricesQueries *db.Queries
	pricesAPI     *pricesclient.Client
	postalQueries *db.Queries
	shortcutAPI   *shortcutclient.Client
	frontdoorAPI  *frontdoorclient.Client
	authService   *auth.Service
	webHandler    *web.Handler
	adsService    *ads.Service
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, authService *auth.Service) *Server {
	pricesQueries := db.New(pool)
	postalQueries := db.New(pool)
	shortcutQueries := db.New(pool)
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
			CUID:   dbToken.ShortcutTokenCuid,
			Token:  dbToken.ShortcutTokenToken,
			Loaded: dbToken.ShortcutTokenLoaded,
		}
		return tokens, nil
	}
	tokenStore := func(ctx context.Context, tokens *shortcutclient.Tokens, expiresAt time.Time) error {
		_, err := shortcutQueries.InsertShortcutToken(ctx, db.InsertShortcutTokenParams{
			ShortcutTokenCuid:      tokens.CUID,
			ShortcutTokenToken:     tokens.Token,
			ShortcutTokenLoaded:    tokens.Loaded,
			ShortcutTokenExpiresAt: expiresAt,
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
	adsService := ads.NewService(pool)
	webHandler := web.NewHandler(adsService, cfg.Shortcut.SitemapBase, cfg.Frontdoor.SitemapBase, logger)
	return &Server{
		logger:        logger.With("component", "server"),
		cfg:           cfg,
		pricesQueries: pricesQueries,
		pricesAPI:     pricesClient,
		postalQueries: postalQueries,
		shortcutAPI:   shortcutClient,
		frontdoorAPI:  frontdoorClient,
		authService:   authService,
		webHandler:    webHandler,
		adsService:    adsService,
	}
}

func (s *Server) Handler(mux *http.ServeMux, api huma.API) http.Handler {
	s.addRoutes(api)
	if s.cfg.WebStaticDir != "" {
		registerSPA(mux, s.cfg.WebStaticDir)
	} else {
		s.webHandler.Register(mux)
	}
	var handler http.Handler = mux
	handler = s.loggingMiddleware(handler)
	return handler
}

// registerSPA serves a React SPA from dir. API paths are handled by the mux
// already; any other path either serves a static asset or falls back to index.html.
func registerSPA(mux *http.ServeMux, dir string) {
	fs := http.FileServer(http.Dir(dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Let the file server try first; if the asset doesn't exist serve index.html.
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
