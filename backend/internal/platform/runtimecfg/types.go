package runtimecfg

import "time"

type DatabaseConfig struct {
	URL               string
	MaxConns          int
	MinConns          int
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type AppleAuthConfig struct {
	BundleID       string
	PrivateKey     string
	PrivateKeyID   string
	TeamID         string
	WebServiceID   string
	WebRedirectURI string
}

type JWTAuthConfig struct {
	PrivateKey  string
	Issuer      string
	UIDHashSalt string
}

type PasskeyConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

type OAuthConfig struct {
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	CookieSigningKey string
}

type GeoIPConfig struct {
	MMDBPath string
}

type AuthConfig struct {
	Apple   AppleAuthConfig
	JWT     JWTAuthConfig
	Passkey *PasskeyConfig
	OAuth   *OAuthConfig
	GeoIP   *GeoIPConfig
}

type StorageConfig struct {
	Endpoint      string
	AccessKeyID   string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
	UseTLS        bool
}

type ClickHouseConfig struct {
	DSN             string
	Database        string
	RequestTimeout  time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
}

type HTTPConfig struct {
	APIBaseURL               string
	APIPublicBaseURL         string
	WebBaseURL               string
	AppStoreID               string
	EnableMCPOAuth           bool
	OAuthCookieSigningKey    string
	OpenAIAppsChallengeToken string
}

type ExternalProvidersConfig struct {
	BrandfetchClientID      string
	EAN13ImageSourceBaseURL string
	TavilyAPIKey            string
	LinkupAPIKey            string
}

type EmailConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

type OpenRouterConfig struct {
	APIKey          string
	GlobalTokensDay int
	GlobalTokensMin int
	Model           string
	ReserveTokens   int
}
