package config

import (
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
	DB              DBConfig      `envPrefix:"DB_"`
	Prices          PricesConfig
	Shortcut        ShortcutConfig
	Frontdoor       FrontdoorConfig
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

type DBConfig struct {
	Host     string `env:"HOST,required"`
	Port     string `env:"PORT,required"`
	User     string `env:"USER,required"`
	Password string `env:"PASSWORD,required"`
	Name     string `env:"NAME,required"`
	SSLMode  string `env:"SSLMODE,required"`
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

func Load() (Config, error) {
	_ = godotenv.Load(".env.local", ".env")
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Name, c.DB.SSLMode)
}
