package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type JWTConfig struct {
	SigningKey string
	Issuer     string
}

type JWTService struct {
	signingKey jwk.Key
	issuer     string
}

func NewJWTService(cfg JWTConfig) (*JWTService, error) {
	var key jwk.Key
	var err error
	if cfg.SigningKey != "" {
		keyBytes, decErr := base64.StdEncoding.DecodeString(cfg.SigningKey)
		if decErr != nil {
			return nil, fmt.Errorf("decode signing key: %w", decErr)
		}
		key, err = jwk.FromRaw(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("parse signing key: %w", err)
		}
	} else {
		rawKey := make([]byte, 32)
		if _, err := rand.Read(rawKey); err != nil {
			return nil, fmt.Errorf("generate signing key: %w", err)
		}
		key, err = jwk.FromRaw(rawKey)
		if err != nil {
			return nil, fmt.Errorf("create signing key: %w", err)
		}
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.HS256); err != nil {
		return nil, fmt.Errorf("set algorithm: %w", err)
	}
	return &JWTService{
		signingKey: key,
		issuer:     cfg.Issuer,
	}, nil
}

func (s *JWTService) SignAccessToken(claims AccessTokenClaims) (string, error) {
	token, err := claims.ToJWT()
	if err != nil {
		return "", fmt.Errorf("build access token: %w", err)
	}
	if err := token.Set(jwt.IssuerKey, s.issuer); err != nil {
		return "", fmt.Errorf("set issuer: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, s.signingKey))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return string(signed), nil
}

func (s *JWTService) SignRefreshToken(claims RefreshTokenClaims) (string, error) {
	token, err := claims.ToJWT()
	if err != nil {
		return "", fmt.Errorf("build refresh token: %w", err)
	}
	if err := token.Set(jwt.IssuerKey, s.issuer); err != nil {
		return "", fmt.Errorf("set issuer: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, s.signingKey))
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}
	return string(signed), nil
}

func (s *JWTService) VerifyAccessToken(_ context.Context, tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.Parse([]byte(tokenString),
		jwt.WithKey(jwa.HS256, s.signingKey),
		jwt.WithValidate(true),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil {
		if isExpiredTokenError(err) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return ParseAccessTokenClaims(token)
}

func (s *JWTService) VerifyRefreshToken(_ context.Context, tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.Parse([]byte(tokenString),
		jwt.WithKey(jwa.HS256, s.signingKey),
		jwt.WithValidate(true),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil {
		if isExpiredTokenError(err) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return ParseRefreshTokenClaims(token)
}

func (s *JWTService) Issuer() string {
	return s.issuer
}

func isExpiredTokenError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "exp not satisfied")
}
