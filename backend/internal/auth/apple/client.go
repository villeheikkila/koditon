package apple

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	tokenURL          = "https://appleid.apple.com/auth/token"
	jwksURL           = "https://appleid.apple.com/auth/keys"
	issuer            = "https://appleid.apple.com"
	clientSecretExp   = 180 * 24 * time.Hour // 6 months
	jwkRefreshMinutes = 10
)

type Client struct {
	httpClient       *http.Client
	bundleID         string
	teamID           string
	privateKeyID     string
	privateKey       jwk.Key
	jwkCache         *jwk.Cache
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
	if err := privateKey.Set(jwk.AlgorithmKey, jwa.ES256); err != nil {
		return nil, fmt.Errorf("set algorithm: %w", err)
	}
	jwkCache := jwk.NewCache(ctx)
	if err := jwkCache.Register(jwksURL, jwk.WithMinRefreshInterval(jwkRefreshMinutes*time.Minute)); err != nil {
		return nil, fmt.Errorf("register jwks cache: %w", err)
	}
	refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := jwkCache.Refresh(refreshCtx, jwksURL); err != nil {
		return nil, fmt.Errorf("initial jwks fetch: %w", err)
	}
	client := &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		bundleID:     cfg.BundleID,
		teamID:       cfg.TeamID,
		privateKeyID: cfg.PrivateKeyID,
		privateKey:   privateKey,
		jwkCache:     jwkCache,
	}
	client.clientSecretFunc = client.generateClientSecret
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
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, c.privateKey))
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
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%w: %s", ErrTokenExchange, errResp.Error)
		}
		return nil, fmt.Errorf("%w: status %d", ErrTokenExchange, resp.StatusCode)
	}
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &tokenResp, nil
}

func (c *Client) VerifyIDToken(ctx context.Context, idToken string, nonce string) (*IdentityToken, error) {
	keySet, err := c.jwkCache.Get(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("get apple jwks: %w", err)
	}
	parsed, err := jwt.Parse([]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(c.bundleID),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims := parsed.PrivateClaims()
	identity := &IdentityToken{
		Issuer:    parsed.Issuer(),
		Subject:   parsed.Subject(),
		Audience:  c.bundleID,
		IssuedAt:  parsed.IssuedAt().Unix(),
		ExpiresAt: parsed.Expiration().Unix(),
	}
	if v, ok := claims["nonce"].(string); ok {
		identity.Nonce = v
	}
	if v, ok := claims["nonce_supported"].(bool); ok {
		identity.NonceSupported = v
	}
	if v, ok := claims["email"].(string); ok {
		identity.Email = v
	}
	if v, ok := claims["email_verified"].(string); ok {
		identity.EmailVerified = v
	}
	if v, ok := claims["is_private_email"].(string); ok {
		identity.IsPrivateEmail = v
	}
	if v, ok := claims["real_user_status"].(float64); ok {
		identity.RealUserStatus = int(v)
	}
	if v, ok := claims["auth_time"].(float64); ok {
		identity.AuthTime = int64(v)
	}
	if v, ok := claims["transfer_sub"].(string); ok {
		identity.TransferSub = v
	}
	if nonce != "" {
		hashedNonce := sha256.Sum256([]byte(nonce))
		hashedNonceHex := hex.EncodeToString(hashedNonce[:])
		if identity.Nonce != hashedNonceHex {
			return nil, ErrInvalidNonce
		}
	}
	return identity, nil
}

func (c *Client) BundleID() string {
	return c.bundleID
}

func Issuer() string {
	return issuer
}
