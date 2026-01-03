package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	ClaimSessionID       = "sid"
	ClaimTokenType       = "typ"
	ClaimTokenCounter    = "cnt"
	ClaimAppleRefresh    = "art"
	ClaimAppleRefreshExp = "are"
	ClaimRoles           = "roles"
	ClaimFeatureFlags    = "flags"
	TokenTypeAccess      = "access"
	TokenTypeRefresh     = "refresh"
	AccessTokenExpiry    = 15 * time.Minute
	RefreshTokenExpiry   = 365 * 24 * time.Hour
)

type AccessTokenClaims struct {
	UserID       uuid.UUID
	SessionID    uuid.UUID
	Roles        []string
	FeatureFlags []string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

func (c AccessTokenClaims) ToJWT() (jwt.Token, error) {
	token := jwt.New()
	if err := token.Set(jwt.SubjectKey, c.UserID.String()); err != nil {
		return nil, err
	}
	if err := token.Set(ClaimSessionID, c.SessionID.String()); err != nil {
		return nil, err
	}
	if err := token.Set(ClaimTokenType, TokenTypeAccess); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuedAtKey, c.IssuedAt); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.ExpirationKey, c.ExpiresAt); err != nil {
		return nil, err
	}
	if len(c.Roles) > 0 {
		if err := token.Set(ClaimRoles, c.Roles); err != nil {
			return nil, err
		}
	}
	if len(c.FeatureFlags) > 0 {
		if err := token.Set(ClaimFeatureFlags, c.FeatureFlags); err != nil {
			return nil, err
		}
	}
	return token, nil
}

func ParseAccessTokenClaims(token jwt.Token) (*AccessTokenClaims, error) {
	sub := token.Subject()
	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, ErrInvalidToken
	}
	sidVal, ok := token.Get(ClaimSessionID)
	if !ok {
		return nil, ErrInvalidToken
	}
	sidStr, ok := sidVal.(string)
	if !ok {
		return nil, ErrInvalidToken
	}
	sessionID, err := uuid.Parse(sidStr)
	if err != nil {
		return nil, ErrInvalidToken
	}
	typVal, ok := token.Get(ClaimTokenType)
	if !ok || typVal != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	claims := &AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		IssuedAt:  token.IssuedAt(),
		ExpiresAt: token.Expiration(),
	}
	if rolesVal, ok := token.Get(ClaimRoles); ok {
		if rolesSlice, ok := rolesVal.([]any); ok {
			for _, r := range rolesSlice {
				if rs, ok := r.(string); ok {
					claims.Roles = append(claims.Roles, rs)
				}
			}
		}
	}
	if flagsVal, ok := token.Get(ClaimFeatureFlags); ok {
		if flagsSlice, ok := flagsVal.([]any); ok {
			for _, f := range flagsSlice {
				if fs, ok := f.(string); ok {
					claims.FeatureFlags = append(claims.FeatureFlags, fs)
				}
			}
		}
	}
	return claims, nil
}

type RefreshTokenClaims struct {
	SessionID         uuid.UUID
	Counter           int64
	IssuedAt          time.Time
	ExpiresAt         time.Time
	AppleRefreshToken string
	AppleRefreshExp   int64
}

func (c RefreshTokenClaims) ToJWT() (jwt.Token, error) {
	token := jwt.New()
	if err := token.Set(jwt.SubjectKey, c.SessionID.String()); err != nil {
		return nil, err
	}
	if err := token.Set(ClaimTokenType, TokenTypeRefresh); err != nil {
		return nil, err
	}
	if err := token.Set(ClaimTokenCounter, c.Counter); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuedAtKey, c.IssuedAt); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.ExpirationKey, c.ExpiresAt); err != nil {
		return nil, err
	}
	if c.AppleRefreshToken != "" {
		if err := token.Set(ClaimAppleRefresh, c.AppleRefreshToken); err != nil {
			return nil, err
		}
		if err := token.Set(ClaimAppleRefreshExp, c.AppleRefreshExp); err != nil {
			return nil, err
		}
	}
	return token, nil
}

func ParseRefreshTokenClaims(token jwt.Token) (*RefreshTokenClaims, error) {
	sub := token.Subject()
	sessionID, err := uuid.Parse(sub)
	if err != nil {
		return nil, ErrInvalidToken
	}
	typVal, ok := token.Get(ClaimTokenType)
	if !ok || typVal != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	cntVal, ok := token.Get(ClaimTokenCounter)
	if !ok {
		return nil, ErrInvalidToken
	}
	var counter int64
	switch v := cntVal.(type) {
	case float64:
		counter = int64(v)
	case int64:
		counter = v
	case int:
		counter = int64(v)
	default:
		return nil, ErrInvalidToken
	}
	claims := &RefreshTokenClaims{
		SessionID: sessionID,
		Counter:   counter,
		IssuedAt:  token.IssuedAt(),
		ExpiresAt: token.Expiration(),
	}
	if artVal, ok := token.Get(ClaimAppleRefresh); ok {
		if art, ok := artVal.(string); ok {
			claims.AppleRefreshToken = art
		}
	}
	if areVal, ok := token.Get(ClaimAppleRefreshExp); ok {
		switch v := areVal.(type) {
		case float64:
			claims.AppleRefreshExp = int64(v)
		case int64:
			claims.AppleRefreshExp = v
		case int:
			claims.AppleRefreshExp = int64(v)
		}
	}
	return claims, nil
}
