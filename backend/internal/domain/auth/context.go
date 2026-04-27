package auth

import (
	"context"

	"koditon/internal/platform/util"
)

type contextKey string

const (
	ContextKeyUserID      contextKey = "auth:user_id"
	ContextKeyUserIDInt64 contextKey = "auth:user_id_int64"
	ContextKeySessionID   contextKey = "auth:session_id"
	ContextKeyClaims      contextKey = "auth:claims"
)

func WithClaims(ctx context.Context, claims *AccessTokenClaims) context.Context {
	if claims == nil {
		return ctx
	}
	newCtx := context.WithValue(ctx, ContextKeyClaims, claims)
	newCtx = context.WithValue(newCtx, ContextKeyUserID, util.EncodeUUIDBase62(claims.UserID))
	newCtx = context.WithValue(newCtx, ContextKeyUserIDInt64, claims.UserIDInt64)
	newCtx = context.WithValue(newCtx, ContextKeySessionID, util.EncodeUUIDBase62(claims.SessionID))
	return newCtx
}

func GetClaimsFromContext(ctx context.Context) *AccessTokenClaims {
	claims, ok := ctx.Value(ContextKeyClaims).(*AccessTokenClaims)
	if !ok {
		return nil
	}
	return claims
}
