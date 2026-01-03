package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type contextKey string

const (
	ContextKeyUserID    contextKey = "auth:user_id"
	ContextKeySessionID contextKey = "auth:session_id"
	ContextKeyClaims    contextKey = "auth:claims"
)

func NewAuthMiddleware(api huma.API, authService *Service) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		var requiredScopes []string
		isAuthRequired := false
		for _, opScheme := range ctx.Operation().Security {
			if scopes, ok := opScheme["bearer"]; ok {
				isAuthRequired = true
				requiredScopes = scopes
				break
			}
		}
		if !isAuthRequired {
			next(ctx)
			return
		}
		token := extractBearerToken(ctx)
		if token == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "missing authorization token")
			return
		}
		claims, err := authService.VerifyAccessToken(ctx.Context(), token)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid token"
			switch {
			case errors.Is(err, ErrTokenExpired):
				msg = "token expired"
			case errors.Is(err, ErrSessionRevoked):
				msg = "session revoked"
			case errors.Is(err, ErrInvalidToken):
				msg = "invalid token"
			}
			huma.WriteErr(api, ctx, status, msg)
			return
		}
		// store claims in context
		newCtx := context.WithValue(ctx.Context(), ContextKeyClaims, claims)
		newCtx = context.WithValue(newCtx, ContextKeyUserID, claims.UserID.String())
		newCtx = context.WithValue(newCtx, ContextKeySessionID, claims.SessionID.String())
		_ = requiredScopes // scopes can be used for role-based access control
		next(huma.WithContext(ctx, newCtx))
	}
}

func extractBearerToken(ctx huma.Context) string {
	header := ctx.Header("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func GetClaimsFromContext(ctx context.Context) *AccessTokenClaims {
	claims, ok := ctx.Value(ContextKeyClaims).(*AccessTokenClaims)
	if !ok {
		return nil
	}
	return claims
}

func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

func GetSessionIDFromContext(ctx context.Context) string {
	sessionID, ok := ctx.Value(ContextKeySessionID).(string)
	if !ok {
		return ""
	}
	return sessionID
}

func GetRoles(ctx context.Context) []string {
	claims := GetClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}
	return claims.Roles
}

func GetFeatureFlags(ctx context.Context) []string {
	claims := GetClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}
	return claims.FeatureFlags
}

func HasRole(ctx context.Context, role string) bool {
	for _, r := range GetRoles(ctx) {
		if r == role {
			return true
		}
	}
	return false
}

func HasFeatureFlag(ctx context.Context, flag string) bool {
	for _, f := range GetFeatureFlags(ctx) {
		if f == flag {
			return true
		}
	}
	return false
}
