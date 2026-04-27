package auth

import "errors"

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenRevoked        = errors.New("token revoked")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionRevoked      = errors.New("session revoked")
	ErrSessionExpired      = errors.New("session expired")
	ErrUserNotFound        = errors.New("user not found")
	ErrIdentityNotFound    = errors.New("identity not found")
	ErrMissingIDToken      = errors.New("missing id token")
	ErrTokenReuse          = errors.New("refresh token reuse detected")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrSessionNotOwned     = errors.New("session not owned by user")
	ErrPasskeyConfig       = errors.New("passkey not configured")
	ErrPasskeyChallenge    = errors.New("invalid or expired passkey challenge")
	ErrPasskeyNotFound     = errors.New("passkey not found")
	ErrPasskeyLast         = errors.New("cannot delete last passkey")
	ErrOAuthInvalidGrant   = errors.New("oauth invalid grant")
	ErrOAuthInvalidRequest = errors.New("oauth invalid request")
	ErrOAuthPending        = errors.New("oauth authorization pending")
	ErrOAuthAccessDenied   = errors.New("oauth access denied")
	ErrOAuthExpiredToken   = errors.New("oauth expired token")
	ErrInvalidScope        = errors.New("invalid scope")
)
