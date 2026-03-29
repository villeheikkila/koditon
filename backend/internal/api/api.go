package api

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"koditon-go/internal/ads"
	"koditon-go/internal/auth"
	"koditon-go/internal/config"
	"koditon-go/internal/db"
	frontdoorclient "koditon-go/internal/frontdoor/client"
	shortcutclient "koditon-go/internal/shortcut/client"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type API struct {
	logger        *slog.Logger
	cfg           config.Config
	pool          *pgxpool.Pool
	redis         *redis.Client
	pricesQueries *db.Queries
	postalQueries *db.Queries
	authService   *auth.Service
	adsService    *ads.Service
	shortcutAPI   *shortcutclient.Client
	frontdoorAPI  *frontdoorclient.Client
}

func New(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, redisClient *redis.Client, authService *auth.Service, adsService *ads.Service) *API {
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
		logger:        logger.With("component", "api"),
		cfg:           cfg,
		pool:          pool,
		redis:         redisClient,
		pricesQueries: db.New(pool),
		postalQueries: db.New(pool),
		authService:   authService,
		adsService:    adsService,
		shortcutAPI:   shortcutClient,
		frontdoorAPI:  frontdoorClient,
	}
}

func (a *API) AddRoutes(humaAPI huma.API) {
	addRoutes(a, humaAPI)
}
