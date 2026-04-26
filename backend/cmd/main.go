package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"koditon-go/internal/auth"
	"koditon-go/internal/config"
	"koditon-go/internal/consumers"
	db "koditon-go/internal/db"
	"koditon-go/internal/frontdoor"
	"koditon-go/internal/logging"
	"koditon-go/internal/mcpserver"
	"koditon-go/internal/oauthapi"
	"koditon-go/internal/postal"
	"koditon-go/internal/prices"
	"koditon-go/internal/requestid"
	"koditon-go/internal/runtimecfg"
	"koditon-go/internal/server"
	"koditon-go/internal/shortcut"
	"koditon-go/internal/telegram"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/redis/go-redis/v9"
	openrouter "github.com/revrost/go-openrouter"
)

func main() {
	ctx := context.Background()
	if len(os.Args) > 1 && os.Args[1] == "verify-env" {
		if err := runVerifyEnv(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	_ []string,
	_ func(string) string,
	_ io.Reader,
	_, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(stderr, cfg)
	appLogger := logging.With(logger.With("component", "app"), logging.Op("app.start"))
	appLogger.InfoContext(ctx, "starting application",
		"env", cfg.Environment,
		"log_level", cfg.LogLevel,
		"mode", cfg.Mode.String(),
	)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	appLogger.DebugContext(ctx, "database connection established")
	var consumer *consumers.Consumer
	if cfg.Mode.Consumer {
		openRouterClient := openrouter.NewClient(cfg.OpenRouter.APIKey)
		pricesService, err := prices.NewService(
			pool,
			cfg.Prices.BaseURL,
			openRouterClient,
		)
		if err != nil {
			return fmt.Errorf("create prices service: %w", err)
		}
		shortcutService := shortcut.NewService(
			pool,
			logger,
			cfg.Shortcut.BaseURL,
			cfg.Shortcut.DocsBaseURL,
			cfg.Shortcut.AdBaseURL,
			cfg.Shortcut.UserAgent,
			cfg.Shortcut.SitemapBase,
		)
		frontdoorService := frontdoor.NewService(
			pool,
			logger,
			cfg.Frontdoor.BaseURL,
			cfg.Frontdoor.UserAgent,
			cfg.Frontdoor.Cookie,
			cfg.Frontdoor.SitemapBase,
		)
		postalService := postal.NewService(pool)
		consumer = consumers.New(
			logger,
			pool,
			pricesService,
			shortcutService,
			frontdoorService,
			postalService,
		)
		consumerConfig := consumers.DefaultConfig()
		if err := consumer.Start(ctx, consumerConfig); err != nil {
			return fmt.Errorf("start consumer: %w", err)
		}
	}
	var httpServer *http.Server
	var errCh chan error
	if cfg.Mode.API {
		redisClient := redis.NewClient(&redis.Options{
			Addr: cfg.Redis.Addr,
		})
		authCfg := &runtimecfg.AuthConfig{
			JWT: runtimecfg.JWTAuthConfig{
				PrivateKey:  cfg.Auth.JWTSigningKey,
				Issuer:      cfg.Auth.JWTIssuer,
				UIDHashSalt: cfg.Auth.JWTUIDHashSalt,
			},
			Apple: runtimecfg.AppleAuthConfig{
				BundleID:       cfg.Auth.Apple.BundleID,
				TeamID:         cfg.Auth.Apple.TeamID,
				PrivateKeyID:   cfg.Auth.Apple.PrivateKeyID,
				PrivateKey:     cfg.Auth.Apple.PrivateKey,
				WebServiceID:   cfg.Auth.Apple.WebServiceID,
				WebRedirectURI: cfg.Auth.Apple.WebRedirectURI,
			},
		}
		if cfg.Auth.OAuthCookieKey != "" {
			authCfg.OAuth = &runtimecfg.OAuthConfig{
				CookieSigningKey: cfg.Auth.OAuthCookieKey,
				AccessTokenTTL:   cfg.Auth.OAuthATTL,
				RefreshTokenTTL:  cfg.Auth.OAuthRTTL,
			}
		}
		if cfg.Auth.PasskeyRPID != "" {
			origins := strings.Split(cfg.Auth.PasskeyRPOrigins, ",")
			authCfg.Passkey = &runtimecfg.PasskeyConfig{
				RPDisplayName: cfg.Auth.PasskeyRPName,
				RPID:          cfg.Auth.PasskeyRPID,
				RPOrigins:     origins,
			}
		}
		authService, err := auth.NewService(ctx, auth.ServiceConfig{
			Pool:        pool,
			Auth:        authCfg,
			RedisClient: redisClient,
			Logger:      logger,
		})
		if err != nil {
			return fmt.Errorf("create auth service: %w", err)
		}
		srv := server.New(logger, cfg, pool, redisClient, authService)
		mux := http.NewServeMux()
		if cfg.APIPublicBaseURL != "" && cfg.Auth.OAuthCookieKey != "" {
			oauthHandler, err := oauthapi.New(logger, oauthapi.Config{
				HTTP: runtimecfg.HTTPConfig{
					APIPublicBaseURL:      cfg.APIPublicBaseURL,
					WebBaseURL:            cfg.WebBaseURL,
					OAuthCookieSigningKey: cfg.Auth.OAuthCookieKey,
				},
				Queries: db.New(pool),
			}, authService, nil)
			if err != nil {
				return fmt.Errorf("create oauth handler: %w", err)
			}
			mux.Handle("/oauth/", oauthHandler)
			mux.Handle("/.well-known/", oauthHandler)
		}
		mcpSrv := mcpserver.New(pool, cfg, logger, authService)
		mux.Handle("/mcp", mcpSrv)
		mux.Handle("/mcp/", mcpSrv)
		mux.Handle("/.well-known/openai-apps-challenge", mcpSrv)
		mux.Handle("/health", mcpSrv)
		apiConfig := huma.DefaultConfig("Koditon API", "0.1.0")
		auth.RegisterSecurityScheme(&apiConfig, cfg.APIPublicBaseURL)
		api := humago.New(mux, apiConfig)
		httpServer = &http.Server{
			Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
			Handler:           srv.Handler(mux, api),
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
			BaseContext: func(net.Listener) context.Context {
				return ctx
			},
		}
		errCh = make(chan error, 1)
		go func() {
			logging.With(appLogger, logging.Op("app.http.listen")).InfoContext(ctx, "server listening", "addr", httpServer.Addr)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
				return
			}
			errCh <- nil
		}()
	}
	select {
	case <-ctx.Done():
		logging.With(appLogger, logging.Op("app.shutdown")).InfoContext(ctx, "shutdown signal received")
	case err := <-errCh:
		if err != nil {
			if consumer != nil {
				consumer.Stop()
			}
			return fmt.Errorf("http server: %w", err)
		}
		if consumer != nil {
			consumer.Stop()
		}
		return nil
	}
	// graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	var shutdownErrs []error
	shutdownLogger := logging.With(appLogger, logging.Op("app.shutdown"))
	shutdownLogger.DebugContext(ctx, "stopping consumer")
	if consumer != nil {
		consumer.Stop()
		shutdownLogger.DebugContext(ctx, "consumer stopped")
	}
	if httpServer != nil {
		shutdownLogger.DebugContext(ctx, "shutting down http server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			shutdownLogger.ErrorContext(shutdownCtx, "http server shutdown failed", "error", err, "outcome", logging.OutcomeError)
			shutdownErrs = append(shutdownErrs, fmt.Errorf("http server shutdown: %w", err))
		} else {
			shutdownLogger.DebugContext(shutdownCtx, "http server stopped")
		}
		if err := <-errCh; err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("http server: %w", err))
		}
	}
	shutdownLogger.DebugContext(ctx, "closing database pool")
	pool.Close()
	shutdownLogger.DebugContext(ctx, "database pool closed")
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	shutdownLogger.InfoContext(ctx, "graceful shutdown complete", "outcome", logging.OutcomeSuccess)
	return nil
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return os.Getenv("TERM") != "dumb" &&
		(fd == os.Stdout.Fd() || fd == os.Stderr.Fd())
}

func newLogger(w io.Writer, cfg config.Config) *slog.Logger {
	isTTY := isTerminal(w)
	opts := &tint.Options{
		Level:      cfg.SlogLevel(),
		TimeFormat: "15:04:05",
		NoColor:    !isTTY,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey && len(groups) == 0 {
				return slog.Attr{Key: a.Key, Value: slog.StringValue(formatLevel(a.Value.Any().(slog.Level)))}
			}
			return a
		},
	}
	if cfg.Environment.IsDevelopment() {
		opts.AddSource = true
	}
	handler := slog.Handler(tint.NewHandler(w, opts))
	if cfg.Telegram.BotToken != "" && cfg.Telegram.ChatID != "" {
		handler = telegram.NewHandler(
			cfg.Telegram.BotToken,
			cfg.Telegram.ChatID,
			slog.LevelWarn,
			handler,
		)
	}
	handler = requestid.NewHandler(handler)
	return slog.New(handler)
}

func formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DBG"
	case slog.LevelInfo:
		return "INF"
	case slog.LevelWarn:
		return "WRN"
	case slog.LevelError:
		return "ERR"
	default:
		return level.String()
	}
}

type productionModeCheck struct {
	name    string
	appMode string
}

func runVerifyEnv(args []string) error {
	fs := flag.NewFlagSet("verify-env", flag.ContinueOnError)
	allModes := fs.Bool("all-modes", false, "Validate env for all production runtime profiles (consumer, api, consumer+api)")
	mode := fs.String("mode", "", "Validate env as the given mode (consumer, api, consumer,api)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *allModes && *mode != "" {
		return fmt.Errorf("--mode and --all-modes cannot be used together")
	}

	baseEnv := config.CurrentEnv()
	checks := []productionModeCheck{
		{name: "consumer,api", appMode: "consumer,api"},
		{name: "consumer", appMode: "consumer"},
		{name: "api", appMode: "api"},
	}

	switch {
	case *allModes:
		for _, check := range checks {
			if err := verifyEnvTarget(os.Stdout, check.name, mergedEnv(baseEnv, productionEnvOverrides(check))); err != nil {
				return err
			}
		}
	case *mode != "":
		check, err := productionCheckForMode(*mode)
		if err != nil {
			return err
		}
		if err := verifyEnvTarget(os.Stdout, check.name, mergedEnv(baseEnv, productionEnvOverrides(check))); err != nil {
			return err
		}
	default:
		if err := verifyEnvTarget(os.Stdout, "current env", baseEnv); err != nil {
			return err
		}
	}
	return nil
}

func verifyEnvTarget(stdout io.Writer, name string, values map[string]string) error {
	if _, err := config.FromEnvMap(values); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	_, err := fmt.Fprintf(stdout, "ok: %s\n", name)
	return err
}

func productionCheckForMode(raw string) (productionModeCheck, error) {
	mode := strings.TrimSpace(strings.ToLower(raw))
	switch mode {
	case "consumer,api", "api,consumer":
		return productionModeCheck{name: "consumer,api", appMode: "consumer,api"}, nil
	case "consumer":
		return productionModeCheck{name: "consumer", appMode: "consumer"}, nil
	case "api":
		return productionModeCheck{name: "api", appMode: "api"}, nil
	default:
		return productionModeCheck{}, fmt.Errorf("invalid mode %q: must be consumer, api, or consumer,api", raw)
	}
}

func productionEnvOverrides(check productionModeCheck) map[string]string {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = fmt.Sprintf(
			"postgres://postgres:%s@db:5432/koditon?sslmode=disable",
			strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD")),
		)
	}
	return map[string]string{
		"APP_HOST":             "0.0.0.0",
		"APP_PORT":             "8080",
		"APP_ENV":              "production",
		"APP_SHUTDOWN_TIMEOUT": "10s",
		"APP_MODE":             check.appMode,
		"LOG_LEVEL":            "info",
		"DATABASE_URL":         databaseURL,
	}
}

func mergedEnv(base, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overrides))
	maps.Copy(merged, base)
	maps.Copy(merged, overrides)
	return merged
}
