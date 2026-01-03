package auth

import "errors"

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenRevoked     = errors.New("token revoked")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionRevoked   = errors.New("session revoked")
	ErrSessionExpired   = errors.New("session expired")
	ErrUserNotFound     = errors.New("user not found")
	ErrIdentityNotFound = errors.New("identity not found")
	ErrMissingIDToken   = errors.New("missing id token")
	ErrTokenReuse       = errors.New("refresh token reuse detected")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrSessionNotOwned  = errors.New("session not owned by user")
)
