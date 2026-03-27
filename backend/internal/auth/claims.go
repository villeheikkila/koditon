package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"koditon-go/internal/util"
)

const (
	ClaimSessionID    = "sid"
	ClaimUserIDHash   = "uid"
	ClaimTokenType    = "typ"
	ClaimScope        = "scope"
	ClaimRoles        = "roles"
	ClaimFeatureFlags = "flags"
	ClaimAudience     = "aud"
	TokenTypeAccess   = "access"
	AccessTokenExpiry = 15 * time.Minute
)

type AccessTokenClaims struct {
	UserID       uuid.UUID
	UserIDHash   string
	UserIDInt64  int64
	SessionID    uuid.UUID
	Roles        []string
	FeatureFlags []string
	Scopes       []string
	Audience     string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

func (c AccessTokenClaims) ToJWT() (jwt.Token, error) {
	token := jwt.New()
	if err := token.Set(jwt.SubjectKey, util.EncodeUUIDBase62(c.UserID)); err != nil {
		return nil, err
	}
	if err := token.Set(ClaimSessionID, util.EncodeUUIDBase62(c.SessionID)); err != nil {
		return nil, err
	}
	if c.UserIDHash == "" {
		return nil, ErrInvalidToken
	}
	if err := token.Set(ClaimUserIDHash, c.UserIDHash); err != nil {
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
	if len(c.Scopes) > 0 {
		if err := token.Set(ClaimScope, joinScopes(c.Scopes)); err != nil {
			return nil, err
		}
	}
	if c.Audience != "" {
		if err := token.Set(ClaimAudience, c.Audience); err != nil {
			return nil, err
		}
	}
	return token, nil
}

func ParseAccessTokenClaims(token jwt.Token) (*AccessTokenClaims, error) {
	sub, ok := token.Subject()
	if !ok {
		return nil, ErrInvalidToken
	}
	userID, err := util.DecodeUUIDBase62(sub)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var sidStr string
	if err := token.Get(ClaimSessionID, &sidStr); err != nil || sidStr == "" {
		return nil, ErrInvalidToken
	}
	sessionID, err := util.DecodeUUIDBase62(sidStr)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var uidStr string
	if err := token.Get(ClaimUserIDHash, &uidStr); err != nil || uidStr == "" {
		return nil, ErrInvalidToken
	}
	var tokenType string
	if err := token.Get(ClaimTokenType, &tokenType); err != nil || tokenType != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	issuedAt, ok := token.IssuedAt()
	if !ok {
		return nil, ErrInvalidToken
	}
	expiresAt, ok := token.Expiration()
	if !ok {
		return nil, ErrInvalidToken
	}
	claims := &AccessTokenClaims{
		UserID:     userID,
		UserIDHash: uidStr,
		SessionID:  sessionID,
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
	}
	var rolesSlice []string
	if err := token.Get(ClaimRoles, &rolesSlice); err == nil {
		claims.Roles = append(claims.Roles, rolesSlice...)
	} else {
		var genericRoles []any
		if err := token.Get(ClaimRoles, &genericRoles); err == nil {
			for _, r := range genericRoles {
				if rs, ok := r.(string); ok {
					claims.Roles = append(claims.Roles, rs)
				}
			}
		}
	}
	var flagsSlice []string
	if err := token.Get(ClaimFeatureFlags, &flagsSlice); err == nil {
		claims.FeatureFlags = append(claims.FeatureFlags, flagsSlice...)
	} else {
		var genericFlags []any
		if err := token.Get(ClaimFeatureFlags, &genericFlags); err == nil {
			for _, f := range genericFlags {
				if fs, ok := f.(string); ok {
					claims.FeatureFlags = append(claims.FeatureFlags, fs)
				}
			}
		}
	}
	var scopeText string
	if err := token.Get(ClaimScope, &scopeText); err == nil {
		claims.Scopes = splitScopes(scopeText)
	}
	var audienceText string
	if err := token.Get(ClaimAudience, &audienceText); err == nil {
		claims.Audience = audienceText
	} else {
		var audienceList []string
		if err := token.Get(ClaimAudience, &audienceList); err == nil {
			if len(audienceList) > 0 {
				claims.Audience = audienceList[0]
			}
		} else {
			var genericAudience []any
			if err := token.Get(ClaimAudience, &genericAudience); err == nil {
				for _, item := range genericAudience {
					if text, ok := item.(string); ok && text != "" {
						claims.Audience = text
						break
					}
				}
			}
		}
	}
	return claims, nil
}

func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	out := scopes[0]
	for i := 1; i < len(scopes); i++ {
		out += " " + scopes[i]
	}
	return out
}

func splitScopes(scopeText string) []string {
	scopeText = strings.TrimSpace(scopeText)
	if scopeText == "" {
		return nil
	}
	parts := strings.Fields(scopeText)
	if len(parts) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		scopes = append(scopes, part)
	}
	return scopes
}
