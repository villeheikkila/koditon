package api

import (
	"context"
	"errors"
	"log/slog"
	"time"

	frontdoorclient "koditon/internal/clients/frontdoor"
	shortcutclient "koditon/internal/clients/shortcut"
	"koditon/internal/db"
	"koditon/internal/domain/ads"
	"koditon/internal/domain/auth"
	"koditon/internal/domain/emailauth"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/config"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	logger            *slog.Logger
	cfg               config.Config
	pool              *pgxpool.Pool
	pricesQueries     *db.Queries
	postalQueries     *db.Queries
	authService       *auth.Service
	emailAuthService  *emailauth.Service
	adsService        *ads.Service
	propertiesService *properties.Service
	shortcutAPI       *shortcutclient.Client
	frontdoorAPI      *frontdoorclient.Client
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, authService *auth.Service, emailAuthService *emailauth.Service, adsService *ads.Service) *API {
	shortcutQueries := db.New(pool)
	tokenLoad := func(ctx context.Context) (*shortcutclient.Tokens, error) {
		dbToken, err := shortcutQueries.GetValidShortcutToken(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("no valid token found")
			}
			return nil, err
		}
		return &shortcutclient.Tokens{
			CUID:   dbToken.ShortcutTokenCuid,
			Token:  dbToken.ShortcutTokenToken,
			Loaded: dbToken.ShortcutTokenLoaded,
		}, nil
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
	return &API{
		logger:            logger.With("component", "api"),
		cfg:               cfg,
		pool:              pool,
		pricesQueries:     db.New(pool),
		postalQueries:     db.New(pool),
		authService:       authService,
		emailAuthService:  emailAuthService,
		adsService:        adsService,
		propertiesService: properties.NewService(pool),
		shortcutAPI:       shortcutClient,
		frontdoorAPI:      frontdoorClient,
	}
}

func (a *API) AddRoutes(humaAPI huma.API) {
	addRoutes(a, humaAPI)
}
