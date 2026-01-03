package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

func (h *Handlers) RegisterRoutes(api huma.API, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Post(api, "/auth/anonymous", h.SignInAnonymous, func(op *huma.Operation) {
		op.OperationID = "auth-sign-in-anonymous"
		op.Summary = "Sign in anonymously"
		op.Description = "Create an anonymous user and obtain access and refresh tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/apple", h.SignInWithApple, func(op *huma.Operation) {
		op.OperationID = "auth-sign-in-with-apple"
		op.Summary = "Sign in with Apple"
		op.Description = "Exchange an Apple authorization code for access and refresh tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/refresh", h.RefreshTokens, func(op *huma.Operation) {
		op.OperationID = "auth-refresh-tokens"
		op.Summary = "Refresh tokens"
		op.Description = "Exchange a refresh token for new access and refresh tokens"
		op.Tags = []string{"Authentication"}
	})
	huma.Post(api, "/auth/sign-out", h.SignOut, func(op *huma.Operation) {
		op.OperationID = "auth-sign-out"
		op.Summary = "Sign out"
		op.Description = "Revoke the current session"
		op.Tags = []string{"Authentication"}
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		op.Middlewares = huma.Middlewares{authMiddleware}
	})
	huma.Get(api, "/auth/me", h.GetCurrentUserInfo, func(op *huma.Operation) {
		op.OperationID = "auth-get-current-user"
		op.Summary = "Get current user info"
		op.Description = "Returns information about the authenticated user including feature flags"
		op.Tags = []string{"Authentication"}
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		op.Middlewares = huma.Middlewares{authMiddleware}
	})
}

func RegisterSecurityScheme(config *huma.Config) {
	if config.Components.SecuritySchemes == nil {
		config.Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
	}
	config.Components.SecuritySchemes["bearer"] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT access token obtained from sign-in",
	}
}

func AuthMiddlewareFactory(api huma.API, service *Service) func(huma.Context, func(huma.Context)) {
	return NewAuthMiddleware(api, service)
}

func RequireAuth(api huma.API, service *Service) huma.Middlewares {
	return huma.Middlewares{NewAuthMiddleware(api, service)}
}

func ProtectedRoute(api huma.API, service *Service) func(*huma.Operation) {
	middleware := NewAuthMiddleware(api, service)
	return func(op *huma.Operation) {
		op.Security = []map[string][]string{
			{"bearer": {}},
		}
		op.Middlewares = append(op.Middlewares, middleware)
	}
}

func GetCurrentUser(ctx context.Context) *UserInfo {
	claims := GetClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}
	return &UserInfo{
		UserID:    claims.UserID.String(),
		SessionID: claims.SessionID.String(),
	}
}
