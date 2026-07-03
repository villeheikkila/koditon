package config

import (
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"koditon/internal/platform/httpratelimit"

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
	DBMaxConns      int32         `env:"DB_MAX_CONNS" envDefault:"10"`
	DBMinConns      int32         `env:"DB_MIN_CONNS" envDefault:"2"`
	DBMaxLifetime   time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"30m"`
	DBMaxIdleTime   time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"5m"`
	DBHealthPeriod  time.Duration `env:"DB_HEALTH_CHECK_PERIOD" envDefault:"1m"`

	TelemetryEnabled          bool    `env:"OTEL_ENABLED"`
	TelemetrySDKDisabled      bool    `env:"OTEL_SDK_DISABLED"`
	TelemetryServiceName      string  `env:"OTEL_SERVICE_NAME" envDefault:""`
	TelemetryServiceVersion   string  `env:"OTEL_SERVICE_VERSION" envDefault:""`
	TelemetryServiceInstance  string  `env:"OTEL_SERVICE_INSTANCE_ID" envDefault:""`
	TelemetryExporterEndpoint string  `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	TelemetryExporterProtocol string  `env:"OTEL_EXPORTER_OTLP_PROTOCOL" envDefault:""`
	TelemetryExporterHeaders  string  `env:"OTEL_EXPORTER_OTLP_HEADERS" envDefault:""`
	TelemetryOTLPEndpoint     string  `env:"OTEL_OTLP_ENDPOINT" envDefault:""`
	TelemetryOTLPProtocol     string  `env:"OTEL_OTLP_PROTOCOL" envDefault:""`
	TelemetryOTLPInsecure     bool    `env:"OTEL_OTLP_INSECURE"`
	TelemetryTracesSampler    string  `env:"OTEL_TRACES_SAMPLER" envDefault:""`
	TelemetryTracesSamplerArg string  `env:"OTEL_TRACES_SAMPLER_ARG" envDefault:""`
	TelemetrySampleRatio      float64 `env:"OTEL_SAMPLE_RATIO"`

	AuthJWTSigningKey    string        `env:"AUTH_JWT_SIGNING_KEY" envDefault:""`
	AuthJWTIssuer        string        `env:"AUTH_JWT_ISSUER" envDefault:""`
	AuthJWTUIDHashSalt   string        `env:"AUTH_UID_HASH_SALT" envDefault:""`
	AuthOAuthCookieKey   string        `env:"AUTH_OAUTH_COOKIE_SIGNING_KEY" envDefault:""`
	AuthOAuthATTL        time.Duration `env:"AUTH_OAUTH_ACCESS_TOKEN_TTL" envDefault:"15m"`
	AuthOAuthRTTL        time.Duration `env:"AUTH_OAUTH_REFRESH_TOKEN_TTL" envDefault:"8760h"`
	AuthPasskeyRPName    string        `env:"AUTH_PASSKEY_RP_DISPLAY_NAME" envDefault:"Koditon"`
	AuthPasskeyRPID      string        `env:"AUTH_PASSKEY_RP_ID" envDefault:""`
	AuthPasskeyRPOrigins string        `env:"AUTH_PASSKEY_RP_ORIGINS" envDefault:""`

	AuthAppleBundleID       string `env:"AUTH_APPLE_BUNDLE_ID" envDefault:""`
	AuthAppleTeamID         string `env:"AUTH_APPLE_TEAM_ID" envDefault:""`
	AuthApplePrivateKeyID   string `env:"AUTH_APPLE_PRIVATE_KEY_ID" envDefault:""`
	AuthApplePrivateKey     string `env:"AUTH_APPLE_PRIVATE_KEY" envDefault:""`
	AuthAppleWebServiceID   string `env:"AUTH_APPLE_WEB_SERVICE_ID" envDefault:""`
	AuthAppleWebRedirectURI string `env:"AUTH_APPLE_WEB_REDIRECT_URI" envDefault:""`

	ShortcutBaseURL        string  `env:"SHORTCUT_BASE_URL" envDefault:""`
	ShortcutDocsBaseURL    string  `env:"SHORTCUT_DOCS_BASE_URL" envDefault:""`
	ShortcutAdBaseURL      string  `env:"SHORTCUT_AD_BASE_URL" envDefault:""`
	ShortcutUserAgent      string  `env:"SHORTCUT_USER_AGENT" envDefault:""`
	ShortcutSitemapBase    string  `env:"SHORTCUT_SITEMAP_BASE_URL" envDefault:""`
	ShortcutRateLimitRPS   float64 `env:"SHORTCUT_RATE_LIMIT_RPS" envDefault:"2"`
	ShortcutRateLimitBurst int     `env:"SHORTCUT_RATE_LIMIT_BURST" envDefault:"1"`

	FrontdoorBaseURL        string  `env:"FRONTDOOR_BASE_URL" envDefault:""`
	FrontdoorUserAgent      string  `env:"FRONTDOOR_USER_AGENT" envDefault:""`
	FrontdoorCookie         string  `env:"FRONTDOOR_COOKIE" envDefault:""`
	FrontdoorSitemapBase    string  `env:"FRONTDOOR_SITEMAP_BASE_URL" envDefault:""`
	FrontdoorRateLimitRPS   float64 `env:"FRONTDOOR_RATE_LIMIT_RPS" envDefault:"2"`
	FrontdoorRateLimitBurst int     `env:"FRONTDOOR_RATE_LIMIT_BURST" envDefault:"1"`

	OpenRouterAPIKey              string `env:"OPENROUTER_API_KEY" envDefault:""`
	OpenAIAPIKey                  string `env:"OPENAI_API_KEY" envDefault:""`
	OpenAIManagerCertificateModel string `env:"OPENAI_MANAGER_CERTIFICATE_MODEL" envDefault:"gpt-5"`

	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN" envDefault:""`
	TelegramChatID   string `env:"TELEGRAM_CHAT_ID" envDefault:""`
	ResendAPIKey     string `env:"RESEND_API_KEY" envDefault:""`
	ResendFromEmail  string `env:"RESEND_FROM_EMAIL" envDefault:""`
	ResendFromName   string `env:"RESEND_FROM_NAME" envDefault:"Koditon"`

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
	r.OpenAIAPIKey = sanitizeSecretValue(r.OpenAIAPIKey)
	r.TelegramBotToken = sanitizeSecretValue(r.TelegramBotToken)
	r.ResendAPIKey = sanitizeSecretValue(r.ResendAPIKey)
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
		Database: DatabaseConfig{
			MaxConns:          r.DBMaxConns,
			MinConns:          r.DBMinConns,
			MaxConnLifetime:   r.DBMaxLifetime,
			MaxConnIdleTime:   r.DBMaxIdleTime,
			HealthCheckPeriod: r.DBHealthPeriod,
		},
		Telemetry: r.telemetryConfig(),
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
		Shortcut: ShortcutConfig{
			BaseURL:     r.ShortcutBaseURL,
			DocsBaseURL: r.ShortcutDocsBaseURL,
			AdBaseURL:   r.ShortcutAdBaseURL,
			UserAgent:   r.ShortcutUserAgent,
			SitemapBase: r.ShortcutSitemapBase,
			RateLimit: httpratelimit.Config{
				RequestsPerSecond: r.ShortcutRateLimitRPS,
				Burst:             r.ShortcutRateLimitBurst,
			},
		},
		Frontdoor: FrontdoorConfig{
			BaseURL:     r.FrontdoorBaseURL,
			UserAgent:   r.FrontdoorUserAgent,
			Cookie:      r.FrontdoorCookie,
			SitemapBase: r.FrontdoorSitemapBase,
			RateLimit: httpratelimit.Config{
				RequestsPerSecond: r.FrontdoorRateLimitRPS,
				Burst:             r.FrontdoorRateLimitBurst,
			},
		},
		OpenRouter: OpenRouterConfig{
			APIKey: r.OpenRouterAPIKey,
		},
		OpenAI: OpenAIConfig{
			APIKey:                  r.OpenAIAPIKey,
			ManagerCertificateModel: r.OpenAIManagerCertificateModel,
		},
		Telegram: TelegramConfig{
			BotToken: r.TelegramBotToken,
			ChatID:   r.TelegramChatID,
		},
		Email: EmailConfig{
			ResendAPIKey:    r.ResendAPIKey,
			ResendFromEmail: r.ResendFromEmail,
			ResendFromName:  r.ResendFromName,
		},
		WebBaseURL:               r.WebBaseURL,
		WebStaticDir:             r.WebStaticDir,
		MCPAuthToken:             r.MCPAuthToken,
		APIPublicBaseURL:         r.APIPublicBaseURL,
		OpenAIAppsChallengeToken: r.OpenAIAppsChallengeToken,
		CORSAllowedOrigins:       r.CORSAllowedOrigins,
	}
}

func (r rawConfig) telemetryConfig() *TelemetryConfig {
	if !r.telemetryConfigured() {
		return nil
	}
	return &TelemetryConfig{
		ServiceName:       r.TelemetryServiceName,
		ServiceVersion:    r.TelemetryServiceVersion,
		ServiceInstanceID: r.TelemetryServiceInstance,
		OTLPEndpoint: firstNonEmpty(
			r.TelemetryExporterEndpoint,
			r.TelemetryOTLPEndpoint,
		),
		OTLPProtocol: firstNonEmpty(
			r.TelemetryExporterProtocol,
			r.TelemetryOTLPProtocol,
		),
		OTLPInsecure: r.TelemetryOTLPInsecure,
		OTLPHeaders:  parseOTLPHeaders(r.TelemetryExporterHeaders),
		Sampler:      r.TelemetryTracesSampler,
		SamplerArg:   r.TelemetryTracesSamplerArg,
		SampleRatio:  r.telemetrySampleRatio(),
	}
}

func (r rawConfig) telemetryConfigured() bool {
	if r.TelemetrySDKDisabled {
		return false
	}
	return r.TelemetryEnabled ||
		strings.TrimSpace(r.TelemetryServiceName) != "" ||
		strings.TrimSpace(r.TelemetryServiceVersion) != "" ||
		strings.TrimSpace(r.TelemetryServiceInstance) != "" ||
		strings.TrimSpace(r.TelemetryExporterEndpoint) != "" ||
		strings.TrimSpace(r.TelemetryExporterProtocol) != "" ||
		strings.TrimSpace(r.TelemetryExporterHeaders) != "" ||
		strings.TrimSpace(r.TelemetryOTLPEndpoint) != "" ||
		strings.TrimSpace(r.TelemetryOTLPProtocol) != "" ||
		strings.TrimSpace(r.TelemetryTracesSampler) != "" ||
		strings.TrimSpace(r.TelemetryTracesSamplerArg) != ""
}

func (r rawConfig) telemetrySampleRatio() float64 {
	if strings.TrimSpace(r.TelemetryTracesSamplerArg) != "" {
		return 1
	}
	if r.TelemetrySampleRatio != 0 {
		return r.TelemetrySampleRatio
	}
	return 1
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseOTLPHeaders(value string) map[string]string {
	headers := map[string]string{}
	for part := range strings.SplitSeq(value, ",") {
		key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key != "" && val != "" {
			headers[key] = val
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
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
	Database                 DatabaseConfig
	Telemetry                *TelemetryConfig
	Auth                     AuthConfig
	Shortcut                 ShortcutConfig
	Frontdoor                FrontdoorConfig
	OpenRouter               OpenRouterConfig
	OpenAI                   OpenAIConfig
	Telegram                 TelegramConfig
	Email                    EmailConfig
	WebBaseURL               string
	WebStaticDir             string
	MCPAuthToken             string
	APIPublicBaseURL         string
	OpenAIAppsChallengeToken string
	CORSAllowedOrigins       string
}

type DatabaseConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type TelemetryConfig struct {
	ServiceName       string
	ServiceVersion    string
	ServiceInstanceID string
	OTLPEndpoint      string
	OTLPProtocol      string
	OTLPInsecure      bool
	OTLPHeaders       map[string]string
	Sampler           string
	SamplerArg        string
	SampleRatio       float64
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

type ShortcutConfig struct {
	BaseURL     string
	DocsBaseURL string
	AdBaseURL   string
	UserAgent   string
	SitemapBase string
	RateLimit   httpratelimit.Config
}

type FrontdoorConfig struct {
	BaseURL     string
	UserAgent   string
	Cookie      string
	SitemapBase string
	RateLimit   httpratelimit.Config
}

type OpenRouterConfig struct {
	APIKey string
}

type OpenAIConfig struct {
	APIKey                  string
	ManagerCertificateModel string
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type EmailConfig struct {
	ResendAPIKey    string
	ResendFromEmail string
	ResendFromName  string
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

func (c Config) Validate() error {
	var errs []error
	requireValue(&errs, "APP_HOST", c.Host)
	requireValue(&errs, "APP_PORT", c.Port)
	requireValue(&errs, "DATABASE_URL", c.DatabaseURL)
	if port, err := strconv.Atoi(strings.TrimSpace(c.Port)); err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be a valid TCP port"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("APP_SHUTDOWN_TIMEOUT must be positive"))
	}
	if c.Database.MaxConns <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS must be positive"))
	}
	if c.Database.MinConns < 0 {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS must not be negative"))
	}
	if c.Database.MinConns > c.Database.MaxConns {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS"))
	}
	if c.Database.MaxConnLifetime <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONN_LIFETIME must be positive"))
	}
	if c.Database.MaxConnIdleTime <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONN_IDLE_TIME must be positive"))
	}
	if c.Database.HealthCheckPeriod <= 0 {
		errs = append(errs, fmt.Errorf("DB_HEALTH_CHECK_PERIOD must be positive"))
	}
	if !isValidLogLevel(c.LogLevel) {
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be debug, info, warn, warning, or error"))
	}
	validateURL(&errs, "DATABASE_URL", c.DatabaseURL, "postgres", "postgresql")
	validateRateLimit(&errs, "SHORTCUT", c.Shortcut.RateLimit)
	validateRateLimit(&errs, "FRONTDOOR", c.Frontdoor.RateLimit)
	c.validateTelemetry(&errs)
	if c.Mode.API {
		c.validateAPI(&errs)
	}
	if c.Mode.Consumer {
		c.validateConsumer(&errs)
	}
	validateOptionalURL(&errs, "WEB_BASE_URL", c.WebBaseURL, "http", "https")
	validateOptionalURL(&errs, "API_PUBLIC_BASE_URL", c.APIPublicBaseURL, "http", "https")
	validateCORSOrigins(&errs, c.CORSAllowedOrigins)
	return errors.Join(errs...)
}

func (c Config) validateTelemetry(errs *[]error) {
	if c.Telemetry == nil {
		return
	}
	protocol := strings.ToLower(strings.TrimSpace(c.Telemetry.OTLPProtocol))
	switch protocol {
	case "", "http", "http/protobuf", "grpc":
	default:
		*errs = append(*errs, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf, http, or grpc"))
	}
	if protocol == "grpc" {
		validateOptionalHostPort(errs, "OTEL_EXPORTER_OTLP_ENDPOINT", c.Telemetry.OTLPEndpoint)
	} else {
		validateOptionalURL(errs, "OTEL_EXPORTER_OTLP_ENDPOINT", c.Telemetry.OTLPEndpoint, "http", "https")
	}
	if c.Telemetry.SampleRatio < 0 || c.Telemetry.SampleRatio > 1 {
		*errs = append(*errs, fmt.Errorf("OTEL_SAMPLE_RATIO must be between 0 and 1"))
	}
	if strings.TrimSpace(c.Telemetry.SamplerArg) != "" {
		ratio, err := strconv.ParseFloat(strings.TrimSpace(c.Telemetry.SamplerArg), 64)
		if err != nil || ratio < 0 || ratio > 1 {
			*errs = append(*errs, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be a number between 0 and 1"))
		}
	}
}

func validateRateLimit(errs *[]error, prefix string, cfg httpratelimit.Config) {
	if cfg.RequestsPerSecond < 0 {
		*errs = append(*errs, fmt.Errorf("%s_RATE_LIMIT_RPS must not be negative", prefix))
	}
	if cfg.Burst < 0 {
		*errs = append(*errs, fmt.Errorf("%s_RATE_LIMIT_BURST must not be negative", prefix))
	}
}

func validateOptionalHostPort(errs *[]error, name string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	if !strings.Contains(trimmed, "://") {
		if strings.Contains(trimmed, " ") || !strings.Contains(trimmed, ":") {
			*errs = append(*errs, fmt.Errorf("%s must be a valid URL or host:port", name))
		}
		return
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		if parsed.Host == "" {
			*errs = append(*errs, fmt.Errorf("%s must be a valid URL or host:port", name))
		}
		return
	}
	*errs = append(*errs, fmt.Errorf("%s must be a valid URL or host:port", name))
}

func (c Config) validateAPI(errs *[]error) {
	requireValue(errs, "AUTH_JWT_SIGNING_KEY", c.Auth.JWTSigningKey)
	requireValue(errs, "AUTH_JWT_ISSUER", c.Auth.JWTIssuer)
	c.validateAppleAuth(errs)
	if c.Auth.OAuthATTL <= 0 {
		*errs = append(*errs, fmt.Errorf("AUTH_OAUTH_ACCESS_TOKEN_TTL must be positive"))
	}
	if c.Auth.OAuthRTTL <= 0 {
		*errs = append(*errs, fmt.Errorf("AUTH_OAUTH_REFRESH_TOKEN_TTL must be positive"))
	}
	if c.Auth.OAuthCookieKey != "" {
		requireValue(errs, "API_PUBLIC_BASE_URL", c.APIPublicBaseURL)
		validateURL(errs, "API_PUBLIC_BASE_URL", c.APIPublicBaseURL, "http", "https")
		validateOptionalURL(errs, "WEB_BASE_URL", c.WebBaseURL, "http", "https")
	}
	if strings.TrimSpace(c.Auth.PasskeyRPID) != "" {
		requireValue(errs, "AUTH_PASSKEY_RP_ORIGINS", c.Auth.PasskeyRPOrigins)
		validateOrigins(errs, "AUTH_PASSKEY_RP_ORIGINS", c.Auth.PasskeyRPOrigins)
		if c.Environment == EnvProduction && isLocalHost(c.Auth.PasskeyRPID) {
			*errs = append(*errs, fmt.Errorf("AUTH_PASSKEY_RP_ID must not be localhost in production"))
		}
	}
}

func (c Config) validateConsumer(errs *[]error) {
	requireURL(errs, "SHORTCUT_BASE_URL", c.Shortcut.BaseURL)
	requireURL(errs, "SHORTCUT_DOCS_BASE_URL", c.Shortcut.DocsBaseURL)
	requireURL(errs, "SHORTCUT_AD_BASE_URL", c.Shortcut.AdBaseURL)
	requireValue(errs, "SHORTCUT_USER_AGENT", c.Shortcut.UserAgent)
	requireURL(errs, "SHORTCUT_SITEMAP_BASE_URL", c.Shortcut.SitemapBase)
	requireURL(errs, "FRONTDOOR_BASE_URL", c.Frontdoor.BaseURL)
	requireValue(errs, "FRONTDOOR_USER_AGENT", c.Frontdoor.UserAgent)
	requireValue(errs, "FRONTDOOR_COOKIE", c.Frontdoor.Cookie)
	requireURL(errs, "FRONTDOOR_SITEMAP_BASE_URL", c.Frontdoor.SitemapBase)
	requireValue(errs, "OPENROUTER_API_KEY", c.OpenRouter.APIKey)
}

func (c Config) validateAppleAuth(errs *[]error) {
	requireValue(errs, "AUTH_APPLE_BUNDLE_ID", c.Auth.Apple.BundleID)
	requireValue(errs, "AUTH_APPLE_TEAM_ID", c.Auth.Apple.TeamID)
	requireValue(errs, "AUTH_APPLE_PRIVATE_KEY_ID", c.Auth.Apple.PrivateKeyID)
	requireValue(errs, "AUTH_APPLE_PRIVATE_KEY", c.Auth.Apple.PrivateKey)
	if strings.TrimSpace(c.Auth.Apple.PrivateKey) != "" {
		block, _ := pem.Decode([]byte(c.Auth.Apple.PrivateKey))
		if block == nil {
			*errs = append(*errs, fmt.Errorf("AUTH_APPLE_PRIVATE_KEY must be PEM encoded"))
		}
	}
	if c.Auth.Apple.WebServiceID != "" {
		validateOptionalURL(errs, "AUTH_APPLE_WEB_REDIRECT_URI", c.Auth.Apple.WebRedirectURI, "http", "https")
	}
}

func requireURL(errs *[]error, name string, value string) {
	requireValue(errs, name, value)
	validateURL(errs, name, value, "http", "https")
}

func requireValue(errs *[]error, name string, value string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", name))
	}
}

func validateOptionalURL(errs *[]error, name string, value string, schemes ...string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	validateURL(errs, name, value, schemes...)
}

func validateURL(errs *[]error, name string, value string, schemes ...string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		*errs = append(*errs, fmt.Errorf("%s must be a valid URL", name))
		return
	}
	if len(schemes) == 0 {
		return
	}
	if slices.Contains(schemes, parsed.Scheme) {
		return
	}
	*errs = append(*errs, fmt.Errorf("%s must use one of these URL schemes: %s", name, strings.Join(schemes, ", ")))
}

func validateOrigins(errs *[]error, name string, value string) {
	parts := strings.SplitSeq(value, ",")
	for part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		validateURL(errs, name, origin, "http", "https")
	}
}

func validateCORSOrigins(errs *[]error, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	validateOrigins(errs, "CORS_ALLOWED_ORIGINS", value)
}

func isValidLogLevel(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug", "info", "warn", "warning", "error":
		return true
	default:
		return false
	}
}

func isLocalHost(value string) bool {
	host := strings.TrimSpace(strings.ToLower(value))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
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
	cfg := raw.toConfig()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// FromEnv parses configuration from the current process environment.
func FromEnv() (Config, error) {
	return FromEnvMap(CurrentEnv())
}

// Load loads .env.local and .env files then parses configuration from the
// resulting environment. Use FromEnvMap in tests instead.
func Load() (Config, error) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) != string(EnvProduction) {
		_ = godotenv.Load(".env.local", ".env")
	}
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
