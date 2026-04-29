package app

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

	db "koditon/internal/db"
	"koditon/internal/domain/auth"
	"koditon/internal/domain/emailauth"
	"koditon/internal/platform/buildinfo"
	"koditon/internal/platform/config"
	"koditon/internal/platform/email"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/runtimecfg"
	"koditon/internal/platform/runtimekv"
	"koditon/internal/platform/schema"
	"koditon/internal/sync/consumers"
	"koditon/internal/sync/frontdoor"
	"koditon/internal/sync/postal"
	"koditon/internal/sync/prices"
	"koditon/internal/sync/shortcut"
	"koditon/internal/transport/health"
	server "koditon/internal/transport/httpserver"
	mcpserver "koditon/internal/transport/mcp"
	oauthapi "koditon/internal/transport/oauth"
	"koditon/internal/transport/telegram"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	openrouter "github.com/revrost/go-openrouter"
)

// Run starts the API and/or consumer application for the configured mode.
func Run(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) > 1 && args[1] == "verify-env" {
		return runVerifyEnv(args[2:], stdout, getenv)
	}
	return run(ctx, args, getenv, stdin, stdout, stderr)
}

func run(
	ctx context.Context,
	_ []string,
	_ func(string) string,
	_ io.Reader,
	_, stderr io.Writer,
) (runErr error) {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(stderr, cfg)
	appLogger := logging.With(logger.With("component", "app"), logging.Op("app.start"))
	info := buildinfo.Current()
	lifecycle := newLifecycle(appLogger)
	defer func() {
		if runErr != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cleanupCancel()
			runErr = errors.Join(runErr, lifecycle.Cleanup(cleanupCtx))
		}
	}()
	appLogger.InfoContext(ctx, "starting application",
		"env", cfg.Environment,
		"log_level", cfg.LogLevel,
		"mode", cfg.Mode.String(),
		"version", info.Version,
		"commit", info.Commit,
		"build_time", info.BuildTime,
	)
	pool, err := newDatabasePool(ctx, cfg)
	if err != nil {
		return err
	}
	lifecycle.Defer("database pool", func(context.Context) error {
		pool.Close()
		return nil
	})
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := schema.Check(ctx, db.New(pool)); err != nil {
		return fmt.Errorf("check schema: %w", err)
	}
	appLogger.DebugContext(ctx, "database connection established")
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
		consumer := consumers.New(
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
		lifecycle.Defer("consumer", func(context.Context) error {
			consumer.Stop()
			return nil
		})
	}
	var httpServer *http.Server
	var errCh chan error
	if cfg.Mode.API {
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
		keySetStore := runtimekv.New(db.New(pool))
		authService, err := auth.NewService(ctx, auth.ServiceConfig{
			Pool:             pool,
			Auth:             authCfg,
			AppleKeySetStore: keySetStore,
			Logger:           logger,
		})
		if err != nil {
			return fmt.Errorf("create auth service: %w", err)
		}
		lifecycle.Defer("auth service", func(context.Context) error {
			return authService.Close()
		})
		emailService := email.NewService(email.NewLoggerSender(logger.With("component", "email")))
		if cfg.Email.ResendAPIKey != "" {
			resendSender, err := email.NewResendSender(cfg.Email.ResendAPIKey, cfg.Email.ResendFromEmail, cfg.Email.ResendFromName)
			if err != nil {
				return fmt.Errorf("create resend email sender: %w", err)
			}
			emailService = email.NewService(resendSender)
		}
		emailAuthService := emailauth.NewService(emailauth.ServiceConfig{
			Logger:          logger,
			Queries:         db.New(pool),
			EmailService:    emailService,
			HTTP:            runtimecfg.HTTPConfig{APIPublicBaseURL: cfg.APIPublicBaseURL, WebBaseURL: cfg.WebBaseURL},
			EmitConsoleLink: cfg.Environment == config.EnvDevelopment,
		})
		srv := server.New(logger, cfg, pool, authService, emailAuthService)
		mux := http.NewServeMux()
		health.New(pool, cfg.Mode, info).Register(mux)
		var oauthHandler *oauthapi.Handler
		if cfg.APIPublicBaseURL != "" && cfg.Auth.OAuthCookieKey != "" {
			oauthHandler, err = oauthapi.New(logger, oauthapi.Config{
				HTTP: runtimecfg.HTTPConfig{
					APIPublicBaseURL:      cfg.APIPublicBaseURL,
					WebBaseURL:            cfg.WebBaseURL,
					OAuthCookieSigningKey: cfg.Auth.OAuthCookieKey,
				},
				Queries:   db.New(pool),
				EmailAuth: emailAuthService,
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
		apiConfig := huma.DefaultConfig("Koditon API", "0.1.0")
		auth.RegisterSecurityScheme(&apiConfig, cfg.APIPublicBaseURL)
		api := humago.New(mux, apiConfig)
		if oauthHandler != nil {
			oauthHandler.RegisterRoutes(api)
		}
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
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	var shutdownErrs []error
	shutdownLogger := logging.With(appLogger, logging.Op("app.shutdown"))
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
	if err := lifecycle.Cleanup(shutdownCtx); err != nil {
		shutdownErrs = append(shutdownErrs, err)
	}
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	shutdownLogger.InfoContext(ctx, "graceful shutdown complete", "outcome", logging.OutcomeSuccess)
	return nil
}

func newDatabasePool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database pool config: %w", err)
	}
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns
	poolCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.Database.HealthCheckPeriod
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}

type lifecycleCleanup struct {
	name string
	fn   func(context.Context) error
}

type lifecycleStack struct {
	logger   *slog.Logger
	cleanups []lifecycleCleanup
	closed   bool
}

func newLifecycle(logger *slog.Logger) *lifecycleStack {
	return &lifecycleStack{logger: logging.With(logger, logging.Op("app.cleanup"))}
}

func (l *lifecycleStack) Defer(name string, fn func(context.Context) error) {
	l.cleanups = append(l.cleanups, lifecycleCleanup{name: name, fn: fn})
}

func (l *lifecycleStack) Cleanup(ctx context.Context) error {
	if l.closed {
		return nil
	}
	l.closed = true
	var errs []error
	for i := len(l.cleanups) - 1; i >= 0; i-- {
		cleanup := l.cleanups[i]
		l.logger.DebugContext(ctx, "running cleanup", "resource", cleanup.name)
		if err := cleanup.fn(ctx); err != nil {
			l.logger.ErrorContext(ctx, "cleanup failed", "resource", cleanup.name, "error", err, "outcome", logging.OutcomeError)
			errs = append(errs, fmt.Errorf("%s cleanup: %w", cleanup.name, err))
		}
	}
	return errors.Join(errs...)
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

func runVerifyEnv(args []string, stdout io.Writer, getenv func(string) string) error {
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
			if err := verifyEnvTarget(stdout, check.name, mergedEnv(baseEnv, productionEnvOverrides(check, getenv))); err != nil {
				return err
			}
		}
	case *mode != "":
		check, err := productionCheckForMode(*mode)
		if err != nil {
			return err
		}
		if err := verifyEnvTarget(stdout, check.name, mergedEnv(baseEnv, productionEnvOverrides(check, getenv))); err != nil {
			return err
		}
	default:
		if err := verifyEnvTarget(stdout, "current env", baseEnv); err != nil {
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

func productionEnvOverrides(check productionModeCheck, getenv func(string) string) map[string]string {
	databaseURL := strings.TrimSpace(getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = fmt.Sprintf(
			"postgres://postgres:%s@db:5432/koditon?sslmode=disable",
			strings.TrimSpace(getenv("POSTGRES_PASSWORD")),
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
