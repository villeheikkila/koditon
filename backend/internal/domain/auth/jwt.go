package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"koditon-go/internal/platform/util"
)

type JWTConfig struct {
	PrivateKey  string
	Issuer      string
	UIDHashSalt string
}

type JWTService struct {
	privateKey   jwk.Key
	publicKey    jwk.Key
	publicKeySet jwk.Set
	issuer       string
	uidHasher    *util.IDHasher
	mu           sync.RWMutex
	access       map[string]accessTokenCacheEntry
}

func NewJWTService(cfg JWTConfig) (*JWTService, error) {
	uidHasher, err := util.NewIDHasher(cfg.UIDHashSalt)
	if err != nil {
		return nil, fmt.Errorf("create uid hasher: %w", err)
	}
	var ecKey *ecdsa.PrivateKey
	if cfg.PrivateKey != "" {
		ecKey, err = parseECPrivateKeyPEM(cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parse jwt private key: %w", err)
		}
	} else {
		ecKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate ephemeral ec key: %w", err)
		}
	}
	privJWK, err := jwk.Import(ecKey)
	if err != nil {
		return nil, fmt.Errorf("create private jwk: %w", err)
	}
	if err := privJWK.Set(jwk.AlgorithmKey, jwa.ES256()); err != nil {
		return nil, fmt.Errorf("set private key algorithm: %w", err)
	}
	thumb, err := privJWK.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("compute key thumbprint: %w", err)
	}
	kid := fmt.Sprintf("%x", thumb[:8])
	if err := privJWK.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, fmt.Errorf("set kid: %w", err)
	}
	pubJWK, err := privJWK.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("derive public jwk: %w", err)
	}
	pubSet := jwk.NewSet()
	if err := pubSet.AddKey(pubJWK); err != nil {
		return nil, fmt.Errorf("add public key to set: %w", err)
	}
	return &JWTService{
		privateKey:   privJWK,
		publicKey:    pubJWK,
		publicKeySet: pubSet,
		issuer:       cfg.Issuer,
		uidHasher:    uidHasher,
		access:       make(map[string]accessTokenCacheEntry),
	}, nil
}

func parseECPrivateKeyPEM(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		ecKey, ecErr := x509.ParseECPrivateKey(block.Bytes)
		if ecErr != nil {
			return nil, fmt.Errorf("parse private key (pkcs8: %w, ec: %v)", err, ecErr)
		}
		return ecKey, nil
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an EC private key")
	}
	return ecKey, nil
}

func (s *JWTService) SignAccessToken(claims AccessTokenClaims) (string, error) {
	token, err := claims.ToJWT()
	if err != nil {
		return "", fmt.Errorf("build access token: %w", err)
	}
	if err := token.Set(jwt.IssuerKey, s.issuer); err != nil {
		return "", fmt.Errorf("set issuer: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), s.privateKey))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return string(signed), nil
}

func (s *JWTService) VerifyAccessToken(_ context.Context, tokenString string) (*AccessTokenClaims, error) {
	if claims, ok := s.getAccessFromCache(tokenString); ok {
		return claims, nil
	}
	token, err := jwt.Parse([]byte(tokenString),
		jwt.WithKey(jwa.ES256(), s.publicKey),
		jwt.WithValidate(true),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil {
		if isExpiredTokenError(err) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims, err := ParseAccessTokenClaims(token)
	if err != nil {
		return nil, err
	}
	userID, err := s.uidHasher.DecodeInt64(claims.UserIDHash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims.UserIDInt64 = userID
	s.setAccessCache(tokenString, claims)
	return claims, nil
}

func (s *JWTService) Issuer() string {
	return s.issuer
}

func (s *JWTService) PublicKeySet() jwk.Set {
	return s.publicKeySet
}

func isExpiredTokenError(err error) bool {
	return errors.Is(err, jwt.TokenExpiredError())
}

type accessTokenCacheEntry struct {
	claims    *AccessTokenClaims
	expiresAt time.Time
}

func (s *JWTService) getAccessFromCache(tokenString string) (*AccessTokenClaims, bool) {
	now := time.Now()
	s.mu.RLock()
	entry, ok := s.access[tokenString]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if entry.expiresAt.IsZero() || now.After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.access, tokenString)
		s.mu.Unlock()
		return nil, false
	}
	return entry.claims, true
}

func (s *JWTService) setAccessCache(tokenString string, claims *AccessTokenClaims) {
	if claims == nil || claims.ExpiresAt.IsZero() {
		return
	}
	if time.Now().After(claims.ExpiresAt) {
		return
	}
	s.mu.Lock()
	s.access[tokenString] = accessTokenCacheEntry{
		claims:    claims,
		expiresAt: claims.ExpiresAt,
	}
	s.mu.Unlock()
}
