package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"koditon-go/internal/auth"
	"koditon-go/internal/auth/apple"
	"koditon-go/internal/config"
	"koditon-go/internal/consumers"
	"koditon-go/internal/frontdoor"
	"koditon-go/internal/postal"
	"koditon-go/internal/prices"
	"koditon-go/internal/server"
	"koditon-go/internal/shortcut"
	"koditon-go/internal/telegram"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	openrouter "github.com/revrost/go-openrouter"
)

func main() {
	ctx := context.Background()
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
	slog.SetDefault(logger)
	appLogger := logger.With("component", "app")
	appLogger.Info("starting application",
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
	appLogger.Debug("database connection established")
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
		var appleConfig *apple.Config
		if cfg.Auth.Apple.IsConfigured() {
			appleConfig = &apple.Config{
				BundleID:     cfg.Auth.Apple.BundleID,
				TeamID:       cfg.Auth.Apple.TeamID,
				PrivateKeyID: cfg.Auth.Apple.PrivateKeyID,
				PrivateKey:   cfg.Auth.Apple.PrivateKey,
			}
		}
		authService, err := auth.NewService(ctx, auth.ServiceConfig{
			Pool: pool,
			JWT: auth.JWTConfig{
				SigningKey: cfg.Auth.JWTSigningKey,
				Issuer:     cfg.Auth.JWTIssuer,
			},
			Apple:  appleConfig,
			Logger: logger,
		})
		if err != nil {
			return fmt.Errorf("create auth service: %w", err)
		}
		srv := server.New(logger, cfg, pool, authService)
		mux := http.NewServeMux()
		apiConfig := huma.DefaultConfig("Koditon API", "0.1.0")
		auth.RegisterSecurityScheme(&apiConfig)
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
			appLogger.Info("server listening", "addr", httpServer.Addr)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
				return
			}
			errCh <- nil
		}()
	}
	select {
	case <-ctx.Done():
		appLogger.Info("shutdown signal received")
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
	appLogger.Debug("stopping consumer")
	if consumer != nil {
		consumer.Stop()
		appLogger.Debug("consumer stopped")
	}
	if httpServer != nil {
		appLogger.Debug("shutting down http server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			appLogger.Error("http server shutdown failed", tint.Err(err))
			shutdownErrs = append(shutdownErrs, fmt.Errorf("http server shutdown: %w", err))
		} else {
			appLogger.Debug("http server stopped")
		}
		if err := <-errCh; err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("http server: %w", err))
		}
	}
	appLogger.Debug("closing database pool")
	pool.Close()
	appLogger.Debug("database pool closed")
	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	appLogger.Info("graceful shutdown complete")
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
