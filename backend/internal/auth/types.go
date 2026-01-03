package auth

import (
	"koditon-go/internal/auth/db"
)

type AuthProvider = db.AuthAuthProvider

const (
	AuthProviderApple     = db.AuthAuthProviderApple
	AuthProviderAnonymous = db.AuthAuthProviderAnonymous
)

type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  int64
	RefreshToken          string
	RefreshTokenExpiresAt int64
}

type UserInfo struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}
