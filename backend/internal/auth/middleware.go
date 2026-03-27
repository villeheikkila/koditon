package auth

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// AuthMiddlewareFactory returns a huma middleware that validates Bearer tokens.
func AuthMiddlewareFactory(api huma.API, svc *Service) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		raw := ctx.Header("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "Bearer "))
		if token == "" {
			huma.WriteErr(api, ctx, 401, "missing bearer token")
			return
		}
		claims, err := svc.VerifyAccessToken(ctx.Context(), token)
		if err != nil {
			huma.WriteErr(api, ctx, 401, "invalid bearer token")
			return
		}
		newCtx := WithClaims(ctx.Context(), claims)
		ctx = huma.WithContext(ctx, newCtx)
		next(ctx)
	}
}
