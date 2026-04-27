package auth

import "time"

const (
	defaultOAuthAccessTokenTTL  = 15 * time.Minute
	defaultOAuthRefreshTokenTTL = 30 * 24 * time.Hour
)

type RefreshReuseAction string

const (
	RefreshReuseActionRevokeSession RefreshReuseAction = "revoke_session"
)

type Policy struct {
	AppAccessTokenTTL    time.Duration
	OAuthAccessTokenTTL  time.Duration
	OAuthRefreshTokenTTL time.Duration
	OAuthRefreshOnReuse  RefreshReuseAction
}

type PolicyConfig struct {
	AppAccessTokenTTL    time.Duration
	OAuthAccessTokenTTL  time.Duration
	OAuthRefreshTokenTTL time.Duration
	OAuthRefreshOnReuse  RefreshReuseAction
}

func defaultPolicy() Policy {
	return Policy{
		AppAccessTokenTTL:    AccessTokenExpiry,
		OAuthAccessTokenTTL:  defaultOAuthAccessTokenTTL,
		OAuthRefreshTokenTTL: defaultOAuthRefreshTokenTTL,
		OAuthRefreshOnReuse:  RefreshReuseActionRevokeSession,
	}
}

func newPolicy(cfg PolicyConfig) Policy {
	policy := defaultPolicy()
	if cfg.AppAccessTokenTTL > 0 {
		policy.AppAccessTokenTTL = cfg.AppAccessTokenTTL
	}
	if cfg.OAuthAccessTokenTTL > 0 {
		policy.OAuthAccessTokenTTL = cfg.OAuthAccessTokenTTL
	}
	if cfg.OAuthRefreshTokenTTL > 0 {
		policy.OAuthRefreshTokenTTL = cfg.OAuthRefreshTokenTTL
	}
	if cfg.OAuthRefreshOnReuse != "" {
		policy.OAuthRefreshOnReuse = cfg.OAuthRefreshOnReuse
	}
	return policy
}
