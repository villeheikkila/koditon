package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func (e Environment) IsDevelopment() bool {
	return e == EnvDevelopment
}

type Config struct {
	Host            string        `env:"APP_HOST,required"`
	Port            string        `env:"APP_PORT,required"`
	ShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT,required"`
	Environment     Environment   `env:"APP_ENV" envDefault:"development"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	Mode            AppMode       `env:"APP_MODE" envDefault:"consumer,api"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	Auth            AuthConfig
	Prices          PricesConfig
	Shortcut        ShortcutConfig
	Frontdoor       FrontdoorConfig
	OpenRouter      OpenRouterConfig
	Telegram        TelegramConfig
}

func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type AppMode struct {
	Consumer bool
	API      bool
}

func (m AppMode) String() string {
	parts := make([]string, 0, 2)
	if m.Consumer {
		parts = append(parts, "consumer")
	}
	if m.API {
		parts = append(parts, "api")
	}
	return strings.Join(parts, ",")
}

func (m *AppMode) UnmarshalText(text []byte) error {
	raw := strings.TrimSpace(strings.ToLower(string(text)))
	if raw == "" {
		return errors.New("app mode must include consumer or api")
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(parts) == 0 {
		return errors.New("app mode must include consumer or api")
	}
	var consumerSet, apiSet bool
	for _, part := range parts {
		switch part {
		case "consumer":
			consumerSet = true
		case "api":
			apiSet = true
		default:
			return fmt.Errorf("invalid app mode: %s", part)
		}
	}
	if !consumerSet && !apiSet {
		return errors.New("app mode must include consumer or api")
	}
	m.Consumer = consumerSet
	m.API = apiSet
	return nil
}

type PricesConfig struct {
	BaseURL string `env:"PRICES_BASE_URL,required"`
}

type ShortcutConfig struct {
	BaseURL     string `env:"SHORTCUT_BASE_URL,required"`
	DocsBaseURL string `env:"SHORTCUT_DOCS_BASE_URL,required"`
	AdBaseURL   string `env:"SHORTCUT_AD_BASE_URL,required"`
	UserAgent   string `env:"SHORTCUT_USER_AGENT,required"`
	SitemapBase string `env:"SHORTCUT_SITEMAP_BASE_URL,required"`
}

type FrontdoorConfig struct {
	BaseURL     string `env:"FRONTDOOR_BASE_URL,required"`
	UserAgent   string `env:"FRONTDOOR_USER_AGENT,required"`
	Cookie      string `env:"FRONTDOOR_COOKIE,required"`
	SitemapBase string `env:"FRONTDOOR_SITEMAP_BASE_URL,required"`
}

type OpenRouterConfig struct {
	APIKey string `env:"OPENROUTER_API_KEY,required"`
}

type TelegramConfig struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`
	ChatID   string `env:"TELEGRAM_CHAT_ID"`
}

type AuthConfig struct {
	JWTSigningKey string `env:"AUTH_JWT_SIGNING_KEY"`
	JWTIssuer     string `env:"AUTH_JWT_ISSUER" envDefault:"koditon"`
	Apple         AppleAuthConfig
}

type AppleAuthConfig struct {
	BundleID     string `env:"AUTH_APPLE_BUNDLE_ID"`
	TeamID       string `env:"AUTH_APPLE_TEAM_ID"`
	PrivateKeyID string `env:"AUTH_APPLE_PRIVATE_KEY_ID"`
	PrivateKey   string `env:"AUTH_APPLE_PRIVATE_KEY"`
}

func (c AppleAuthConfig) IsConfigured() bool {
	return c.BundleID != "" && c.TeamID != "" && c.PrivateKeyID != "" && c.PrivateKey != ""
}

func Load() (Config, error) {
	_ = godotenv.Load(".env.local", ".env")
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
