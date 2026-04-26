package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
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

func ParseEnvironment(raw string) (Environment, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case string(EnvDevelopment):
		return EnvDevelopment, nil
	case string(EnvProduction):
		return EnvProduction, nil
	default:
		if value == "" {
			return "", errors.New("environment is required")
		}
		return "", fmt.Errorf("invalid environment: %s", raw)
	}
}

func (e *Environment) UnmarshalText(text []byte) error {
	parsed, err := ParseEnvironment(string(text))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

// rawConfig is the flat env-var struct used for parsing. All env tags live here.
type rawConfig struct {
	Host            string        `env:"APP_HOST,required"`
	Port            string        `env:"APP_PORT,required"`
	ShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT,required"`
	Environment     Environment   `env:"APP_ENV,required"`
	LogLevel        string        `env:"LOG_LEVEL,required"`
	Mode            AppMode       `env:"APP_MODE,required"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`

	AuthJWTSigningKey    string        `env:"AUTH_JWT_SIGNING_KEY,required"`
	AuthJWTIssuer        string        `env:"AUTH_JWT_ISSUER,required"`
	AuthJWTUIDHashSalt   string        `env:"AUTH_UID_HASH_SALT" envDefault:""`
	AuthOAuthCookieKey   string        `env:"AUTH_OAUTH_COOKIE_SIGNING_KEY" envDefault:""`
	AuthOAuthATTL        time.Duration `env:"AUTH_OAUTH_ACCESS_TOKEN_TTL" envDefault:"15m"`
	AuthOAuthRTTL        time.Duration `env:"AUTH_OAUTH_REFRESH_TOKEN_TTL" envDefault:"8760h"`
	AuthPasskeyRPName    string        `env:"AUTH_PASSKEY_RP_DISPLAY_NAME" envDefault:"Koditon"`
	AuthPasskeyRPID      string        `env:"AUTH_PASSKEY_RP_ID" envDefault:"localhost"`
	AuthPasskeyRPOrigins string        `env:"AUTH_PASSKEY_RP_ORIGINS" envDefault:"http://localhost:3000"`

	AuthAppleBundleID       string `env:"AUTH_APPLE_BUNDLE_ID,required"`
	AuthAppleTeamID         string `env:"AUTH_APPLE_TEAM_ID,required"`
	AuthApplePrivateKeyID   string `env:"AUTH_APPLE_PRIVATE_KEY_ID,required"`
	AuthApplePrivateKey     string `env:"AUTH_APPLE_PRIVATE_KEY,required"`
	AuthAppleWebServiceID   string `env:"AUTH_APPLE_WEB_SERVICE_ID" envDefault:""`
	AuthAppleWebRedirectURI string `env:"AUTH_APPLE_WEB_REDIRECT_URI" envDefault:""`

	PricesBaseURL string `env:"PRICES_BASE_URL,required"`

	ShortcutBaseURL     string `env:"SHORTCUT_BASE_URL,required"`
	ShortcutDocsBaseURL string `env:"SHORTCUT_DOCS_BASE_URL,required"`
	ShortcutAdBaseURL   string `env:"SHORTCUT_AD_BASE_URL,required"`
	ShortcutUserAgent   string `env:"SHORTCUT_USER_AGENT,required"`
	ShortcutSitemapBase string `env:"SHORTCUT_SITEMAP_BASE_URL,required"`

	FrontdoorBaseURL     string `env:"FRONTDOOR_BASE_URL,required"`
	FrontdoorUserAgent   string `env:"FRONTDOOR_USER_AGENT,required"`
	FrontdoorCookie      string `env:"FRONTDOOR_COOKIE,required"`
	FrontdoorSitemapBase string `env:"FRONTDOOR_SITEMAP_BASE_URL,required"`

	OpenRouterAPIKey string `env:"OPENROUTER_API_KEY,required"`

	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramChatID   string `env:"TELEGRAM_CHAT_ID,required"`

	RedisAddr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`

	TelemetryOTLPEndpoint string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	TelemetryServiceName  string  `env:"OTEL_SERVICE_NAME" envDefault:"koditon"`
	TelemetryOTLPProtocol string  `env:"OTEL_EXPORTER_OTLP_PROTOCOL" envDefault:"grpc"`
	TelemetryOTLPInsecure bool    `env:"OTEL_EXPORTER_OTLP_INSECURE" envDefault:"false"`
	TelemetrySampleRatio  float64 `env:"OTEL_SAMPLE_RATIO" envDefault:"1.0"`

	WebBaseURL               string `env:"WEB_BASE_URL" envDefault:""`
	WebStaticDir             string `env:"WEB_STATIC_DIR" envDefault:""`
	MCPAuthToken             string `env:"MCP_AUTH_TOKEN" envDefault:""`
	APIPublicBaseURL         string `env:"API_PUBLIC_BASE_URL" envDefault:""`
	OpenAIAppsChallengeToken string `env:"OPENAI_APPS_CHALLENGE_TOKEN" envDefault:""`
	CORSAllowedOrigins       string `env:"CORS_ALLOWED_ORIGINS" envDefault:""`
}

func (r *rawConfig) sanitize() {
	r.AuthJWTSigningKey = sanitizeSecretValue(r.AuthJWTSigningKey)
	r.AuthOAuthCookieKey = sanitizeSecretValue(r.AuthOAuthCookieKey)
	r.OpenRouterAPIKey = sanitizeSecretValue(r.OpenRouterAPIKey)
	r.TelegramBotToken = sanitizeSecretValue(r.TelegramBotToken)
	r.FrontdoorCookie = sanitizeSecretValue(r.FrontdoorCookie)

	if r.AuthApplePrivateKey != "" {
		r.AuthApplePrivateKey = strings.Trim(r.AuthApplePrivateKey, "\"")
		r.AuthApplePrivateKey = strings.ReplaceAll(r.AuthApplePrivateKey, "\\n", "\n")
	}
}

func (r rawConfig) toConfig() Config {
	return Config{
		Host:            r.Host,
		Port:            r.Port,
		ShutdownTimeout: r.ShutdownTimeout,
		Environment:     r.Environment,
		LogLevel:        r.LogLevel,
		Mode:            r.Mode,
		DatabaseURL:     r.DatabaseURL,
		Auth: AuthConfig{
			JWTSigningKey:    r.AuthJWTSigningKey,
			JWTIssuer:        r.AuthJWTIssuer,
			JWTUIDHashSalt:   r.AuthJWTUIDHashSalt,
			OAuthCookieKey:   r.AuthOAuthCookieKey,
			OAuthATTL:        r.AuthOAuthATTL,
			OAuthRTTL:        r.AuthOAuthRTTL,
			PasskeyRPName:    r.AuthPasskeyRPName,
			PasskeyRPID:      r.AuthPasskeyRPID,
			PasskeyRPOrigins: r.AuthPasskeyRPOrigins,
			Apple: AppleAuthConfig{
				BundleID:       r.AuthAppleBundleID,
				TeamID:         r.AuthAppleTeamID,
				PrivateKeyID:   r.AuthApplePrivateKeyID,
				PrivateKey:     r.AuthApplePrivateKey,
				WebServiceID:   r.AuthAppleWebServiceID,
				WebRedirectURI: r.AuthAppleWebRedirectURI,
			},
		},
		Prices: PricesConfig{
			BaseURL: r.PricesBaseURL,
		},
		Shortcut: ShortcutConfig{
			BaseURL:     r.ShortcutBaseURL,
			DocsBaseURL: r.ShortcutDocsBaseURL,
			AdBaseURL:   r.ShortcutAdBaseURL,
			UserAgent:   r.ShortcutUserAgent,
			SitemapBase: r.ShortcutSitemapBase,
		},
		Frontdoor: FrontdoorConfig{
			BaseURL:     r.FrontdoorBaseURL,
			UserAgent:   r.FrontdoorUserAgent,
			Cookie:      r.FrontdoorCookie,
			SitemapBase: r.FrontdoorSitemapBase,
		},
		OpenRouter: OpenRouterConfig{
			APIKey: r.OpenRouterAPIKey,
		},
		Telegram: TelegramConfig{
			BotToken: r.TelegramBotToken,
			ChatID:   r.TelegramChatID,
		},
		Redis: RedisConfig{
			Addr: r.RedisAddr,
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint: r.TelemetryOTLPEndpoint,
			ServiceName:  r.TelemetryServiceName,
			OTLPProtocol: r.TelemetryOTLPProtocol,
			OTLPInsecure: r.TelemetryOTLPInsecure,
			SampleRatio:  r.TelemetrySampleRatio,
		},
		WebBaseURL:               r.WebBaseURL,
		WebStaticDir:             r.WebStaticDir,
		MCPAuthToken:             r.MCPAuthToken,
		APIPublicBaseURL:         r.APIPublicBaseURL,
		OpenAIAppsChallengeToken: r.OpenAIAppsChallengeToken,
		CORSAllowedOrigins:       r.CORSAllowedOrigins,
	}
}

// Config is the structured application configuration. Its shape is stable; all
// env-var parsing details live in rawConfig.
type Config struct {
	Host                     string
	Port                     string
	ShutdownTimeout          time.Duration
	Environment              Environment
	LogLevel                 string
	Mode                     AppMode
	DatabaseURL              string
	Auth                     AuthConfig
	Prices                   PricesConfig
	Shortcut                 ShortcutConfig
	Frontdoor                FrontdoorConfig
	OpenRouter               OpenRouterConfig
	Telegram                 TelegramConfig
	Redis                    RedisConfig
	Telemetry                TelemetryConfig
	WebBaseURL               string
	WebStaticDir             string
	MCPAuthToken             string
	APIPublicBaseURL         string
	OpenAIAppsChallengeToken string
	CORSAllowedOrigins       string
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
	BaseURL string
}

type ShortcutConfig struct {
	BaseURL     string
	DocsBaseURL string
	AdBaseURL   string
	UserAgent   string
	SitemapBase string
}

type FrontdoorConfig struct {
	BaseURL     string
	UserAgent   string
	Cookie      string
	SitemapBase string
}

type OpenRouterConfig struct {
	APIKey string
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type AuthConfig struct {
	JWTSigningKey    string
	JWTIssuer        string
	JWTUIDHashSalt   string
	OAuthCookieKey   string
	OAuthATTL        time.Duration
	OAuthRTTL        time.Duration
	PasskeyRPName    string
	PasskeyRPID      string
	PasskeyRPOrigins string
	Apple            AppleAuthConfig
}

type AppleAuthConfig struct {
	BundleID       string
	TeamID         string
	PrivateKeyID   string
	PrivateKey     string
	WebServiceID   string
	WebRedirectURI string
}

func (c AppleAuthConfig) IsConfigured() bool {
	return c.BundleID != "" && c.TeamID != "" && c.PrivateKeyID != "" && c.PrivateKey != ""
}

type RedisConfig struct {
	Addr string
}

type TelemetryConfig struct {
	OTLPEndpoint string
	ServiceName  string
	OTLPProtocol string
	OTLPInsecure bool
	SampleRatio  float64
}

// CurrentEnv captures the current process environment as a map.
func CurrentEnv() map[string]string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	return values
}

// FromEnvMap parses configuration from an explicit env map. Use this in tests.
func FromEnvMap(values map[string]string) (Config, error) {
	var raw rawConfig
	if err := env.ParseWithOptions(&raw, env.Options{Environment: values}); err != nil {
		return Config{}, err
	}
	raw.sanitize()
	return raw.toConfig(), nil
}

// FromEnv parses configuration from the current process environment.
func FromEnv() (Config, error) {
	return FromEnvMap(CurrentEnv())
}

// Load loads .env.local and .env files then parses configuration from the
// resulting environment. Use FromEnvMap in tests instead.
func Load() (Config, error) {
	_ = godotenv.Load(".env.local", ".env")
	return FromEnv()
}

func sanitizeSecretValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
	}
	return trimmed
}
