package apple

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/redis/go-redis/v9"
)

const (
	tokenURL        = "https://appleid.apple.com/auth/token"
	jwksURL         = "https://appleid.apple.com/auth/keys"
	issuer          = "https://appleid.apple.com"
	clientSecretExp = 180 * 24 * time.Hour
	defaultRedisKey = "koditon:auth:apple:jwks"
	redisJWKSMaxAge = 24 * time.Hour
)

type Client struct {
	httpClient       *http.Client
	bundleID         string
	teamID           string
	privateKeyID     string
	privateKey       jwk.Key
	redisClient      *redis.Client
	redisKey         string
	keySetMu         sync.RWMutex
	keySet           jwk.Set
	clientSecretFunc func() (string, error)
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	privateKey, err := jwk.ParseKey([]byte(cfg.PrivateKey), jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	if err := privateKey.Set(jwk.KeyIDKey, cfg.PrivateKeyID); err != nil {
		return nil, fmt.Errorf("set key id: %w", err)
	}
	if err := privateKey.Set(jwk.AlgorithmKey, jwa.ES256()); err != nil {
		return nil, fmt.Errorf("set algorithm: %w", err)
	}
	client := &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		bundleID:     cfg.BundleID,
		teamID:       cfg.TeamID,
		privateKeyID: cfg.PrivateKeyID,
		privateKey:   privateKey,
		redisClient:  cfg.RedisClient,
		redisKey:     strings.TrimSpace(cfg.RedisKey),
	}
	if client.redisKey == "" {
		client.redisKey = defaultRedisKey
	}
	client.clientSecretFunc = client.generateClientSecret
	if _, err := client.loadKeySetFromRedis(ctx); err != nil {
		return nil, fmt.Errorf("load apple jwks from redis: %w", err)
	}
	return client, nil
}

func (c *Client) generateClientSecret() (string, error) {
	now := time.Now()
	token, err := jwt.NewBuilder().
		Issuer(c.teamID).
		Subject(c.bundleID).
		Audience([]string{issuer}).
		IssuedAt(now).
		Expiration(now.Add(clientSecretExp)).
		Build()
	if err != nil {
		return "", fmt.Errorf("build client secret jwt: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), c.privateKey))
	if err != nil {
		return "", fmt.Errorf("sign client secret: %w", err)
	}
	return string(signed), nil
}

func (c *Client) ExchangeAuthorizationCode(ctx context.Context, authCode string) (*TokenResponse, error) {
	clientSecret, err := c.clientSecretFunc()
	if err != nil {
		return nil, fmt.Errorf("generate client secret: %w", err)
	}
	form := url.Values{}
	form.Set("client_id", c.bundleID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", authCode)
	return c.sendTokenRequest(ctx, form)
}

func (c *Client) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	clientSecret, err := c.clientSecretFunc()
	if err != nil {
		return nil, fmt.Errorf("generate client secret: %w", err)
	}
	form := url.Values{}
	form.Set("client_id", c.bundleID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return c.sendTokenRequest(ctx, form)
}

func (c *Client) sendTokenRequest(ctx context.Context, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, &TokenExchangeError{
				StatusCode:       resp.StatusCode,
				ErrorCode:        errResp.Error,
				ErrorDescription: strings.TrimSpace(errResp.ErrorDescription),
			}
		}
		return nil, &TokenExchangeError{StatusCode: resp.StatusCode}
	}
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &tokenResp, nil
}

func (c *Client) VerifyIDToken(ctx context.Context, idToken string, nonce string) (*IdentityToken, error) {
	keySet, err := c.currentKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("get apple jwks: %w", err)
	}
	parsed, err := c.parseIDToken(idToken, keySet)
	if err != nil {
		refreshedKeySet, refreshErr := c.refreshKeySet(ctx)
		if refreshErr != nil {
			return nil, fmt.Errorf("%w: %v (refresh failed: %v)", ErrInvalidToken, err, refreshErr)
		}
		parsed, err = c.parseIDToken(idToken, refreshedKeySet)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}
	}
	issuerValue, ok := parsed.Issuer()
	if !ok {
		return nil, fmt.Errorf("%w: missing issuer", ErrInvalidToken)
	}
	subjectValue, ok := parsed.Subject()
	if !ok {
		return nil, fmt.Errorf("%w: missing subject", ErrInvalidToken)
	}
	issuedAtValue, ok := parsed.IssuedAt()
	if !ok {
		return nil, fmt.Errorf("%w: missing issued at", ErrInvalidToken)
	}
	expiresAtValue, ok := parsed.Expiration()
	if !ok {
		return nil, fmt.Errorf("%w: missing expiration", ErrInvalidToken)
	}
	identity := &IdentityToken{
		Issuer:    issuerValue,
		Subject:   subjectValue,
		Audience:  c.bundleID,
		IssuedAt:  issuedAtValue.Unix(),
		ExpiresAt: expiresAtValue.Unix(),
	}
	var nonceClaim string
	if err := parsed.Get("nonce", &nonceClaim); err == nil {
		identity.Nonce = nonceClaim
	}
	var nonceSupported bool
	if err := parsed.Get("nonce_supported", &nonceSupported); err == nil {
		identity.NonceSupported = nonceSupported
	}
	var emailClaim string
	if err := parsed.Get("email", &emailClaim); err == nil {
		identity.Email = emailClaim
	}
	var emailVerified string
	if err := parsed.Get("email_verified", &emailVerified); err == nil {
		identity.EmailVerified = emailVerified
	}
	var isPrivateEmail string
	if err := parsed.Get("is_private_email", &isPrivateEmail); err == nil {
		identity.IsPrivateEmail = isPrivateEmail
	}
	var realUserStatus int
	if err := parsed.Get("real_user_status", &realUserStatus); err == nil {
		identity.RealUserStatus = realUserStatus
	} else {
		var realUserStatusFloat float64
		if err := parsed.Get("real_user_status", &realUserStatusFloat); err == nil {
			identity.RealUserStatus = int(realUserStatusFloat)
		}
	}
	var authTime int64
	if err := parsed.Get("auth_time", &authTime); err == nil {
		identity.AuthTime = authTime
	} else {
		var authTimeFloat float64
		if err := parsed.Get("auth_time", &authTimeFloat); err == nil {
			identity.AuthTime = int64(authTimeFloat)
		}
	}
	var transferSub string
	if err := parsed.Get("transfer_sub", &transferSub); err == nil {
		identity.TransferSub = transferSub
	}
	if nonce != "" {
		if identity.Nonce != nonce && identity.Nonce != hashNonceHex(nonce) && identity.Nonce != hashNonceBase64URL(nonce) {
			return nil, ErrInvalidNonce
		}
	}
	return identity, nil
}

func (c *Client) parseIDToken(idToken string, keySet jwk.Set) (jwt.Token, error) {
	return jwt.Parse([]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(c.bundleID),
	)
}

func (c *Client) currentKeySet(ctx context.Context) (jwk.Set, error) {
	if keySet := c.cachedKeySet(); keySet != nil {
		return keySet, nil
	}
	if keySet, err := c.loadKeySetFromRedis(ctx); err != nil {
		return nil, err
	} else if keySet != nil {
		return keySet, nil
	}
	return c.refreshKeySet(ctx)
}

func (c *Client) cachedKeySet() jwk.Set {
	c.keySetMu.RLock()
	defer c.keySetMu.RUnlock()
	return c.keySet
}

func (c *Client) setCachedKeySet(keySet jwk.Set) {
	c.keySetMu.Lock()
	defer c.keySetMu.Unlock()
	c.keySet = keySet
}

func (c *Client) loadKeySetFromRedis(ctx context.Context) (jwk.Set, error) {
	if c.redisClient == nil {
		return nil, nil
	}
	payload, err := c.redisClient.Get(ctx, c.redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return c.parseAndCacheKeySet(payload)
}

func (c *Client) refreshKeySet(ctx context.Context) (jwk.Set, error) {
	payload, err := c.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}
	keySet, err := c.parseAndCacheKeySet(payload)
	if err != nil {
		return nil, err
	}
	if c.redisClient != nil {
		if err := c.redisClient.Set(ctx, c.redisKey, payload, redisJWKSMaxAge).Err(); err != nil {
			return nil, err
		}
	}
	return keySet, nil
}

func (c *Client) fetchJWKS(ctx context.Context) ([]byte, error) {
	refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(refreshCtx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %q: %w", jwksURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %q: status %d", jwksURL, resp.StatusCode)
	}
	return body, nil
}

func (c *Client) parseAndCacheKeySet(payload []byte) (jwk.Set, error) {
	keySet, err := jwk.Parse(payload)
	if err != nil {
		return nil, fmt.Errorf("parse apple jwks: %w", err)
	}
	c.setCachedKeySet(keySet)
	return keySet, nil
}

func hashNonceHex(nonce string) string {
	hashed := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(hashed[:])
}

func hashNonceBase64URL(nonce string) string {
	hashed := sha256.Sum256([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(hashed[:])
}

func (c *Client) BundleID() string {
	return c.bundleID
}

func Issuer() string {
	return issuer
}

func (c *Client) ExchangeAuthorizationCodeWeb(ctx context.Context, authCode, redirectURI string) (*TokenResponse, error) {
	clientSecret, err := c.clientSecretFunc()
	if err != nil {
		return nil, fmt.Errorf("generate client secret: %w", err)
	}
	form := url.Values{}
	form.Set("client_id", c.bundleID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", authCode)
	form.Set("redirect_uri", redirectURI)
	return c.sendTokenRequest(ctx, form)
}

