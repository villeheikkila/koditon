package auth

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// ScopedAuthMiddlewareFactory returns a factory that creates per-route huma
// middlewares enforcing both token validity and the given required scopes.
func ScopedAuthMiddlewareFactory(api huma.API, svc *Service) func(requiredScopes []string) func(huma.Context, func(huma.Context)) {
	return func(requiredScopes []string) func(huma.Context, func(huma.Context)) {
		return func(ctx huma.Context, next func(huma.Context)) {
			raw := ctx.Header("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "Bearer "))
			if token == "" {
				_ = huma.WriteErr(api, ctx, 401, "missing bearer token")
				return
			}
			claims, err := svc.VerifyAccessToken(ctx.Context(), token)
			if err != nil {
				_ = huma.WriteErr(api, ctx, 401, "invalid bearer token")
				return
			}
			if len(requiredScopes) > 0 {
				allowed := LimitScopes(ScopesForRoles(claims.Roles), claims.Scopes)
				if !HasScopes(allowed, requiredScopes) {
					_ = huma.WriteErr(api, ctx, 403, "insufficient scope")
					return
				}
			}
			ctx = huma.WithContext(ctx, WithClaims(ctx.Context(), claims))
			next(ctx)
		}
	}
}
