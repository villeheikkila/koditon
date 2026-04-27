package authz

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"koditon/internal/domain/auth"
)

const AuthTypeBearer = "bearer"

type VerifyError struct {
	Status  int
	Message string
	Cause   error
}

type VerifyResult struct {
	Context  context.Context
	AuthType string
}

// VerifyAuthorization verifies the Authorization header and required scopes,
// returning a context containing authenticated claims on success.
func VerifyAuthorization(ctx context.Context, authService *auth.Service, authorizationHeader string, requiredScopes []string, requiredAudience string) (*VerifyResult, *VerifyError) {
	if token := extractTokenByScheme(authorizationHeader, "Bearer"); token != "" {
		claims, err := authService.VerifyAccessToken(ctx, token)
		if err != nil {
			return nil, mapAccessTokenError(err)
		}
		if err := validateAudience(claims, requiredAudience); err != nil {
			return nil, err
		}
		if err := validateScopes(claims, requiredScopes); err != nil {
			return nil, err
		}
		return &VerifyResult{Context: auth.WithClaims(ctx, claims), AuthType: AuthTypeBearer}, nil
	}
	return nil, &VerifyError{Status: http.StatusUnauthorized, Message: "missing authorization token"}
}

func validateAudience(claims *auth.AccessTokenClaims, requiredAudience string) *VerifyError {
	requiredAudience = strings.TrimSpace(requiredAudience)
	if requiredAudience == "" {
		return nil
	}
	if claims == nil || strings.TrimSpace(claims.Audience) != requiredAudience {
		return &VerifyError{Status: http.StatusUnauthorized, Message: "invalid token audience"}
	}
	return nil
}

func validateScopes(claims *auth.AccessTokenClaims, requiredScopes []string) *VerifyError {
	if len(requiredScopes) == 0 {
		return nil
	}
	allowedScopes := auth.LimitScopes(auth.ScopesForRoles(claims.Roles), claims.Scopes)
	if auth.HasScopes(allowedScopes, requiredScopes) {
		return nil
	}
	return &VerifyError{Status: http.StatusForbidden, Message: "insufficient scope"}
}

func mapAccessTokenError(err error) *VerifyError {
	status := http.StatusUnauthorized
	msg := "invalid token"
	switch {
	case errors.Is(err, auth.ErrTokenExpired):
		msg = "token expired"
	case errors.Is(err, auth.ErrSessionRevoked):
		msg = "session revoked"
	case errors.Is(err, auth.ErrTokenRevoked):
		msg = "token revoked"
	case errors.Is(err, auth.ErrInvalidToken):
		msg = "invalid token"
	}
	return &VerifyError{Status: status, Message: msg, Cause: err}
}

func extractTokenByScheme(header, scheme string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], scheme) {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
